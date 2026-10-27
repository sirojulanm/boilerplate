package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirojulanam/boilerplate/internal/constant"
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

func decodeEntries(t *testing.T, buffer *bytes.Buffer) []map[string]any {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(buffer.String()), "\n")
	entries := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("decode log entry %q: %v", line, err)
		}
		entries = append(entries, entry)
	}
	return entries
}

func TestMiddlewareLogsEveryRequest(t *testing.T) {
	log, buffer := newBufferedLogger()
	router := newRouterWithLogger(stubHealth{result: domain.Health{Status: domain.StatusOK}}, log)
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, constant.HealthPath, nil))

	entries := decodeEntries(t, buffer)
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1: %s", len(entries), buffer.String())
	}
	entry := entries[0]

	if entry["msg"] != "http request" {
		t.Fatalf("msg = %v, want http request", entry["msg"])
	}
	if entry["path"] != constant.HealthPath {
		t.Fatalf("path = %v, want %s", entry["path"], constant.HealthPath)
	}
	if entry["method"] != http.MethodGet {
		t.Fatalf("method = %v, want GET", entry["method"])
	}
	if entry["status"] != float64(http.StatusOK) {
		t.Fatalf("status = %v, want %d", entry["status"], http.StatusOK)
	}
	if entry["request_id"] == "" {
		t.Fatal("request_id must be present in the access log")
	}
}

func TestMiddlewareLogsNotFoundAsWarn(t *testing.T) {
	log, buffer := newBufferedLogger()
	router := newRouterWithLogger(stubHealth{result: domain.Health{Status: domain.StatusOK}}, log)
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))

	entries := decodeEntries(t, buffer)
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1: %s", len(entries), buffer.String())
	}

	if entries[0]["level"] != "warning" {
		t.Fatalf("level = %v, want warning", entries[0]["level"])
	}
	if entries[0]["status"] != float64(http.StatusNotFound) {
		t.Fatalf("status = %v, want %d", entries[0]["status"], http.StatusNotFound)
	}
}

func TestMiddlewarePropagatesIncomingRequestID(t *testing.T) {
	log, buffer := newBufferedLogger()
	router := newRouterWithLogger(stubHealth{result: domain.Health{Status: domain.StatusOK}}, log)

	request := httptest.NewRequest(http.MethodGet, constant.HealthPath, nil)
	request.Header.Set(constant.RequestIDHeader, "trace-123")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if got := response.Header().Get(constant.RequestIDHeader); got != "trace-123" {
		t.Fatalf("%s = %q, want trace-123", constant.RequestIDHeader, got)
	}

	entries := decodeEntries(t, buffer)
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1: %s", len(entries), buffer.String())
	}
	if entries[0]["request_id"] != "trace-123" {
		t.Fatalf("request_id = %v, want trace-123", entries[0]["request_id"])
	}
}

func TestMiddlewareGeneratesRequestID(t *testing.T) {
	log, buffer := newBufferedLogger()
	router := newRouterWithLogger(stubHealth{result: domain.Health{Status: domain.StatusOK}}, log)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, constant.HealthPath, nil))

	id := response.Header().Get(constant.RequestIDHeader)
	if id == "" {
		t.Fatalf("%s header must be set", constant.RequestIDHeader)
	}

	entries := decodeEntries(t, buffer)
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1: %s", len(entries), buffer.String())
	}
	if entries[0]["request_id"] != id {
		t.Fatalf("request_id = %v, want %s", entries[0]["request_id"], id)
	}
}

func TestPanicIsRecoveredAndLogged(t *testing.T) {
	log, buffer := newBufferedLogger()
	router := newRouterWithLogger(stubHealth{result: domain.Health{Status: domain.StatusOK}}, log)

	router.GET("/panic", func(*gin.Context) {
		panic("boom")
	})

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusInternalServerError)
	}

	var panicEntry, accessEntry map[string]any
	for _, entry := range decodeEntries(t, buffer) {
		switch entry["msg"] {
		case "panic recovered":
			panicEntry = entry
		case "http request":
			accessEntry = entry
		}
	}

	if panicEntry == nil {
		t.Fatalf("panic was not logged: %s", buffer.String())
	}
	if stack, _ := panicEntry["stack"].(string); stack == "" {
		t.Fatal("panic log must include a stack trace")
	}

	if accessEntry == nil {
		t.Fatalf("access log missing after panic: %s", buffer.String())
	}
	if accessEntry["status"] != float64(http.StatusInternalServerError) {
		t.Fatalf("access log status = %v, want %d", accessEntry["status"], http.StatusInternalServerError)
	}
}
