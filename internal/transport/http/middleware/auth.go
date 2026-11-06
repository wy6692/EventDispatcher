package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"eventdispatcher/internal/transport/http/response"
	"eventdispatcher/pkg/errcode"
)

// AdminAuth 管理后台鉴权中间件，基于恒定时间比较防时序攻击。
func AdminAuth(expectedToken string) func(http.Handler) http.Handler {
	expectedHash := sha256.Sum256([]byte(expectedToken))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			tokenHash := sha256.Sum256([]byte(token))
			if subtle.ConstantTimeCompare(tokenHash[:], expectedHash[:]) != 1 {
				response.Error(w, r, errcode.ErrUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractBearerToken(r *http.Request) string {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
