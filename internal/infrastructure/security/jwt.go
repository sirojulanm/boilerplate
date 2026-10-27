package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirojulanam/boilerplate/internal/domain"
)

const tokenLeeway = 30 * time.Second

// JWTIssuer signs and verifies HS256 access tokens.
type JWTIssuer struct {
	secret   []byte
	issuer   string
	audience string
	ttl      time.Duration
	now      func() time.Time
}

// NewJWTIssuer builds a token issuer. The secret must be strong enough for
// HS256; configuration also enforces this at startup.
func NewJWTIssuer(secret []byte, issuer, audience string, ttl time.Duration) (*JWTIssuer, error) {
	if len(secret) < constant.MinJWTSecretLength {
		return nil, fmt.Errorf("jwt secret must be at least %d bytes", constant.MinJWTSecretLength)
	}
	if issuer == "" {
		return nil, errors.New("jwt issuer is required")
	}
	if audience == "" {
		return nil, errors.New("jwt audience is required")
	}
	if ttl <= 0 {
		return nil, errors.New("jwt ttl must be positive")
	}

	return &JWTIssuer{
		secret:   append([]byte(nil), secret...),
		issuer:   issuer,
		audience: audience,
		ttl:      ttl,
		now:      time.Now,
	}, nil
}

// Issue creates a signed access token for the user.
func (j *JWTIssuer) Issue(user domain.User) (domain.Token, error) {
	now := j.now().UTC()
	expiresAt := now.Add(j.ttl)

	claims := jwt.RegisteredClaims{
		Issuer:    j.issuer,
		Subject:   user.ID.String(),
		Audience:  jwt.ClaimStrings{j.audience},
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
		ID:        uuid.NewString(),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
	if err != nil {
		return domain.Token{}, fmt.Errorf("sign token: %w", err)
	}

	return domain.Token{
		AccessToken: signed,
		TokenType:   constant.BearerScheme,
		ExpiresAt:   expiresAt,
		ExpiresIn:   int64(j.ttl.Seconds()),
	}, nil
}

// Verify validates the signature, issuer, audience, and expiry of a token.
func (j *JWTIssuer) Verify(rawToken string) (domain.Identity, error) {
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(
		rawToken,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, domain.ErrInvalidToken
			}
			return j.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(j.issuer),
		jwt.WithAudience(j.audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(tokenLeeway),
		jwt.WithTimeFunc(j.now),
	)
	if err != nil || !token.Valid {
		return domain.Identity{}, domain.ErrInvalidToken
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return domain.Identity{}, domain.ErrInvalidToken
	}

	return domain.Identity{UserID: userID}, nil
}
