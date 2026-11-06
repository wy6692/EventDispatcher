package handler

import (
	"errors"
	"io"
	"net/http"

	"eventdispatcher/internal/service"
	"eventdispatcher/pkg/errcode"
)

type IngestHandler struct {
	svc *service.IngestService
}

func NewIngestHandler(svc *service.IngestService) *IngestHandler {
	return &IngestHandler{svc: svc}
}

func (h *IngestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.verifyURL(w, r)
	case http.MethodPost:
		h.receiveEvent(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *IngestHandler) verifyURL(w http.ResponseWriter, r *http.Request) {
	accountKey := r.PathValue("accountKey")
	q := r.URL.Query()
	timestamp := q.Get("timestamp")
	nonce := q.Get("nonce")
	signature := q.Get("signature")
	echostr := q.Get("echostr")
	encryptType := q.Get("encrypt_type")
	msgSignature := q.Get("msg_signature")

	if timestamp == "" || nonce == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	echo, err := h.svc.VerifyURL(r.Context(), accountKey, signature, timestamp, nonce, echostr, encryptType, msgSignature)
	if err != nil {
		h.writeWeChatError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(echo))
}

func (h *IngestHandler) receiveEvent(w http.ResponseWriter, r *http.Request) {
	accountKey := r.PathValue("accountKey")
	if r.Body == nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil || len(body) == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	timestamp := q.Get("timestamp")
	nonce := q.Get("nonce")
	signature := q.Get("signature")
	encryptType := q.Get("encrypt_type")
	msgSignature := q.Get("msg_signature")

	if timestamp == "" || nonce == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	_, err = h.svc.IngestPayload(r.Context(), accountKey, signature, timestamp, nonce, encryptType, msgSignature, body)
	if err != nil {
		h.writeWeChatError(w, err)
		return
	}

	// 微信官方协议：收到后回复文本 success
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("success"))
}

func (h *IngestHandler) writeWeChatError(w http.ResponseWriter, err error) {
	var appErr *errcode.Error
	if errors.As(err, &appErr) {
		http.Error(w, appErr.Message, appErr.HTTPStatus)
		return
	}
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
