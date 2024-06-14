// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package auth forwards authorization information from an incoming REST request to an outgoing store request.
package auth

import (
	"context"
	"net/http"
)

// Context returns an authorization-forwarding context for an incoming request.
func Context(req *http.Request) context.Context {
	auth := req.Header.Get(authorization)
	return context.WithValue(req.Context(), authKey{}, auth)
}

// Wrap adds authorization-forwarding for outgoing requests with an authorization-forwarding context.
func Wrap(next http.RoundTripper) http.RoundTripper {
	return &roundTripper{next: next}
}

type roundTripper struct{ next http.RoundTripper }

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	if auth, ok := ctx.Value(authKey{}).(string); ok {
		req.Header.Set(authorization, auth)
	}
	return rt.next.RoundTrip(req)
}

type authKey struct{}

const authorization = "Authorization"
