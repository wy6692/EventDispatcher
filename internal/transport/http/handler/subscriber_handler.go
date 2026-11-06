package handler

import (
	"net/http"

	"eventdispatcher/internal/service"
	"eventdispatcher/internal/transport/http/response"
	"eventdispatcher/pkg/errcode"
)

type SubscriberHandler struct {
	svc *service.SubscriberService
}

func NewSubscriberHandler(svc *service.SubscriberService) *SubscriberHandler {
	return &SubscriberHandler{svc: svc}
}

func (h *SubscriberHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto service.SubscriberDTO
	if err := decodeJSON(r, &dto); err != nil {
		response.Error(w, r, errcode.ErrInvalidParam.WithMessage("请求 JSON 无效"))
		return
	}
	res, err := h.svc.Create(r.Context(), dto)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Created(w, r, res)
}

func (h *SubscriberHandler) List(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("accountId")
	list, err := h.svc.ListByAccount(r.Context(), accountID)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, r, map[string]any{"items": list})
}

func (h *SubscriberHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := h.svc.Get(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, r, res)
}

func (h *SubscriberHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var dto service.SubscriberDTO
	if err := decodeJSON(r, &dto); err != nil {
		response.Error(w, r, errcode.ErrInvalidParam.WithMessage("请求 JSON 无效"))
		return
	}
	res, err := h.svc.Update(r.Context(), id, dto)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, r, res)
}

func (h *SubscriberHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
