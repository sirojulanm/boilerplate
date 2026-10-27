package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirojulanam/opencodeproject/internal/domain"
	"github.com/sirojulanam/opencodeproject/internal/usecase"
	"github.com/sirupsen/logrus"
)

type stubHealth struct {
	result domain.Health
}

func (s stubHealth) Check(context.Context) domain.Health { return s.result }

func newTestLogger() *logrus.Logger {
	log := logrus.New()
	log.SetOutput(io.Discard)
	return log
}

func newTestRouter(uc usecase.Health) *gin.Engine {
	gin.SetMode(gin.TestMode)
	return NewRouter(NewHealthHandler(uc), newTestLogger())
}

func TestHealthEndpointReturnsOK(t *testing.T) {
	router := newTestRouter(stubHealth{result: domain.Health{
		Status:  domain.StatusOK,
		Service: "opencodeproject",
		Version: "1.0.0",
	}})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}

	var body domain.Health
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != domain.StatusOK {
		t.Fatalf("body status = %q, want %q", body.Status, domain.StatusOK)
	}
}

func TestHealthEndpointReturns503WhenDegraded(t *testing.T) {
	router := newTestRouter(stubHealth{result: domain.Health{Status: domain.StatusDegraded}})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestRootRedirectsToSwagger(t *testing.T) {
	router := newTestRouter(stubHealth{result: domain.Health{Status: domain.StatusOK}})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusTemporaryRedirect)
	}
}
