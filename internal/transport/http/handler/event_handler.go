package handler

import (
	"net/http"
	"strconv"

	"eventdispatcher/internal/domain"
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
	page := parsePage(r.URL.Query().Get("page"))
	limit := parseLimit(r.URL.Query().Get("limit"), 15)
	filter := domain.EventFilter{
		AccountID: r.URL.Query().Get("accountId"),
		MsgType:   r.URL.Query().Get("msgType"),
		Event:     r.URL.Query().Get("event"),
		FromUser:  r.URL.Query().Get("fromUser"),
		EventID:   r.URL.Query().Get("eventId"),
	}
	list, total, err := h.svc.List(r.Context(), filter, page, limit)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	totalPages := 1
	if total > 0 && limit > 0 {
		totalPages = (total + limit - 1) / limit
	}
	response.Success(w, r, map[string]any{
		"items":      list,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
	})
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

func (h *EventHandler) Overview(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.GetOverviewStats(r.Context())
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, r, stats)
}

func parsePage(raw string) int {
	if raw == "" {
		return 1
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 1
	}
	return n
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
