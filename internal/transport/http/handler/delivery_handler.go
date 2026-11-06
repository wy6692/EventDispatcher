package handler

import (
	"net/http"

	"eventdispatcher/internal/domain"
	"eventdispatcher/internal/service"
	"eventdispatcher/internal/transport/http/response"
)

type DeliveryHandler struct {
	svc *service.DeliveryService
}

func NewDeliveryHandler(svc *service.DeliveryService) *DeliveryHandler {
	return &DeliveryHandler{svc: svc}
}

func (h *DeliveryHandler) List(w http.ResponseWriter, r *http.Request) {
	page := parsePage(r.URL.Query().Get("page"))
	limit := parseLimit(r.URL.Query().Get("limit"), 15)
	filter := domain.DeliveryFilter{
		EventID:      r.URL.Query().Get("eventId"),
		SubscriberID: r.URL.Query().Get("subscriberId"),
		Status:       r.URL.Query().Get("status"),
		DeliveryID:   r.URL.Query().Get("deliveryId"),
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

func (h *DeliveryHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := h.svc.Get(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, r, res)
}

func (h *DeliveryHandler) Retry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := h.svc.Reset(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, r, res)
}
