package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/sirojulanam/boilerplate/internal/domain"
)

type fakeChecker struct {
	name   string
	detail string
	err    error
}

func (f fakeChecker) Name() string { return f.name }

func (f fakeChecker) Check(context.Context) (string, error) { return f.detail, f.err }

func TestHealthCheckReportsOK(t *testing.T) {
	service := NewHealth("test-service", "1.0.0", fakeChecker{name: "runtime", detail: "uptime=1s"})

	got := service.Check(context.Background())

	if got.Status != domain.StatusOK {
		t.Fatalf("status = %q, want %q", got.Status, domain.StatusOK)
	}
	if got.Service != "test-service" || got.Version != "1.0.0" {
		t.Fatalf("unexpected metadata: %+v", got)
	}
	if len(got.Checks) != 1 || got.Checks[0].Status != "up" {
		t.Fatalf("unexpected checks: %+v", got.Checks)
	}
	if got.Timestamp.IsZero() {
		t.Fatal("timestamp must be set")
	}
}

func TestHealthCheckReportsDegraded(t *testing.T) {
	service := NewHealth("test-service", "1.0.0",
		fakeChecker{name: "runtime", detail: "uptime=1s"},
		fakeChecker{name: "database", err: errors.New("connection refused")},
	)

	got := service.Check(context.Background())

	if got.Status != domain.StatusDegraded {
		t.Fatalf("status = %q, want %q", got.Status, domain.StatusDegraded)
	}
	if got.Checks[1].Status != "down" || got.Checks[1].Detail != "connection refused" {
		t.Fatalf("unexpected failing check: %+v", got.Checks[1])
	}
}
