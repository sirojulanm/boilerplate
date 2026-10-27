package domain

import (
	"context"
	"time"
)

// Status is the overall health state of the service.
type Status string

const (
	// StatusOK means every dependency check passed.
	StatusOK Status = "ok"
	// StatusDegraded means at least one dependency check failed.
	StatusDegraded Status = "degraded"
)

// Check is the result of probing a single dependency.
type Check struct {
	Name   string `json:"name" example:"runtime"`
	Status string `json:"status" example:"up"`
	Detail string `json:"detail,omitempty" example:"uptime=1h2m3s"`
}

// Health is the health snapshot returned to API clients.
type Health struct {
	Status    Status    `json:"status" example:"ok"`
	Service   string    `json:"service" example:"opencodeproject"`
	Version   string    `json:"version" example:"1.0.0"`
	Timestamp time.Time `json:"timestamp" example:"2026-10-27T22:00:00Z"`
	Checks    []Check   `json:"checks"`
}

// Checker is a dependency that can be probed. The usecase layer depends on
// this port, while infrastructure packages provide the implementations.
type Checker interface {
	Name() string
	Check(ctx context.Context) (detail string, err error)
}
