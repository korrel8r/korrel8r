// package alert implements korrel8 interfaces on prometheus alerts.
package metric

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const Domain korrel8.Domain = "metric"

type Store struct {
	c   api.Client
	api promv1.API
}

func NewStore(host string) (*Store, error) {
	var err error
	s := &Store{}
	s.c, err = api.NewClient(api.Config{Address: "https://" + host})
	if err != nil {
		return nil, err
	}
	s.api = promv1.NewAPI(s.c)
	return s, nil
}

type Class struct{} // Only one class

func (c Class) Domain() korrel8.Domain { return Domain }
func (c Class) String() string         { return string(Domain) }

// FIXME use streams not samples? Only keep metadata?
// Switch to metadata queries? But do we need promQL flexibility (time intervals etc)
type Object struct{ *model.Sample }

func (o Object) Identifier() korrel8.Identifier { return o.Metric }
func (o Object) Class() korrel8.Class           { return Class{} }
func (o Object) Native() any                    { return o.Value }

// QueryObject a JSON object representing a Loki query.
// Time values are RFC3339 format: "2006-01-02T15:04:05.999999999Z07:00"
type QueryObject struct {
	Query string     `json:"query,omitempty"` // LogQL log query
	Start *time.Time `json:"start,omitempty"` // Start of time interval, RFC3339 format
	End   *time.Time `json:"end,omitempty"`   // End of time interval, RFC3339 format
}

func (qo QueryObject) String() string {
	b, err := json.Marshal(qo)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// Query is a JSON-marshalled QueryObject.
func (s Store) Query(ctx context.Context, query string) (result []korrel8.Object, err error) {
	if query == "" {
		query = "{}" // Allow empty string as empty object
	}
	qo := QueryObject{}
	if err := json.Unmarshal([]byte(query), &qo); err != nil {
		return nil, err
	}
	if qo.End == nil {
		now := time.Now()
		qo.End = &now
	}
	if qo.Start == nil {
		qo.Start = qo.End
	}
	var v model.Value
	if qo.Start == qo.End {
		v, _, err = s.api.Query(ctx, qo.Query, *qo.Start)
	} else {
		v, _, err = s.api.QueryRange(ctx, qo.Query, promv1.Range{Start: *qo.Start, End: *qo.End})
	}
	if err != nil {
		return nil, err
	}
	switch v := v.(type) {
	case model.Matrix:
		for _, ss := range v {
			// FIXME Inefficient, should we keep as sample stream? What about logs also?
			for _, v := range ss.Values {
				result = append(result, Object{&model.Sample{Metric: ss.Metric, Value: v.Value, Timestamp: v.Timestamp}})
			}
		}
	case model.Vector:
		result = make([]korrel8.Object, len(v))
		for i := 0; i < len(v); i++ {
			result[i] = Object{v[i]}
		}
	default:
		return nil, fmt.Errorf("unexpected query result: %v", v)
	}
	return result, nil
}
