package main

import (
	"context"
	"errors"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	_ "github.com/sirojulanam/boilerplate/docs"
	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirojulanam/boilerplate/internal/delivery/httpapi"
	"github.com/sirojulanam/boilerplate/internal/helper"
	"github.com/sirojulanam/boilerplate/internal/infrastructure/checker"
	"github.com/sirojulanam/boilerplate/internal/infrastructure/logger"
	"github.com/sirojulanam/boilerplate/internal/usecase"
)

// version is overridden at build time:
//
//	go build -ldflags "-X main.version=1.2.3" ./cmd/api
var version = constant.DefaultVersion

// @title           Boilerplate API
// @version         1.0
// @description     Generic health-check API template built with a clean, interface-driven architecture.
// @schemes         http
// @host            localhost:1111
// @BasePath        /api/v1
func main() {
	appLog, err := logger.New(logger.Config{
		Level:    helper.Env("LOG_LEVEL", constant.DefaultLogLevel),
		Dir:      helper.Env("LOG_DIR", constant.DefaultLogDir),
		Filename: constant.DefaultLogFilename,
	})
	if err != nil {
		// The file logger is unavailable, so emit a JSON logrus entry to stdout.
		fallback := logrus.New()
		fallback.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339})
		fallback.WithError(err).Error("initialize logger")
		os.Exit(1)
	}
	defer func() { _ = appLog.Close() }()

	if err := run(appLog); err != nil {
		appLog.WithError(err).Error("service stopped")
		_ = appLog.Close()
		os.Exit(1)
	}
}

func run(appLog *logger.Logger) error {
	startedAt := time.Now()
	port := helper.Env("PORT", constant.DefaultPort)
	serviceName := helper.Env("SERVICE_NAME", constant.DefaultServiceName)

	appLog.WithFields(logrus.Fields{
		"service": serviceName,
		"version": version,
		"port":    port,
	}).Info("starting service")

	httpapi.Configure(appLog.Logger)

	healthUsecase := usecase.NewHealth(
		serviceName,
		version,
		checker.NewRuntime(startedAt),
	)
	router := httpapi.NewRouter(
		httpapi.NewHealthHandler(healthUsecase, appLog.Logger),
		appLog.Logger,
	)

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

	// Bind before announcing startup so a busy port is reported as a real
	// failure instead of a misleading "server started" entry.
	listener, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", server.Addr, err)
	}

	appLog.WithFields(logrus.Fields{
		"addr":    server.Addr,
		"health":  "http://localhost:" + port + constant.HealthPath,
		"swagger": "http://localhost:" + port + constant.SwaggerUIPath,
	}).Info("server started")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()

	select {
	case <-ctx.Done():
		appLog.Info("shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	}
}
