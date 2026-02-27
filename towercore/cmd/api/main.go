package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"

	"towercore/internal/api/routes"
	"towercore/internal/infrastructure/config"
	"towercore/internal/infrastructure/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)
	router := routes.NewRouter(cfg, log)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Infof("starting http server on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("failed to shutdown http server: %v", err)
	}

	log.Info("http server stopped")
}
