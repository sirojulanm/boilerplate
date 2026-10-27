package main

import (
	"context"
	"errors"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	_ "github.com/sirojulanam/opencodeproject/docs"
	"github.com/sirojulanam/opencodeproject/internal/delivery/httpapi"
	"github.com/sirojulanam/opencodeproject/internal/infrastructure/checker"
	"github.com/sirojulanam/opencodeproject/internal/infrastructure/logger"
	"github.com/sirojulanam/opencodeproject/internal/usecase"
)

const (
	serviceName = "opencodeproject"
	version     = "1.0.0"
)

// @title           OpenCode Project API
// @version         1.0
// @description     Simple health-check API built with a clean, interface-driven architecture.
// @schemes         http
// @host            localhost:1111
// @BasePath        /api/v1
func main() {
	startedAt := time.Now()
	port := env("PORT", "1111")

	appLog, err := logger.New(logger.Config{
		Level:    env("LOG_LEVEL", "info"),
		Dir:      env("LOG_DIR", "logs"),
		Filename: "app.log",
	})
	if err != nil {
		// A logger could not be built, so fall back to a plain logrus instance.
		logrus.New().WithError(err).Fatal("failed to initialize logger")
	}
	defer func() { _ = appLog.Close() }()

	appLog.WithFields(logrus.Fields{
		"service": serviceName,
		"version": version,
		"port":    port,
	}).Info("starting service")

	healthUsecase := usecase.NewHealth(
		serviceName,
		version,
		checker.NewRuntime(startedAt),
	)
	router := httpapi.NewRouter(httpapi.NewHealthHandler(healthUsecase), appLog.Logger)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		// net/http internal errors are forwarded to logrus as well.
		ErrorLog: stdlog.New(appLog.WriterLevel(logrus.ErrorLevel), "", 0),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		appLog.WithFields(logrus.Fields{
			"addr":    server.Addr,
			"health":  "http://localhost:" + port + "/api/v1/health",
			"swagger": "http://localhost:" + port + "/swagger/index.html",
		}).Info("server started")

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLog.WithError(err).Error("server failed")
			stop()
		}
	}()

	<-ctx.Done()
	appLog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		appLog.WithError(err).Error("graceful shutdown failed")
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
