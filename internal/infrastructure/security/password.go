package security

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher implements domain.PasswordHasher using bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher builds a hasher with the requested bcrypt cost.
func NewBcryptHasher(cost int) (*BcryptHasher, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return nil, fmt.Errorf("bcrypt cost must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost)
	}
	return &BcryptHasher{cost: cost}, nil
}

// Hash derives a bcrypt hash from a plaintext password.
func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(hash), nil
}

// Verify reports whether password matches hash. Invalid hashes and mismatches
// both return false.
func (h *BcryptHasher) Verify(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
