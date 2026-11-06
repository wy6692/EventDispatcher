package handler

import (
	"net/http"
	"strconv"

	"eventdispatcher/internal/service"
	"eventdispatcher/internal/transport/http/response"
)

type EventHandler struct {
	svc *service.EventService
}

func NewEventHandler(svc *service.EventService) *EventHandler {
	return &EventHandler{svc: svc}
}

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 50)
	accountID := r.URL.Query().Get("accountId")
	list, err := h.svc.List(r.Context(), accountID, limit)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, r, map[string]any{"items": list})
}

func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := h.svc.Get(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, r, res)
}

func parseLimit(raw string, defaultLimit int) int {
	if raw == "" {
		return defaultLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 || n > 200 {
		return defaultLimit
	}
	return n
}
