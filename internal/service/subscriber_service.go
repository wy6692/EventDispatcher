package service

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"unicode/utf8"

	"eventdispatcher/internal/domain"
	"eventdispatcher/pkg/errcode"
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
