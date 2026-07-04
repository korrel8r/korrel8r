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

	start := time.Now()
	detail := log.V(9).Enabled() // Extra detail

	values := []any{
		"method", c.Request.Method,
		"url", c.Request.URL,
	}
	var rw *responseWriter
	if detail {
		rw = newResponseWriter(c.Writer)
		c.Writer = rw // Save the response
		values = append(values,
			"from", c.Request.RemoteAddr,
			"body", copyBody(c.Request),
		)
	}

	log.V(3).Info("REST Request", values...)

	defer func() {
		values = append(values,
			"status", c.Writer.Status(),
			"latency", time.Since(start),
		)
		if detail {
			values = append(values,
				"request", copyBody(c.Request),
				"response", rw.String(),
			)
		}
		if len(c.Errors.Errors()) > 0 {
			values = append(values, "errors", c.Errors.Errors())
		}
		if c.IsAborted() || c.Writer.Status()/100 != 2 {
			log.V(2).Info("REST Request failed", values...)
		} else {
			log.V(3).Info("REST Request OK", values...)
		}
	}()
	c.Next()
}
