package handler

import (
	"context"
	"net/http"

	"eventdispatcher/internal/transport/http/response"
	"eventdispatcher/pkg/errcode"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	pinger Pinger
}

func NewHealthHandler(pinger Pinger) *HealthHandler {
	return &HealthHandler{pinger: pinger}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	if h.pinger != nil {
		if err := h.pinger.Ping(r.Context()); err != nil {
			response.Error(w, r, errcode.ErrServiceUnavailable.WithMessage("数据库连接不可用"))
			return
		}
	}
	response.Success(w, r, map[string]string{"status": "ok"})
}
