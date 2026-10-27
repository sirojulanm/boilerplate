package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirojulanam/boilerplate/internal/domain"
	"github.com/sirojulanam/boilerplate/internal/usecase"
	"github.com/sirupsen/logrus"
)

type stubAuth struct {
	register func(context.Context, usecase.RegisterInput) (domain.User, error)
	token    func(context.Context, string, string) (usecase.TokenResult, error)
	profile  func(context.Context, uuid.UUID) (domain.User, error)
}

func (s stubAuth) Register(ctx context.Context, input usecase.RegisterInput) (domain.User, error) {
	if s.register == nil {
		return domain.User{}, nil
	}
	return s.register(ctx, input)
}

func (s stubAuth) GenerateToken(ctx context.Context, email, password string) (usecase.TokenResult, error) {
	if s.token == nil {
		return usecase.TokenResult{}, nil
	}
	return s.token(ctx, email, password)
}

func (s stubAuth) Profile(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	if s.profile == nil {
		return domain.User{}, nil
	}
	return s.profile(ctx, userID)
}

func newAuthRouter(auth usecase.Auth, verifier domain.TokenVerifier, log *logrus.Logger) *gin.Engine {
	gin.SetMode(gin.TestMode)
	return NewRouter(Dependencies{
		Health:        NewHealthHandler(stubHealth{result: domain.Health{Status: domain.StatusOK}}, log),
		Auth:          NewAuthHandler(auth, log),
		TokenVerifier: verifier,
		Log:           log,
	})
}

