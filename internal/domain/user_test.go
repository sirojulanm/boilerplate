package domain_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sirojulanam/boilerplate/internal/domain"
)

func TestNewUserNormalizesInput(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.FixedZone("WIB", 7*60*60))

	user, err := domain.NewUser("  Jane Doe  ", "  JANE@Example.COM  ", "hash", now)
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	if user.Name != "Jane Doe" {
		t.Fatalf("name = %q, want Jane Doe", user.Name)
	}
	if user.Email != "jane@example.com" {
		t.Fatalf("email = %q, want jane@example.com", user.Email)
	}
	if user.ID == [16]byte{} {
		t.Fatal("user id must be set")
	}
	if !user.CreatedAt.Equal(now.UTC()) || !user.UpdatedAt.Equal(now.UTC()) {
		t.Fatalf("timestamps = %v / %v, want %v", user.CreatedAt, user.UpdatedAt, now.UTC())
	}
}

func TestNewUserRejectsInvalidInput(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		userName string
		email    string
		hash     string
		want     error
	}{
		{name: "empty name", userName: " ", email: "jane@example.com", hash: "hash", want: domain.ErrInvalidName},
		{name: "long name", userName: strings.Repeat("a", 101), email: "jane@example.com", hash: "hash", want: domain.ErrInvalidName},
		{name: "empty email", userName: "Jane", email: "", hash: "hash", want: domain.ErrInvalidEmail},
		{name: "email with display name", userName: "Jane", email: "Jane <jane@example.com>", hash: "hash", want: domain.ErrInvalidEmail},
		{name: "empty hash", userName: "Jane", email: "jane@example.com", hash: "", want: domain.ErrInvalidPasswordHash},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := domain.NewUser(test.userName, test.email, test.hash, now)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	valid := []string{"jane@example.com", "a.b+tag@sub.example.co.id"}
	for _, email := range valid {
		if err := domain.ValidateEmail(email); err != nil {
			t.Fatalf("ValidateEmail(%q) = %v, want nil", email, err)
		}
	}

	invalid := []string{"", "jane", "jane@", "jane @example.com", "Jane <jane@example.com>"}
	for _, email := range invalid {
		if err := domain.ValidateEmail(email); !errors.Is(err, domain.ErrInvalidEmail) {
			t.Fatalf("ValidateEmail(%q) = %v, want ErrInvalidEmail", email, err)
		}
	}
}

func TestValidatePasswordBounds(t *testing.T) {
	if err := domain.ValidatePassword("1234567"); !errors.Is(err, domain.ErrWeakPassword) {
		t.Fatalf("7-byte password error = %v, want ErrWeakPassword", err)
	}
	if err := domain.ValidatePassword("12345678"); err != nil {
		t.Fatalf("8-byte password error = %v, want nil", err)
	}
	if err := domain.ValidatePassword(strings.Repeat("a", 72)); err != nil {
		t.Fatalf("72-byte password error = %v, want nil", err)
	}
	if err := domain.ValidatePassword(strings.Repeat("a", 73)); !errors.Is(err, domain.ErrWeakPassword) {
		t.Fatalf("73-byte password error = %v, want ErrWeakPassword", err)
	}
}

func TestUserDoesNotSerializePasswordHash(t *testing.T) {
	user, err := domain.NewUser("Jane", "jane@example.com", "super-secret-hash", time.Now())
	if err != nil {
		t.Fatalf("NewUser() error = %v", err)
	}

	payload, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(payload), "super-secret-hash") {
		t.Fatalf("serialized user leaked password hash: %s", payload)
	}
}
