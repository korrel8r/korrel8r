// package loki generates qgueries for logs stored in Loki or LokiStack
package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/korrel8/korrel8/internal/pkg/openshift"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/uri"
	"golang.org/x/exp/slices"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	_ korrel8.Domain       = Domain
	_ korrel8.Store        = &Store{}
	_ korrel8.RefClasser   = Domain
	_ korrel8.RefConverter = &Store{}
)

var Domain = domain{}

type domain struct{}

func (domain) String() string                  { return "loki" }
func (domain) Class(name string) korrel8.Class { return classMap[name] }
func (domain) Classes() []korrel8.Class        { return classes }

func (domain) RefClass(ref uri.Reference) (korrel8.Class, error) {
	m := lokiStackPath.FindStringSubmatch(ref.Path)
	if len(m) != 2 {
		return nil, fmt.Errorf("not a valid %v reference: %v", Domain, ref)
	}
	return Class(m[1]), nil
}

// Plain converts a LokiStack reference to a plain loki reference.
func Plain(ref uri.Reference) uri.Reference {
	// TODO add a log_type test to the plain query.
	return uri.Reference{Path: lokiStackPath.ReplaceAllString(ref.Path, ""), RawQuery: ref.RawQuery}
}

type Class string

func (c Class) Domain() korrel8.Domain { return Domain }
func (c Class) String() string         { return string(c) }
func (c Class) New() korrel8.Object    { return Object("") }

var _ korrel8.Class = Class("") // Implements interface.

type Object string // Log record

const (
	Application    = "application"
	Infrastructure = "infrastructure"
	Audit          = "audit"
)

var (
	classNames = []string{Application, Infrastructure, Audit}
	classes    = []korrel8.Class{Class(Application), Class(Infrastructure), Class(Audit)}
	classMap   = map[string]korrel8.Class{Application: Class(Application), Infrastructure: Class(Infrastructure), Audit: Class(Audit)}
)

type Store struct {
	c    *http.Client
	base url.URL
}

func NewStore(baseURL *url.URL, c *http.Client) (*Store, error) {
	return &Store{c: c, base: *baseURL}, nil
}

func (s *Store) Resolve(ref uri.Reference) *url.URL { return ref.Resolve(&s.base) }

func (s *Store) String() string { return s.base.String() }

func (s *Store) Get(ctx context.Context, ref uri.Reference, result korrel8.Appender) error {
	u := s.base.ResolveReference(ref.URL())
	resp, err := s.c.Get(u.String())
	if err != nil {
		return fmt.Errorf("%w: %v", err, u)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%v: %v", resp.Status, u)
	}
	qr := queryResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&qr); err != nil {
		return err
	}
	if qr.Status != "success" {
		return fmt.Errorf("expected 'status: success' in %v", qr)
	}
	if qr.Data.ResultType != "streams" {
		return fmt.Errorf("expected 'resultType: streams' in %v", qr)
	}
	// Interleave and sort the stream results.
	var logs [][]string // Each log is [timestamp,logline]
	for _, sv := range qr.Data.Result {
		logs = append(logs, sv.Values...)
	}
	slices.SortStableFunc(logs, func(a, b []string) bool { return a[0] < b[0] })
	for _, tl := range logs { // tl is [time, line]
		result.Append(Object(tl[1]))
	}
	return nil
}

// RefStoreToConsole converts a LokiStak ref to a console URL
func (s *Store) RefStoreToConsole(c korrel8.Class, ref uri.Reference) (uri.Reference, error) {
	rc, err := Domain.RefClass(ref)
	if err != nil {
		return uri.Empty, err
	}
	if rc != c {
		return uri.Empty, fmt.Errorf("class missmatch: expected %v in %v", korrel8.FullName(c), ref)
	}
	v := url.Values{}
	v.Add("q", ref.Query().Get("query"))
	v.Add("tenant", c.String())
	return uri.Reference{Path: "/monitoring/logs", RawQuery: v.Encode()}, nil
}

func (*Store) RefConsoleToStore(ref uri.Reference) (korrel8.Class, uri.Reference, error) {
	c := Class(ref.Query().Get("tenant"))
	return c, NewLokiStackRef(c, ref.Query().Get("q"), nil), nil
}

// queryResponse is the response to a loki query.
type queryResponse struct {
	Status string    `json:"status"`
	Data   queryData `json:"data"`
}

// queryData holds the data for a query
type queryData struct {
	ResultType string         `json:"resultType"`
	Result     []streamValues `json:"result"`
}

// streamValues is a set of log values ["time", "line"] for a log stream.
type streamValues struct {
	Stream map[string]string `json:"stream"` // Labels for the stream
	Values [][]string        `json:"values"`
}

var lokiStackPath = regexp.MustCompile(fmt.Sprintf("/api/logs/v1/(%v)", strings.Join(classNames, "|")))

// PlainStore re-writes observatorium-style URIs as plain Loki URIs.
type PlainStore struct{ *Store }

func NewPlainStore(baseURL *url.URL, c *http.Client) (PlainStore, error) {
	s, err := NewStore(baseURL, c)
	return PlainStore{s}, err
}

func (s *PlainStore) Get(ctx context.Context, ref uri.Reference, result korrel8.Appender) error {
	return s.Store.Get(ctx, Plain(ref), result)
}

func NewOpenshiftLokiStackStore(ctx context.Context, c client.Client, cfg *rest.Config) (*Store, error) {
	host, err := openshift.RouteHost(ctx, c, openshift.LokiStackNSName)
	if err != nil {
		return nil, err
	}
	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}
	return NewStore(&url.URL{Scheme: "https", Host: host}, hc)
}

func NewPlainRef(logQL string, constraint *korrel8.Constraint) uri.Reference {
	v := url.Values{}
	v.Add("query", logQL)
	v.Add("direction", "forward")
	if constraint != nil {
		if constraint.Limit != nil {
			v.Add("limit", fmt.Sprintf("%v", *constraint.Limit))
		}
		if constraint.Start != nil {
			v.Add("start", fmt.Sprintf("%v", constraint.Start.UnixNano()))
		}
		if constraint.End != nil {
			v.Add("end", fmt.Sprintf("%v", constraint.End.UnixNano()))
		}
	}
	return uri.Reference{Path: "/loki/api/v1/query_range", RawQuery: v.Encode()}
}

func NewLokiStackRef(class Class, logQL string, constraint *korrel8.Constraint) uri.Reference {
	u := NewPlainRef(logQL, constraint)
	u.Path = path.Join("/api/logs/v1/", class.String(), u.Path)
	return u
}
