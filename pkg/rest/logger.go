// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"bytes"
	"net/http"
	"time"
)

// logger records REST requests at the configured verbosity.
func (a *API) logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !log.V(3).Enabled() {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		common := []any{"method", r.Method, "url", r.URL}
		if sn, _ := a.session(r); sn != nil {
			common = append(common, "session", sn.ID)
		}
		detail := log.V(9).Enabled()
		rw := newResponseWriter(w)
		if detail {
			rw.body = &bytes.Buffer{}
			common = append(common, "from", r.RemoteAddr, "body", copyBody(r))
		}
		log.V(3).Info("REST Request", common...)
		next.ServeHTTP(rw, r)
		values := append(common, "status", rw.status, "latency", time.Since(start))
		if detail {
			values = append(values, "response", rw.body.String())
		}
		log.V(3).Info("REST Response", values...)
	})
}
