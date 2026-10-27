package httpapi

import "github.com/sirojulanam/boilerplate/internal/domain"

type stubTokenVerifier struct {
	identity domain.Identity
	err      error
}

func (s stubTokenVerifier) Verify(string) (domain.Identity, error) {
	return s.identity, s.err
}
