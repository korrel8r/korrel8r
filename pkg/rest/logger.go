// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/pkg/rest/auth"
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
	if log.V(3).Enabled() {
		auth := c.Request.Header.Get(auth.Header)
		log = log.WithValues("hasAuth", auth != "")
	}
	log.V(3).Info("Request received", "body", copyBody(c.Request))
	// Wrap the ResponseWriter to capture the response
	rw := newResponseWriter(c.Writer)
	c.Writer = rw
	start := time.Now()

	defer func() {
		latency := time.Since(start)
		log = log.WithValues("status", c.Writer.Status(), "latency", latency)
		if c.IsAborted() {
			log = log.WithValues("errors", c.Errors.Errors())
		}
		if log.V(5).Enabled() { // Response is big, trace at per-object level.
			log = log.WithValues("response", rw.String())
		}
		if c.IsAborted() || c.Writer.Status()/100 != 2 {
			log.V(2).Info("Request failed")
		} else {
			log.V(3).Info("Request success")
		}
	}()
	c.Next()
}
