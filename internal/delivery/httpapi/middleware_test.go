package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirojulanam/boilerplate/internal/domain"
	"github.com/sirupsen/logrus"
)

func newBufferedLogger() (*logrus.Logger, *bytes.Buffer) {
	buffer := &bytes.Buffer{}
	log := logrus.New()
	log.SetOutput(buffer)
	log.SetFormatter(&logrus.JSONFormatter{})
	return log, buffer
}

func TestMiddlewareLogsEveryRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	log, buffer := newBufferedLogger()
	router := NewRouter(NewHealthHandler(stubHealth{result: domain.Health{Status: domain.StatusOK}}), log)
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, HealthPath, nil))

	var entry map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &entry); err != nil {
		t.Fatalf("decode log entry %q: %v", buffer.String(), err)
	}

	if entry["msg"] != "http request" {
		t.Fatalf("msg = %v, want http request", entry["msg"])
	}
	if entry["path"] != HealthPath {
		t.Fatalf("path = %v, want %s", entry["path"], HealthPath)
	}
	if entry["method"] != http.MethodGet {
		t.Fatalf("method = %v, want GET", entry["method"])
	}
	if entry["status"] != float64(http.StatusOK) {
		t.Fatalf("status = %v, want %d", entry["status"], http.StatusOK)
	}
}

func TestMiddlewareLogsNotFoundAsWarn(t *testing.T) {
	gin.SetMode(gin.TestMode)

	log, buffer := newBufferedLogger()
	router := NewRouter(NewHealthHandler(stubHealth{result: domain.Health{Status: domain.StatusOK}}), log)
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))

	var entry map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &entry); err != nil {
		t.Fatalf("decode log entry %q: %v", buffer.String(), err)
	}

	if entry["level"] != "warning" {
		t.Fatalf("level = %v, want warning", entry["level"])
	}
	if entry["status"] != float64(http.StatusNotFound) {
		t.Fatalf("status = %v, want %d", entry["status"], http.StatusNotFound)
	}
}
