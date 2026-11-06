package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"eventdispatcher/internal/service"
	"eventdispatcher/internal/transport/http/response"
	"eventdispatcher/pkg/errcode"
)

type AccountHandler struct {
	svc *service.AccountService
}

func NewAccountHandler(svc *service.AccountService) *AccountHandler {
	return &AccountHandler{svc: svc}
}

func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto service.AccountDTO
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

func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context())
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, r, map[string]any{"items": list})
}

func (h *AccountHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := h.svc.Get(r.Context(), id)
	if err != nil {
		response.Error(w, r, err)
		return
	}
	response.Success(w, r, res)
}

func (h *AccountHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var dto service.AccountDTO
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

func (h *AccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		response.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	return dec.Decode(dst)
}
