package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirojulanam/boilerplate/internal/domain"
	"github.com/sirojulanam/boilerplate/internal/usecase"
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
		Service: "boilerplate",
		Version: "1.0.0",
	}})

	request := httptest.NewRequest(http.MethodGet, HealthPath, nil)
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

	request := httptest.NewRequest(http.MethodGet, HealthPath, nil)
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
	if location := response.Header().Get("Location"); location != SwaggerUIPath {
		t.Fatalf("location = %q, want %q", location, SwaggerUIPath)
	}
}

func TestSwaggerUIIsServed(t *testing.T) {
	router := newTestRouter(stubHealth{result: domain.Health{Status: domain.StatusOK}})

	request := httptest.NewRequest(http.MethodGet, SwaggerUIPath, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestUnknownRouteReturnsJSON404(t *testing.T) {
	router := newTestRouter(stubHealth{result: domain.Health{Status: domain.StatusOK}})

	request := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusNotFound)
	}

	var body ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != "route not found" {
		t.Fatalf("error = %q, want route not found", body.Error)
	}
}

func TestWrongMethodReturnsJSON405(t *testing.T) {
	router := newTestRouter(stubHealth{result: domain.Health{Status: domain.StatusOK}})

	request := httptest.NewRequest(http.MethodPost, HealthPath, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
