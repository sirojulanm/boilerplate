package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirojulanam/boilerplate/internal/domain"
)

type fakeUserRepository struct {
	createErr error
	created   *domain.User

	byEmail    domain.User
	byEmailErr error

	byID    domain.User
	byIDErr error
}

func (f *fakeUserRepository) Create(_ context.Context, user domain.User) error {
	f.created = &user
	return f.createErr
}

func (f *fakeUserRepository) FindByEmail(context.Context, string) (domain.User, error) {
	return f.byEmail, f.byEmailErr
}

func (f *fakeUserRepository) FindByID(context.Context, uuid.UUID) (domain.User, error) {
	return f.byID, f.byIDErr
}

type fakeHasher struct {
	hash        string
	hashErr     error
	hashCalls   int
	verify      bool
	verifyHash  string
	verifyPass  string
	verifyCalls int
}

func (f *fakeHasher) Hash(string) (string, error) {
	f.hashCalls++
	return f.hash, f.hashErr
}

func (f *fakeHasher) Verify(hash, password string) bool {
	f.verifyCalls++
	f.verifyHash = hash
	f.verifyPass = password
	return f.verify
}

type fakeTokenIssuer struct {
	token domain.Token
	err   error
	user  domain.User
}

func (f *fakeTokenIssuer) Issue(user domain.User) (domain.Token, error) {
	f.user = user
	return f.token, f.err
}

func newAuthForTest(repo domain.UserRepository, hasher domain.PasswordHasher, issuer domain.TokenIssuer) *authService {
	return NewAuth(repo, hasher, issuer).(*authService)
}

func TestRegisterHashesPasswordAndNormalizesEmail(t *testing.T) {
	repo := &fakeUserRepository{}
	hasher := &fakeHasher{hash: "hashed-password"}
	service := newAuthForTest(repo, hasher, &fakeTokenIssuer{})
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	user, err := service.Register(context.Background(), RegisterInput{
		Name:     " Jane Doe ",
		Email:    " JANE@Example.COM ",
		Password: "correct-horse-battery",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if user.Email != "jane@example.com" {
		t.Fatalf("email = %q, want normalized email", user.Email)
	}
	if user.PasswordHash != "hashed-password" {
		t.Fatalf("password hash = %q, want hashed-password", user.PasswordHash)
	}
	if !user.CreatedAt.Equal(now) {
		t.Fatalf("created_at = %v, want %v", user.CreatedAt, now)
	}
	if repo.created == nil || repo.created.ID != user.ID {
		t.Fatal("created user was not persisted")
	}
	if hasher.hashCalls != 1 {
		t.Fatalf("hash calls = %d, want 1", hasher.hashCalls)
	}
}

func TestRegisterRejectsInvalidInputBeforeHashing(t *testing.T) {
	tests := []struct {
		name  string
		input RegisterInput
		want  error
	}{
		{
			name:  "empty name",
			input: RegisterInput{Name: " ", Email: "jane@example.com", Password: "correct-horse-battery"},
			want:  domain.ErrInvalidName,
		},
		{
			name:  "invalid email",
			input: RegisterInput{Name: "Jane", Email: "not-an-email", Password: "correct-horse-battery"},
			want:  domain.ErrInvalidEmail,
		},
		{
			name:  "short password",
			input: RegisterInput{Name: "Jane", Email: "jane@example.com", Password: "short"},
			want:  domain.ErrWeakPassword,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &fakeUserRepository{}
			hasher := &fakeHasher{hash: "hashed"}
			service := newAuthForTest(repo, hasher, &fakeTokenIssuer{})

			_, err := service.Register(context.Background(), test.input)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			if hasher.hashCalls != 0 {
				t.Fatal("password must not be hashed for invalid input")
			}
			if repo.created != nil {
				t.Fatal("invalid user must not be persisted")
			}
		})
	}
}

func TestRegisterPropagatesDuplicateEmail(t *testing.T) {
	repo := &fakeUserRepository{createErr: domain.ErrEmailTaken}
	service := newAuthForTest(repo, &fakeHasher{hash: "hashed"}, &fakeTokenIssuer{})

	_, err := service.Register(context.Background(), RegisterInput{
		Name:     "Jane",
		Email:    "jane@example.com",
		Password: "correct-horse-battery",
	})
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("error = %v, want ErrEmailTaken", err)
	}
}

