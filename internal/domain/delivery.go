package domain

import (
	"context"
	"time"
)

// 投递任务状态枚举
const (
	DeliveryStatusPending  = "pending"  // 待投递/重试等待中
	DeliveryStatusInflight = "inflight" // 投递中
	DeliveryStatusSuccess  = "success"  // 投递成功 (2xx)
	DeliveryStatusFailed   = "failed"   // 已达最大重试次数，彻底失败
)

// Delivery 表示向特定订阅方投递事件的任务状态机实体。
type Delivery struct {
	ID             string    `json:"id"`
	EventID        string    `json:"eventId"`
	SubscriberID   string    `json:"subscriberId"`
	SubscriberName string    `json:"subscriberName,omitempty"`
	Status         string    `json:"status"`
	Attempt        int       `json:"attempt"`
	NextRetryAt    time.Time `json:"nextRetryAt"`
	LastStatusCode int       `json:"lastStatusCode"`
	LastError      string    `json:"lastError"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// DueJob 表示已被调度器抢占、准备立即出站投递的完整任务。
type DueJob struct {
	ID           string
	EventID      string
	SubscriberID string
	CallbackURL  string
	Secret       string
	Timeout      time.Duration
	MaxAttempts  int
	Attempt      int
	Payload      []byte
}

// DeliveryRepository 定义投递状态机持久化契约接口。
type DeliveryRepository interface {
	ClaimDue(ctx context.Context, now time.Time, limit int) ([]DueJob, error)
	Settle(ctx context.Context, id, status string, statusCode int, errMsg string, nextRetryAt, updatedAt time.Time) error
	RequeueInflight(ctx context.Context, now time.Time) (int64, error)
	RequeueUnsent(ctx context.Context, id string, now time.Time) error
	Reset(ctx context.Context, id string, now time.Time) error
	GetByID(ctx context.Context, id string) (*Delivery, error)
	List(ctx context.Context, eventID, status string, limit int) ([]*Delivery, error)
}
