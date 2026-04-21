// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package auth manages bearer token authentication for korrel8r.
package auth

import (
	"context"
	"net/http"
	"strings"
)

// Token extracts a bearer token from a context, if there is one.
func Token(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(authKey{}).(string)
	return token, ok
}

// WithToken adds a bearer token to a context.
func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authKey{}, token)
}

// Context returns a context carrying the bearer token from an HTTP request's Authorization header.
// If the header is not a Bearer token, the returned context has no token.
func Context(req *http.Request) context.Context {
	header := req.Header.Get(headerKey)
	if token, ok := strings.CutPrefix(header, "Bearer "); ok {
		return WithToken(req.Context(), token)
	}
	return req.Context()
}

// Wrap adds bearer token forwarding to outgoing HTTP requests.
// If the request context carries a bearer token (via WithToken), it is set as the Authorization header.
func Wrap(next http.RoundTripper) http.RoundTripper {
	return &roundTripper{next: next}
}

type roundTripper struct{ next http.RoundTripper }

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if token, ok := Token(req.Context()); ok {
		req.Header.Set(headerKey, "Bearer "+token)
	}
	return rt.next.RoundTrip(req)
}

type authKey struct{}

const headerKey = "Authorization"
