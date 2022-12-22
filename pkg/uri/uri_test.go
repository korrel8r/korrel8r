package uri_test

import (
	"strings"
	"testing"

	"github.com/korrel8/korrel8/pkg/uri"
	"github.com/stretchr/testify/assert"
)

// TestRoundTrip tests valid URIs that round-trip from Parse to String
func TestRoundTrip(t *testing.T) {
	for _, s := range []string{
		"path/to/somewhere",
		"/path/to/somewhere",
		"path?query",
		"?query",
		"",
	} {
		t.Run(s, func(t *testing.T) {
			r, err := uri.Parse(s)
			if assert.NoError(t, err) {
				assert.Equal(t, s, r.String())
			}
		})
	}
}

// TesNotRoundTrip test valid URIs that dont round-trip
func TestNotRoundTrip(t *testing.T) {
	for _, x := range []struct{ s, want string }{
		{"path/to/somewhere?", "path/to/somewhere"},
	} {
		t.Run(x.s, func(t *testing.T) {
			r, err := uri.Parse(x.s)
			if assert.NoError(t, err) {
				assert.Equal(t, x.want, r.String())
			}
		})
	}
}

func TestBad(t *testing.T) {
	for _, s := range []string{
		"scheme:",
		"scheme:x",
		"scheme://abspath",
		"//abspath",
		"path#fragment",
		"#fragment",
	} {
		t.Run(s, func(t *testing.T) {
			_, err := uri.Parse(s)
			if assert.Error(t, err) {
				assert.True(t, strings.HasSuffix(err.Error(), s))
			}
		})
	}
}
