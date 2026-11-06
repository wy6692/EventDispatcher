package service

import (
	"context"
	"encoding/json"
	"time"

	"eventdispatcher/internal/domain"
)

type EventDTO struct {
	ID         string          `json:"id"`
	AccountID  string          `json:"accountId"`
	DedupeKey  string          `json:"dedupeKey"`
	MsgType    string          `json:"msgType"`
	Event      string          `json:"event"`
	EventKey   string          `json:"eventKey"`
	FromUser   string          `json:"fromUser"`
	ToUser     string          `json:"toUser"`
	CreateTime int64           `json:"createTime"`
	ReceivedAt string          `json:"receivedAt"`
	Payload    json.RawMessage `json:"payload"`
}

type EventService struct {
	repo domain.EventRepository
}

func NewEventService(repo domain.EventRepository) *EventService {
	return &EventService{repo: repo}
}

func (s *EventService) Get(ctx context.Context, id string) (*EventDTO, error) {
	evt, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toEventDTO(evt), nil
}

func (s *EventService) List(ctx context.Context, accountID string, limit int) ([]*EventDTO, error) {
	list, err := s.repo.List(ctx, accountID, limit)
	if err != nil {
		return nil, err
	}
	res := make([]*EventDTO, 0, len(list))
	for _, e := range list {
		res = append(res, toEventDTO(e))
	}
	return res, nil
}

func toEventDTO(e *domain.Event) *EventDTO {
	if e == nil {
		return nil
	}
	return &EventDTO{
		ID:         e.ID,
		AccountID:  e.AccountID,
		DedupeKey:  e.DedupeKey,
		MsgType:    e.MsgType,
		Event:      e.Event,
		EventKey:   e.EventKey,
		FromUser:   e.FromUser,
		ToUser:     e.ToUser,
		CreateTime: e.CreateTime,
		ReceivedAt: e.ReceivedAt.Format(time.RFC3339),
		Payload:    json.RawMessage(e.PayloadJSON),
	}
}
