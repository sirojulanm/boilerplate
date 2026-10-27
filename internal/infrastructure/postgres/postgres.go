package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config controls the GORM connection pool.
type Config struct {
	URL            string
	MaxConns       int32
	ConnectTimeout time.Duration
	Log            *logrus.Logger
}

// Open creates a GORM database handle backed by PostgreSQL and verifies
// connectivity before returning.
func Open(ctx context.Context, cfg Config) (*gorm.DB, error) {
	database, err := gorm.Open(postgres.New(postgres.Config{DSN: cfg.URL}), &gorm.Config{
		Logger:         newGormLogger(cfg.Log),
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("access database pool: %w", err)
	}

	if cfg.MaxConns > 0 {
		sqlDB.SetMaxOpenConns(int(cfg.MaxConns))
		sqlDB.SetMaxIdleConns(int(cfg.MaxConns))
	}
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	timeout := cfg.ConnectTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return database, nil
}
