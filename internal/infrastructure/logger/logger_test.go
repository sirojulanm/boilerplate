package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCreatesDirectoryAndWritesFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "logs")

	log, err := New(Config{Level: "debug", Dir: dir, Filename: "test.log"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	log.WithField("endpoint", "/api/v1/health").Info("request handled")

	if err := log.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "test.log"))
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(content), `"endpoint":"/api/v1/health"`) {
		t.Fatalf("log file missing entry: %s", content)
	}
}

func TestNewRejectsInvalidLevel(t *testing.T) {
	if _, err := New(Config{Level: "loud", Dir: t.TempDir()}); err == nil {
		t.Fatal("expected error for invalid level")
	}
}
