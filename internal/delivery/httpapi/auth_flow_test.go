package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirojulanam/boilerplate/internal/domain"
	"github.com/sirojulanam/boilerplate/internal/infrastructure/security"
	"github.com/sirojulanam/boilerplate/internal/usecase"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type memoryUserRepository struct {
	mu     sync.RWMutex
	byID   map[uuid.UUID]domain.User
	byMail map[string]domain.User
}

func newMemoryUserRepository() *memoryUserRepository {
	return &memoryUserRepository{
		byID:   make(map[uuid.UUID]domain.User),
		byMail: make(map[string]domain.User),
	}
}

func (m *memoryUserRepository) Create(_ context.Context, user domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.byMail[user.Email]; exists {
		return domain.ErrEmailTaken
	}
	m.byID[user.ID] = user
	m.byMail[user.Email] = user
	return nil
}

func (m *memoryUserRepository) FindByEmail(_ context.Context, email string) (domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.byMail[domain.NormalizeEmail(email)]
	if !exists {
		return domain.User{}, domain.ErrUserNotFound
	}
	return user, nil
}

func (m *memoryUserRepository) FindByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.byID[id]
	if !exists {
		return domain.User{}, domain.ErrUserNotFound
	}
	return user, nil
}

func newAuthFlowRouter(t *testing.T, log *bytes.Buffer) *gin.Engine {
	t.Helper()

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	if log != nil {
		logger.SetOutput(log)
	} else {
		logger.SetOutput(io.Discard)
	}

	hasher, err := security.NewBcryptHasher(bcrypt.MinCost)
	if err != nil {
		t.Fatalf("NewBcryptHasher() error = %v", err)
	}
	tokens, err := security.NewJWTIssuer([]byte("0123456789abcdef0123456789abcdef"), "test-issuer", "test-audience", time.Hour)
	if err != nil {
		t.Fatalf("NewJWTIssuer() error = %v", err)
	}

	gin.SetMode(gin.TestMode)
	return NewRouter(Dependencies{
		Health:        NewHealthHandler(stubHealth{result: domain.Health{Status: domain.StatusOK}}, logger),
		Auth:          NewAuthHandler(usecase.NewAuth(newMemoryUserRepository(), hasher, tokens), logger),
		TokenVerifier: tokens,
		Log:           logger,
	})
}

func decodeEnvelope[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()

	var body T
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return body
}

func TestAuthFlowRegisterTokenProfile(t *testing.T) {
	router := newAuthFlowRouter(t, nil)

	register := postJSON(t, router, constant.RegisterPath, `{
		"name": "Jane Doe",
		"email": "Jane@Example.com",
		"password": "correct-horse-battery"
	}`)
	if register.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d: %s", register.Code, http.StatusCreated, register.Body.String())
	}
	registered := decodeEnvelope[UserResponse](t, register)
	if registered.Data.Email != "jane@example.com" {
		t.Fatalf("registered email = %q, want normalized email", registered.Data.Email)
	}
	if registered.RequestID == "" || registered.RequestID != register.Header().Get(constant.RequestIDHeader) {
		t.Fatalf("register request_id mismatch: body=%q header=%q", registered.RequestID, register.Header().Get(constant.RequestIDHeader))
	}

	token := postJSON(t, router, constant.TokenPath, `{
		"email": "jane@example.com",
		"password": "correct-horse-battery"
	}`)
	if token.Code != http.StatusOK {
		t.Fatalf("token status = %d, want %d: %s", token.Code, http.StatusOK, token.Body.String())
	}
	issued := decodeEnvelope[TokenResponse](t, token)
	if issued.Data.AccessToken == "" || issued.Data.TokenType != constant.BearerScheme {
		t.Fatalf("unexpected token payload: %+v", issued.Data)
	}
	if issued.Data.User.ID != registered.Data.ID {
		t.Fatalf("token user id = %s, want %s", issued.Data.User.ID, registered.Data.ID)
	}

	request := httptest.NewRequest(http.MethodGet, constant.MePath, nil)
	request.Header.Set("Authorization", constant.BearerScheme+" "+issued.Data.AccessToken)
	profile := httptest.NewRecorder()
	router.ServeHTTP(profile, request)

	if profile.Code != http.StatusOK {
		t.Fatalf("profile status = %d, want %d: %s", profile.Code, http.StatusOK, profile.Body.String())
	}
	me := decodeEnvelope[UserResponse](t, profile)
	if me.Data.ID != registered.Data.ID || me.Data.Email != "jane@example.com" {
		t.Fatalf("profile = %+v, want id=%s", me.Data, registered.Data.ID)
	}

	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, constant.MePath, nil))
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d, want %d", missing.Code, http.StatusUnauthorized)
	}
	unauthorized := decodeEnvelope[ErrorResponse](t, missing)
	if unauthorized.Success || unauthorized.Error.Code != constant.CodeUnauthorized {
		t.Fatalf("unexpected unauthorized envelope: %+v", unauthorized)
	}
}

func TestAuthFlowWrongPasswordIsGeneric(t *testing.T) {
	router := newAuthFlowRouter(t, nil)

	register := postJSON(t, router, constant.RegisterPath, `{
		"name": "Jane Doe",
		"email": "jane@example.com",
		"password": "correct-horse-battery"
	}`)
	if register.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d", register.Code, http.StatusCreated)
	}

	token := postJSON(t, router, constant.TokenPath, `{
		"email": "jane@example.com",
		"password": "wrong-password"
	}`)
	if token.Code != http.StatusUnauthorized {
		t.Fatalf("token status = %d, want %d: %s", token.Code, http.StatusUnauthorized, token.Body.String())
	}

	body := decodeEnvelope[ErrorResponse](t, token)
	if body.Error.Code != constant.CodeInvalidCredentials || body.Error.Message != constant.ErrInvalidCredentials {
		t.Fatalf("unexpected error envelope: %+v", body)
	}
}

func TestAuthFlowLogsRequestProcessResponse(t *testing.T) {
	buffer := &bytes.Buffer{}
	router := newAuthFlowRouter(t, buffer)
	postJSON(t, router, constant.RegisterPath, `{
		"name": "Jane Doe",
		"email": "jane@example.com",
		"password": "correct-horse-battery"
	}`)

	entries := decodeEntries(t, buffer)
	if len(entries) < 4 {
		t.Fatalf("expected request, process, and response logs, got: %s", buffer.String())
	}
	if entries[0]["event"] != constant.LogEventRequest {
		t.Fatalf("first event = %v, want request", entries[0]["event"])
	}
	last := entries[len(entries)-1]
	if last["event"] != constant.LogEventResponse {
		t.Fatalf("last event = %v, want response", last["event"])
	}

	requestID := entries[0]["request_id"]
	for _, entry := range entries {
		if entry["request_id"] != requestID {
			t.Fatalf("request_id mismatch: %v vs %v", entry["request_id"], requestID)
		}
	}
	if _, ok := last["status"]; !ok {
		t.Fatal("response log must include status")
	}
}
