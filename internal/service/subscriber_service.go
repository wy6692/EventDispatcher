package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"eventdispatcher/internal/domain"
	"eventdispatcher/pkg/errcode"
	"eventdispatcher/pkg/id"
)

type SubscriberDTO struct {
	ID          string          `json:"id,omitempty"`
	AccountID   string          `json:"accountId"`
	Name        string          `json:"name"`
	CallbackURL string          `json:"callbackUrl"`
	Secret      string          `json:"secret,omitempty"`
	SecretSet   bool            `json:"secretSet,omitempty"`
	Enabled     *bool           `json:"enabled,omitempty"`
	TimeoutMs   int             `json:"timeoutMs"`
	MaxAttempts int             `json:"maxAttempts"`
	Filter      json.RawMessage `json:"filter,omitempty"`
	CreatedAt   string          `json:"createdAt,omitempty"`
	UpdatedAt   string          `json:"updatedAt,omitempty"`
}

type SubscriberService struct {
	repo        domain.SubscriberRepository
	accountRepo domain.AccountRepository
}

func NewSubscriberService(repo domain.SubscriberRepository, accountRepo domain.AccountRepository) *SubscriberService {
	return &SubscriberService{repo: repo, accountRepo: accountRepo}
}

func (s *SubscriberService) Create(ctx context.Context, dto SubscriberDTO) (*SubscriberDTO, error) {
	name := strings.TrimSpace(dto.Name)
	accID := strings.TrimSpace(dto.AccountID)
	cbURL := strings.TrimSpace(dto.CallbackURL)
	secret := dto.Secret

	if _, err := s.accountRepo.GetByID(ctx, accID); err != nil {
		return nil, err
	}

	timeoutMs := dto.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 3000
	}
	maxAttempts := dto.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 6
	}
	enabled := true
	if dto.Enabled != nil {
		enabled = *dto.Enabled
	}

	filterJSON, err := canonicalFilter(dto.Filter)
	if err != nil {
		return nil, err
	}

	if err := validateSubscriberFields(name, accID, cbURL, secret, timeoutMs, maxAttempts); err != nil {
		return nil, err
	}

	sub := &domain.Subscriber{
		AccountID:   accID,
		Name:        name,
		CallbackURL: cbURL,
		Secret:      secret,
		Enabled:     enabled,
		TimeoutMs:   timeoutMs,
		MaxAttempts: maxAttempts,
		FilterJSON:  filterJSON,
	}

	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, err
	}

	return toSubscriberDTO(sub), nil
}

func (s *SubscriberService) Get(ctx context.Context, id string) (*SubscriberDTO, error) {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toSubscriberDTO(sub), nil
}

func (s *SubscriberService) ListByAccount(ctx context.Context, accountID string) ([]*SubscriberDTO, error) {
	if accountID != "" {
		if _, err := s.accountRepo.GetByID(ctx, accountID); err != nil {
			return nil, err
		}
	}
	list, err := s.repo.ListByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	res := make([]*SubscriberDTO, 0, len(list))
	for _, sub := range list {
		res = append(res, toSubscriberDTO(sub))
	}
	return res, nil
}

func (s *SubscriberService) Update(ctx context.Context, id string, dto SubscriberDTO) (*SubscriberDTO, error) {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if dto.Name != "" {
		current.Name = strings.TrimSpace(dto.Name)
	}
	if dto.AccountID != "" {
		current.AccountID = strings.TrimSpace(dto.AccountID)
		if _, err := s.accountRepo.GetByID(ctx, current.AccountID); err != nil {
			return nil, err
		}
	}
	if dto.CallbackURL != "" {
		current.CallbackURL = strings.TrimSpace(dto.CallbackURL)
	}
	if dto.Secret != "" {
		current.Secret = dto.Secret
	}
	if dto.Enabled != nil {
		current.Enabled = *dto.Enabled
	}
	if dto.TimeoutMs > 0 {
		current.TimeoutMs = dto.TimeoutMs
	}
	if dto.MaxAttempts > 0 {
		current.MaxAttempts = dto.MaxAttempts
	}
	if len(dto.Filter) > 0 {
		filterJSON, err := canonicalFilter(dto.Filter)
		if err != nil {
			return nil, err
		}
		current.FilterJSON = filterJSON
	}

	if err := validateSubscriberFields(current.Name, current.AccountID, current.CallbackURL, current.Secret, current.TimeoutMs, current.MaxAttempts); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, current); err != nil {
		return nil, err
	}

	return toSubscriberDTO(current), nil
}

