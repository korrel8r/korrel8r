// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"net/http"
	"net/http/pprof"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/session"
)

// WebProfile enables profiling HTTP endpoints only when explicitly requested.
// It returns the paths it registered.
func WebProfile(router *http.ServeMux, sessions session.Manager) []string {
	var paths []string
	register := func(pattern string, handler http.Handler) {
		_, path, _ := strings.Cut(pattern, " ")
		paths = append(paths, path)
		router.Handle(pattern, Recover(session.Middleware(sessions)(handler)))
	}
	register("GET /debug/pprof/", http.HandlerFunc(pprof.Index))
	register("GET /debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	register("GET /debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	register("GET /debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	register("POST /debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	register("GET /debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	for _, name := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
		register("GET /debug/pprof/"+name, pprof.Handler(name))
	}
	return paths
}
