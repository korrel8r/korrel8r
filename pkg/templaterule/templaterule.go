// package templaterule implements korrel8.Rule as a Go template.
package templaterule

import (
	"fmt"
	"net/url"
	"text/template"

	"bytes"

	"github.com/korrel8/korrel8/internal/pkg/decoder"
	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/korrel8/korrel8/pkg/korrel8"
)

var log = logging.Log

// Rule implements korrel8.Rule as a Go template that generate a query string from the start object.
// The template should return the empty string if the rule does not apply to the start object.
type Rule struct {
	*template.Template
	start, goal korrel8.Class
}

// New rule using a template to convert the start object to a goal query.
func New(name string, start, goal korrel8.Class, tmpl string) (*Rule, error) {
	t, err := template.New(name).Funcs(funcs).Option("missingkey=error").Parse(tmpl)
	return &Rule{Template: t, start: start, goal: goal}, err
}

func (r Rule) String() string       { return r.Template.Name() }
func (r Rule) Start() korrel8.Class { return r.start }
func (r Rule) Goal() korrel8.Class  { return r.goal }

// Apply the rule by applying the template.
// The template will be executed with start as the "." context object.
// A function "constraint" returns the constraint.
func (r *Rule) Apply(start korrel8.Object, c *korrel8.Constraint) (*korrel8.Query, error) {
	b := &bytes.Buffer{}
	err := r.Template.Funcs(map[string]any{"constraint": func() *korrel8.Constraint { return c }}).Execute(b, start)
	if err != nil {
		return nil, err
	}
	return url.Parse(b.String())
}

var _ korrel8.Rule = &Rule{}

// Decode a template rule from JSON or YAML.
func Decode(decoder *decoder.Decoder, parseClass func(string) (korrel8.Class, error)) (*Rule, error) {
	sr := struct { // Serialized rule
		Name     string `json:"name"`
		Start    string `json:"start"`
		Goal     string `json:"goal"`
		Template string `json:"template"`
	}{}
	if err := decoder.Decode(&sr); err != nil {
		return nil, err
	}
	if sr.Name == "" || sr.Template == "" || sr.Goal == "" || sr.Start == "" {
		return nil, fmt.Errorf("invalid rule: %+v", sr)
	}
	start, err := parseClass(sr.Start)
	if err != nil {
		return nil, err
	}
	goal, err := parseClass(sr.Goal)
	if err != nil {
		return nil, err
	}
	return New(sr.Name, start, goal, sr.Template)
}
