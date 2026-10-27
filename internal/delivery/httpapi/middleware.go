package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirojulanam/boilerplate/internal/constant"
	"github.com/sirojulanam/boilerplate/internal/helper"
	"github.com/sirupsen/logrus"
)

const maxRequestIDLength = 128

// withRequestID reuses an incoming X-Request-ID or generates a new one. The ID
// is stored on the context so every request, process, and response log entry
// can be correlated.
func withRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(constant.RequestIDHeader)
		if id == "" || len(id) > maxRequestIDLength {
			id = helper.NewRequestID()
		}

		c.Set(constant.RequestIDKey, id)
		c.Header(constant.RequestIDHeader, id)
		c.Next()
	}
}

// withBodyLimit rejects bodies larger than limit while they are being read.
func withBodyLimit(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}

// withLogging writes two ordered entries per request: one when the request
// arrives and one when the response is sent. Process logs from handlers land
// between them, all sharing the same request_id. The response entry is deferred
// so a recovered panic still produces a final status/latency record.
func withLogging(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path = path + "?" + raw
		}

		requestID := c.GetString(constant.RequestIDKey)
		baseFields := logrus.Fields{
			"request_id": requestID,
			"method":     c.Request.Method,
			"path":       path,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}

		requestFields := logrus.Fields{"event": constant.LogEventRequest}
		for key, value := range baseFields {
			requestFields[key] = value
		}
		requestFields["content_length"] = c.Request.ContentLength
		log.WithFields(requestFields).Info("request received")

		defer func() {
			status := c.Writer.Status()

			responseFields := logrus.Fields{
				"event":         constant.LogEventResponse,
				"request_id":    requestID,
				"method":        c.Request.Method,
				"path":          path,
				"status":        status,
				"latency_ms":    helper.DurationMS(time.Since(start)),
				"bytes_written": c.Writer.Size(),
				"client_ip":     c.ClientIP(),
				"user_agent":    c.Request.UserAgent(),
			}

			entry := log.WithFields(responseFields)
			switch {
			case len(c.Errors) > 0:
				entry.WithField("errors", c.Errors.String()).Error("response sent")
			case status >= http.StatusInternalServerError:
				entry.Error("response sent")
			case status >= http.StatusBadRequest:
				entry.Warn("response sent")
			default:
				entry.Info("response sent")
			}
		}()

		c.Next()
	}
}
