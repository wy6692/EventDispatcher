package domain

import (
	"context"
	"time"
)

// Account 表示微信公众号（服务号/订阅号）领域实体。
type Account struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	AccountKey     string    `json:"accountKey"`
	AppID          string    `json:"appId"`
	Token          string    `json:"-"`
	EncodingAESKey string    `json:"-"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// AccountRepository 定义账号持久化契约接口。
type AccountRepository interface {
	Create(ctx context.Context, account *Account) error
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByKey(ctx context.Context, accountKey string) (*Account, error)
	List(ctx context.Context) ([]*Account, error)
	Update(ctx context.Context, account *Account) error
	Delete(ctx context.Context, id string) error
}