func (s *SubscriberService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func validateSubscriberFields(name, accID, cbURL, secret string, timeoutMs, maxAttempts int) error {
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 64 {
		return errcode.ErrInvalidParam.WithMessage("name 长度必须在 1 到 64 之间")
	}
	if accID == "" {
		return errcode.ErrInvalidParam.WithMessage("accountId 不能为空")
	}
	parsed, err := url.ParseRequestURI(cbURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errcode.ErrInvalidParam.WithMessage("callbackUrl 必须是合法的 http/https 地址")
	}
	if len(secret) < 1 || len(secret) > 256 {
		return errcode.ErrInvalidParam.WithMessage("secret 长度必须在 1 到 256 之间")
	}
	if timeoutMs < 100 || timeoutMs > 30000 {
		return errcode.ErrInvalidParam.WithMessage("timeoutMs 必须在 100 到 30000 之间")
	}
	if maxAttempts < 1 || maxAttempts > 10 {
		return errcode.ErrInvalidParam.WithMessage("maxAttempts 必须在 1 到 10 之间")
	}
	return nil
}

func canonicalFilter(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "{}", nil
	}
	var f domain.Filter
	if err := json.Unmarshal(raw, &f); err != nil {
		return "", errcode.ErrInvalidParam.WithMessage("filter 必须是合法的 JSON 对象")
	}
	b, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func toSubscriberDTO(s *domain.Subscriber) *SubscriberDTO {
	if s == nil {
		return nil
	}
	filter := json.RawMessage(s.FilterJSON)
	if len(filter) == 0 {
		filter = json.RawMessage("{}")
	}
	enabled := s.Enabled
	return &SubscriberDTO{
		ID:          s.ID,
		AccountID:   s.AccountID,
		Name:        s.Name,
		CallbackURL: s.CallbackURL,
		SecretSet:   s.Secret != "",
		Enabled:     &enabled,
		TimeoutMs:   s.TimeoutMs,
		MaxAttempts: s.MaxAttempts,
		Filter:      filter,
		CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// SubscriberTestResult 表示对下游订阅方发起连通性测试的实时响应结果。
type SubscriberTestResult struct {
	Success      bool              `json:"success"`
	StatusCode   int               `json:"statusCode"`
	StatusText   string            `json:"statusText"`
	LatencyMs    int64             `json:"latencyMs"`
	TargetURL    string            `json:"targetUrl"`
	ResponseBody string            `json:"responseBody"`
	Error        string            `json:"error,omitempty"`
	EventID      string            `json:"eventId"`
	Timestamp    string            `json:"timestamp"`
	HeadersSent  map[string]string `json:"headersSent"`
}

// SubscriberTestDirectDTO 用于无需先持久化订阅方直接测试出站连接。
type SubscriberTestDirectDTO struct {
	AccountID   string `json:"accountId"`
	CallbackURL string `json:"callbackUrl"`
	Secret      string `json:"secret"`
	TimeoutMs   int    `json:"timeoutMs"`
}

// Test 对已存在的下游订阅方发送测试事件验证连通性。
func (s *SubscriberService) Test(ctx context.Context, id string) (*SubscriberTestResult, error) {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	accountKey := "test_account"
	appID := "wx_test_app"
	if acc, err := s.accountRepo.GetByID(ctx, sub.AccountID); err == nil && acc != nil {
		accountKey = acc.AccountKey
		appID = acc.AppID
	}

	return s.sendTestProbe(ctx, sub.CallbackURL, sub.Secret, sub.TimeoutMs, accountKey, appID)
}

// TestDirect 对指定参数的下游地址发送测试事件验证连通性。
func (s *SubscriberService) TestDirect(ctx context.Context, dto SubscriberTestDirectDTO) (*SubscriberTestResult, error) {
	cbURL := strings.TrimSpace(dto.CallbackURL)
	if cbURL == "" {
		return nil, errcode.ErrInvalidParam.WithMessage("callbackUrl 不能为空")
	}

	accountKey := "test_account"
	appID := "wx_test_app"
	if dto.AccountID != "" {
		if acc, err := s.accountRepo.GetByID(ctx, dto.AccountID); err == nil && acc != nil {
			accountKey = acc.AccountKey
			appID = acc.AppID
		}
	}

	return s.sendTestProbe(ctx, cbURL, dto.Secret, dto.TimeoutMs, accountKey, appID)
}

func (s *SubscriberService) sendTestProbe(ctx context.Context, callbackURL, secret string, timeoutMs int, accountKey, appID string) (*SubscriberTestResult, error) {
	testEvtID, err := id.New("evt_test_")
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	ts := strconv.FormatInt(now.Unix(), 10)

	payloadObj := map[string]any{
		"id":         testEvtID,
		"accountKey": accountKey,
		"appId":      appID,
		"msgType":    "event",
		"event":      "test_ping",
		"eventKey":   "connectivity_test",
		"fromUser":   "EventDispatcher_Probe",
		"toUser":     appID,
		"createTime": now.Unix(),
		"receivedAt": now.Format(time.RFC3339),
		"fields": map[string]string{
			"ToUserName":   appID,
			"FromUserName": "EventDispatcher_Probe",
			"CreateTime":   ts,
			"MsgType":      "event",
			"Event":        "test_ping",
			"EventKey":     "connectivity_test",
			"Content":      "EventDispatcher 下游连通性测试 (Connectivity Test)",
		},
	}

	payloadBytes, err := json.Marshal(payloadObj)
	if err != nil {
		return nil, err
	}

	sig := SignPayload(secret, ts, payloadBytes)

	if timeoutMs <= 0 {
		timeoutMs = 3000
	}
	if timeoutMs > 10000 {
		timeoutMs = 10000
	}

	probeCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodPost, callbackURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return &SubscriberTestResult{
			Success:    false,
			StatusCode: 0,
			StatusText: "Invalid URL",
			TargetURL:  callbackURL,
			Error:      fmt.Sprintf("创建请求失败: %v", err),
			EventID:    testEvtID,
			Timestamp:  ts,
		}, nil
	}

	headersSent := map[string]string{
		"Content-Type":           "application/json",
		"User-Agent":             "event-dispatcher/1 (connectivity-probe)",
		"X-Dispatcher-Event-Id":  testEvtID,
		"X-Dispatcher-Timestamp": ts,
		"X-Dispatcher-Signature": sig,
	}
	for k, v := range headersSent {
		req.Header.Set(k, v)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &SubscriberTestResult{
			Success:     false,
			StatusCode:  0,
			StatusText:  "Network Error",
			LatencyMs:   latency,
			TargetURL:   callbackURL,
			Error:       err.Error(),
			EventID:     testEvtID,
			Timestamp:   ts,
			HeadersSent: headersSent,
		}, nil
	}
	defer resp.Body.Close()

	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	isSuccess := resp.StatusCode >= 200 && resp.StatusCode < 300
	statusText := http.StatusText(resp.StatusCode)
	if statusText == "" {
		statusText = "Unknown Status"
	}

	errMsg := ""
	if !isSuccess {
		errMsg = fmt.Sprintf("HTTP %d (%s)", resp.StatusCode, statusText)
		if s := sanitizeString(string(snippet)); s != "" {
			if len(s) > 256 {
				s = s[:256] + "..."
			}
			errMsg += ": " + s
		}
	}

	return &SubscriberTestResult{
		Success:      isSuccess,
		StatusCode:   resp.StatusCode,
		StatusText:   statusText,
		LatencyMs:    latency,
		TargetURL:    callbackURL,
		ResponseBody: string(snippet),
		Error:        errMsg,
		EventID:      testEvtID,
		Timestamp:    ts,
		HeadersSent:  headersSent,
	}, nil
}
