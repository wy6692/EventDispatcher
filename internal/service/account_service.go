package service

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"eventdispatcher/internal/domain"
	"eventdispatcher/pkg/errcode"
)

var validKeyRegex = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// AccountDTO 用于前端请求与响应的数据传输对象。
type AccountDTO struct {
	ID                string `json:"id,omitempty"`
	Name              string `json:"name"`
	AccountKey        string `json:"accountKey"`
	AppID             string `json:"appId"`
	Token             string `json:"token,omitempty"`
	EncodingAESKey    string `json:"encodingAesKey,omitempty"`
	TokenSet          bool   `json:"tokenSet,omitempty"`
	EncodingAESKeySet bool   `json:"encodingAesKeySet,omitempty"`
	CreatedAt         string `json:"createdAt,omitempty"`
	UpdatedAt         string `json:"updatedAt,omitempty"`
}

type AccountService struct {
	repo domain.AccountRepository
}

func NewAccountService(repo domain.AccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

func (s *AccountService) Create(ctx context.Context, dto AccountDTO) (*AccountDTO, error) {
	name := strings.TrimSpace(dto.Name)
	key := strings.TrimSpace(dto.AccountKey)
	appID := strings.TrimSpace(dto.AppID)
	token := dto.Token
	aesKey := dto.EncodingAESKey

	if err := validateAccountFields(name, key, appID, token, aesKey); err != nil {
		return nil, err
	}

	acc := &domain.Account{
		Name:           name,
		AccountKey:     key,
		AppID:          appID,
		Token:          token,
		EncodingAESKey: aesKey,
	}

	if err := s.repo.Create(ctx, acc); err != nil {
		return nil, err
	}

	return toAccountDTO(acc), nil
}

func (s *AccountService) Get(ctx context.Context, id string) (*AccountDTO, error) {
	acc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toAccountDTO(acc), nil
}

func (s *AccountService) GetByKey(ctx context.Context, key string) (*domain.Account, error) {
	return s.repo.GetByKey(ctx, key)
}

func (s *AccountService) List(ctx context.Context) ([]*AccountDTO, error) {
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]*AccountDTO, 0, len(list))
	for _, acc := range list {
		res = append(res, toAccountDTO(acc))
	}
	return res, nil
}

func (s *AccountService) Update(ctx context.Context, id string, dto AccountDTO) (*AccountDTO, error) {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if dto.Name != "" {
		current.Name = strings.TrimSpace(dto.Name)
	}
	if dto.AccountKey != "" {
		current.AccountKey = strings.TrimSpace(dto.AccountKey)
	}
	if dto.AppID != "" {
		current.AppID = strings.TrimSpace(dto.AppID)
	}
	if dto.Token != "" {
		current.Token = dto.Token
	}
	if dto.EncodingAESKey != "" {
		current.EncodingAESKey = dto.EncodingAESKey
	}

	if err := validateAccountFields(current.Name, current.AccountKey, current.AppID, current.Token, current.EncodingAESKey); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, current); err != nil {
		return nil, err
	}

	return toAccountDTO(current), nil
}

func (s *AccountService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func validateAccountFields(name, key, appID, token, aesKey string) error {
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 64 {
		return errcode.ErrInvalidParam.WithMessage("name 长度必须在 1 到 64 之间")
	}
	if !validKeyRegex.MatchString(key) {
		return errcode.ErrInvalidParam.WithMessage("accountKey 格式无效，只允许 1-64 位字母、数字、下划线和连字符")
	}
	if len(appID) < 1 || len(appID) > 64 {
		return errcode.ErrInvalidParam.WithMessage("appId 长度必须在 1 到 64 之间")
	}
	if len(token) < 3 || len(token) > 32 {
		return errcode.ErrInvalidParam.WithMessage("token 长度必须在 3 到 32 之间")
	}
	if aesKey != "" && len(aesKey) != 43 {
		return errcode.ErrInvalidParam.WithMessage("encodingAesKey 必须是 43 位字符")
	}
	return nil
}

func toAccountDTO(a *domain.Account) *AccountDTO {
	if a == nil {
		return nil
	}
	return &AccountDTO{
		ID:                a.ID,
		Name:              a.Name,
		AccountKey:        a.AccountKey,
		AppID:             a.AppID,
		TokenSet:          a.Token != "",
		EncodingAESKeySet: a.EncodingAESKey != "",
		CreatedAt:         a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:         a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
