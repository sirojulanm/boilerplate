package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
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

func middlewareEntries(t *testing.T, buffer *bytes.Buffer) []map[string]any {
	t.Helper()

	entries := make([]map[string]any, 0)
	for _, entry := range decodeEntries(t, buffer) {
		switch entry["event"] {
		case constant.LogEventRequest, constant.LogEventResponse:
			entries = append(entries, entry)
		}
	}
	return entries
}

func TestMiddlewareLogsRequestThenResponse(t *testing.T) {
	log, buffer := newBufferedLogger()
	router := newRouterWithLogger(stubHealth{result: domain.Health{Status: domain.StatusOK}}, log)
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, constant.HealthPath, nil))

	entries := middlewareEntries(t, buffer)
	if len(entries) != 2 {
		t.Fatalf("middleware entries = %d, want 2: %s", len(entries), buffer.String())
	}

	request, response := entries[0], entries[1]
	if request["event"] != constant.LogEventRequest || request["msg"] != "request received" {
		t.Fatalf("first entry is not the request log: %v", request)
	}
	if response["event"] != constant.LogEventResponse || response["msg"] != "response sent" {
		t.Fatalf("second entry is not the response log: %v", response)
	}
	if request["request_id"] != response["request_id"] {
		t.Fatalf("request/response ids differ: %v vs %v", request["request_id"], response["request_id"])
	}

	if response["path"] != constant.HealthPath {
		t.Fatalf("path = %v, want %s", response["path"], constant.HealthPath)
	}
	if response["method"] != http.MethodGet {
		t.Fatalf("method = %v, want GET", response["method"])
	}
	if response["status"] != float64(http.StatusOK) {
		t.Fatalf("status = %v, want %d", response["status"], http.StatusOK)
	}
	if response["latency_ms"] == nil {
		t.Fatal("response log must include latency_ms")
	}
	if response["bytes_written"] == nil {
		t.Fatal("response log must include bytes_written")
	}
}

func TestMiddlewareKeepsRequestProcessResponseOrder(t *testing.T) {
	log, buffer := newBufferedLogger()
	router := newRouterWithLogger(stubHealth{result: domain.Health{
		Status: domain.StatusOK,
		Checks: []domain.Check{domain.NewCheck("runtime", "uptime=1s", nil)},
	}}, log)

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, constant.HealthPath, nil))

	entries := decodeEntries(t, buffer)
	events := make([]string, 0, len(entries))
	requestIDs := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		events = append(events, entry["event"].(string))
		requestIDs[entry["request_id"].(string)] = struct{}{}
	}

	want := []string{constant.LogEventRequest, constant.LogEventProcess, constant.LogEventProcess, constant.LogEventResponse}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("event order = %v, want %v: %s", events, want, buffer.String())
	}
	if len(requestIDs) != 1 {
		t.Fatalf("log entries must share one request id, got %v", requestIDs)
	}
}

func TestMiddlewareLogsNotFoundAsWarn(t *testing.T) {
	log, buffer := newBufferedLogger()
	router := newRouterWithLogger(stubHealth{result: domain.Health{Status: domain.StatusOK}}, log)
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))

	entries := middlewareEntries(t, buffer)
	if len(entries) != 2 {
		t.Fatalf("middleware entries = %d, want 2: %s", len(entries), buffer.String())
	}

	response := entries[1]
	if response["level"] != "warning" {
		t.Fatalf("level = %v, want warning", response["level"])
	}
	if response["status"] != float64(http.StatusNotFound) {
		t.Fatalf("status = %v, want %d", response["status"], http.StatusNotFound)
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

	for _, entry := range middlewareEntries(t, buffer) {
		if entry["request_id"] != "trace-123" {
			t.Fatalf("request_id = %v, want trace-123", entry["request_id"])
		}
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

	for _, entry := range middlewareEntries(t, buffer) {
		if entry["request_id"] != id {
			t.Fatalf("request_id = %v, want %s", entry["request_id"], id)
		}
	}
}

func TestMiddlewareReplacesOversizedRequestID(t *testing.T) {
	log, buffer := newBufferedLogger()
	router := newRouterWithLogger(stubHealth{result: domain.Health{Status: domain.StatusOK}}, log)

	request := httptest.NewRequest(http.MethodGet, constant.HealthPath, nil)
	request.Header.Set(constant.RequestIDHeader, strings.Repeat("x", maxRequestIDLength+1))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	id := response.Header().Get(constant.RequestIDHeader)
	if id == strings.Repeat("x", maxRequestIDLength+1) || id == "" {
		t.Fatalf("oversized request id was not replaced: %q", id)
	}

	for _, entry := range middlewareEntries(t, buffer) {
		if entry["request_id"] != id {
			t.Fatalf("request_id = %v, want %s", entry["request_id"], id)
		}
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

	var panicEntry, responseEntry map[string]any
	for _, entry := range decodeEntries(t, buffer) {
		switch entry["msg"] {
		case "panic recovered":
			panicEntry = entry
		case "response sent":
			responseEntry = entry
		}
	}

	if panicEntry == nil {
		t.Fatalf("panic was not logged: %s", buffer.String())
	}
	if stack, _ := panicEntry["stack"].(string); stack == "" {
		t.Fatal("panic log must include a stack trace")
	}

	if responseEntry == nil {
		t.Fatalf("response log missing after panic: %s", buffer.String())
	}
	if responseEntry["status"] != float64(http.StatusInternalServerError) {
		t.Fatalf("response log status = %v, want %d", responseEntry["status"], http.StatusInternalServerError)
	}
}
