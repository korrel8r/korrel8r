// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"time"

	"github.com/gin-gonic/gin"
)

// logger is a Gin handler to log requests for debugging.
func (a *API) logger(c *gin.Context) {
	if !log.V(2).Enabled() {
		return // Nothing to do for V < 2
	}
	var rw *responseWriter
	if log.V(9).Enabled() {
		rw = newResponseWriter(c.Writer) // Save the response
		c.Writer = rw
	}
	start := time.Now()

	defer func() {
		latency := time.Since(start)
		status := c.Writer.Status()
		values := []any{
			"method", c.Request.Method,
			"url", c.Request.URL,
			"status", status,
			"latency", latency,
		}
		if rw != nil { // Response was saved, extra detail
			values = append(values,
				"from", c.Request.RemoteAddr,
				"body", copyBody(c.Request),
				"response", rw.String(),
			)
		}
		if len(c.Errors.Errors()) > 0 {
			values = append(values, "errors", c.Errors.Errors())
		}
		if c.IsAborted() || c.Writer.Status()/100 != 2 {
			log.V(2).Info("Request failed", values...)
		} else {
			log.V(3).Info("Request OK", values...)
		}
	}()
	c.Next()
}
