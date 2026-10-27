package security

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirojulanam/boilerplate/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func TestNewJWTIssuerValidatesConfig(t *testing.T) {
	tests := []struct {
		name     string
		secret   []byte
		issuer   string
		audience string
		ttl      time.Duration
	}{
		{name: "short secret", secret: []byte("short"), issuer: "issuer", audience: "audience", ttl: time.Hour},
		{name: "empty issuer", secret: []byte(testSecret), audience: "audience", ttl: time.Hour},
		{name: "empty audience", secret: []byte(testSecret), issuer: "issuer", ttl: time.Hour},
		{name: "zero ttl", secret: []byte(testSecret), issuer: "issuer", audience: "audience"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewJWTIssuer(test.secret, test.issuer, test.audience, test.ttl); err == nil {
				t.Fatal("NewJWTIssuer() error = nil, want validation error")
			}
		})
	}
}

func TestJWTIssueAndVerify(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	issuer := newTestIssuer(t)
	issuer.now = func() time.Time { return now }

	user := domain.User{ID: uuid.New(), Email: "jane@example.com"}
	token, err := issuer.Issue(user)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if token.AccessToken == "" {
		t.Fatal("access token must not be empty")
	}
	if token.TokenType != "Bearer" {
		t.Fatalf("token type = %q, want Bearer", token.TokenType)
	}
	if !token.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("expires_at = %v, want %v", token.ExpiresAt, now.Add(time.Hour))
	}
	if token.ExpiresIn != 3600 {
		t.Fatalf("expires_in = %d, want 3600", token.ExpiresIn)
	}

	identity, err := issuer.Verify(token.AccessToken)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if identity.UserID != user.ID {
		t.Fatalf("user id = %s, want %s", identity.UserID, user.ID)
	}
}

func TestJWTVerifyRejectsInvalidTokens(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	issuer := newTestIssuer(t)
	issuer.now = func() time.Time { return now }

	user := domain.User{ID: uuid.New()}
	valid, err := issuer.Issue(user)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	otherIssuer := newTestIssuer(t)
	otherIssuer.now = func() time.Time { return now }
	otherIssuer.secret = []byte("abcdef0123456789abcdef0123456789")

	expiredIssuer := newTestIssuer(t)
	expiredIssuer.now = func() time.Time { return now.Add(-2 * time.Hour) }
	expired, err := expiredIssuer.Issue(user)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	hs512 := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.RegisteredClaims{
		Issuer:    issuer.issuer,
		Subject:   user.ID.String(),
		Audience:  jwt.ClaimStrings{issuer.audience},
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(now),
	})
	hs512Token, err := hs512.SignedString(issuer.secret)
	if err != nil {
		t.Fatalf("sign HS512 token: %v", err)
	}

	noExpiry := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:   issuer.issuer,
		Subject:  user.ID.String(),
		Audience: jwt.ClaimStrings{issuer.audience},
		IssuedAt: jwt.NewNumericDate(now),
	})
	noExpiryToken, err := noExpiry.SignedString(issuer.secret)
	if err != nil {
		t.Fatalf("sign no-expiry token: %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{name: "tampered", token: valid.AccessToken + "x"},
		{name: "wrong secret", token: mustIssue(t, otherIssuer, user)},
		{name: "expired", token: expired.AccessToken},
		{name: "wrong algorithm", token: hs512Token},
		{name: "missing expiry", token: noExpiryToken},
		{name: "garbage", token: "not-a-token"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := issuer.Verify(test.token)
			if !errors.Is(err, domain.ErrInvalidToken) {
				t.Fatalf("Verify() error = %v, want ErrInvalidToken", err)
			}
		})
	}
}

func TestJWTVerifyRejectsWrongIssuerOrAudience(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	issuer := newTestIssuer(t)
	issuer.now = func() time.Time { return now }
	user := domain.User{ID: uuid.New()}

	for _, test := range []struct {
		name     string
		issuer   string
		audience string
	}{
		{name: "wrong issuer", issuer: "other", audience: issuer.audience},
		{name: "wrong audience", issuer: issuer.issuer, audience: "other"},
	} {
		t.Run(test.name, func(t *testing.T) {
			other := newTestIssuer(t)
			other.now = func() time.Time { return now }
			other.issuer = test.issuer
			other.audience = test.audience

			token := mustIssue(t, other, user)
			if _, err := issuer.Verify(token); !errors.Is(err, domain.ErrInvalidToken) {
				t.Fatalf("Verify() error = %v, want ErrInvalidToken", err)
			}
		})
	}
}

func TestBcryptHasher(t *testing.T) {
	hasher, err := NewBcryptHasher(bcrypt.MinCost)
	if err != nil {
		t.Fatalf("NewBcryptHasher() error = %v", err)
	}

	hash, err := hasher.Hash("correct-horse-battery")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if hash == "correct-horse-battery" {
		t.Fatal("hash must not equal the plaintext password")
	}
	if !hasher.Verify(hash, "correct-horse-battery") {
		t.Fatal("Verify() = false, want true")
	}
	if hasher.Verify(hash, "wrong-password") {
		t.Fatal("Verify() = true for wrong password")
	}
	if hasher.Verify("not-a-bcrypt-hash", "correct-horse-battery") {
		t.Fatal("Verify() = true for invalid hash")
	}
}

func TestNewBcryptHasherValidatesCost(t *testing.T) {
	for _, cost := range []int{bcrypt.MinCost - 1, bcrypt.MaxCost + 1} {
		if _, err := NewBcryptHasher(cost); err == nil {
			t.Fatalf("NewBcryptHasher(%d) error = nil, want validation error", cost)
		}
	}
}

func newTestIssuer(t *testing.T) *JWTIssuer {
	t.Helper()

	issuer, err := NewJWTIssuer([]byte(testSecret), "issuer", "audience", time.Hour)
	if err != nil {
		t.Fatalf("NewJWTIssuer() error = %v", err)
	}
	return issuer
}

func mustIssue(t *testing.T, issuer *JWTIssuer, user domain.User) string {
	t.Helper()

	token, err := issuer.Issue(user)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	return token.AccessToken
}
