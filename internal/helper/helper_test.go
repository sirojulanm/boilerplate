package helper_test

import (
	"testing"
	"time"

	"github.com/sirojulanam/boilerplate/internal/helper"
)

func TestEnvReturnsValue(t *testing.T) {
	t.Setenv("HELPER_TEST_KEY", "value")

	if got := helper.Env("HELPER_TEST_KEY", "fallback"); got != "value" {
		t.Fatalf("Env() = %q, want value", got)
	}
}

func TestEnvFallsBackWhenEmpty(t *testing.T) {
	t.Setenv("HELPER_TEST_KEY", "")

	if got := helper.Env("HELPER_TEST_KEY", "fallback"); got != "fallback" {
		t.Fatalf("Env() = %q, want fallback", got)
	}
}

func TestNewRequestIDIsRandomAndNonEmpty(t *testing.T) {
	first := helper.NewRequestID()
	second := helper.NewRequestID()

	if first == "" || second == "" {
		t.Fatal("request ID must not be empty")
	}
	if first == second {
		t.Fatalf("request IDs must be unique: %q", first)
	}
	if len(first) != 16 {
		t.Fatalf("request ID length = %d, want 16", len(first))
	}
}

func TestDurationMS(t *testing.T) {
	if got := helper.DurationMS(1500 * time.Microsecond); got != 1.5 {
		t.Fatalf("DurationMS() = %v, want 1.5", got)
	}
}
