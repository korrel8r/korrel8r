// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package log is a korrel8r domain for log records.
//
// Logs can be stored in Loki, LokiStack, or they can be retrieved directly from the Kubernetes API server.
//
// # Class
//
// There are 3 classes corresponding to the 3 openshift logging log types:
//
//   - log:application
//   - log:infrastructure
//   - log:audit
//
// # Object
//
// A log object is a map of string attributes.
// The set of attribute names and values may vary depending on how the logs were collected.
// Attribute names contain only ASCII letters, digits, underscores, and colons, and cannot start with a digit.
//
// For Loki logs, all Loki stream and structured metadata labels are included as attributes.
// If the log body is a JSON object, all nested fields paths are flattened into attribute names.
//
// Special attributes:
//
//   - "body" contains the original log message.
//   - "timestamp" is the time the log was produced (if known) in RFC3999 format.
//   - "observed_timestamp" is the time the log was stored in RFC3999 format.
//
// Viaq logs have attributes like: "kubernetes_namespace_name", "kubernetes_pod_name"
//
// OTEL logs have attributes like: "k8s_namespace_name", "k8s_pod_name"
//
// # Query
//
// A query starts with the log class, followed by one of the following:
//   - A [LogQL] expression: LogQL queries can only be used to retrieve stored logs
//   - A container selector: This can be used for stored logs and/or direct API log access.
//
// This is a literal [LogQL] expression that will be passed to the Loki store.
//
//	log:infrastructure:{ kubernetes_namespace_name="openshift-cluster-version", kubernetes_pod_name=~".*-operator-.*" }
//
// A container selector is a JSON map of the form:
//
//	{
//	  "namespace": "pod_namespace",
//	  "name": "pod_name",
//	  "labels": { "label_name": "label_value", ... },
//	  "fields": { "field_name": "field_value", ... },
//	  "containers": ["container_name", "another_container_name", ...],
//	}
//
// For example: to get all logs from pods in namespace "app" that have containers named "foo" or "bar"
//
//	log:infrastructure:{ "namespace": "app", "labels":["foo", "bar"]}
//
// # Store Configuration
//
//	domain: log
//	lokiStack: https://url_of_lokistack
//	direct: true
//
// This will connect to a Lokistack instance and try to get logs from there. If it fails
// it will fall back to using the API server directly.
// You can configure a store with just `direct: true` to use the API server only,
// or with just `lokiStack` to use the Loki store only.
//
// # Template functions
//
// The following functions can be used in rule templates when the log domain is available:
//
//	logTypeForNamespace
//	  Takes a namespace string argument.
//	  Returns the log type: "application" or "infrastructure"
//
//	logSafeLabel
//	  Replace all characters other than alphanumerics, '_' and ':' with '_'.
//
//	logSafeLabels
//	  Takes a map[string]string argument.
//	  Returns a map where each key is replaced by calling logSafeLabel()
//
// [LogQL]: https://grafana.com/docs/loki/latest/query/
package log

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
)

var (
	// Verify implementing interfaces.
	_ korrel8r.Domain    = Domain
	_ korrel8r.Store     = &Store{}
	_ korrel8r.Query     = &Query{}
	_ korrel8r.Class     = Class("")
	_ korrel8r.Previewer = Class("")
)

var Domain = &domain{
	impl.NewDomain("log", "Records from container and node logs.",
		Class(Application), Class(Infrastructure), Class(Audit)),
}

type domain struct{ *impl.Domain }

func (d *domain) Query(query string) (korrel8r.Query, error) { return NewQuery(query) }

const (
	StoreKeyLoki      = "loki"
	StoreKeyLokiStack = "lokiStack"
	StoreKeyDirect    = "direct"
)

func (*domain) Store(s any) (korrel8r.Store, error) {
	cs, err := impl.TypeAssert[config.Store](s)
	if err != nil {
		return nil, err
	}
	ks, err := k8s.NewStore(nil, nil)
	if err != nil {
		return nil, err
	}
	return NewStore(cs, ks)
}

type Store = impl.TryStores

