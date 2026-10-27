package domain

import (
	"context"
	"time"
)

// Status is the aggregate health state of the service.
type Status string

const (
	// StatusOK means every dependency check passed.
	StatusOK Status = "ok"
	// StatusDegraded means at least one dependency check failed.
	StatusDegraded Status = "degraded"
)

// IsHealthy reports whether the aggregate state is healthy.
func (s Status) IsHealthy() bool {
	return s == StatusOK
}

// CheckStatus is the state of a single dependency probe.
type CheckStatus string

const (
	// CheckUp means the dependency responded successfully.
	CheckUp CheckStatus = "up"
	// CheckDown means the dependency failed its probe.
	CheckDown CheckStatus = "down"
)

// Check is the result of probing a single dependency.
type Check struct {
	Name   string      `json:"name" example:"runtime"`
	Status CheckStatus `json:"status" example:"up"`
	Detail string      `json:"detail,omitempty" example:"uptime=1h2m3s"`
}

// Healthy reports whether this check passed.
func (c Check) Healthy() bool {
	return c.Status == CheckUp
}

// Health is the aggregate health snapshot returned to API clients.
type Health struct {
	Status    Status    `json:"status" example:"ok"`
	Service   string    `json:"service" example:"boilerplate"`
	Version   string    `json:"version" example:"1.0.0"`
	Timestamp time.Time `json:"timestamp" example:"2026-10-27T22:00:00Z"`
	Checks    []Check   `json:"checks"`
}

// NewHealth builds a health snapshot and derives its status from the checks.
// Keeping this invariant in the domain prevents callers from reporting a
// healthy service while a dependency is down.
func NewHealth(service, version string, at time.Time, checks []Check) Health {
	if checks == nil {
		checks = []Check{}
	}

	status := StatusOK
	for _, check := range checks {
		if !check.Healthy() {
			status = StatusDegraded
			break
		}
	}

	return Health{
		Status:    status,
		Service:   service,
		Version:   version,
		Timestamp: at.UTC(),
		Checks:    checks,
	}
}

// NewCheck translates a probe result into a domain check.
func NewCheck(name, detail string, err error) Check {
	if err != nil {
		return Check{Name: name, Status: CheckDown, Detail: err.Error()}
	}
	return Check{Name: name, Status: CheckUp, Detail: detail}
}

// Checker is a dependency that can be probed. The usecase layer depends on
// this port, while infrastructure packages provide the implementations.
type Checker interface {
	Name() string
	Check(ctx context.Context) (detail string, err error)
}
