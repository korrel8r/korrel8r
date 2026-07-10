// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"time"

	"github.com/gin-gonic/gin"
)

// logger is a Gin handler to log requests for debugging.
func (a *API) logger(c *gin.Context) {
	if log.V(3).Enabled() {
		start := time.Now()
		detail := log.V(9).Enabled() // Extra detail
		common := []any{"method", c.Request.Method, "url", c.Request.URL}
		if sn, _ := a.session(c); sn != nil {
			common = append(common, "session", sn.ID)
		}

		// Log on receiving request
		values := common
		var rw *responseWriter
		if detail {
			rw = newResponseWriter(c.Writer)
			c.Writer = rw // Save the response
			values = append(values, "from", c.Request.RemoteAddr, "body", copyBody(c.Request))
		}
		log.V(3).Info("REST Request", values...)

		// Log before sending response
		defer func() {
			values := append(common, "status", c.Writer.Status(), "latency", time.Since(start))
			if len(c.Errors.Errors()) > 0 {
				values = append(values, "errors", c.Errors.Errors())
			}
			if detail {
				values = append(values, "response", rw.String())
			}
			log.V(3).Info("REST Response", values...)
		}()
	}
	c.Next()
}
