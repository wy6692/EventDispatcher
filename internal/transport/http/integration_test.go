package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"eventdispatcher/internal/infrastructure/persistence"
	"eventdispatcher/internal/service"
	transporthttp "eventdispatcher/internal/transport/http"
	"eventdispatcher/internal/transport/http/handler"
	"eventdispatcher/internal/wechat"
)

func TestEndToEndPipeline(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
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

	adminToken := "test-admin-secret"
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

	// 1. 测试健康检查与 Web UI 页面服务
	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("health check failed: status=%d, err=%v", resp.StatusCode, err)
	}
	resp.Body.Close()

	respUI, err := http.Get(srv.URL + "/ui")
	if err != nil || respUI.StatusCode != http.StatusOK {
		t.Fatalf("web UI failed: status=%d, err=%v", respUI.StatusCode, err)
	}
	uiBody, _ := io.ReadAll(respUI.Body)
	respUI.Body.Close()
	if !bytes.Contains(uiBody, []byte("EventDispatcher")) {
		t.Fatalf("web UI HTML unexpected content")
	}

	// 2. 创建账号
	token := "wechat_token_123"
	aesKey := "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"
	accBody := map[string]string{
		"name":           "测试服务号",
		"accountKey":     "main",
		"appId":          "wx_test_appid",
		"token":          token,
		"encodingAesKey": aesKey,
	}
	bodyBytes, _ := json.Marshal(accBody)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/admin/accounts", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("create account failed: status=%d", resp.StatusCode)
	}

	var createAccResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&createAccResp)
	resp.Body.Close()
	accountID := createAccResp.Data.ID
	if accountID == "" {
		t.Fatalf("account ID is empty in response")
	}

	// 3. 测试微信 GET URL 校验
	timestamp := "1730874600"
	nonce := "noncestr123"
	echostr := "echo_challenge_string"
	sig := wechat.Signature(token, timestamp, nonce)
	verifyURL := fmt.Sprintf("%s/w/main?signature=%s&timestamp=%s&nonce=%s&echostr=%s", srv.URL, sig, timestamp, nonce, echostr)
	resp, err = http.Get(verifyURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("wechat verify URL failed: status=%d", resp.StatusCode)
	}
	echoBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(echoBody) != echostr {
		t.Fatalf("expected echostr %s, got %s", echostr, string(echoBody))
	}

	// 4. 模拟下游 Webhook 接收服务
	deliveredPayloads := make(chan []byte, 5)
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		deliveredPayloads <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	// 5. 创建订阅方
	subBody := map[string]any{
		"name":        "菜单消费程序",
		"accountId":   accountID,
		"callbackUrl": downstream.URL,
		"secret":      "menu_secret_val",
		"filter": map[string]any{
			"msgTypes": []string{"event"},
			"events":   []string{"CLICK"},
		},
	}
	subBytes, _ := json.Marshal(subBody)
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/admin/subscribers", bytes.NewReader(subBytes))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("create subscriber failed: status=%d", resp.StatusCode)
	}
	resp.Body.Close()

	// 6. 微信 POST 事件推送
	xmlPayload := `<xml>
		<ToUserName><![CDATA[gh_test]]></ToUserName>
		<FromUserName><![CDATA[oX_user123]]></FromUserName>
		<CreateTime>1730874600</CreateTime>
		<MsgType><![CDATA[event]]></MsgType>
		<Event><![CDATA[CLICK]]></Event>
		<EventKey><![CDATA[MENU_ABOUT]]></EventKey>
	</xml>`
	postSig := wechat.Signature(token, timestamp, nonce)
	postURL := fmt.Sprintf("%s/w/main?signature=%s&timestamp=%s&nonce=%s", srv.URL, postSig, timestamp, nonce)
	resp, err = http.Post(postURL, "text/xml", bytes.NewReader([]byte(xmlPayload)))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("wechat post event failed: status=%d", resp.StatusCode)
	}
	postRespBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(postRespBody) != "success" {
		t.Fatalf("expected 'success', got %s", string(postRespBody))
	}

	// 7. 执行后台调度并检查下游投递
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := deliveryService.ProcessBatch(ctx); err != nil {
		t.Fatalf("process batch delivery failed: %v", err)
	}

	select {
	case payload := <-deliveredPayloads:
		var eventMap map[string]any
		if err := json.Unmarshal(payload, &eventMap); err != nil {
			t.Fatalf("invalid payload json: %v", err)
		}
		if eventMap["event"] != "CLICK" || eventMap["eventKey"] != "MENU_ABOUT" {
			t.Fatalf("unexpected event data: %s", string(payload))
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for downstream delivery")
	}
}
