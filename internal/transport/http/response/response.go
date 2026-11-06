package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"eventdispatcher/pkg/errcode"
	"eventdispatcher/pkg/logger"
)

// Response 是企业级对外统一响应体结构。
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	TraceID string `json:"traceId,omitempty"`
}

// JSON 向客户端输出规范 JSON。
func JSON(w http.ResponseWriter, r *http.Request, httpStatus, code int, message string, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(httpStatus)

	traceID := logger.GetTraceID(r.Context())
	resp := Response{
		Code:    code,
		Message: message,
		Data:    data,
		TraceID: traceID,
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(resp)
}

// Success 输出业务成功响应 (HTTP 200, Code 0)。
func Success(w http.ResponseWriter, r *http.Request, data any) {
	JSON(w, r, http.StatusOK, 0, "success", data)
}

// Created 输出资源创建成功响应 (HTTP 201, Code 0)。
func Created(w http.ResponseWriter, r *http.Request, data any) {
	JSON(w, r, http.StatusCreated, 0, "created", data)
}

// Error 根据错误类型自动映射 HTTP 状态码与业务错误码。
func Error(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *errcode.Error
	if errors.As(err, &appErr) {
		JSON(w, r, appErr.HTTPStatus, appErr.Code, appErr.Message, nil)
		return
	}
	JSON(w, r, http.StatusInternalServerError, errcode.ErrInternalServer.Code, "系统内部繁忙，请稍后重试", nil)
}
