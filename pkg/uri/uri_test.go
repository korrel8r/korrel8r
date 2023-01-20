package uri_test

import (
	"strings"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/uri"
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
		{"scheme:x", ""},
		{"scheme:/x", "/x"},
		{"scheme://host:3/abspath#fragment", "/abspath"},
		{"//host:2/abspath", "/abspath"},
	} {
		t.Run(x.s, func(t *testing.T) {
			r, err := uri.Parse(x.s)
			if assert.NoError(t, err) {
				assert.Equal(t, x.want, r.Path)
			}
		})
	}
}

func TestRelativeTo(t *testing.T) {
	for _, x := range [][3]string{
		{"/a/b/c", "/a/", "b/c"},
		{"/a/b/c/", "a/b", "c"},
		{"/a/b/c", "/a", "b/c"},
		{"/a/b/c/", "a/b/", "c"},
	} {
		t.Run(strings.Join(x[:], ":"), func(t *testing.T) {
			ref, err := uri.Reference{Path: x[0], RawQuery: "a=b"}.RelativeTo(x[1])
			if assert.NoError(t, err) {
				assert.Equal(t, x[2], ref.Path)
				assert.Equal(t, "a=b", ref.RawQuery)
			}
		})
	}
}
