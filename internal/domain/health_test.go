package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/sirojulanam/boilerplate/internal/domain"
)

func TestNewHealthAllChecksUp(t *testing.T) {
	at := time.Date(2026, 10, 27, 12, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	health := domain.NewHealth("boilerplate", "1.0.0", at, []domain.Check{
		domain.NewCheck("runtime", "uptime=1s", nil),
	})

	if !health.Status.IsHealthy() {
		t.Fatalf("status = %q, want healthy", health.Status)
	}
	if health.Timestamp.Location() != time.UTC {
		t.Fatalf("timestamp location = %v, want UTC", health.Timestamp.Location())
	}
	if len(health.Checks) != 1 || !health.Checks[0].Healthy() {
		t.Fatalf("unexpected checks: %+v", health.Checks)
	}
}

func TestNewHealthDegradedWhenAnyCheckFails(t *testing.T) {
	health := domain.NewHealth("boilerplate", "1.0.0", time.Now(), []domain.Check{
		domain.NewCheck("runtime", "uptime=1s", nil),
		domain.NewCheck("database", "", errors.New("connection refused")),
	})

	if health.Status != domain.StatusDegraded {
		t.Fatalf("status = %q, want %q", health.Status, domain.StatusDegraded)
	}
	if health.Status.IsHealthy() {
		t.Fatal("degraded status must not be healthy")
	}

	failed := health.Checks[1]
	if failed.Status != domain.CheckDown || failed.Detail != "connection refused" {
		t.Fatalf("unexpected failing check: %+v", failed)
	}
}

func TestNewHealthWithoutChecksIsHealthy(t *testing.T) {
	health := domain.NewHealth("boilerplate", "1.0.0", time.Now(), nil)

	if health.Status != domain.StatusOK {
		t.Fatalf("status = %q, want %q", health.Status, domain.StatusOK)
	}
	if health.Checks == nil {
		t.Fatal("checks must be an empty slice, not nil")
	}
}

func TestNewCheckPreservesDetail(t *testing.T) {
	check := domain.NewCheck("runtime", "uptime=5s", nil)

	if check.Status != domain.CheckUp || check.Detail != "uptime=5s" {
		t.Fatalf("unexpected check: %+v", check)
	}
}
