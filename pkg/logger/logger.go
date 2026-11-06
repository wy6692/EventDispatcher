package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
)

type ctxKey string

const traceIDKey ctxKey = "trace_id"

// WithTraceID 将链路追踪 ID 注入上下文。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// GetTraceID 从上下文中获取链路追踪 ID。
func GetTraceID(ctx context.Context) string {
	if val, ok := ctx.Value(traceIDKey).(string); ok {
		return val
	}
	return ""
}

// Handler 是支持从 Context 自动提取 trace_id 属性的 slog.Handler。
type Handler struct {
	slog.Handler
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if traceID := GetTraceID(ctx); traceID != "" {
		r.AddAttrs(slog.String("traceId", traceID))
	}
	return h.Handler.Handle(ctx, r)
}

// New 创建企业级结构化日志实例。
func New(w io.Writer, level slog.Level, jsonFormat bool) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}
	opts := &slog.HandlerOptions{
		Level: level,
	}
	var base slog.Handler
	if jsonFormat {
		base = slog.NewJSONHandler(w, opts)
	} else {
		base = slog.NewTextHandler(w, opts)
	}
	return slog.New(&Handler{Handler: base})
}
