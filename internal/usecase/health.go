package usecase

import (
	"context"
	"time"

	"github.com/sirojulanam/boilerplate/internal/domain"
)

// Health is the application behaviour for health reporting. Delivery layers
// depend on this interface instead of a concrete implementation.
type Health interface {
	Check(ctx context.Context) domain.Health
}

type healthService struct {
	service  string
	version  string
	checkers []domain.Checker
	now      func() time.Time
}

// NewHealth builds the health usecase. Every checker is optional; when none is
// provided the service still reports itself as healthy.
func NewHealth(service, version string, checkers ...domain.Checker) Health {
	return &healthService{
		service:  service,
		version:  version,
		checkers: checkers,
		now:      time.Now,
	}
}

// Check probes every registered checker and lets the domain derive the
// aggregate status from the results.
func (s *healthService) Check(ctx context.Context) domain.Health {
	checks := make([]domain.Check, 0, len(s.checkers))

	for _, checker := range s.checkers {
		detail, err := checker.Check(ctx)
		checks = append(checks, domain.NewCheck(checker.Name(), detail, err))
	}

	return domain.NewHealth(s.service, s.version, s.now(), checks)
}
