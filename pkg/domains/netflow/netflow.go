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
	"hash/fnv"
	"maps"
	"net/http"
	"net/url"
	"strings"

	"github.com/korrel8r/korrel8r/internal/pkg/loki"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
)

var (
	// Verify implementing interfaces.
	_ korrel8r.Domain = Domain
	_ korrel8r.Store  = &store{}
	_ korrel8r.Store  = &stackStore{}
	_ korrel8r.Query  = Query("")
	_ korrel8r.Class  = Class{}
)

// Domain for log records produced by openshift-logging.
//
// There are several possible log store configurations:
// - Default LokiStack store on current Openshift cluster: `{}`
// - Remote LokiStack: `{ "lokiStack": "https://url-of-lokistack"}`
// - Plain Loki store: `{ "loki": "https://url-of-loki"}`
var Domain = domain{Domain: impl.NewDomain("netflow", "Network flows from source nodes to destination nodes.", Class{})}

type domain struct{ *impl.Domain }

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

func (domain) TemplateFuncs() map[string]any {
	return map[string]any{
		// Convert a netflow type field to a k8s class.
		// Need to add the "apps" group to the Deployment type.
		"netflowTypeToK8s": func(t string) (string, error) {
			if t == "Deployment" {
				return "k8s:Deployment.v1.apps", nil
			} else {
				return fmt.Sprintf("k8s:%v.v1", t), nil
			}
		},
	}
}

// There is only a single class, named "netflow	".
type Class struct{}

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) Name() string            { return "network" }
func (c Class) String() string          { return korrel8r.ClassString(c) }

func (c Class) Unmarshal(data []byte) (korrel8r.Object, error) {
	return impl.UnmarshalAs[Object](data)
}

// ID is a hash constructed from "interesting" attributes defined in idKeys
func (c Class) ID(ko korrel8r.Object) any {
	o, _ := ko.(Object)
	hash := fnv.New64()
	for _, k := range idKeys {
		fmt.Fprintf(hash, "%v", o[k])
	}
	return hash.Sum64()
}

// Object is a map holding netflow entries
type Object map[string]any

func NewObject(entry *loki.Log) Object {
	var label_object, o Object
	o = make(map[string]any)
	_ = json.Unmarshal([]byte(entry.Body), &o)
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
func (q Query) String() string        { return korrel8r.QueryString(q) }

// NewLokiStackStore returns a store that uses a LokiStack observatorium-style URLs.
func NewLokiStackStore(base *url.URL, h *http.Client) (korrel8r.Store, error) {
	return &stackStore{store: store{Client: loki.New(h, base), Store: impl.NewStore(Domain)}}, nil
}

// NewPlainLokiStore returns a store that uses plain Loki URLs.
func NewPlainLokiStore(base *url.URL, h *http.Client) (korrel8r.Store, error) {
	return &store{Client: loki.New(h, base), Store: impl.NewStore(Domain)}, nil
}

type store struct {
	*loki.Client
	*impl.Store
}

func (store) Domain() korrel8r.Domain { return Domain }
func (s *store) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}
	return s.Client.Get(ctx, q.Data(), c, func(e *loki.Log) { result.Append(NewObject(e)) })
}

type stackStore struct{ store }

func (stackStore) Domain() korrel8r.Domain { return Domain }
func (s *stackStore) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) error {
	q, err := impl.TypeAssert[Query](query)
	if err != nil {
		return err
	}

	return s.GetStack(ctx, q.Data(), "network", c, func(e *loki.Log) { result.Append(NewObject(e)) })
}

// Attributes to use when constructing an ID for de-duplication.
// Choose attributes that help identify endpoints and don't vary during a single conversation.
var idKeys = []string{
	"SrcK8S_Name",
	"SrcSubnetLabel",
	"AgentIP",
	"DstK8S_OwnerType",
	"Proto",
	"SrcK8S_OwnerType",
	"DnsErrno",
	"DstPort",
	"DstSubnetLabel",
	"SrcK8S_Namespace",
	"SrcK8S_Type",
	"DstK8S_Name",
	"DstMac",
	"K8S_FlowLayer",
	"SrcAddr",
	"SrcPort",
	"service_name",
	"DstK8S_HostName",
	"DstK8S_Type",
	"SrcMac",
	"DstAddr",
	"DstK8S_HostIP",
	"DstK8S_Namespace",
	"DstK8S_OwnerName",
	"IfDirections",
	"SrcK8S_HostIP",
	"SrcK8S_HostName",
	"SrcK8S_OwnerName",
	"Dscp",
}
