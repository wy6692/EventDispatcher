package domain

import (
	"context"
	"time"
)

// Event 表示从微信网关接收到的标准化事件实体。
type Event struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"accountId"`
	AccountKey  string    `json:"accountKey,omitempty"`
	AppID       string    `json:"appId,omitempty"`
	DedupeKey   string    `json:"dedupeKey"`
	MsgType     string    `json:"msgType"`
	Event       string    `json:"event"`
	EventKey    string    `json:"eventKey"`
	FromUser    string    `json:"fromUser"`
	ToUser      string    `json:"toUser"`
	CreateTime  int64     `json:"createTime"`
	PayloadJSON []byte    `json:"payloadJson"`
	ReceivedAt  time.Time `json:"receivedAt"`
}

// EventFilter 定义事件多条件查询过滤器。
type EventFilter struct {
	AccountID string `json:"accountId,omitempty"`
	MsgType   string `json:"msgType,omitempty"`
	Event     string `json:"event,omitempty"`
	FromUser  string `json:"fromUser,omitempty"`
	EventID   string `json:"eventId,omitempty"`
}

// EventRepository 定义事件持久化契约接口。
type EventRepository interface {
	SaveIncoming(ctx context.Context, event *Event, subscriberIDs []string) (created bool, err error)
	GetByID(ctx context.Context, id string) (*Event, error)
	List(ctx context.Context, filter EventFilter, page, limit int) (items []*Event, total int, err error)
}
