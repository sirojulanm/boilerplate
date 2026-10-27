package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirupsen/logrus"
)

// Config controls how the application logger is built.
type Config struct {
	// Level is a logrus level name such as debug, info, warn, or error.
	Level string
	// Dir is the directory that receives the log file. It is created if missing.
	Dir string
	// Filename is the log file name inside Dir.
	Filename string
}

// Logger wraps logrus and owns the log file handle.
type Logger struct {
	*logrus.Logger
	file *os.File
}

// New builds a JSON logger that writes to both stdout and Dir/Filename.
func New(cfg Config) (*Logger, error) {
	if cfg.Level == "" {
		cfg.Level = constant.DefaultLogLevel
	}
	if cfg.Dir == "" {
		cfg.Dir = constant.DefaultLogDir
	}
	if cfg.Filename == "" {
		cfg.Filename = constant.DefaultLogFilename
	}

	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level %q: %w", cfg.Level, err)
	}

	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log directory %q: %w", cfg.Dir, err)
	}

	path := filepath.Join(cfg.Dir, cfg.Filename)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file %q: %w", path, err)
	}

	instance := logrus.New()
	instance.SetLevel(level)
	instance.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339})
	instance.SetOutput(io.MultiWriter(os.Stdout, file))

	return &Logger{Logger: instance, file: file}, nil
}

// Close flushes pending output and releases the log file.
func (l *Logger) Close() error {
	if l.file == nil {
		return nil
	}
	return l.file.Close()
}
