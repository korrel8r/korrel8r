// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package auth manages bearer token authentication for korrel8r.
package auth

import (
	"context"
	"net/http"
	"strings"
)

// ContextToken returns a bearer token from a context, or "" if there is none.
func ContextToken(ctx context.Context) string {
	token, _ := ctx.Value(authKey{}).(string)
	return token
}

// WithToken adds a bearer token to the context. No-op if token == ""
func WithToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, authKey{}, token)
}

// HeaderToken returns the bearer token from an HTTP Authorization header, or "" if none.
func HeaderToken(h http.Header) string {
	if token, ok := strings.CutPrefix(h.Get(headerKey), "Bearer "); ok {
		return token
	}
	return ""
}

// UpdateRequest adds a token to the request context, returns a new request.
func UpdateRequest(req *http.Request) *http.Request {
	if token := HeaderToken(req.Header); token != "" {
		return req.WithContext(WithToken(req.Context(), token))
	}
	return req
}

// Wrap adds bearer token forwarding to outgoing HTTP requests.
// If the request context carries a bearer token (via WithToken), it is set as the Authorization header.
func Wrap(next http.RoundTripper) http.RoundTripper {
	return &roundTripper{next: next}
}

type roundTripper struct{ next http.RoundTripper }

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if token := ContextToken(req.Context()); token != "" {
		req.Header.Set(headerKey, "Bearer "+token)
	}
	return rt.next.RoundTrip(req)
}

type authKey struct{}

const headerKey = "Authorization"
