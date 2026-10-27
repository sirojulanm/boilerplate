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
	"github.com/sirojulanam/boilerplate/internal/config"
	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirojulanam/boilerplate/internal/delivery/httpapi"
	"github.com/sirojulanam/boilerplate/internal/helper"
	"github.com/sirojulanam/boilerplate/internal/infrastructure/checker"
	"github.com/sirojulanam/boilerplate/internal/infrastructure/logger"
	"github.com/sirojulanam/boilerplate/internal/infrastructure/postgres"
	"github.com/sirojulanam/boilerplate/internal/infrastructure/security"
	"github.com/sirojulanam/boilerplate/internal/usecase"
)

// version is overridden at build time:
//
//	go build -ldflags "-X main.version=1.2.3" ./cmd/api
var version = constant.DefaultVersion

// @title           Boilerplate API
// @version         1.0
// @description     Generic PostgreSQL-backed API template with JWT authentication, health checks, and structured logging.
// @schemes         http
// @host            localhost:1111
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT access token.
func main() {
	if err := helper.LoadEnvFile(helper.Env("ENV_FILE", ".env")); err != nil {
		fallbackLogger().WithError(err).Error("load environment file")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fallbackLogger().WithError(err).Error("load configuration")
		os.Exit(1)
	}

	appLog, err := logger.New(logger.Config{
		Level:    cfg.LogLevel,
		Dir:      cfg.LogDir,
		Filename: constant.DefaultLogFilename,
	})
	if err != nil {
		fallbackLogger().WithError(err).Error("initialize logger")
		os.Exit(1)
	}
	defer func() { _ = appLog.Close() }()

	if err := run(cfg, appLog); err != nil {
		appLog.WithError(err).Error("service stopped")
		_ = appLog.Close()
		os.Exit(1)
	}
}

func run(cfg config.Config, appLog *logger.Logger) error {
	startedAt := time.Now()

	appLog.WithFields(logrus.Fields{
		"service":  cfg.ServiceName,
		"version":  version,
		"port":     cfg.Port,
		"database": cfg.Database.Name,
	}).Info("starting service")

	httpapi.Configure(appLog.Logger)

	if cfg.Database.AutoMigrate {
		appLog.Info("applying database migrations")
		if err := postgres.MigrateUp(cfg.Database.URL); err != nil {
			return fmt.Errorf("migrate database: %w", err)
		}
		appLog.Info("database migrations applied")
	}

	pool, err := postgres.Open(context.Background(), postgres.Config{
		URL:            cfg.Database.URL,
		MaxConns:       cfg.Database.MaxConns,
		ConnectTimeout: cfg.Database.ConnectTimeout,
		Log:            appLog.Logger,
	})
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}

	sqlDB, err := pool.DB()
	if err != nil {
		return fmt.Errorf("access database pool: %w", err)
	}
	defer sqlDB.Close()

	appLog.WithFields(logrus.Fields{
		"database":  cfg.Database.Name,
		"max_conns": cfg.Database.MaxConns,
	}).Info("database connected")

	hasher, err := security.NewBcryptHasher(constant.BcryptCost)
	if err != nil {
		return fmt.Errorf("initialize password hasher: %w", err)
	}

	tokens, err := security.NewJWTIssuer(
		[]byte(cfg.Auth.JWTSecret),
		cfg.Auth.JWTIssuer,
		cfg.Auth.JWTAudience,
		cfg.Auth.JWTTTL,
	)
	if err != nil {
		return fmt.Errorf("initialize token issuer: %w", err)
	}

	users := postgres.NewUserRepository(pool)
	authUsecase := usecase.NewAuth(users, hasher, tokens)
	healthUsecase := usecase.NewHealth(
		cfg.ServiceName,
		version,
		checker.NewRuntime(startedAt),
		checker.NewPostgres(sqlDB),
	)

	router := httpapi.NewRouter(httpapi.Dependencies{
		Health:        httpapi.NewHealthHandler(healthUsecase, appLog.Logger),
		Auth:          httpapi.NewAuthHandler(authUsecase, appLog.Logger),
		TokenVerifier: tokens,
		Log:           appLog.Logger,
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
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
		"health":  "http://localhost:" + cfg.Port + constant.HealthPath,
		"swagger": "http://localhost:" + cfg.Port + constant.SwaggerUIPath,
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

func fallbackLogger() *logrus.Logger {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339})
	return log
}