func TestRegisterWrapsHashFailure(t *testing.T) {
	repo := &fakeUserRepository{}
	hasher := &fakeHasher{hashErr: errors.New("bcrypt failure")}
	service := newAuthForTest(repo, hasher, &fakeTokenIssuer{})

	_, err := service.Register(context.Background(), RegisterInput{
		Name:     "Jane",
		Email:    "jane@example.com",
		Password: "correct-horse-battery",
	})
	if err == nil || !strings.Contains(err.Error(), "hash password") {
		t.Fatalf("error = %v, want wrapped hash failure", err)
	}
	if repo.created != nil {
		t.Fatal("user must not be persisted when hashing fails")
	}
}

func TestGenerateTokenReturnsIssuedToken(t *testing.T) {
	user := domain.User{
		ID:           uuid.New(),
		Name:         "Jane",
		Email:        "jane@example.com",
		PasswordHash: "hashed",
	}
	wantToken := domain.Token{
		AccessToken: "signed",
		TokenType:   "Bearer",
		ExpiresAt:   time.Now().Add(time.Hour),
		ExpiresIn:   3600,
	}

	repo := &fakeUserRepository{byEmail: user}
	hasher := &fakeHasher{verify: true}
	issuer := &fakeTokenIssuer{token: wantToken}
	service := newAuthForTest(repo, hasher, issuer)

	result, err := service.GenerateToken(context.Background(), " JANE@example.com ", "correct-horse-battery")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if result.Token != wantToken {
		t.Fatalf("token = %+v, want %+v", result.Token, wantToken)
	}
	if result.User.ID != user.ID {
		t.Fatalf("user id = %s, want %s", result.User.ID, user.ID)
	}
	if hasher.verifyPass != "correct-horse-battery" {
		t.Fatalf("verified password = %q", hasher.verifyPass)
	}
	if issuer.user.ID != user.ID {
		t.Fatal("issuer did not receive the authenticated user")
	}
}

func TestGenerateTokenHidesUnknownEmail(t *testing.T) {
	repo := &fakeUserRepository{byEmailErr: domain.ErrUserNotFound}
	hasher := &fakeHasher{verify: false}
	service := newAuthForTest(repo, hasher, &fakeTokenIssuer{})

	_, err := service.GenerateToken(context.Background(), "missing@example.com", "correct-horse-battery")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
	if hasher.verifyCalls != 1 || hasher.verifyHash != dummyPasswordHash {
		t.Fatalf("dummy hash verification not performed: calls=%d hash=%q", hasher.verifyCalls, hasher.verifyHash)
	}
}

func TestGenerateTokenRejectsWrongPassword(t *testing.T) {
	repo := &fakeUserRepository{
		byEmail: domain.User{ID: uuid.New(), Email: "jane@example.com", PasswordHash: "hashed"},
	}
	hasher := &fakeHasher{verify: false}
	service := newAuthForTest(repo, hasher, &fakeTokenIssuer{})

	_, err := service.GenerateToken(context.Background(), "jane@example.com", "wrong-password")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestGenerateTokenPropagatesRepositoryFailure(t *testing.T) {
	repo := &fakeUserRepository{byEmailErr: errors.New("database down")}
	service := newAuthForTest(repo, &fakeHasher{}, &fakeTokenIssuer{})

	_, err := service.GenerateToken(context.Background(), "jane@example.com", "correct-horse-battery")
	if err == nil || !strings.Contains(err.Error(), "database down") {
		t.Fatalf("error = %v, want repository failure", err)
	}
}

func TestGenerateTokenWrapsIssuerFailure(t *testing.T) {
	repo := &fakeUserRepository{
		byEmail: domain.User{ID: uuid.New(), Email: "jane@example.com", PasswordHash: "hashed"},
	}
	hasher := &fakeHasher{verify: true}
	issuer := &fakeTokenIssuer{err: errors.New("signing failed")}
	service := newAuthForTest(repo, hasher, issuer)

	_, err := service.GenerateToken(context.Background(), "jane@example.com", "correct-horse-battery")
	if err == nil || !strings.Contains(err.Error(), "issue token") {
		t.Fatalf("error = %v, want wrapped issuer failure", err)
	}
	if errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatal("issuer failure must not become invalid credentials")
	}
}

func TestProfileReturnsUser(t *testing.T) {
	user := domain.User{ID: uuid.New(), Name: "Jane", Email: "jane@example.com"}
	repo := &fakeUserRepository{byID: user}
	service := newAuthForTest(repo, &fakeHasher{}, &fakeTokenIssuer{})

	got, err := service.Profile(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("Profile() error = %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("user id = %s, want %s", got.ID, user.ID)
	}
}

func TestProfileHidesMissingUser(t *testing.T) {
	repo := &fakeUserRepository{byIDErr: domain.ErrUserNotFound}
	service := newAuthForTest(repo, &fakeHasher{}, &fakeTokenIssuer{})

	_, err := service.Profile(context.Background(), uuid.New())
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}
