package logger

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/mattn/go-isatty"
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

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{Handler: h.Handler.WithGroup(name)}
}

// New 创建日志实例。文本模式输出对齐的控制台格式；jsonFormat 为 true 时输出 JSON。
func New(w io.Writer, level slog.Level, jsonFormat bool) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}
	var base slog.Handler
	if jsonFormat {
		base = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	} else {
		color := wantColor(w)
		if color {
			enableANSI(w)
		}
		base = newConsoleHandler(w, level, color)
	}
	return slog.New(&Handler{Handler: base})
}

func wantColor(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}
