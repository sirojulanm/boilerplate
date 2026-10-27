package httpapi

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirojulanam/boilerplate/internal/helper"
	"github.com/sirupsen/logrus"
)

// withRequestID reuses an incoming X-Request-ID or generates a new one. The ID
// is stored on the context so every log entry for the request can be traced.
func withRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(constant.RequestIDHeader)
		if id == "" {
			id = helper.NewRequestID()
		}

		c.Set(constant.RequestIDKey, id)
		c.Header(constant.RequestIDHeader, id)
		c.Next()
	}
}

// withLogging records every request: method, path, status, latency, client,
// user agent, and request ID. Entries with status >= 400 are logged at
// warn/error level. The log call is deferred so a recovered panic is still
// recorded with its final status code.
func withLogging(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}

		defer func() {
			status := c.Writer.Status()
			entry := log.WithFields(logrus.Fields{
				"request_id": c.GetString(constant.RequestIDKey),
				"method":     c.Request.Method,
				"path":       path,
				"status":     status,
				"latency_ms": helper.DurationMS(time.Since(start)),
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
		}()

		c.Next()
	}
}
