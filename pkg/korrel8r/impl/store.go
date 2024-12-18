// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Get and decode a REST response, for stores that use raw HTTP clients.
// body is decoded from the response, it must point to a JSON decodable type.
func Get[T any](ctx context.Context, u *url.URL, hc *http.Client, body T) (err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		if b, err := io.ReadAll(resp.Body); err == nil && len(b) > 0 {
			return fmt.Errorf("%v: %v", resp.Status, string(b))
		}
		return fmt.Errorf("%v", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(body)
}
