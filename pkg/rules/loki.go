package rules

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alanconway/korrel8/pkg/k8s"
	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/alanconway/korrel8/pkg/loki"
	v1 "k8s.io/api/core/v1"
)

// FIXME sort out rule construction
// FIXME generalize constraint propagation.

type rule struct{ korrel8.Rule }

func (r rule) Apply(start korrel8.Object, c *korrel8.Constraint) (result korrel8.Queries, err error) {
	result, err = r.Rule.Apply(start, c)
	for i, q := range result {
		result[i] = addConstraint(q, c)
	}
	return result, err
}

// FIXME
func (r rule) String() string { return fmt.Sprintf("%v", r.Rule) }

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

func addConstraint(q string, c *korrel8.Constraint) string {
	if c == nil {
		return q
	}

	return QueryObject{
		Query: q,
		Start: c.After,
		End:   c.Before,
	}.String()
}

// FIXME need test for constraints

func K8sToLoki() []korrel8.Rule {
	return []korrel8.Rule{
		rule{newTemplate("PodLogs", k8s.ClassOf(&v1.Pod{}), loki.Class{},
			`{kubernetes_namespace_name="{{.ObjectMeta.Namespace}}",kubernetes_pod_name="{{.ObjectMeta.Name}}"}`),
		},
	}
}
