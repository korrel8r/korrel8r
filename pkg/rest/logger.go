// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// logger is a Gin handler to log requests for debugging.
func (a *API) logger(c *gin.Context) {
	if !log.V(2).Enabled() {
		return // Nothing to do for V < 2
	}
	log := log // Local variable, can assign without changing global log.
	log = log.WithValues(
		"method", c.Request.Method,
		"url", c.Request.URL,
		"from", c.Request.RemoteAddr,
	)
	log.V(4).Info("Request received", "body", copyBody(c.Request))
	// Wrap the ResponseWriter to capture the response
	rw := newResponseWriter(c.Writer)
	c.Writer = rw
	start := time.Now()

	defer func() {
		latency := time.Since(start)
		status := c.Writer.Status()
		log = log.WithValues("code", status, "text", http.StatusText(status), "latency", latency)
		if len(c.Errors.Errors()) > 0 {
			log = log.WithValues("errors", c.Errors.Errors())
		}
		if log.V(5).Enabled() { // Response is big, trace at per-object level.
			log = log.WithValues("response", rw.String())
		}
		if c.IsAborted() || c.Writer.Status()/100 != 2 {
			log.V(2).Info("Request failed")
		} else {
			log.V(3).Info("Request succeeded")
		}
	}()
	c.Next()
}
