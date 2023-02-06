package graph

import (
	"fmt"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

// Result accumulates the result of applying a rule or rules.
type Result struct {
	Queries unique.JSONList[korrel8r.Query]
	Objects int // Count objects retrieved by all queries
	Class   korrel8r.Class
}

func NewResult(class korrel8r.Class) *Result {
	return &Result{
		Queries: unique.NewJSONList[korrel8r.Query](),
		Class:   class,
	}
}

func (r *Result) String() string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "Queries:\n")
	for _, q := range r.Queries.List {
		fmt.Fprintf(b, "- %v\n", unique.JSONString(q))
	}
	fmt.Fprintf(b, "Objects: %v\n", r.Objects)
	return b.String()
}
