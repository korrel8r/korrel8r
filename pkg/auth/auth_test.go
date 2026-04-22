// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package auth_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/auth"
	"github.com/stretchr/testify/assert"
)

type dummyRoundTripper struct{ *http.Request }

func (d *dummyRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	d.Request = r
	return nil, nil
}

func TestRequestToken(t *testing.T) {
	assert.Equal(t, "my-token", auth.HeaderToken(http.Header{"Authorization": {"Bearer my-token"}}))
	assert.Equal(t, "", auth.HeaderToken(http.Header{"Authorization": {"Basic foo"}}))
	assert.Equal(t, "", auth.HeaderToken(http.Header{}))
}

func TestWithToken_RoundTrip(t *testing.T) {
	drt := dummyRoundTripper{}
	rt := auth.Wrap(&drt)
	for _, x := range []struct {
		in, out, want string
	}{
		{in: "Bearer my-token", out: "", want: "Bearer my-token"},
		{in: "", out: "", want: ""},
		{in: "Bearer my-token", out: "Basic bad:stuff", want: "Bearer my-token"},
		{in: "", out: "Basic bad:stuff", want: "Basic bad:stuff"},
	} {
		t.Run(fmt.Sprintf("%v", x), func(t *testing.T) {
			token := auth.HeaderToken(http.Header{"Authorization": {x.in}})
			ctx := auth.WithToken(context.Background(), token)
			out, err := http.NewRequestWithContext(ctx, "GET", "/", nil)
			out.Header.Set("Authorization", x.out)
			if assert.NoError(t, err) {
				_, _ = rt.RoundTrip(out)
				assert.Equal(t, x.want, drt.Header.Get("Authorization"))
			}
		})
	}
}
