// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// Store is a minimal implementation of korrel8r.Store.
// New domains can embed it and redefine methods as needed.
// [korrel8r.Store.Get] is not provided
type Store struct {
	domain korrel8r.Domain
}

func NewStore(d korrel8r.Domain) *Store { return &Store{d} }

func (s *Store) Domain() korrel8r.Domain                 { return s.domain }
func (s *Store) StoreClasses() ([]korrel8r.Class, error) { return s.domain.Classes(), nil }

// Get and decode a REST response, for stores that use raw HTTP clients.
// body is decoded from the response, it must point to a JSON decodable type.
func Get[T any](ctx context.Context, u *url.URL, hc *http.Client, timeout time.Duration, body T) (err error) {
	if timeout > 0 {
		hc2 := *hc // Make a copy
		hc2.Timeout = timeout
		hc = &hc2
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := hc.Do(req)
	if err != nil {
		if inner, ok := err.(*url.Error); ok {
			if u, err := url.Parse(inner.URL); err == nil {
				// Simplify the URL to just the host+path
				inner.URL = (&url.URL{Scheme: u.Scheme, Host: u.Host, Path: u.Path}).String()
			}
		}
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		if b, err := io.ReadAll(resp.Body); err == nil && len(b) > 0 {
			return fmt.Errorf("%v: %v", resp.Status, string(b))
		}
		return fmt.Errorf("%v", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(body)
}
