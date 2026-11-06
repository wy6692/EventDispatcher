package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:web
var webFS embed.FS

// Handler 返回开箱即用的 Web 管理控制台静态资源文件服务。
func Handler() http.Handler {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. 若请求 /ui，标准化重定向到 /ui/，保证前端相对路径 (css/..., js/...) 能够正确解析
		if r.URL.Path == "/ui" {
			http.Redirect(w, r, "/ui/", http.StatusFound)
			return
		}

		// 2. 剥离 /ui 前缀后转交给内置静态文件服务
		p := strings.TrimPrefix(r.URL.Path, "/ui")
		if p == "" {
			p = "/"
		}

		r2 := new(http.Request)
		*r2 = *r
		urlCopy := *r.URL
		urlCopy.Path = p
		r2.URL = &urlCopy

		// 增加无缓存头，避免前端修改后浏览器强缓存
		w.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(w, r2)
	})
}