func NewStore(cs config.Store, k8sStore *k8s.Store) (*Store, error) {
	var stores impl.TryStores // Collect loki and pod store
	loki, lokiStack, direct := cs[StoreKeyLoki], cs[StoreKeyLokiStack], cs[StoreKeyDirect]

	if loki != "" && lokiStack != "" {
		return nil, fmt.Errorf("can't set both loki and lokiStack URLs")
	}
	hc, err := k8s.NewHTTPClient(cs)
	if err != nil {
		return nil, err
	}
	if loki != "" {
		u, err := url.Parse(loki)
		if err != nil {
			return nil, err
		}
		stores = append(stores, NewLokiStore(u, hc))
	}
	if lokiStack != "" {
		u, err := url.Parse(lokiStack)
		if err != nil {
			return nil, err
		}
		stores = append(stores, NewLokiStackStore(u, hc))
	}

	if ok, err := strconv.ParseBool(direct); direct != "" && err != nil {
		return nil, err
	} else if ok {
		direct, err := newDirectStore(k8sStore)
		if err != nil {
			return nil, err
		}
		stores = append(stores, direct)
	}

	if len(stores) == 0 {
		return nil, errors.New("must set at least one of loki, lokiStack or direct")
	}
	return &stores, nil
}

type Class string

func (c Class) Domain() korrel8r.Domain                     { return Domain }
func (c Class) Name() string                                { return string(c) }
func (c Class) String() string                              { return impl.ClassString(c) }
func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) { return impl.UnmarshalAs[Object](b) }
func (c Class) Preview(o korrel8r.Object) (line string)     { return Preview(o) }

func Preview(x korrel8r.Object) string {
	if o, _ := x.(Object); o != nil {
		return o[AttrBody]
	}
	return ""
}

type Object map[string]string

func (o Object) Body() string                          { return o["body"] }
func (o Object) ObservedTimestamp() (time.Time, error) { return ParseTime(o["observed_timestamp"]) }
func (o Object) Timestamp() (time.Time, error)         { return ParseTime(o["timestamp"]) }
func (o Object) SortTime() (time.Time, error) {
	ts, err := ParseTime(o[AttrTimestamp])
	if err != nil {
		ts, err = ParseTime(o[AttrObservedTimestamp])
	}
	return ts, err
}

// ParseTime parses a timestamp in RFC3999 or Unix nanosecond format.
func ParseTime(ts string) (time.Time, error) {
	tt, err := time.Parse(time.RFC3339Nano, ts)
	if err == nil {
		return tt, nil
	}
	if n, err := strconv.ParseInt(ts, 10, 64); err == nil {
		return time.Unix(0, n), nil
	}
	return time.Time{}, err
}

type Query struct {
	logQL  string
	direct *ContainerSelector
	class  Class
}

func (q *Query) Class() korrel8r.Class { return q.class }
func (q *Query) String() string        { return impl.QueryString(q) }
func (q *Query) Data() string {
	if q.direct != nil {
		d, _ := json.Marshal(q.direct)
		return string(d)
	}
	return q.logQL
}

func NewQuery(query string) (*Query, error) {
	class, selector, err := impl.ParseQuery(Domain, query)
	if err != nil {
		return nil, err
	}
	q := &Query{class: class.(Class)}
	// Try to unmarshal selector to direct pod selector.
	var direct ContainerSelector
	if err := impl.Unmarshal([]byte(selector), &direct); err == nil {
		q.direct = &direct
		// FIXME defer LogQL conversion to store, when we know the label sets in use.
		q.logQL = q.direct.LogQL()
	} else { // Otherwise assume LogQL
		q.logQL = selector
	}
	return q, nil
}

const (
	Application    = "application"
	Infrastructure = "infrastructure"
	Audit          = "audit"
)

// Separate attributes from timestamp/body intrinsics?
// Document attributes

const (
	AttrObservedTimestamp = "observed_timestamp"
	AttrTimestamp         = "timestamp"
	Attr_Timestamp        = "_timestamp"
	AttrBody              = "body"
	AttrMessage           = "message"

	AttrK8sPodName       = "k8s_pod_name"
	AttrK8sNamespaceName = "k8s_namespace_name"
	AttrK8sContainerName = "k8s_container_name"

	AttrKubernetesPodName       = "kubernetes_pod_name"
	AttrKubernetesNamespaceName = "kubernetes_namespace_name"
	AttrKubernetesContainerName = "kubernetes_container_name"
)
