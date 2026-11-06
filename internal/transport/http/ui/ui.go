package ui

import (
	_ "embed"
	"net/http"
)

//go:embed index.html
var IndexHTML []byte

// Handler 返回开箱即用的 Web 管理控制台静态单页面。
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(IndexHTML)
	}
}
