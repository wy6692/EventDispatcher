package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"eventdispatcher/internal/domain"
)

type DeliveryDTO struct {
	ID             string `json:"id"`
	EventID        string `json:"eventId"`
	SubscriberID   string `json:"subscriberId"`
	SubscriberName string `json:"subscriberName,omitempty"`
	Status         string `json:"status"`
	Attempt        int    `json:"attempt"`
	NextRetryAt    string `json:"nextRetryAt"`
	LastStatusCode int    `json:"lastStatusCode"`
	LastError      string `json:"lastError"`
	UpdatedAt      string `json:"updatedAt"`
}

type DeliveryService struct {
	repo   domain.DeliveryRepository
	client *http.Client
	logger *slog.Logger
}

func NewDeliveryService(repo domain.DeliveryRepository, logger *slog.Logger) *DeliveryService {
	if logger == nil {
		logger = slog.Default()
	}
	return &DeliveryService{
		repo: repo,
		client: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		logger: logger,
	}
}

// StartWorker 启动后台轮询投递调度器。
func (s *DeliveryService) StartWorker(ctx context.Context) {
	n, err := s.repo.RequeueInflight(ctx, time.Now().UTC())
	if err != nil && ctx.Err() == nil {
		s.logger.Error("failed to requeue inflight tasks on startup", slog.String("error", err.Error()))
	} else if n > 0 {
		s.logger.Info("requeued inflight delivery tasks", slog.Int64("count", n))
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		if err := s.ProcessBatch(ctx); err != nil && ctx.Err() == nil {
			s.logger.Error("deliver batch scan failed", slog.String("error", err.Error()))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// ProcessBatch 拉取并执行一批到期任务。
func (s *DeliveryService) ProcessBatch(ctx context.Context) error {
	jobs, err := s.repo.ClaimDue(ctx, time.Now().UTC(), 20)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if ctx.Err() != nil {
			s.requeue(job.ID)
			continue
		}
		s.executeJob(ctx, job)
	}
	return nil
}

func (s *DeliveryService) executeJob(ctx context.Context, job domain.DueJob) {
	timeout := job.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, job.CallbackURL, bytes.NewReader(job.Payload))
	if err != nil {
		s.settleFailure(ctx, job, 0, "invalid callback url")
		return
	}

	sig := SignPayload(job.Secret, ts, job.Payload)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "event-dispatcher/1")
	req.Header.Set("X-Dispatcher-Event-Id", job.EventID)
	req.Header.Set("X-Dispatcher-Timestamp", ts)
	req.Header.Set("X-Dispatcher-Signature", sig)

	resp, err := s.client.Do(req)
	if ctx.Err() != nil {
		s.requeue(job.ID)
		return
	}
	if err != nil {
		s.settleFailure(ctx, job, 0, "webhook request error")
		return
	}
	defer resp.Body.Close()

	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	_, _ = io.Copy(io.Discard, resp.Body)

	now := time.Now().UTC()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := s.repo.Settle(ctx, job.ID, domain.DeliveryStatusSuccess, resp.StatusCode, "", now, now); err != nil {
			s.logger.Error("failed to settle delivery success", slog.String("deliveryId", job.ID), slog.String("error", err.Error()))
			return
		}
		s.logger.Info("delivered successfully",
			slog.String("deliveryId", job.ID),
			slog.String("eventId", job.EventID),
			slog.Int("attempt", job.Attempt),
			slog.Int("statusCode", resp.StatusCode),
		)
		return
	}

	errMsg := fmt.Sprintf("http status %d", resp.StatusCode)
	if text := sanitizeString(string(snippet)); text != "" {
		if len(text) > 128 {
			text = text[:128]
		}
		errMsg += ": " + text
	}
	s.settleFailure(ctx, job, resp.StatusCode, errMsg)
}

func (s *DeliveryService) settleFailure(ctx context.Context, job domain.DueJob, statusCode int, errMsg string) {
	now := time.Now().UTC()
	status := domain.DeliveryStatusPending
	next := now.Add(CalculateBackoff(job.Attempt))
	if job.Attempt >= job.MaxAttempts {
		status = domain.DeliveryStatusFailed
		next = now
	}

	if err := s.repo.Settle(ctx, job.ID, status, statusCode, errMsg, next, now); err != nil {
		s.logger.Error("failed to settle delivery failure", slog.String("deliveryId", job.ID), slog.String("error", err.Error()))
		return
	}

	s.logger.Warn("delivery attempt failed",
		slog.String("deliveryId", job.ID),
		slog.String("eventId", job.EventID),
		slog.Int("attempt", job.Attempt),
		slog.Int("statusCode", statusCode),
		slog.String("finalStatus", status),
	)
}

func (s *DeliveryService) requeue(deliveryID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.repo.RequeueUnsent(ctx, deliveryID, time.Now().UTC())
}

// Reset 手动重放失败的投递任务。
func (s *DeliveryService) Reset(ctx context.Context, id string) (*DeliveryDTO, error) {
	now := time.Now().UTC()
	if err := s.repo.Reset(ctx, id, now); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *DeliveryService) Get(ctx context.Context, id string) (*DeliveryDTO, error) {
	dlv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toDeliveryDTO(dlv), nil
}

func (s *DeliveryService) List(ctx context.Context, eventID, status string, limit int) ([]*DeliveryDTO, error) {
	list, err := s.repo.List(ctx, eventID, status, limit)
	if err != nil {
		return nil, err
	}
	res := make([]*DeliveryDTO, 0, len(list))
	for _, item := range list {
		res = append(res, toDeliveryDTO(item))
	}
	return res, nil
}

// SignPayload 计算 HMAC-SHA256 签名。
func SignPayload(secret, timestamp string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// CalculateBackoff 计算指数阶梯退避间隔。
func CalculateBackoff(attempt int) time.Duration {
	steps := []time.Duration{
		1 * time.Second,
		5 * time.Second,
		30 * time.Second,
		2 * time.Minute,
		10 * time.Minute,
	}
	idx := attempt - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(steps) {
		idx = len(steps) - 1
	}
	return steps[idx]
}

func sanitizeString(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.TrimSpace(s)
}

func toDeliveryDTO(d *domain.Delivery) *DeliveryDTO {
	if d == nil {
		return nil
	}
	return &DeliveryDTO{
		ID:             d.ID,
		EventID:        d.EventID,
		SubscriberID:   d.SubscriberID,
		SubscriberName: d.SubscriberName,
		Status:         d.Status,
		Attempt:        d.Attempt,
		NextRetryAt:    d.NextRetryAt.Format(time.RFC3339),
		LastStatusCode: d.LastStatusCode,
		LastError:      d.LastError,
		UpdatedAt:      d.UpdatedAt.Format(time.RFC3339),
	}
}
