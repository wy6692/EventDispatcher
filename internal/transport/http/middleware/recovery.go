package middleware

import (
	"log/slog"
	"net/http"

	"eventdispatcher/internal/transport/http/response"
	"eventdispatcher/pkg/errcode"
)

// Recovery 捕获 panic 并转换为 500 统一响应。
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if logger != nil {
						logger.ErrorContext(r.Context(), "panic recovered in http handler",
							slog.Any("recover", rec),
							slog.String("path", r.URL.Path),
						)
					}
					response.Error(w, r, errcode.ErrInternalServer)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
