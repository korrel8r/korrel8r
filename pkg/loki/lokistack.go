package loki

import (
	"context"
	"fmt"
	"net/url"
	"path"

	"regexp"

	"github.com/korrel8/korrel8/internal/pkg/openshift"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	matchLogType = regexp.MustCompile(`[{,]\s*log_type\s*(=~?)\s*"([^"]*)"\s*[,}]`)
	logTypes     = []string{"application", "infrastructure", "audit"}
)

// LokiStackStore is a store using the LokiStack observatorium API in openshift tenancy mode.
type LokiStackStore struct {
	store *LokiStore
}

// NewLokiStackStore creates a store using the LokiStack observatorium API in openshift tenancy mode.
func NewLokiStackStore(ctx context.Context, c client.Client, cfg *rest.Config) (*LokiStackStore, error) {
	host, err := openshift.RouteHost(ctx, c, openshift.LokiStackNSName)
	if err != nil {
		return nil, err
	}
	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}
	lokiStore, err := NewLokiStore(&url.URL{Scheme: "https", Host: host, Path: "/api/logs/v1"}, hc)
	return &LokiStackStore{store: lokiStore}, err
}

// query executes a LogQL query_range call.
// The query is split into up to 3 queries for each of the log_type tenants, based on the log_type expression.
func (s *LokiStackStore) Get(ctx context.Context, q korrel8.Query, result korrel8.Result) error {
	query, ok := q.(*Query)
	if !ok {
		return fmt.Errorf("%v store expects %T but got %T", Domain, query, q)
	}
	for _, tenant := range query.LogTypes() { // Each matched log type
		tenantURL := s.store.baseURL
		tenantURL.Path = path.Join(tenantURL.Path, tenant, "loki/api/v1")
		if err := s.store.get(ctx, q, result, &tenantURL); err != nil {
			return err
		}
	}
	return nil
}

// Parse the query to find the selected log types for openshift tenancy mode.
func (query *Query) LogTypes() []string {
	m := matchLogType.FindStringSubmatch(string(*query))
	var matching []string
	if m == nil { // No log_type matches all.
		return logTypes
	}
	operator, value := m[1], m[2]
	switch operator {
	case "=": // Equality match
		for _, lt := range logTypes {
			if value == lt {
				matching = append(matching, lt)
			}
		}
	case "=~": // Regexp match
		if regex, err := regexp.Compile(value); err == nil {
			for _, lt := range logTypes {
				if regex.MatchString(lt) {
					matching = append(matching, lt)
				}
			}
		}
	}
	return matching
}
