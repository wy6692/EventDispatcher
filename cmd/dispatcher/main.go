package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eventdispatcher/internal/config"
	"eventdispatcher/internal/infrastructure/persistence"
	"eventdispatcher/internal/service"
	transporthttp "eventdispatcher/internal/transport/http"
	"eventdispatcher/internal/transport/http/handler"
	"eventdispatcher/pkg/logger"
)

func main() {
	appLogger := logger.New(os.Stdout, slog.LevelInfo, false)
	slog.SetDefault(appLogger)

	cfg, err := config.FromEnv()
	if err != nil {
		appLogger.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// 1. 初始化持久化基础设施
	db, err := persistence.Open(cfg.DBPath)
	if err != nil {
		appLogger.Error("failed to initialize sqlite storage", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	// 2. 装配仓储层
	accountRepo := persistence.NewAccountRepository(db)
	subscriberRepo := persistence.NewSubscriberRepository(db)
	eventRepo := persistence.NewEventRepository(db)
	deliveryRepo := persistence.NewDeliveryRepository(db)

	// 3. 装配领域业务服务层
	accountService := service.NewAccountService(accountRepo)
	subscriberService := service.NewSubscriberService(subscriberRepo, accountRepo)
	eventService := service.NewEventService(eventRepo)
	deliveryService := service.NewDeliveryService(deliveryRepo, appLogger)
	ingestService := service.NewIngestService(accountRepo, subscriberRepo, eventRepo, appLogger)

	// 4. 装配传输层 Handler 与 HTTP 路由
	router := transporthttp.NewRouter(transporthttp.RouterConfig{
		AdminToken:        cfg.AdminToken,
		Logger:            appLogger,
		AccountHandler:    handler.NewAccountHandler(accountService),
		SubscriberHandler: handler.NewSubscriberHandler(subscriberService),
		EventHandler:      handler.NewEventHandler(eventService),
		DeliveryHandler:   handler.NewDeliveryHandler(deliveryService),
		IngestHandler:     handler.NewIngestHandler(ingestService),
		HealthHandler:     handler.NewHealthHandler(db),
	})

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 5. 优雅启停机制
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		deliveryService.StartWorker(ctx)
	}()

	go func() {
		<-ctx.Done()
		appLogger.Info("initiating graceful shutdown...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	appLogger.Info("EventDispatcher server started", slog.String("addr", cfg.Addr))
	err = httpServer.ListenAndServe()
	stop()
	<-workerDone

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		appLogger.Error("server terminated with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
	appLogger.Info("EventDispatcher exited cleanly")
}
