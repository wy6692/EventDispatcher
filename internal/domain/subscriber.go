package domain

import (
	"context"
	"time"
)

// Subscriber 表示下游订阅方领域实体。
type Subscriber struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"accountId"`
	Name        string    `json:"name"`
	CallbackURL string    `json:"callbackUrl"`
	Secret      string    `json:"-"`
	Enabled     bool      `json:"enabled"`
	TimeoutMs   int       `json:"timeoutMs"`
	MaxAttempts int       `json:"maxAttempts"`
	FilterJSON  string    `json:"filterJson"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Timeout 获取实际请求超时时间。
func (s *Subscriber) Timeout() time.Duration {
	if s.TimeoutMs <= 0 {
		return 3 * time.Second
	}
	return time.Duration(s.TimeoutMs) * time.Millisecond
}

// SubscriberRepository 定义订阅方持久化契约接口。
type SubscriberRepository interface {
	Create(ctx context.Context, sub *Subscriber) error
	GetByID(ctx context.Context, id string) (*Subscriber, error)
	ListByAccount(ctx context.Context, accountID string) ([]*Subscriber, error)
	ListEnabledByAccount(ctx context.Context, accountID string) ([]*Subscriber, error)
	Update(ctx context.Context, sub *Subscriber) error
	Delete(ctx context.Context, id string) error
}
