package http_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"eventdispatcher/internal/domain"
	"eventdispatcher/internal/infrastructure/persistence"
	"eventdispatcher/internal/service"
	transporthttp "eventdispatcher/internal/transport/http"
	"eventdispatcher/internal/transport/http/handler"
	"eventdispatcher/internal/wechat"
)

type recordedWebhook struct {
	Headers http.Header
	Body    []byte
	Event   map[string]any
}

// TestFullFeatureSuite 执行针对网关核心特性的全功能集成测试
func TestFullFeatureSuite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "full_feature_test.db")
	db, err := persistence.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	defer db.Close()

	accountRepo := persistence.NewAccountRepository(db)
	subscriberRepo := persistence.NewSubscriberRepository(db)
	eventRepo := persistence.NewEventRepository(db)
	deliveryRepo := persistence.NewDeliveryRepository(db)

	accountService := service.NewAccountService(accountRepo)
	subscriberService := service.NewSubscriberService(subscriberRepo, accountRepo)
	eventService := service.NewEventService(eventRepo)
	deliveryService := service.NewDeliveryService(deliveryRepo, nil)
	ingestService := service.NewIngestService(accountRepo, subscriberRepo, eventRepo, nil)

	adminToken := "secret-super-admin-token-123456"
	router := transporthttp.NewRouter(transporthttp.RouterConfig{
		AdminToken:        adminToken,
		AccountHandler:    handler.NewAccountHandler(accountService),
		SubscriberHandler: handler.NewSubscriberHandler(subscriberService),
		EventHandler:      handler.NewEventHandler(eventService),
		DeliveryHandler:   handler.NewDeliveryHandler(deliveryService),
		IngestHandler:     handler.NewIngestHandler(ingestService),
		HealthHandler:     handler.NewHealthHandler(db),
	})

	srv := httptest.NewServer(router)
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	// 辅助发送请求函数
	doAdminRequest := func(method, path string, body any, token string) (*http.Response, []byte) {
		var reqBody io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			reqBody = bytes.NewReader(b)
		}
		req, _ := http.NewRequest(method, srv.URL+path, reqBody)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, reqErr := client.Do(req)
		if reqErr != nil {
			t.Fatalf("HTTP %s %s failed: %v", method, path, reqErr)
		}
		defer resp.Body.Close()
		respData, _ := io.ReadAll(resp.Body)
		return resp, respData
	}

	// ==========================================
	// 1. 基础端点 & 鉴权校验测试
	// ==========================================
	t.Run("1_AuthAndHealthCheck", func(t *testing.T) {
		// 1.1 健康检查端点
		resp, body := doAdminRequest(http.MethodGet, "/healthz", nil, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for healthz, got %d", resp.StatusCode)
		}
		var healthResp struct {
			Code int `json:"code"`
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		_ = json.Unmarshal(body, &healthResp)
		if healthResp.Data.Status != "ok" {
			t.Fatalf("unexpected health response: %s", string(body))
		}

		// 1.2 未携带 Token 访问 Admin 接口应返回 401
		resp, _ = doAdminRequest(http.MethodGet, "/admin/accounts", nil, "")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 for unauthorized access, got %d", resp.StatusCode)
		}

		// 1.3 携带错误 Token 访问 Admin 接口应返回 401
		resp, _ = doAdminRequest(http.MethodGet, "/admin/accounts", nil, "wrong-token")
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 for wrong token, got %d", resp.StatusCode)
		}

		// 1.4 携带正确 Token 访问应返回 200
		resp, _ = doAdminRequest(http.MethodGet, "/admin/accounts", nil, adminToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for valid token, got %d", resp.StatusCode)
		}
	})

	// ==========================================
	// 2. 账号管理 (Account CRUD & 脱敏)
	// ==========================================
	var accountID string
	accountKey := "wx_enterprise"
	appID := "wx9876543210fedcba"
	wechatToken := "wechat_demo_token_key"
	encodingAESKey := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"

	t.Run("2_AccountCRUD", func(t *testing.T) {
		// 2.1 创建账号
		createPayload := map[string]string{
			"name":           "企业服务号",
			"accountKey":     accountKey,
			"appId":          appID,
			"token":          wechatToken,
			"encodingAesKey": encodingAESKey,
		}
		resp, body := doAdminRequest(http.MethodPost, "/admin/accounts", createPayload, adminToken)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create account failed: %s", string(body))
		}
		var res struct {
			Code int `json:"code"`
			Data struct {
				ID             string `json:"id"`
				Name           string `json:"name"`
				AccountKey     string `json:"accountKey"`
				AppID          string `json:"appId"`
				Token          string `json:"token"`
				EncodingAESKey string `json:"encodingAesKey"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &res); err != nil {
			t.Fatalf("unmarshal account response failed: %v", err)
		}
		accountID = res.Data.ID
		if accountID == "" {
			t.Fatalf("empty account ID returned")
		}

		// 验证脱敏逻辑：Token 和 EncodingAESKey 不应完全明文暴露
		if res.Data.Token == wechatToken {
			t.Fatalf("account token should be masked")
		}

		// 2.2 重复创建相同 accountKey 应当冲突失败 (409)
		resp, _ = doAdminRequest(http.MethodPost, "/admin/accounts", createPayload, adminToken)
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("expected 409 conflict for duplicate accountKey, got %d", resp.StatusCode)
		}

		// 2.3 获取单个账号信息
		resp, body = doAdminRequest(http.MethodGet, "/admin/accounts/"+accountID, nil, adminToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get account failed: %s", string(body))
		}

		// 2.4 获取账号列表
		resp, body = doAdminRequest(http.MethodGet, "/admin/accounts", nil, adminToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list accounts failed: %s", string(body))
		}
		var listRes struct {
			Data struct {
				Items []map[string]any `json:"items"`
			} `json:"data"`
		}
		_ = json.Unmarshal(body, &listRes)
		if len(listRes.Data.Items) != 1 {
			t.Fatalf("expected 1 account in list, got %d", len(listRes.Data.Items))
		}
	})

	// ==========================================
	// 3. 模拟下游服务体系 (3 个不同规则的下游业务)
	// ==========================================
	var mu sync.Mutex
	downstreamReceived := map[string][]recordedWebhook{
		"crm":      {},
		"service":  {},
		"audit":    {},
		"flakyApp": {},
	}

	flakyCounter := 0
	downstreamSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var evt map[string]any
		_ = json.Unmarshal(body, &evt)

		mu.Lock()
		defer mu.Unlock()

		item := recordedWebhook{
			Headers: r.Header.Clone(),
			Body:    body,
			Event:   evt,
		}

		switch r.URL.Path {
		case "/webhook/crm":
			downstreamReceived["crm"] = append(downstreamReceived["crm"], item)
			w.WriteHeader(http.StatusOK)
		case "/webhook/service":
			downstreamReceived["service"] = append(downstreamReceived["service"], item)
			w.WriteHeader(http.StatusOK)
		case "/webhook/audit":
			downstreamReceived["audit"] = append(downstreamReceived["audit"], item)
			w.WriteHeader(http.StatusOK)
		case "/webhook/flaky":
			flakyCounter++
			if flakyCounter == 1 {
				// 第一次投递时模拟 500 异常
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("internal database failure"))
				return
			}
			downstreamReceived["flakyApp"] = append(downstreamReceived["flakyApp"], item)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer downstreamSrv.Close()

	crmSecret := "crm_webhook_hmac_secret_key"
	serviceSecret := "service_webhook_hmac_secret_key"
	auditSecret := "audit_webhook_hmac_secret_key"
	flakySecret := "flaky_webhook_hmac_secret_key"

	// ==========================================
	// 4. 订阅者配置 (Subscriber CRUD & 规则绑定)
	// ==========================================
	var crmSubID, serviceSubID, auditSubID, flakySubID string
	t.Run("4_SubscriberSetup", func(t *testing.T) {
		// 4.1 校验不存在的 accountId 创建订阅者应报错 404
		invalidSubPayload := map[string]any{
			"name":        "非法账号订阅者",
			"accountId":   "acc_not_exists",
			"callbackUrl": downstreamSrv.URL + "/webhook/crm",
		}
		resp, _ := doAdminRequest(http.MethodPost, "/admin/subscribers", invalidSubPayload, adminToken)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 when creating subscriber with non-existent accountId, got %d", resp.StatusCode)
		}

		// 4.2 校验非法 callbackUrl 应报错 400
		badURLPayload := map[string]any{
			"name":        "非法URL订阅者",
			"accountId":   accountID,
			"callbackUrl": "not-a-valid-http-url",
		}
		resp, _ = doAdminRequest(http.MethodPost, "/admin/subscribers", badURLPayload, adminToken)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for bad callbackUrl, got %d", resp.StatusCode)
		}

		// 4.3 创建 CRM 订阅者：只消费带参二维码关注事件与扫描事件 (eventKeyPrefix: qrscene_)
		crmSub := map[string]any{
			"name":        "用户增长CRM中心",
			"accountId":   accountID,
			"callbackUrl": downstreamSrv.URL + "/webhook/crm",
			"secret":      crmSecret,
			"filter": map[string]any{
				"msgTypes":         []string{"event"},
				"events":           []string{"subscribe", "SCAN"},
				"eventKeyPrefixes": []string{"qrscene_"},
			},
		}
		resp, body := doAdminRequest(http.MethodPost, "/admin/subscribers", crmSub, adminToken)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create CRM subscriber failed: %s", string(body))
		}
		var crmResp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(body, &crmResp)
		crmSubID = crmResp.Data.ID

		// 4.4 创建客服系统订阅者：只消费普通文本消息 (msgTypes: ["text"])
		serviceSub := map[string]any{
			"name":        "智能在线客服系统",
			"accountId":   accountID,
			"callbackUrl": downstreamSrv.URL + "/webhook/service",
			"secret":      serviceSecret,
			"filter": map[string]any{
				"msgTypes": []string{"text"},
			},
		}
		resp, body = doAdminRequest(http.MethodPost, "/admin/subscribers", serviceSub, adminToken)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create Service subscriber failed: %s", string(body))
		}
		var srvResp struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(body, &srvResp)
		serviceSubID = srvResp.Data.ID

		// 4.5 创建合规审计系统订阅者：消费全部事件与消息 (nil 或 empty filter)
		auditSub := map[string]any{
			"name":        "企业合规与安全审计中心",
			"accountId":   accountID,
			"callbackUrl": downstreamSrv.URL + "/webhook/audit",
			"secret":      auditSecret,
			"filter":      map[string]any{},
		}
		resp, body = doAdminRequest(http.MethodPost, "/admin/subscribers", auditSub, adminToken)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create Audit subscriber failed: %s", string(body))
		}
		var auditRes struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(body, &auditRes)
		auditSubID = auditRes.Data.ID

		// 4.6 创建用于测试失败重试的订阅者
		flakySub := map[string]any{
			"name":        "异常下游系统",
			"accountId":   accountID,
			"callbackUrl": downstreamSrv.URL + "/webhook/flaky",
			"secret":      flakySecret,
			"filter": map[string]any{
				"msgTypes": []string{"event"},
				"events":   []string{"FLAKY_TEST"},
			},
		}
		resp, body = doAdminRequest(http.MethodPost, "/admin/subscribers", flakySub, adminToken)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create Flaky subscriber failed: %s", string(body))
		}
		var flakyRes struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(body, &flakyRes)
		flakySubID = flakyRes.Data.ID

		if crmSubID == "" || serviceSubID == "" || auditSubID == "" || flakySubID == "" {
			t.Fatalf("one or more subscriber IDs are empty")
		}
	})

	// ==========================================
	// 5. 微信网关接入验证 (GET /w/{accountKey})
	// ==========================================
	t.Run("5_WeChatVerifyURL", func(t *testing.T) {
		ts := "1730000000"
		nonce := "random_nonce_123"
		echostr := "wechat_challenge_echo_payload"

		// 5.1 正确的明文校验
		sig := wechat.Signature(wechatToken, ts, nonce)
		url := fmt.Sprintf("%s/w/%s?signature=%s&timestamp=%s&nonce=%s&echostr=%s",
			srv.URL, accountKey, sig, ts, nonce, echostr)
		resp, err := client.Get(url)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("verify URL failed, status: %d", resp.StatusCode)
		}
		echoResp, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(echoResp) != echostr {
			t.Fatalf("expected echostr %s, got %s", echostr, string(echoResp))
		}

		// 5.2 错误的签名校验应拒绝 (401)
		badSigURL := fmt.Sprintf("%s/w/%s?signature=bad_sig&timestamp=%s&nonce=%s&echostr=%s",
			srv.URL, accountKey, ts, nonce, echostr)
		resp, _ = client.Get(badSigURL)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401 for bad signature, got %d", resp.StatusCode)
		}

		// 5.3 安全模式 (AES 加密) 接入验证
		plainChallenge := "aes_challenge_test_data"
		encryptedEcho, err := wechat.Encrypt(encodingAESKey, appID, []byte(plainChallenge))
		if err != nil {
			t.Fatalf("failed to encrypt echostr: %v", err)
		}
		msgSig := wechat.Signature(wechatToken, ts, nonce, encryptedEcho)
		aesURL := fmt.Sprintf("%s/w/%s?signature=%s&timestamp=%s&nonce=%s&echostr=%s&encrypt_type=aes&msg_signature=%s",
			srv.URL, accountKey, sig, ts, nonce, encryptedEcho, msgSig)
		resp, err = client.Get(aesURL)
		if err != nil || resp.StatusCode != http.StatusOK {
			t.Fatalf("AES verify URL failed, status: %d", resp.StatusCode)
		}
		aesEchoResp, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(aesEchoResp) != plainChallenge {
			t.Fatalf("expected decrypted challenge %s, got %s", plainChallenge, string(aesEchoResp))
		}
	})

	// 辅助发送微信事件 POST
	postWeChatXML := func(xmlContent string, useAES bool) (*http.Response, string) {
		ts := fmt.Sprintf("%d", time.Now().Unix())
		nonce := "wechat_post_nonce"
		sig := wechat.Signature(wechatToken, ts, nonce)

		var postBody []byte
		var reqURL string

		if useAES {
			encryptedXML, err := wechat.Encrypt(encodingAESKey, appID, []byte(xmlContent))
			if err != nil {
				t.Fatalf("encrypt message failed: %v", err)
			}
			msgSig := wechat.Signature(wechatToken, ts, nonce, encryptedXML)
			wrapperXML := fmt.Sprintf(`<xml>
				<ToUserName><![CDATA[%s]]></ToUserName>
				<Encrypt><![CDATA[%s]]></Encrypt>
			</xml>`, appID, encryptedXML)
			postBody = []byte(wrapperXML)
			reqURL = fmt.Sprintf("%s/w/%s?signature=%s&timestamp=%s&nonce=%s&encrypt_type=aes&msg_signature=%s",
				srv.URL, accountKey, sig, ts, nonce, msgSig)
		} else {
			postBody = []byte(xmlContent)
			reqURL = fmt.Sprintf("%s/w/%s?signature=%s&timestamp=%s&nonce=%s",
				srv.URL, accountKey, sig, ts, nonce)
		}

		resp, err := client.Post(reqURL, "text/xml", bytes.NewReader(postBody))
		if err != nil {
			t.Fatalf("post wechat event failed: %v", err)
		}
		defer resp.Body.Close()
		bodyStr, _ := io.ReadAll(resp.Body)
		return resp, string(bodyStr)
	}

	// 验签下游收到请求的辅助函数
	verifyHMACSignature := func(secret, timestamp, gotSig string, payload []byte) bool {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(timestamp))
		mac.Write([]byte("."))
		mac.Write(payload)
		expected := hex.EncodeToString(mac.Sum(nil))
		return hmac.Equal([]byte(expected), []byte(gotSig))
	}

	// ==========================================
	// 6. 微信明文事件接收、多路由过滤与 HMAC 验签
	// ==========================================
	t.Run("6_PlainEventRoutingAndDelivery", func(t *testing.T) {
		// 6.1 发送文本消息：MsgType: "text", Content: "我想咨询售后服务"
		// 预期规则：客服系统收到、审计系统收到；CRM 系统不应收到
		textXML := `<xml>
			<ToUserName><![CDATA[gh_enterprise_account]]></ToUserName>
			<FromUserName><![CDATA[oX_user_customer_01]]></FromUserName>
			<CreateTime>1730001001</CreateTime>
			<MsgType><![CDATA[text]]></MsgType>
			<Content><![CDATA[我想咨询售后服务]]></Content>
			<MsgId>9900000000000001</MsgId>
		</xml>`
		resp, reply := postWeChatXML(textXML, false)
		if resp.StatusCode != http.StatusOK || reply != "success" {
			t.Fatalf("text msg ingest failed: code=%d, reply=%s", resp.StatusCode, reply)
		}

		// 触发 Worker 调度批处理
		if err := deliveryService.ProcessBatch(context.Background()); err != nil {
			t.Fatalf("process batch failed: %v", err)
		}

		mu.Lock()
		srvCount := len(downstreamReceived["service"])
		auditCount := len(downstreamReceived["audit"])
		crmCount := len(downstreamReceived["crm"])
		mu.Unlock()

		if srvCount != 1 {
			t.Fatalf("customer service should receive 1 text event, got %d", srvCount)
		}
		if auditCount != 1 {
			t.Fatalf("audit service should receive 1 event, got %d", auditCount)
		}
		if crmCount != 0 {
			t.Fatalf("crm service should NOT receive text event, got %d", crmCount)
		}

		// 验证客服系统收到的 HMAC-SHA256 签名与 Header
		mu.Lock()
		srvItem := downstreamReceived["service"][0]
		mu.Unlock()

		srvSig := srvItem.Headers.Get("X-Dispatcher-Signature")
		srvTS := srvItem.Headers.Get("X-Dispatcher-Timestamp")
		srvEvtID := srvItem.Headers.Get("X-Dispatcher-Event-Id")
		if srvSig == "" || srvTS == "" || srvEvtID == "" {
			t.Fatalf("missing required dispatcher headers in downstream request")
		}
		if !verifyHMACSignature(serviceSecret, srvTS, srvSig, srvItem.Body) {
			t.Fatalf("service webhook HMAC signature verification failed")
		}
		if srvItem.Event["msgType"] != "text" {
			t.Fatalf("unexpected msgType in downstream payload: %v", srvItem.Event["msgType"])
		}

		// 6.2 发送带参二维码关注事件：MsgType: "event", Event: "subscribe", EventKey: "qrscene_channel_douyin"
		// 预期规则：CRM 系统收到、审计系统收到；客服系统不应收到
		qrScanXML := `<xml>
			<ToUserName><![CDATA[gh_enterprise_account]]></ToUserName>
			<FromUserName><![CDATA[oX_user_growth_02]]></FromUserName>
			<CreateTime>1730001002</CreateTime>
			<MsgType><![CDATA[event]]></MsgType>
			<Event><![CDATA[subscribe]]></Event>
			<EventKey><![CDATA[qrscene_channel_douyin]]></EventKey>
		</xml>`
		resp, reply = postWeChatXML(qrScanXML, false)
		if resp.StatusCode != http.StatusOK || reply != "success" {
			t.Fatalf("qr subscribe event ingest failed: code=%d, reply=%s", resp.StatusCode, reply)
		}

		if err := deliveryService.ProcessBatch(context.Background()); err != nil {
			t.Fatalf("process batch failed: %v", err)
		}

		mu.Lock()
		crmCount = len(downstreamReceived["crm"])
		srvCount = len(downstreamReceived["service"])
		auditCount = len(downstreamReceived["audit"])
		mu.Unlock()

		if crmCount != 1 {
			t.Fatalf("crm service should receive 1 qrscene event, got %d", crmCount)
		}
		if srvCount != 1 {
			t.Fatalf("customer service should still have only 1 event, got %d", srvCount)
		}
		if auditCount != 2 {
			t.Fatalf("audit service should receive all 2 events, got %d", auditCount)
		}

		// 验证 CRM 系统的 HMAC 签名
		mu.Lock()
		crmItem := downstreamReceived["crm"][0]
		mu.Unlock()
		crmSig := crmItem.Headers.Get("X-Dispatcher-Signature")
		crmTS := crmItem.Headers.Get("X-Dispatcher-Timestamp")
		if !verifyHMACSignature(crmSecret, crmTS, crmSig, crmItem.Body) {
			t.Fatalf("crm webhook HMAC signature verification failed")
		}
	})

	// ==========================================
	// 7. 微信安全模式 (AES 加密) 消息接收与全链路分发
	// ==========================================
	t.Run("7_EncryptedAESMessageIngest", func(t *testing.T) {
		// 7.1 安全模式加密的文本消息
		plainMsg := `<xml>
			<ToUserName><![CDATA[` + appID + `]]></ToUserName>
			<FromUserName><![CDATA[oX_user_vip_03]]></FromUserName>
			<CreateTime>1730001003</CreateTime>
			<MsgType><![CDATA[text]]></MsgType>
			<Content><![CDATA[安全加密通道文本咨询]]></Content>
			<MsgId>9900000000000002</MsgId>
		</xml>`

		resp, reply := postWeChatXML(plainMsg, true)
		if resp.StatusCode != http.StatusOK || reply != "success" {
			t.Fatalf("encrypted msg ingest failed: code=%d, reply=%s", resp.StatusCode, reply)
		}

		if err := deliveryService.ProcessBatch(context.Background()); err != nil {
			t.Fatalf("process batch failed: %v", err)
		}

		mu.Lock()
		srvCount := len(downstreamReceived["service"])
		mu.Unlock()

		if srvCount != 2 {
			t.Fatalf("customer service should receive 2nd text message (from AES encrypted packet), got %d", srvCount)
		}

		mu.Lock()
		latestSrv := downstreamReceived["service"][1]
		mu.Unlock()
		fields, ok := latestSrv.Event["fields"].(map[string]any)
		if !ok || fields["Content"] != "安全加密通道文本咨询" {
			t.Fatalf("unexpected decrypted fields in downstream: %v", fields)
		}
	})

	// ==========================================
	// 8. 微信幂等去重校验 (重发相同事件仅投递一次)
	// ==========================================
	t.Run("8_IdempotencyDeduplication", func(t *testing.T) {
		dupXML := `<xml>
			<ToUserName><![CDATA[gh_enterprise_account]]></ToUserName>
			<FromUserName><![CDATA[oX_user_dup_04]]></FromUserName>
			<CreateTime>1730001004</CreateTime>
			<MsgType><![CDATA[text]]></MsgType>
			<Content><![CDATA[重复发送的消息]]></Content>
			<MsgId>9900000000000003</MsgId>
		</xml>`

		// 第一次投递
		resp1, _ := postWeChatXML(dupXML, false)
		if resp1.StatusCode != http.StatusOK {
			t.Fatalf("first post failed: %d", resp1.StatusCode)
		}

		// 微信因网络延迟重复重推同一报文
		resp2, _ := postWeChatXML(dupXML, false)
		if resp2.StatusCode != http.StatusOK {
			t.Fatalf("second duplicate post failed: %d", resp2.StatusCode)
		}

		// 执行投递
		if err := deliveryService.ProcessBatch(context.Background()); err != nil {
			t.Fatalf("process batch failed: %v", err)
		}

		mu.Lock()
		currentSrvCount := len(downstreamReceived["service"])
		mu.Unlock()

		// 在前一个测试基础上增加 1 条（总共 3 条，绝不能是 4 条）
		if currentSrvCount != 3 {
			t.Fatalf("expected 3 total service deliveries after duplicate post, got %d", currentSrvCount)
		}
	})

	// ==========================================
	// 9. 失败退避重试与人工一键重放测试
	// ==========================================
	t.Run("9_RetryAndManualReplay", func(t *testing.T) {
		flakyXML := `<xml>
			<ToUserName><![CDATA[gh_enterprise_account]]></ToUserName>
			<FromUserName><![CDATA[oX_user_flaky_05]]></FromUserName>
			<CreateTime>1730001005</CreateTime>
			<MsgType><![CDATA[event]]></MsgType>
			<Event><![CDATA[FLAKY_TEST]]></Event>
			<EventKey><![CDATA[retry_key_1]]></EventKey>
		</xml>`

		resp, _ := postWeChatXML(flakyXML, false)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("post flaky event failed")
		}

		// 第一次调度：下游 /webhook/flaky 返回 500 报错
		if err := deliveryService.ProcessBatch(context.Background()); err != nil {
			t.Fatalf("process batch failed: %v", err)
		}

		// 查询 delivery 任务状态
		resp, body := doAdminRequest(http.MethodGet, "/admin/deliveries?status=pending", nil, adminToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list pending deliveries failed: %s", string(body))
		}
		var dlvList struct {
			Data struct {
				Items []map[string]any `json:"items"`
			} `json:"data"`
		}
		_ = json.Unmarshal(body, &dlvList)

		var failedDeliveryID string
		for _, item := range dlvList.Data.Items {
			if item["subscriberId"] == flakySubID {
				failedDeliveryID = item["id"].(string)
				attempt := int(item["attempt"].(float64))
				if attempt != 1 {
					t.Fatalf("expected attempt 1 after failure, got %d", attempt)
				}
				lastCode := int(item["lastStatusCode"].(float64))
				if lastCode != 500 {
					t.Fatalf("expected lastStatusCode 500, got %d", lastCode)
				}
				break
			}
		}
		if failedDeliveryID == "" {
			t.Fatalf("flaky delivery not found in pending retry list")
		}

		// 调用管理接口人工触发重试: POST /admin/deliveries/{id}/retry
		resp, body = doAdminRequest(http.MethodPost, "/admin/deliveries/"+failedDeliveryID+"/retry", nil, adminToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("manual retry failed: %s", string(body))
		}

		// 再次执行调度：下游第二次请求返回 200 成功
		if err := deliveryService.ProcessBatch(context.Background()); err != nil {
			t.Fatalf("process batch failed: %v", err)
		}

		mu.Lock()
		flakyReceivedCount := len(downstreamReceived["flakyApp"])
		mu.Unlock()

		if flakyReceivedCount != 1 {
			t.Fatalf("expected flakyApp to receive 1 successful delivery after retry, got %d", flakyReceivedCount)
		}

		// 再次检查 Delivery 状态应为 success
		resp, body = doAdminRequest(http.MethodGet, "/admin/deliveries/"+failedDeliveryID, nil, adminToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get delivery failed: %s", string(body))
		}
		var finalDlv struct {
			Data struct {
				Status string `json:"status"`
			} `json:"data"`
		}
		_ = json.Unmarshal(body, &finalDlv)
		if finalDlv.Data.Status != domain.DeliveryStatusSuccess {
			t.Fatalf("expected final delivery status to be success, got %s", finalDlv.Data.Status)
		}
	})

	// ==========================================
	// 10. 管理查询接口 (Events & Deliveries)
	// ==========================================
	t.Run("10_ManagementQueries", func(t *testing.T) {
		// 10.1 查询事件列表
		resp, body := doAdminRequest(http.MethodGet, "/admin/events?limit=10", nil, adminToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list events failed: %s", string(body))
		}
		var evtList struct {
			Data struct {
				Items []map[string]any `json:"items"`
			} `json:"data"`
		}
		_ = json.Unmarshal(body, &evtList)
		if len(evtList.Data.Items) == 0 {
			t.Fatalf("expected events in list, got 0")
		}

		firstEvtID := evtList.Data.Items[0]["id"].(string)

		// 10.2 查询单个事件详情
		resp, body = doAdminRequest(http.MethodGet, "/admin/events/"+firstEvtID, nil, adminToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get event failed: %s", string(body))
		}

		// 10.3 查询投递任务列表
		resp, body = doAdminRequest(http.MethodGet, "/admin/deliveries?limit=20", nil, adminToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list deliveries failed: %s", string(body))
		}
	})

	// ==========================================
	// 11. 资源更新与清理 (Update & Delete)
	// ==========================================
	t.Run("11_ResourceUpdatesAndDeletions", func(t *testing.T) {
		// 11.1 更新订阅者 (修改名字与 Filter)
		updateSubPayload := map[string]any{
			"name":        "CRM系统-已更新",
			"accountId":   accountID,
			"callbackUrl": downstreamSrv.URL + "/webhook/crm",
			"secret":      crmSecret,
			"filter": map[string]any{
				"msgTypes": []string{"event"},
			},
		}
		resp, body := doAdminRequest(http.MethodPut, "/admin/subscribers/"+crmSubID, updateSubPayload, adminToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("update subscriber failed: %s", string(body))
		}

		// 11.2 更新服务号信息
		updateAccPayload := map[string]string{
			"name":           "企业服务号-改名",
			"accountKey":     accountKey,
			"appId":          appID,
			"token":          wechatToken,
			"encodingAesKey": encodingAESKey,
		}
		resp, body = doAdminRequest(http.MethodPut, "/admin/accounts/"+accountID, updateAccPayload, adminToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("update account failed: %s", string(body))
		}

		// 11.3 删除异常订阅者
		resp, _ = doAdminRequest(http.MethodDelete, "/admin/subscribers/"+flakySubID, nil, adminToken)
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("delete subscriber failed: expected 204, got %d", resp.StatusCode)
		}

		// 验证该订阅者已不存在
		resp, _ = doAdminRequest(http.MethodGet, "/admin/subscribers/"+flakySubID, nil, adminToken)
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 after deleting subscriber, got %d", resp.StatusCode)
		}
	})
}
