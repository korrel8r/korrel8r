// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package auth forwards authorization information from an incoming HTTP request to an outgoing store request.
package auth

import (
	"context"
	"net/http"
)

// Token extracts an authorization token from a context, if there is one.
func Token(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(authKey{}).(string)
	return token, ok
}

// WithToken adds an authorization token to a context.
func WithToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, authKey{}, token)
}

// Context returns an authorization-forwarding context for an incoming request.
func Context(req *http.Request) context.Context {
	token := req.Header.Get(Header)
	return WithToken(req.Context(), token)
}

// Wrap adds authorization-forwarding for outgoing requests with an authorization-forwarding context.
func Wrap(next http.RoundTripper) http.RoundTripper {
	return &roundTripper{next: next}
}

type roundTripper struct{ next http.RoundTripper }

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	if token, ok := Token(ctx); ok {
		req.Header.Set(Header, token)
	}
	return rt.next.RoundTrip(req)
}

type authKey struct{}

const Header = "Authorization"
