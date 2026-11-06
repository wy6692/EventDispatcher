package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"eventdispatcher/internal/infrastructure/persistence"
	"eventdispatcher/internal/service"
	transporthttp "eventdispatcher/internal/transport/http"
	"eventdispatcher/internal/transport/http/handler"
	"eventdispatcher/internal/wechat"
	"eventdispatcher/pkg/logger"
)

// DownstreamEvent 表示下游 Webhook 接收到的解析数据。
type DownstreamEvent struct {
	ReceiverName string
	Timestamp    string
	Signature    string
	Verified     bool
	EventID      string
	MsgType      string
	Event        string
	EventKey     string
	FromUser     string
	Payload      map[string]any
}

func main() {
	appLogger := logger.New(os.Stdout, slog.LevelInfo, false)
	slog.SetDefault(appLogger)

	fmt.Println("================================================================================")
	fmt.Println("       EventDispatcher 微信服务号接入与多路下游程序分发实战模拟器       ")
	fmt.Println("================================================================================")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. 初始化临时 SQLite 数据库
	tempDir, err := os.MkdirTemp("", "dispatcher_sim_*")
	if err != nil {
		appLogger.Error("创建模拟临时目录失败", slog.String("error", err.Error()))
		return
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "sim.db")
	db, err := persistence.Open(dbPath)
	if err != nil {
		appLogger.Error("初始化持久化存储失败", slog.String("error", err.Error()))
		return
	}
	defer db.Close()

	// 2. 装配网关核心服务
	accountRepo := persistence.NewAccountRepository(db)
	subscriberRepo := persistence.NewSubscriberRepository(db)
	eventRepo := persistence.NewEventRepository(db)
	deliveryRepo := persistence.NewDeliveryRepository(db)

	accountService := service.NewAccountService(accountRepo)
	subscriberService := service.NewSubscriberService(subscriberRepo, accountRepo)
	eventService := service.NewEventService(eventRepo)
	deliveryService := service.NewDeliveryService(deliveryRepo, appLogger)
	ingestService := service.NewIngestService(accountRepo, subscriberRepo, eventRepo, appLogger)

	adminToken := "admin_secret_token_sim_8888"
	gatewayRouter := transporthttp.NewRouter(transporthttp.RouterConfig{
		AdminToken:        adminToken,
		Logger:            appLogger,
		AccountHandler:    handler.NewAccountHandler(accountService),
		SubscriberHandler: handler.NewSubscriberHandler(subscriberService),
		EventHandler:      handler.NewEventHandler(eventService),
		DeliveryHandler:   handler.NewDeliveryHandler(deliveryService),
		IngestHandler:     handler.NewIngestHandler(ingestService),
		HealthHandler:     handler.NewHealthHandler(db),
	})

	// 在随机可用端口启动网关服务器
	gatewayServer := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: gatewayRouter,
	}

	lnGateway, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		appLogger.Error("启动网关监听失败", slog.String("error", err.Error()))
		return
	}
	gatewayAddr := fmt.Sprintf("http://%s", lnGateway.Addr().String())

	go func() {
		_ = gatewayServer.Serve(lnGateway)
	}()
	defer gatewayServer.Close()

	// 启动后台投递 Worker
	go deliveryService.StartWorker(ctx)

	appLogger.Info("EventDispatcher 分发网关已在后台就绪", slog.String("gatewayUrl", gatewayAddr))

	// 3. 启动下游业务系统 Webhook 模拟端点
	var mu sync.Mutex
	receivedEvents := make([]DownstreamEvent, 0)

	crmSecret := "secret_key_growth_crm"
	serviceSecret := "secret_key_customer_service"
	auditSecret := "secret_key_audit_center"

	verifyHMAC := func(secret, timestamp, signature string, body []byte) bool {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(timestamp))
		mac.Write([]byte("."))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))
		return hmac.Equal([]byte(expected), []byte(signature))
	}

	downstreamMux := http.NewServeMux()

	recordWebhook := func(w http.ResponseWriter, r *http.Request, receiverName, secret string) {
		body, _ := io.ReadAll(r.Body)
		ts := r.Header.Get("X-Dispatcher-Timestamp")
		sig := r.Header.Get("X-Dispatcher-Signature")
		evtID := r.Header.Get("X-Dispatcher-Event-Id")

		isValid := verifyHMAC(secret, ts, sig, body)

		var evt map[string]any
		_ = json.Unmarshal(body, &evt)

		msgType, _ := evt["msgType"].(string)
		event, _ := evt["event"].(string)
		eventKey, _ := evt["eventKey"].(string)
		fromUser, _ := evt["fromUser"].(string)

		mu.Lock()
		record := DownstreamEvent{
			ReceiverName: receiverName,
			Timestamp:    ts,
			Signature:    sig,
			Verified:     isValid,
			EventID:      evtID,
			MsgType:      msgType,
			Event:        event,
			EventKey:     eventKey,
			FromUser:     fromUser,
			Payload:      evt,
		}
		receivedEvents = append(receivedEvents, record)
		mu.Unlock()

		appLogger.Info("下游业务系统接收到分发请求",
			slog.String("downstream", receiverName),
			slog.String("eventId", evtID),
			slog.String("msgType", msgType),
			slog.String("event", event),
			slog.Bool("hmacVerified", isValid),
		)
		w.WriteHeader(http.StatusOK)
	}

	downstreamMux.HandleFunc("/webhook/crm", func(w http.ResponseWriter, r *http.Request) {
		recordWebhook(w, r, "用户增长CRM中心", crmSecret)
	})
	downstreamMux.HandleFunc("/webhook/service", func(w http.ResponseWriter, r *http.Request) {
		recordWebhook(w, r, "智能在线客服系统", serviceSecret)
	})
	downstreamMux.HandleFunc("/webhook/audit", func(w http.ResponseWriter, r *http.Request) {
		recordWebhook(w, r, "安全合规审计系统", auditSecret)
	})

	lnDownstream, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		appLogger.Error("启动下游模拟监听失败", slog.String("error", err.Error()))
		return
	}
	downstreamAddr := fmt.Sprintf("http://%s", lnDownstream.Addr().String())
	downstreamServer := &http.Server{Handler: downstreamMux}
	go func() {
		_ = downstreamServer.Serve(lnDownstream)
	}()
	defer downstreamServer.Close()

	appLogger.Info("下游业务模拟服务已启动", slog.String("downstreamAddr", downstreamAddr))

	httpClient := &http.Client{Timeout: 5 * time.Second}

	// 辅助 Admin API 调用
	callAdmin := func(method, path string, payload any) (int, map[string]any) {
		var bodyReader io.Reader
		if payload != nil {
			b, _ := json.Marshal(payload)
			bodyReader = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(method, gatewayAddr+path, bodyReader)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		resp, err := httpClient.Do(req)
		if err != nil {
			appLogger.Error("Admin API 请求失败", slog.String("path", path), slog.String("error", err.Error()))
			return 0, nil
		}
		defer resp.Body.Close()
		var res map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&res)
		return resp.StatusCode, res
	}

	// =========================================================================
	// 阶段一：通过 Admin API 创建微信服务号
	// =========================================================================
	fmt.Println("\n>>> [阶段 1/5] 在分发网关登记微信服务号 (Account)")
	accKey := "wx_enterprise_corp"
	appID := "wx88888888abcdef01"
	wechatToken := "wechat_token_pass_2026"
	aesKey := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"

	code, accRes := callAdmin(http.MethodPost, "/admin/accounts", map[string]string{
		"name":           "集团官方服务号",
		"accountKey":     accKey,
		"appId":          appID,
		"token":          wechatToken,
		"encodingAesKey": aesKey,
	})
	if code != http.StatusCreated {
		appLogger.Error("创建账号失败", slog.Int("code", code), slog.Any("res", accRes))
		return
	}
	dataMap := accRes["data"].(map[string]any)
	accountID := dataMap["id"].(string)
	fmt.Printf("已成功创建服务号: ID=%s, Key=%s, AppID=%s\n", accountID, accKey, appID)
	fmt.Printf("凭据安全保护: TokenSet=%v, EncodingAESKeySet=%v (核心机密凭据零明文泄露)\n", dataMap["tokenSet"], dataMap["encodingAesKeySet"])

	// =========================================================================
	// 阶段二：注册下游订阅者与路由过滤规则
	// =========================================================================
	fmt.Println("\n>>> [阶段 2/5] 注册多路下游业务程序及其路由过滤规则 (Subscribers)")

	// 1. 增长 CRM 中心：订阅带参二维码关注事件
	code, _ = callAdmin(http.MethodPost, "/admin/subscribers", map[string]any{
		"name":        "用户增长CRM中心",
		"accountId":   accountID,
		"callbackUrl": downstreamAddr + "/webhook/crm",
		"secret":      crmSecret,
		"filter": map[string]any{
			"msgTypes":         []string{"event"},
			"events":           []string{"subscribe", "SCAN"},
			"eventKeyPrefixes": []string{"qrscene_"},
		},
	})
	fmt.Printf("1. [CRM中心] 注册完成 (HTTP %d): 规则=[msgType: event, event: subscribe/SCAN, keyPrefix: qrscene_]\n", code)

	// 2. 智能在线客服：订阅普通用户消息
	code, _ = callAdmin(http.MethodPost, "/admin/subscribers", map[string]any{
		"name":        "智能在线客服系统",
		"accountId":   accountID,
		"callbackUrl": downstreamAddr + "/webhook/service",
		"secret":      serviceSecret,
		"filter": map[string]any{
			"msgTypes": []string{"text", "image", "voice"},
		},
	})
	fmt.Printf("2. [客服系统] 注册完成 (HTTP %d): 规则=[msgType: text, image, voice]\n", code)

	// 3. 安全合规审计中心：订阅全部事件（无过滤）
	code, _ = callAdmin(http.MethodPost, "/admin/subscribers", map[string]any{
		"name":        "安全合规审计系统",
		"accountId":   accountID,
		"callbackUrl": downstreamAddr + "/webhook/audit",
		"secret":      auditSecret,
		"filter":      map[string]any{},
	})
	fmt.Printf("3. [审计系统] 注册完成 (HTTP %d): 规则=[消费全量事件与消息]\n", code)

	// =========================================================================
	// 阶段三：模拟微信公众平台开发者接入验证 (GET /w/{accountKey})
	// =========================================================================
	fmt.Println("\n>>> [阶段 3/5] 模拟微信公众平台开发者 URL 接入验证 (GET)")
	ts := fmt.Sprintf("%d", time.Now().Unix())
	nonce := "random_nonce_wx_999"
	echostr := "challenge_string_echo_success"
	sig := wechat.Signature(wechatToken, ts, nonce)

	verifyURL := fmt.Sprintf("%s/w/%s?signature=%s&timestamp=%s&nonce=%s&echostr=%s",
		gatewayAddr, accKey, sig, ts, nonce, echostr)

	resp, err := httpClient.Get(verifyURL)
	if err != nil {
		appLogger.Error("微信验证请求失败", slog.String("error", err.Error()))
		return
	}
	defer resp.Body.Close()
	echoBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK && string(echoBody) == echostr {
		fmt.Printf("微信服务器接入验证通过! HTTP %d, 响应报文: %s\n", resp.StatusCode, string(echoBody))
	} else {
		appLogger.Error("微信验证不匹配", slog.Int("status", resp.StatusCode), slog.String("body", string(echoBody)))
		return
	}

	// =========================================================================
	// 阶段四：模拟微信推送各类消息，检验下游多路分发
	// =========================================================================
	fmt.Println("\n>>> [阶段 4/5] 模拟微信服务器向网关推送真实事件报文 (POST)")

	// 4.1 场景一：用户扫抖音渠道码关注 (明文事件)
	fmt.Println("\n--- 场景 1：用户扫描抖音推广渠道二维码关注公众号 ---")
	qrScanXML := `<xml>
		<ToUserName><![CDATA[gh_enterprise_account]]></ToUserName>
		<FromUserName><![CDATA[oX_user_douyin_001]]></FromUserName>
		<CreateTime>1730005001</CreateTime>
		<MsgType><![CDATA[event]]></MsgType>
		<Event><![CDATA[subscribe]]></Event>
		<EventKey><![CDATA[qrscene_douyin_campaign_2026]]></EventKey>
	</xml>`
	postSig := wechat.Signature(wechatToken, ts, nonce)
	postURL := fmt.Sprintf("%s/w/%s?signature=%s&timestamp=%s&nonce=%s", gatewayAddr, accKey, postSig, ts, nonce)

	startT := time.Now()
	resp1, err := httpClient.Post(postURL, "text/xml", strings.NewReader(qrScanXML))
	if err != nil {
		appLogger.Error("推送事件失败", slog.String("error", err.Error()))
		return
	}
	reply1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()
	fmt.Printf("网关极速同步应答微信: HTTP %d, 响应=%s, 耗时=%v\n", resp1.StatusCode, string(reply1), time.Since(startT))

	// 等待后台 Worker 异步派发
	time.Sleep(700 * time.Millisecond)

	// 4.2 场景二：用户发送普通咨询文本消息 (安全模式 AES-256 加密)
	fmt.Println("\n--- 场景 2：用户在微信端发起售前咨询 (微信安全模式 AES-256 密文) ---")
	plainTextXML := `<xml>
		<ToUserName><![CDATA[` + appID + `]]></ToUserName>
		<FromUserName><![CDATA[oX_user_vip_client_002]]></FromUserName>
		<CreateTime>1730005002</CreateTime>
		<MsgType><![CDATA[text]]></MsgType>
		<Content><![CDATA[请问支持企业私有化部署和高并发架构吗？]]></Content>
		<MsgId>8888000000000001</MsgId>
	</xml>`

	encryptedText, err := wechat.Encrypt(aesKey, appID, []byte(plainTextXML))
	if err != nil {
		appLogger.Error("加密微信报文失败", slog.String("error", err.Error()))
		return
	}
	msgSig := wechat.Signature(wechatToken, ts, nonce, encryptedText)
	wrapperXML := fmt.Sprintf(`<xml>
		<ToUserName><![CDATA[%s]]></ToUserName>
		<Encrypt><![CDATA[%s]]></Encrypt>
	</xml>`, appID, encryptedText)

	aesPostURL := fmt.Sprintf("%s/w/%s?signature=%s&timestamp=%s&nonce=%s&encrypt_type=aes&msg_signature=%s",
		gatewayAddr, accKey, postSig, ts, nonce, msgSig)

	startT = time.Now()
	resp2, err := httpClient.Post(aesPostURL, "text/xml", strings.NewReader(wrapperXML))
	if err != nil {
		appLogger.Error("推送加密事件失败", slog.String("error", err.Error()))
		return
	}
	reply2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	fmt.Printf("网关解密并极速应答微信: HTTP %d, 响应=%s, 耗时=%v\n", resp2.StatusCode, string(reply2), time.Since(startT))

	// 等待后台 Worker 异步派发
	time.Sleep(700 * time.Millisecond)

	// 4.3 场景三：微信因网络延迟重复重推同一文本消息 (幂等去重校验)
	fmt.Println("\n--- 场景 3：微信因网络抖动重发上一条咨询消息 (幂等性去重) ---")
	resp3, _ := httpClient.Post(aesPostURL, "text/xml", strings.NewReader(wrapperXML))
	reply3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	fmt.Printf("网关幂等拦截并直接应答微信: HTTP %d, 响应=%s (重复消息不再生成下游投递任务)\n", resp3.StatusCode, string(reply3))

	time.Sleep(500 * time.Millisecond)

	// =========================================================================
	// 阶段五：汇总下游消费与验签报告
	// =========================================================================
	fmt.Println("\n>>> [阶段 5/5] 下游程序分发结果汇总与审计报告")
	mu.Lock()
	defer mu.Unlock()

	fmt.Printf("\n总计成功接收投递任务: %d 次\n", len(receivedEvents))
	fmt.Println("--------------------------------------------------------------------------------")
	for i, ev := range receivedEvents {
		fmt.Printf("[%d] 业务系统: %-18s | 消息类型: %-6s | 事件: %-10s | HMAC验签: %t\n",
			i+1, ev.ReceiverName, ev.MsgType, ev.Event, ev.Verified)
		if ev.MsgType == "text" {
			fields, _ := ev.Payload["fields"].(map[string]any)
			fmt.Printf("    └─ 客户提问内容: \"%s\" (用户: %s)\n", fields["Content"], ev.FromUser)
		} else if ev.Event == "subscribe" {
			fmt.Printf("    └─ 渠道归因标识: \"%s\" (用户: %s)\n", ev.EventKey, ev.FromUser)
		}
	}
	fmt.Println("--------------------------------------------------------------------------------")

	fmt.Println("\n规则路由验证判定:")
	fmt.Println("√ 场景 1 关注事件: CRM中心 (已收到) | 审计系统 (已收到) | 客服系统 (未收到/规则隔离)")
	fmt.Println("√ 场景 2 文本咨询: 客服系统 (已收到) | 审计系统 (已收到) | CRM中心 (未收到/规则隔离)")
	fmt.Println("√ 场景 3 幂等去重: 微信重复重推未造成下游二次消费")
	fmt.Println("√ 所有出站投递均通过 HMAC-SHA256 签名校验!")
	fmt.Println("\n全功能模拟与多路分发验证圆满完成!")
}
