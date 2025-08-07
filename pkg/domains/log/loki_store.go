// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

import (
	"context"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/loki"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
)

type lokiStore struct {
	*impl.Store
	*loki.Client
}

// NewLokiStore returns a store that uses plain Loki URLs.
func NewLokiStore(base *url.URL, h *http.Client) korrel8r.Store {
	return &lokiStore{Client: loki.New(h, base), Store: impl.NewStore(Domain)}
}

func (s *lokiStore) Get(ctx context.Context, query korrel8r.Query, constraint *korrel8r.Constraint, r korrel8r.Appender) error {
	q, err := impl.TypeAssert[*Query](query)
	if err != nil {
		return err
	}
	return s.Client.Get(ctx, parseJSON(q.logQL), constraint, func(l *loki.Log) { r.Append(newObject(l)) })
}

type lokiStackStore struct{ *lokiStore }

// NewLokiStackStore returns a store that uses a LokiStack observatorium-style URLs.
func NewLokiStackStore(base *url.URL, h *http.Client) korrel8r.Store {
	return &lokiStackStore{&lokiStore{Client: loki.New(h, base), Store: impl.NewStore(Domain)}}
}

func (s *lokiStackStore) Get(ctx context.Context, query korrel8r.Query, constraint *korrel8r.Constraint, r korrel8r.Appender) error {
	q, err := impl.TypeAssert[*Query](query)
	if err != nil {
		return err
	}
	return s.GetStack(ctx, parseJSON(q.logQL), string(q.class), constraint, func(l *loki.Log) { r.Append(newObject(l)) })
}

var jsonRE = regexp.MustCompile(`\|\s*json\b`)

func parseJSON(logql string) string {
	if !jsonRE.MatchString(logql) {
		return logql + "|json"
	}
	return logql
}

func newObject(l *loki.Log) Object {
	o := Object{}
	maps.Copy(o, l.Metadata)
	maps.Copy(o, l.Labels)
	if o[AttrMessage] != "" && o[Attr_Timestamp] != "" { // This is a ViaQ log
		// For Viaq logs, use "message" field as body.
		o[AttrBody] = o[AttrMessage]
	} else {
		// For non-Viaq, use the log body as body.
		o[AttrBody] = l.Body
	}
	// Copy @timestamp or _timestamp fields to timestamp.
	if o[AttrTimestamp] == "" && o[Attr_Timestamp] != "" {
		o[AttrTimestamp] = o[Attr_Timestamp]
	}
	if o[AttrObservedTimestamp] == "" {
		o[AttrObservedTimestamp] = l.Time.Format(time.RFC3339Nano)
	}
	// Ignore JSON parsing errors, expected for non-JSON logs.
	if o["__error__"] == "JSONParserErr" {
		delete(o, "__error__")
		delete(o, "__error_details__")
	}
	return o
}
