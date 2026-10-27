package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirojulanam/boilerplate/internal/domain"
)

// Auth is the application behaviour for registration, token generation, and
// reading the authenticated profile.
type Auth interface {
	Register(ctx context.Context, input RegisterInput) (domain.User, error)
	GenerateToken(ctx context.Context, email, password string) (TokenResult, error)
	Profile(ctx context.Context, userID uuid.UUID) (domain.User, error)
}

// RegisterInput is the validated input for account creation.
type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

// TokenResult bundles an issued token with the account it belongs to.
type TokenResult struct {
	Token domain.Token
	User  domain.User
}

type authService struct {
	creator domain.UserCreator
	query   domain.UserQuery
	hasher  domain.PasswordHasher
	tokens  domain.TokenIssuer
	now     func() time.Time
}

// NewAuth builds the authentication usecase from its ports. The repository
// implements both the write and read interfaces; they are kept separate so the
// query surface stays explicit.
func NewAuth(users domain.UserRepository, hasher domain.PasswordHasher, tokens domain.TokenIssuer) Auth {
	return &authService{
		creator: users,
		query:   users,
		hasher:  hasher,
		tokens:  tokens,
		now:     time.Now,
	}
}

// Register validates the input, hashes the password, and persists the account.
func (s *authService) Register(ctx context.Context, input RegisterInput) (domain.User, error) {
	email := domain.NormalizeEmail(input.Email)

	if err := domain.ValidateName(input.Name); err != nil {
		return domain.User{}, err
	}
	if err := domain.ValidateEmail(email); err != nil {
		return domain.User{}, err
	}
	if err := domain.ValidatePassword(input.Password); err != nil {
		return domain.User{}, err
	}

	hash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := domain.NewUser(input.Name, email, hash, s.now())
	if err != nil {
		return domain.User{}, err
	}

	if err := s.creator.Create(ctx, user); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// GenerateToken verifies credentials and issues a short-lived access token.
// Unknown accounts and wrong passwords return the same generic error so the
// endpoint cannot be used to enumerate registered emails.
func (s *authService) GenerateToken(ctx context.Context, email, password string) (TokenResult, error) {
	email = domain.NormalizeEmail(email)
	if err := domain.ValidateEmail(email); err != nil {
		return TokenResult{}, domain.ErrInvalidCredentials
	}

	user, err := s.query.FindByEmail(ctx, email)
	if errors.Is(err, domain.ErrUserNotFound) {
		// Perform a bcrypt comparison anyway to keep timing similar to a real
		// account lookup. The hash is for a throwaway password.
		s.hasher.Verify(dummyPasswordHash, password)
		return TokenResult{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return TokenResult{}, err
	}

	if !s.hasher.Verify(user.PasswordHash, password) {
		return TokenResult{}, domain.ErrInvalidCredentials
	}

	token, err := s.tokens.Issue(user)
	if err != nil {
		return TokenResult{}, fmt.Errorf("issue token: %w", err)
	}
	return TokenResult{Token: token, User: user}, nil
}

// Profile returns the authenticated account.
func (s *authService) Profile(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	user, err := s.query.FindByID(ctx, userID)
	if errors.Is(err, domain.ErrUserNotFound) {
		return domain.User{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return domain.User{}, err
	}
	return user, nil
}

// dummyPasswordHash is a valid bcrypt hash of an unused password. It exists
// only to equalize the work done for unknown accounts during token generation.
const dummyPasswordHash = "$2a$12$Y8AopPNtq5Du8HGZcfICresMG29C58X62ju3xMZNhG0/GPshA0GCy"
