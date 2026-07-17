// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/korrel8r/korrel8r/pkg/api/auth"
)

// Client is an HTTP client for the korrel8r REST API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a REST client that calls the korrel8r API at baseURL.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: baseURL, httpClient: httpClient}
}

// NewClientForHandler creates a REST client that calls an http.Handler in-process.
// Auth tokens from the request context are forwarded automatically.
func NewClientForHandler(handler http.Handler) *Client {
	return NewClient("http://korrel8r", &http.Client{
		Transport: auth.Wrap(&handlerTransport{handler: handler}),
	})
}

// handlerTransport is an http.RoundTripper that calls an http.Handler directly.
type handlerTransport struct {
	handler http.Handler
}

func (t *handlerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	t.handler.ServeHTTP(w, req)
	return w.Result(), nil
}

func (c *Client) ListDomains(ctx context.Context) ([]api.Domain, error) {
	var domains []api.Domain
	if err := c.get(ctx, "/domains", &domains); err != nil {
		return nil, err
	}
	return domains, nil
}

func (c *Client) ListDomainClasses(ctx context.Context, domain string) ([]string, error) {
	var classes []string
	if err := c.get(ctx, "/domain/"+url.PathEscape(domain)+"/classes", &classes); err != nil {
		return nil, err
	}
	return classes, nil
}

func (c *Client) Help(ctx context.Context, domain string) (string, error) {
	path := "/help"
	if domain != "" {
		path += "/" + url.PathEscape(domain)
	}
	var h api.Help
	if err := c.get(ctx, path, &h); err != nil {
		return "", err
	}
	return h.Documentation, nil
}

func (c *Client) GraphNeighbors(ctx context.Context, params api.Neighbors) (*api.Graph, error) {
	var g api.Graph
	if err := c.post(ctx, "/graphs/neighbors", params, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

func (c *Client) GraphGoals(ctx context.Context, params api.Goals) (*api.Graph, error) {
	var g api.Graph
	if err := c.post(ctx, "/graphs/goals", params, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

func (c *Client) GetObjects(ctx context.Context, query string, constraint *api.Constraint) ([]json.RawMessage, error) {
	u := "/objects?query=" + url.QueryEscape(query)
	if constraint != nil {
		if constraint.Limit != nil {
			u += fmt.Sprintf("&limit=%d", *constraint.Limit)
		}
		if constraint.QueryLimit != nil {
			u += fmt.Sprintf("&queryLimit=%d", *constraint.QueryLimit)
		}
		if constraint.Start != nil {
			u += "&start=" + url.QueryEscape(constraint.Start.Format("2006-01-02T15:04:05Z07:00"))
		}
		if constraint.End != nil {
			u += "&end=" + url.QueryEscape(constraint.End.Format("2006-01-02T15:04:05Z07:00"))
		}
	}
	var objects []json.RawMessage
	if err := c.get(ctx, u, &objects); err != nil {
		return nil, err
	}
	return objects, nil
}

func (c *Client) GetConsole(ctx context.Context) (*api.Console, error) {
	var console api.Console
	if err := c.get(ctx, "/console", &console); err != nil {
		return nil, err
	}
	return &console, nil
}

func (c *Client) ShowInConsole(ctx context.Context, update *api.Console) error {
	return c.put(ctx, "/console/events", update, nil)
}

func (c *Client) get(ctx context.Context, path string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+api.BasePath+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

func (c *Client) post(ctx context.Context, path string, body, result any) error {
	return c.send(ctx, http.MethodPost, path, body, result)
}

func (c *Client) put(ctx context.Context, path string, body, result any) error {
	return c.send(ctx, http.MethodPut, path, body, result)
}

func (c *Client) send(ctx context.Context, method, path string, body, result any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+api.BasePath+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, result)
}

func (c *Client) do(req *http.Request, result any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("korrel8r: %s %s: %s: %s", req.Method, req.URL.Path, resp.Status, body)
	}
	if result != nil && len(body) > 0 {
		return json.Unmarshal(body, result)
	}
	return nil
}
