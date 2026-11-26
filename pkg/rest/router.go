// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/korrel8r/korrel8r/pkg/session"
)

// apiMux adds authentication and instrumentation to generated REST routes.
type apiMux struct {
	mux      *http.ServeMux
	api      *API
	sessions session.Manager
}

func (m *apiMux) ServeHTTP(w http.ResponseWriter, r *http.Request) { m.mux.ServeHTTP(w, r) }
func (m *apiMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	_, path, _ := strings.Cut(pattern, " ")
	m.api.paths = append(m.api.paths, path)
	h := http.Handler(http.HandlerFunc(handler))
	h = m.api.logger(h)
	h = metrics(path, h)
	h = session.Middleware(m.sessions)(h)
	m.mux.Handle(pattern, Recover(h))
}

// Recover turns a handler panic into a 500 response instead of dropping the connection.
// Panic details are logged, not sent to the client.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := newResponseWriter(w)
		defer func() {
			p := recover()
			switch p {
			case nil:
				return
			case http.ErrAbortHandler: // Deliberate abort, let net/http handle it.
				panic(p)
			}
			log.Error(fmt.Errorf("panic: %v", p), "Recovered from panic",
				"method", r.Method, "url", r.URL, "stack", string(debug.Stack()))
			if !rw.wrote { // Response not started, we can still report the error.
				writeJSON(rw, http.StatusInternalServerError, api.Error{Error: "internal error"})
			}
		}()
		next.ServeHTTP(rw, r)
	})
}
