package handler

import (
	"net/http"

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
	limit := parseLimit(r.URL.Query().Get("limit"), 50)
	eventID := r.URL.Query().Get("eventId")
	status := r.URL.Query().Get("status")

	list, err := h.svc.List(r.Context(), eventID, status, limit)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, r, map[string]any{"items": list})
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
