package http

import (
	"log/slog"
	"net/http"

	"eventdispatcher/internal/transport/http/handler"
	"eventdispatcher/internal/transport/http/middleware"
	"eventdispatcher/internal/transport/http/ui"
)

type RouterConfig struct {
	AdminToken string
	Logger     *slog.Logger

	AccountHandler    *handler.AccountHandler
	SubscriberHandler *handler.SubscriberHandler
	EventHandler      *handler.EventHandler
	DeliveryHandler   *handler.DeliveryHandler
	IngestHandler     *handler.IngestHandler
	HealthHandler     *handler.HealthHandler
}

// NewRouter 装配全站路由、全局中间件及鉴权过滤器。
func NewRouter(cfg RouterConfig) http.Handler {
	mux := http.NewServeMux()

	// 1. 公开端点
	mux.HandleFunc("GET /healthz", cfg.HealthHandler.Health)
	mux.HandleFunc("/w/{accountKey}", cfg.IngestHandler.ServeHTTP)

	// Web 管理控制台 (内置多模块静态控制台，无需外部构建工具与 Node.js)
	uiHandler := ui.Handler()
	mux.Handle("GET /ui", uiHandler)
	mux.Handle("GET /ui/", uiHandler)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusFound)
	})

	// 2. 受保护的管理端点 (套用 AdminAuth 中间件)
	adminAuth := middleware.AdminAuth(cfg.AdminToken)

	// Accounts
	mux.Handle("POST /admin/accounts", adminAuth(http.HandlerFunc(cfg.AccountHandler.Create)))
	mux.Handle("GET /admin/accounts", adminAuth(http.HandlerFunc(cfg.AccountHandler.List)))
	mux.Handle("GET /admin/accounts/{id}", adminAuth(http.HandlerFunc(cfg.AccountHandler.Get)))
	mux.Handle("PUT /admin/accounts/{id}", adminAuth(http.HandlerFunc(cfg.AccountHandler.Update)))
	mux.Handle("DELETE /admin/accounts/{id}", adminAuth(http.HandlerFunc(cfg.AccountHandler.Delete)))

	// Subscribers
	mux.Handle("POST /admin/subscribers", adminAuth(http.HandlerFunc(cfg.SubscriberHandler.Create)))
	mux.Handle("POST /admin/subscribers/test", adminAuth(http.HandlerFunc(cfg.SubscriberHandler.TestDirect)))
	mux.Handle("GET /admin/subscribers", adminAuth(http.HandlerFunc(cfg.SubscriberHandler.List)))
	mux.Handle("GET /admin/subscribers/{id}", adminAuth(http.HandlerFunc(cfg.SubscriberHandler.Get)))
	mux.Handle("POST /admin/subscribers/{id}/test", adminAuth(http.HandlerFunc(cfg.SubscriberHandler.Test)))
	mux.Handle("PUT /admin/subscribers/{id}", adminAuth(http.HandlerFunc(cfg.SubscriberHandler.Update)))
	mux.Handle("DELETE /admin/subscribers/{id}", adminAuth(http.HandlerFunc(cfg.SubscriberHandler.Delete)))

	// Events
	mux.Handle("GET /admin/events", adminAuth(http.HandlerFunc(cfg.EventHandler.List)))
	mux.Handle("GET /admin/events/{id}", adminAuth(http.HandlerFunc(cfg.EventHandler.Get)))

	// Deliveries
	mux.Handle("GET /admin/deliveries", adminAuth(http.HandlerFunc(cfg.DeliveryHandler.List)))
	mux.Handle("GET /admin/deliveries/{id}", adminAuth(http.HandlerFunc(cfg.DeliveryHandler.Get)))
	mux.Handle("POST /admin/deliveries/{id}/retry", adminAuth(http.HandlerFunc(cfg.DeliveryHandler.Retry)))

	// 3. 全局中间件链：TraceID -> Recovery
	var handler http.Handler = mux
	handler = middleware.Recovery(cfg.Logger)(handler)
	handler = middleware.Trace(handler)

	return handler
}
