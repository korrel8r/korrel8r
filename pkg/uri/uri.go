// package uri implements relative URI references.
//
// URI references can be represented by using only the relevant fields of the standrad url.URL type,
// but it can be more convenient, type safe, and memory efficient to have a comparable, value receiver type.
package uri

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// Reference implements the "path?query" part of a URL.
// Methods with the same name as url.URL methods behave the same way.
//
// This is a partial implementation of https://datatracker.ietf.org/doc/html/rfc3986#section-4.2
// but does not support references with authority or fragment.
// Reference has all value-receiver methods, and is small enough to pass by value efficiently.
type Reference struct {
	Path     string
	RawQuery string
}

// Parse a URI string as a relative URI reference.
// Only the path and query parts are used in the Reference, other parts are ignored.
func Parse(s string) (Reference, error) {
	u, err := url.Parse(s)
	if err != nil {
		return Reference{}, err
	}
	return Reference{Path: u.Path, RawQuery: u.RawQuery}, nil
}

// String behaves like url.URL.String
func (r Reference) String() string {
	if r.RawQuery == "" {
		return r.Path
	}
	return fmt.Sprintf("%v?%v", r.Path, r.RawQuery)
}

// Values is an alias for url.Values
type Values = url.Values

// Query behaves like url.URL.Query
func (r Reference) Query() Values { v, _ := url.ParseQuery(r.RawQuery); return v }

// URL creates a url.URL equivalent to the reference.
func (r Reference) URL() *url.URL { return &url.URL{Path: r.Path, RawQuery: r.RawQuery} }

// Resolve the Reference relative to a base URL, see url.URL.ResolveReference.
func (r Reference) Resolve(base *url.URL) *url.URL { return base.ResolveReference(r.URL()) }

// TrimPrefix returns a relative URI with the removed from the front of the path.
// Returns an error if r.Path does not start with basePath
func (r Reference) RelativeTo(basePath string) (Reference, error) {
	p, base := path.Join("/", r.Path), path.Join("/", basePath)
	if base != "/" {
		base = base + "/"
	}
	if !strings.HasPrefix(p, base) {
		return Reference{}, fmt.Errorf("path %q is not reltive to %q", r.Path, basePath)
	}
	p = strings.TrimPrefix(p, base)
	return Reference{Path: p, RawQuery: r.RawQuery}, nil
}

// IsReference is true if a URL contains only path?query parts.
func IsReference(u *url.URL) bool { return u.Scheme == "" && u.Host == "" && u.Fragment == "" }

// Make a URI from a path string and un-encoded key, value string pairs for the query.
// Panics if len(keyValuePairs) is odd
func Make(path string, keyValuePairs ...string) Reference {
	if len(keyValuePairs)%2 != 0 {
		panic("uri.Make called with odd numbered list of key, value pairs")
	}
	v := url.Values{}
	for i := 0; i < len(keyValuePairs); i += 2 {
		v.Set(keyValuePairs[i], keyValuePairs[i+1])
	}
	return Reference{Path: path, RawQuery: v.Encode()}
}
