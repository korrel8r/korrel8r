// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package netflow is a domain for network observability flow events stored in Loki or LokiStack.
//
// # Class
//
// There is a single class `netflow:network`
//
// # Object
//
// A log object is a JSON `map[string]any` in [NetFlow] format.
//
// # Query
//
// A query is a [LogQL] query string, prefixed by `netflow:network:`, for example:
//
//	netflow:network:{SrcK8S_Type="Pod", SrcK8S_Namespace="myNamespace"}
//
// # Store
//
// To connect to a netflow lokiStack store use this configuration:
//
//	domain: netflow
//	lokistack: URL_OF_LOKISTACK_PROXY
//
// To connect to plain loki store use:
//
//	domain: netflow
//	loki: URL_OF_LOKI
//
// [LogQL]: https://grafana.com/docs/loki/latest/query/
// [NetFlow]: https://docs.openshift.com/container-platform/latest/observability/network_observability/json-flows-format-reference.html
package netflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/korrel8r/korrel8r/internal/pkg/loki"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"golang.org/x/exp/maps"
)

var (
	// Verify implementing interfaces.
	_ korrel8r.Domain    = Domain
	_ korrel8r.Store     = &store{}
	_ korrel8r.Store     = &stackStore{}
	_ korrel8r.Query     = Query("")
	_ korrel8r.Class     = Class{}
	_ korrel8r.Previewer = Class{}
)

// Domain for log records produced by openshift-logging.
//
// There are several possible log store configurations:
// - Default LokiStack store on current Openshift cluster: `{}`
// - Remote LokiStack: `{ "lokiStack": "https://url-of-lokistack"}`
// - Plain Loki store: `{ "loki": "https://url-of-loki"}`
var Domain = domain{}

type domain struct{}

func (domain) Name() string                     { return "netflow" }
func (d domain) String() string                 { return d.Name() }
func (domain) Description() string              { return "Network flows from source nodes to destination nodes." }
func (domain) Class(name string) korrel8r.Class { return Class{} }
func (domain) Classes() []korrel8r.Class        { return []korrel8r.Class{Class{}} }
func (d domain) Query(s string) (korrel8r.Query, error) {
	_, s, err := impl.ParseQuery(d, s)
	if err != nil {
		return nil, err
	}
	return Query(s), nil
}

const (
	StoreKeyLoki      = "loki"
	StoreKeyLokiStack = "lokiStack"
)

func (domain) Store(s any) (korrel8r.Store, error) {
	cs, err := impl.TypeAssert[config.Store](s)
	if err != nil {
		return nil, err
	}
	hc, err := k8s.NewHTTPClient(cs)
	if err != nil {
		return nil, err
	}

	loki, lokiStack := cs[StoreKeyLoki], cs[StoreKeyLokiStack]
	switch {

	case loki != "" && lokiStack != "":
		return nil, fmt.Errorf("can't set both loki and lokiStack URLs")

	case loki != "":
		u, err := url.Parse(loki)
		if err != nil {
			return nil, err
		}
		return NewPlainLokiStore(u, hc)

	case lokiStack != "":
		u, err := url.Parse(lokiStack)
		if err != nil {
			return nil, err
		}
		return NewLokiStackStore(u, hc)

	default:
		return nil, fmt.Errorf("must set one of loki or lokiStack URLs")
	}
}

// There is only a single class, named "netflow	".
type Class struct{}

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) Name() string            { return "network" }
func (c Class) String() string          { return impl.ClassString(c) }

func (c Class) Unmarshal(data []byte) (korrel8r.Object, error) { return impl.UnmarshalAs[Object](data) }

// Preview extracts the message from a Viaq log record.
func (c Class) Preview(o korrel8r.Object) (line string) { return Preview(o) }

// Preview extracts the message from a Viaq log record.
func Preview(x korrel8r.Object) (line string) {
	if m := x.(Object)["SrcK8S_Namespace"]; m != nil {
		s, _ := m.(string)
		message := "Network Flows from :" + s
		if m = x.(Object)["DstK8S_Namespace"]; m != nil {
			d, _ := m.(string)
			message = message + " to : " + d
		}
		return message
	}
	return ""
}

// Object is a map holding netflow entries
type Object map[string]any

func NewObject(entry *loki.Entry) Object {
	var label_object, o Object
	o = make(map[string]any)
	_ = json.Unmarshal([]byte(entry.Line), &o)
	if entry.Labels != nil {
		label_object = make(map[string]any)
		for k, v := range entry.Labels {
			label_object[k] = v
		}
		maps.Copy(o, label_object)
	}
	return o
}

// Query is a LogQL query string
type Query string

func NewQuery(logQL string) korrel8r.Query { return Query(strings.TrimSpace(logQL)) }

func (q Query) Class() korrel8r.Class { return Class{} }
func (q Query) Data() string          { return string(q) }
func (q Query) String() string        { return impl.QueryString(q) }

// NewLokiStackStore returns a store that uses a LokiStack observatorium-style URLs.
func NewLokiStackStore(base *url.URL, h *http.Client) (korrel8r.Store, error) {
	return &stackStore{store: store{loki.New(h, base)}}, nil
}

// NewPlainLokiStore returns a store that uses plain Loki URLs.
func NewPlainLokiStore(base *url.URL, h *http.Client) (korrel8r.Store, error) {
	return &store{loki.New(h, base)}, nil
}

type store struct{ *loki.Client }

func (store) Domain() korrel8r.Domain { return Domain }
func (s *store) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}
	return s.Client.Get(ctx, q.Data(), c, func(e *loki.Entry) { result.Append(NewObject(e)) })
}

type stackStore struct{ store }

func (stackStore) Domain() korrel8r.Domain { return Domain }
func (s *stackStore) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}

	return s.Client.GetStack(ctx, q.Data(), "network", c, func(e *loki.Entry) { result.Append(NewObject(e)) })
}
