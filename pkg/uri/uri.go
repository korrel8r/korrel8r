// package uri implements relative URI references.
//
// URI references can be represented by using only the relevant fields of the standrad url.URL type,
// but it can be more convenient, type safe, and memory efficient to have a comparable, value receiver type.
package uri

import (
	"fmt"
	"net/url"
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

// Parse a string URI reference.
func Parse(s string) (Reference, error) {
	u, err := url.Parse(s)
	if err == nil && !IsReference(u) {
		err = fmt.Errorf("URL is not a relative URI reference: %v", u)
	}
	if err != nil {
		return Reference{}, err
	}
	return Extract(u), nil
}

// String behaves like url.URL.String
func (r Reference) String() string {
	if r.RawQuery == "" {
		return r.Path
	}
	return fmt.Sprintf("%v?%v", r.Path, r.RawQuery)
}

// Extract extracts a Reference from a URL, ignoring other parts of the URL.
func Extract(u *url.URL) Reference { return Reference{Path: u.Path, RawQuery: u.RawQuery} }

// Values is an alias for url.Values
type Values = url.Values

// Query behaves like url.URL.Query
func (r Reference) Query() Values { v, _ := url.ParseQuery(r.RawQuery); return v }

// URL creates a url.URL containing the reference.
func (r Reference) URL() *url.URL { return &url.URL{Path: r.Path, RawQuery: r.RawQuery} }

// Resolve the Reference relative to a base URL.
func (r Reference) Resolve(base *url.URL) *url.URL { return base.ResolveReference(r.URL()) }

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
