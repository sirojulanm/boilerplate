package checker_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sirojulanam/boilerplate/internal/infrastructure/checker"
)

func TestRuntimeCheckReportsUptime(t *testing.T) {
	runtime := checker.NewRuntime(time.Now().Add(-2 * time.Second))

	if runtime.Name() != "runtime" {
		t.Fatalf("name = %q, want runtime", runtime.Name())
	}

	detail, err := runtime.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !strings.HasPrefix(detail, "uptime=") {
		t.Fatalf("detail = %q, want uptime prefix", detail)
	}
}

func TestRuntimeCheckHonorsCanceledContext(t *testing.T) {
	runtime := checker.NewRuntime(time.Now())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := runtime.Check(ctx); err == nil {
		t.Fatal("expected context cancellation error")
	}
}
