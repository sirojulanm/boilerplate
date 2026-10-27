package checker_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sirojulanam/boilerplate/internal/infrastructure/checker"
)

type fakePinger struct {
	err error
	ctx context.Context
}

func (f *fakePinger) PingContext(ctx context.Context) error {
	f.ctx = ctx
	return f.err
}

func TestPostgresCheckReportsConnected(t *testing.T) {
	pinger := &fakePinger{}
	probe := checker.NewPostgres(pinger)

	if probe.Name() != "postgres" {
		t.Fatalf("name = %q, want postgres", probe.Name())
	}

	detail, err := probe.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if detail != "connected" {
		t.Fatalf("detail = %q, want connected", detail)
	}
	if pinger.ctx == nil {
		t.Fatal("pinger did not receive a context")
	}
	if _, ok := pinger.ctx.Deadline(); !ok {
		t.Fatal("pinger context must carry a timeout")
	}
}

func TestPostgresCheckPropagatesFailure(t *testing.T) {
	want := errors.New("connection refused")
	probe := checker.NewPostgres(&fakePinger{err: want})

	detail, err := probe.Check(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("Check() error = %v, want %v", err, want)
	}
	if detail != "" {
		t.Fatalf("detail = %q, want empty on failure", detail)
	}
}

func TestPostgresCheckHonorsCanceledContext(t *testing.T) {
	probe := checker.NewPostgres(&fakePinger{err: context.Canceled})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := probe.Check(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Check() error = %v, want context.Canceled", err)
	}
}
