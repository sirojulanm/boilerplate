package checker

import (
	"context"
	"time"
)

// Pinger is the subset of a database pool needed for a health probe.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Postgres checks database connectivity for the health endpoint.
type Postgres struct {
	db      Pinger
	timeout time.Duration
}

// NewPostgres builds a PostgreSQL health checker.
func NewPostgres(db Pinger) *Postgres {
	return &Postgres{db: db, timeout: 2 * time.Second}
}

// Name identifies the dependency in health responses.
func (p *Postgres) Name() string {
	return "postgres"
}

// Check pings the database with a bounded timeout.
func (p *Postgres) Check(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	if err := p.db.PingContext(ctx); err != nil {
		return "", err
	}
	return "connected", nil
}
