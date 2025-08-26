// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package log is a korrel8r domain for logs.
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
