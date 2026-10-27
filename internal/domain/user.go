package domain

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/sirojulanam/boilerplate/internal/constant"
)

// Domain errors. Delivery layers translate these into HTTP responses without
// exposing internal details such as database errors.
var (
	ErrInvalidEmail        = errors.New("invalid email")
	ErrInvalidName         = errors.New("invalid name")
	ErrWeakPassword        = errors.New("weak password")
	ErrInvalidPasswordHash = errors.New("invalid password hash")
	ErrEmailTaken          = errors.New("email already registered")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidToken        = errors.New("invalid token")
)

// User is an authenticated account. The password hash is never serialized.
type User struct {
	ID           uuid.UUID `json:"id" example:"6f1c3a2b-6a2f-4f7c-9d0d-3b8c0f0f0f0f"`
	Name         string    `json:"name" example:"Jane Doe"`
	Email        string    `json:"email" example:"jane@example.com"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at" example:"2026-10-27T22:00:00Z"`
	UpdatedAt    time.Time `json:"updated_at" example:"2026-10-27T22:00:00Z"`
}

// NewUser validates and normalizes a new account. Keeping these invariants in
// the domain means every caller produces the same valid user shape.
func NewUser(name, email, passwordHash string, now time.Time) (User, error) {
	name = strings.TrimSpace(name)
	email = NormalizeEmail(email)

	if err := ValidateName(name); err != nil {
		return User{}, err
	}
	if err := ValidateEmail(email); err != nil {
		return User{}, err
	}
	if passwordHash == "" {
		return User{}, ErrInvalidPasswordHash
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return User{}, fmt.Errorf("generate user id: %w", err)
	}

	timestamp := now.UTC()
	return User{
		ID:           id,
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    timestamp,
		UpdatedAt:    timestamp,
	}, nil
}

// NormalizeEmail trims whitespace and lowercases an address so lookups and the
// unique constraint behave consistently.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateEmail accepts a single bare email address.
func ValidateEmail(email string) error {
	if email == "" || len(email) > constant.MaxEmailLength || strings.ContainsAny(email, " \t\r\n") {
		return ErrInvalidEmail
	}

	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return ErrInvalidEmail
	}
	return nil
}

// ValidateName enforces a non-empty, bounded display name.
func ValidateName(name string) error {
	count := utf8.RuneCountInString(strings.TrimSpace(name))
	if count == 0 || count > constant.MaxNameLength {
		return ErrInvalidName
	}
	return nil
}

// ValidatePassword enforces the bcrypt input limit and a minimum length.
func ValidatePassword(password string) error {
	if len(password) < constant.MinPasswordLength || len(password) > constant.MaxPasswordLength {
		return ErrWeakPassword
	}
	return nil
}

// UserCreator is the write port for accounts.
type UserCreator interface {
	Create(ctx context.Context, user User) error
}

// UserQuery is the read port for accounts. Keeping it separate makes the
// query dependency explicit and lets usecases depend on only what they need.
type UserQuery interface {
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByID(ctx context.Context, id uuid.UUID) (User, error)
}

// UserRepository combines the account read and write ports.
type UserRepository interface {
	UserCreator
	UserQuery
}

// PasswordHasher is the password hashing port. Infrastructure provides bcrypt.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) bool
}

// Token is an issued access token and its public metadata.
type Token struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
	ExpiresIn   int64
}

// Identity is the authenticated caller derived from a token.
type Identity struct {
	UserID uuid.UUID
}

// TokenIssuer signs access tokens for authenticated users.
type TokenIssuer interface {
	Issue(user User) (Token, error)
}

// TokenVerifier validates a raw access token and returns its identity.
type TokenVerifier interface {
	Verify(rawToken string) (Identity, error)
}