func postJSON(t *testing.T, router *gin.Engine, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestRegisterEndpointCreatesUser(t *testing.T) {
	user := domain.User{
		ID:        uuid.New(),
		Name:      "Jane Doe",
		Email:     "jane@example.com",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	var gotInput usecase.RegisterInput
	router := newAuthRouter(stubAuth{
		register: func(_ context.Context, input usecase.RegisterInput) (domain.User, error) {
			gotInput = input
			return user, nil
		},
	}, stubTokenVerifier{}, newTestLogger())

	response := postJSON(t, router, constant.RegisterPath, `{
		"name": "Jane Doe",
		"email": "jane@example.com",
		"password": "correct-horse-battery"
	}`)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if gotInput.Email != "jane@example.com" || gotInput.Password != "correct-horse-battery" {
		t.Fatalf("unexpected usecase input: %+v", gotInput)
	}

	rawBody := response.Body.String()
	if strings.Contains(rawBody, "password_hash") || strings.Contains(rawBody, "super-secret") {
		t.Fatalf("response must not expose the password hash: %s", rawBody)
	}

	var body UserResponse
	if err := json.NewDecoder(strings.NewReader(rawBody)).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Success {
		t.Fatal("success must be true")
	}
	if body.Data.ID != user.ID {
		t.Fatalf("user id = %s, want %s", body.Data.ID, user.ID)
	}
}

func TestRegisterEndpointRejectsInvalidJSON(t *testing.T) {
	router := newAuthRouter(stubAuth{}, stubTokenVerifier{}, newTestLogger())
	response := postJSON(t, router, constant.RegisterPath, `{"name":`)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	var body ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Message != constant.ErrInvalidRequest {
		t.Fatalf("error = %q, want %q", body.Error.Message, constant.ErrInvalidRequest)
	}
	if body.Error.Code != constant.CodeInvalidRequest {
		t.Fatalf("error code = %q, want %q", body.Error.Code, constant.CodeInvalidRequest)
	}
	if body.RequestID == "" || body.RequestID != response.Header().Get(constant.RequestIDHeader) {
		t.Fatalf("response request_id must match header: body=%q header=%q", body.RequestID, response.Header().Get(constant.RequestIDHeader))
	}
}

func TestRegisterEndpointReturnsConflictForDuplicateEmail(t *testing.T) {
	router := newAuthRouter(stubAuth{
		register: func(context.Context, usecase.RegisterInput) (domain.User, error) {
			return domain.User{}, domain.ErrEmailTaken
		},
	}, stubTokenVerifier{}, newTestLogger())

	response := postJSON(t, router, constant.RegisterPath, `{
		"name": "Jane Doe",
		"email": "jane@example.com",
		"password": "correct-horse-battery"
	}`)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestRegisterEndpointDoesNotLeakInternalError(t *testing.T) {
	log, buffer := newBufferedLogger()
	router := newAuthRouter(stubAuth{
		register: func(context.Context, usecase.RegisterInput) (domain.User, error) {
			return domain.User{}, errors.New("password=secret database failure")
		},
	}, stubTokenVerifier{}, log)

	response := postJSON(t, router, constant.RegisterPath, `{
		"name": "Jane Doe",
		"email": "jane@example.com",
		"password": "correct-horse-battery"
	}`)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if strings.Contains(response.Body.String(), "password=secret") {
		t.Fatalf("response leaked internal error: %s", response.Body.String())
	}
	if !strings.Contains(buffer.String(), "password=secret") {
		t.Fatal("internal error should be logged")
	}
}

func TestRegisterEndpointRejectsOversizedBody(t *testing.T) {
	router := newAuthRouter(stubAuth{}, stubTokenVerifier{}, newTestLogger())
	response := postJSON(t, router, constant.RegisterPath, `{"name":"`+strings.Repeat("a", constant.MaxRequestBodyBytes)+`"}`)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}

	var body ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != constant.CodeRequestTooLarge {
		t.Fatalf("error code = %q, want %q", body.Error.Code, constant.CodeRequestTooLarge)
	}
}

func TestTokenEndpointReturnsAccessToken(t *testing.T) {
	user := domain.User{ID: uuid.New(), Name: "Jane", Email: "jane@example.com"}
	expiresAt := time.Now().Add(time.Hour).UTC()

	router := newAuthRouter(stubAuth{
		token: func(_ context.Context, email, password string) (usecase.TokenResult, error) {
			if email != "jane@example.com" || password != "correct-horse-battery" {
				t.Fatalf("unexpected credentials: %q / %q", email, password)
			}
			return usecase.TokenResult{
				User: user,
				Token: domain.Token{
					AccessToken: "signed-token",
					TokenType:   constant.BearerScheme,
					ExpiresAt:   expiresAt,
					ExpiresIn:   3600,
				},
			}, nil
		},
	}, stubTokenVerifier{}, newTestLogger())

	response := postJSON(t, router, constant.TokenPath, `{
		"email": "jane@example.com",
		"password": "correct-horse-battery"
	}`)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var body TokenResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Success {
		t.Fatal("success must be true")
	}
	if body.Data.AccessToken != "signed-token" || body.Data.TokenType != constant.BearerScheme {
		t.Fatalf("unexpected token response: %+v", body)
	}
	if body.Data.ExpiresIn != 3600 {
		t.Fatalf("expires_in = %d, want 3600", body.Data.ExpiresIn)
	}
	if body.Data.User.ID != user.ID {
		t.Fatalf("user id = %s, want %s", body.Data.User.ID, user.ID)
	}
}

func TestTokenEndpointReturnsUnauthorizedForBadCredentials(t *testing.T) {
	router := newAuthRouter(stubAuth{
		token: func(context.Context, string, string) (usecase.TokenResult, error) {
			return usecase.TokenResult{}, domain.ErrInvalidCredentials
		},
	}, stubTokenVerifier{}, newTestLogger())

	response := postJSON(t, router, constant.TokenPath, `{
		"email": "jane@example.com",
		"password": "wrong-password"
	}`)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestMeRequiresBearerToken(t *testing.T) {
	router := newAuthRouter(stubAuth{}, stubTokenVerifier{}, newTestLogger())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, constant.MePath, nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if header := response.Header().Get("WWW-Authenticate"); !strings.Contains(header, constant.BearerScheme) {
		t.Fatalf("WWW-Authenticate = %q, want Bearer challenge", header)
	}
}

func TestMeRejectsInvalidToken(t *testing.T) {
	router := newAuthRouter(stubAuth{}, stubTokenVerifier{err: domain.ErrInvalidToken}, newTestLogger())

	request := httptest.NewRequest(http.MethodGet, constant.MePath, nil)
	request.Header.Set("Authorization", "Bearer broken")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestMeReturnsAuthenticatedProfile(t *testing.T) {
	user := domain.User{ID: uuid.New(), Name: "Jane", Email: "jane@example.com"}

	router := newAuthRouter(stubAuth{
		profile: func(_ context.Context, userID uuid.UUID) (domain.User, error) {
			if userID != user.ID {
				t.Fatalf("user id = %s, want %s", userID, user.ID)
			}
			return user, nil
		},
	}, stubTokenVerifier{identity: domain.Identity{UserID: user.ID}}, newTestLogger())

	request := httptest.NewRequest(http.MethodGet, constant.MePath, nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var body UserResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.ID != user.ID {
		t.Fatalf("user id = %s, want %s", body.Data.ID, user.ID)
	}
}
