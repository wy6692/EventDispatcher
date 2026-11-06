package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"eventdispatcher/pkg/logger"
)

// Trace 链路追踪中间件，自动提取或生成 X-Trace-Id 并注入请求上下文与响应头。
func Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-Id")
		if traceID == "" {
			var b [12]byte
			_, _ = rand.Read(b[:])
			traceID = hex.EncodeToString(b[:])
		}
		w.Header().Set("X-Trace-Id", traceID)
		ctx := logger.WithTraceID(r.Context(), traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
