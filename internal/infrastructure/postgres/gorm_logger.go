package postgres

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	gormlogger "gorm.io/gorm/logger"
)

// gormWriter forwards GORM's internal messages to logrus so the application
// keeps a single logging pipeline.
type gormWriter struct {
	log *logrus.Logger
}

func (w gormWriter) Printf(format string, args ...any) {
	w.log.WithField("component", "gorm").Warnf(strings.TrimSuffix(format, "\n"), args...)
}

func newGormLogger(log *logrus.Logger) gormlogger.Interface {
	if log == nil {
		return gormlogger.Discard
	}

	return gormlogger.New(gormWriter{log: log}, gormlogger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  gormlogger.Warn,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
		Colorful:                  false,
	})
}
