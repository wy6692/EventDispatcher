package service

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"eventdispatcher/internal/domain"
	"eventdispatcher/internal/wechat"
	"eventdispatcher/pkg/errcode"
	"eventdispatcher/pkg/id"
)

// IngestService 处理微信网关事件接收、解密验签与广播派发。
type IngestService struct {
	accountRepo    domain.AccountRepository
	subscriberRepo domain.SubscriberRepository
	eventRepo      domain.EventRepository
	logger         *slog.Logger
}

func NewIngestService(
	accRepo domain.AccountRepository,
	subRepo domain.SubscriberRepository,
	evtRepo domain.EventRepository,
	logger *slog.Logger,
) *IngestService {
	if logger == nil {
		logger = slog.Default()
	}
	return &IngestService{
		accountRepo:    accRepo,
		subscriberRepo: subRepo,
		eventRepo:      evtRepo,
		logger:         logger,
	}
}

// VerifyURL 处理微信首次接入验证 (GET /w/{accountKey})。
func (s *IngestService) VerifyURL(ctx context.Context, accountKey, signature, timestamp, nonce, echostr, encryptType, msgSignature string) (string, error) {
	acc, err := s.accountRepo.GetByKey(ctx, accountKey)
	if err != nil {
		return "", err
	}
	cleanEcho := wechat.CleanBase64(echostr)
	isEncrypted := strings.EqualFold(encryptType, "aes") || msgSignature != ""

	if isEncrypted {
		if acc.EncodingAESKey == "" {
			return "", errcode.ErrWeChatMissingKey
		}
		if !validPlainSignature(acc.Token, timestamp, nonce, signature) ||
			!wechat.ValidSignature(wechat.Signature(acc.Token, timestamp, nonce, cleanEcho), msgSignature) {
			return "", errcode.ErrWeChatBadSignature
		}
		plain, err := wechat.Decrypt(acc.EncodingAESKey, acc.AppID, cleanEcho)
		if err != nil {
			return "", errcode.ErrWeChatDecryptFail
		}
		return string(plain), nil
	}

	if !validPlainSignature(acc.Token, timestamp, nonce, signature) {
		return "", errcode.ErrWeChatBadSignature
	}
	return cleanEcho, nil
}

// IngestPayload 处理微信事件推送 (POST /w/{accountKey})。
func (s *IngestService) IngestPayload(ctx context.Context, accountKey, signature, timestamp, nonce, encryptType, msgSignature string, body []byte) (bool, error) {
	acc, err := s.accountRepo.GetByKey(ctx, accountKey)
	if err != nil {
		return false, err
	}

	fields, err := wechat.ParseFields(body)
	if err != nil {
		return false, errcode.ErrWeChatBadXML
	}

	if strings.EqualFold(encryptType, "aes") {
		if acc.EncodingAESKey == "" {
			return false, errcode.ErrWeChatMissingKey
		}
		enc := wechat.CleanBase64(fields["Encrypt"])
		if enc == "" {
			return false, errcode.ErrWeChatBadXML
		}
		if !validPlainSignature(acc.Token, timestamp, nonce, signature) ||
			!wechat.ValidSignature(wechat.Signature(acc.Token, timestamp, nonce, enc), msgSignature) {
			return false, errcode.ErrWeChatBadSignature
		}
		plain, err := wechat.Decrypt(acc.EncodingAESKey, acc.AppID, enc)
		if err != nil {
			return false, errcode.ErrWeChatDecryptFail
		}
		fields, err = wechat.ParseFields(plain)
		if err != nil {
			return false, errcode.ErrWeChatBadXML
		}
	} else if !validPlainSignature(acc.Token, timestamp, nonce, signature) {
		return false, errcode.ErrWeChatBadSignature
	}

	msg, err := wechat.MessageFromFields(fields)
	if err != nil {
		return false, errcode.ErrBadRequest.WithMessage("微信消息字段解析失败")
	}

	eventID, err := id.New("evt_")
	if err != nil {
		return false, err
	}
	receivedAt := time.Now().UTC()

	outboundPayload, err := marshalOutboundEvent(eventID, acc.AccountKey, acc.AppID, receivedAt, msg)
	if err != nil {
		return false, err
	}

	matchedSubIDs, err := s.matchSubscribers(ctx, acc.ID, msg)
	if err != nil {
		return false, err
	}

	evt := &domain.Event{
		ID:          eventID,
		AccountID:   acc.ID,
		AccountKey:  acc.AccountKey,
		AppID:       acc.AppID,
		DedupeKey:   wechat.DedupeKey(acc.AppID, msg),
		MsgType:     msg.MsgType,
		Event:       msg.Event,
		EventKey:    msg.EventKey,
		FromUser:    msg.FromUser,
		ToUser:      msg.ToUser,
		CreateTime:  msg.CreateTime,
		PayloadJSON: outboundPayload,
		ReceivedAt:  receivedAt,
	}

	created, err := s.eventRepo.SaveIncoming(ctx, evt, matchedSubIDs)
	if err != nil {
		return false, err
	}

	s.logger.InfoContext(ctx, "ingest event processed",
		slog.String("accountKey", acc.AccountKey),
		slog.String("msgType", msg.MsgType),
		slog.String("event", msg.Event),
		slog.Bool("duplicate", !created),
		slog.Int("matchedSubscribers", len(matchedSubIDs)),
	)

	return created, nil
}

func (s *IngestService) matchSubscribers(ctx context.Context, accountID string, msg wechat.Message) ([]string, error) {
	subs, err := s.subscriberRepo.ListEnabledByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	var matchedIDs []string
	for _, sub := range subs {
		filter, err := domain.ParseFilter(sub.FilterJSON)
		if err != nil {
			continue
		}
		if filter.Matches(msg.MsgType, msg.Event, msg.EventKey) {
			matchedIDs = append(matchedIDs, sub.ID)
		}
	}
	return matchedIDs, nil
}

func validPlainSignature(token, timestamp, nonce, signature string) bool {
	if signature == "" {
		return false
	}
	return wechat.ValidSignature(wechat.Signature(token, timestamp, nonce), signature)
}

func marshalOutboundEvent(eventID, accountKey, appID string, receivedAt time.Time, msg wechat.Message) ([]byte, error) {
	data := map[string]any{
		"id":         eventID,
		"accountKey": accountKey,
		"appId":      appID,
		"msgType":    msg.MsgType,
		"event":      msg.Event,
		"eventKey":   msg.EventKey,
		"fromUser":   msg.FromUser,
		"toUser":     msg.ToUser,
		"createTime": msg.CreateTime,
		"receivedAt": receivedAt.Format(time.RFC3339),
		"fields":     msg.Fields,
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(data); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
