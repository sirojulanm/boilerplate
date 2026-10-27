package httpapi

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// withLogging records every request: method, path, status, latency, client,
// and user agent. Entries with status >= 400 are logged at warn/error level.
func withLogging(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}

		c.Next()

		status := c.Writer.Status()
		entry := log.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       path,
			"status":     status,
			"latency_ms": float64(time.Since(start).Microseconds()) / 1000,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		})

		switch {
		case len(c.Errors) > 0:
			entry.WithField("errors", c.Errors.String()).Error("http request")
		case status >= 500:
			entry.Error("http request")
		case status >= 400:
			entry.Warn("http request")
		default:
			entry.Info("http request")
		}
	}
}
