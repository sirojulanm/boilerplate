package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirojulanam/boilerplate/internal/domain"
	"github.com/sirojulanam/boilerplate/internal/infrastructure/postgres"
)

// TestUserRepositoryIntegration exercises the GORM repository against a real
// PostgreSQL instance. It is skipped unless TEST_DATABASE_URL is set.
func TestUserRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run the PostgreSQL integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := postgres.MigrateUp(dsn); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	db, err := postgres.Open(ctx, postgres.Config{URL: dsn, MaxConns: 2, ConnectTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	repo := postgres.NewUserRepository(db)
	email := fmt.Sprintf("integration-%s@example.com", uuid.NewString())
	user, err := domain.NewUser("Integration User", email, "bcrypt-hash", time.Now())
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	t.Cleanup(func() {
		if err := db.WithContext(context.Background()).Exec("DELETE FROM users WHERE email = ?", email).Error; err != nil {
			t.Logf("cleanup user: %v", err)
		}
	})

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	byEmail, err := repo.FindByEmail(ctx, strings.ToUpper(email))
	if err != nil {
		t.Fatalf("FindByEmail() error = %v", err)
	}
	if byEmail.ID != user.ID || byEmail.Email != email || byEmail.PasswordHash != user.PasswordHash {
		t.Fatalf("FindByEmail() = %+v, want id=%s email=%s", byEmail, user.ID, email)
	}

	byID, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if byID.ID != user.ID || byID.Email != email {
		t.Fatalf("FindByID() = %+v, want id=%s", byID, user.ID)
	}

	duplicate, err := domain.NewUser("Other User", email, "other-hash", time.Now())
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}
	if err := repo.Create(ctx, duplicate); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("duplicate Create() error = %v, want ErrEmailTaken", err)
	}

	if _, err := repo.FindByEmail(ctx, fmt.Sprintf("missing-%s@example.com", uuid.NewString())); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("FindByEmail() error = %v, want ErrUserNotFound", err)
	}
	if _, err := repo.FindByID(ctx, uuid.New()); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("FindByID() error = %v, want ErrUserNotFound", err)
	}
}
