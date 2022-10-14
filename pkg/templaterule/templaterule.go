// package templaterule implements korrel8.Rule as a Go template.
package templaterule

import (
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/korrel8/korrel8/internal/pkg/decoder"
	"github.com/korrel8/korrel8/pkg/korrel8"
)

// Rule implements korrel8.Rule as a Go template that generate a query string from the start object.
// The template should return the empty string if the rule does not apply to the start object.
type Rule struct {
	*template.Template
	start, goal korrel8.Class
}

// Error returned if a rule is applied to an object that does not have required fields or values.
var ErrRuleDoesNotApply = errors.New("rule does not apply")

// Funcs that are automatically added to templates created by New.
// Rule.Apply() also adds a "constraint" function.
var funcs = map[string]any{
	// doesnotapply fails template evaluation, call when a rule does not apply to its start object.
	"doesnotapply": func() (int, error) { return 0, ErrRuleDoesNotApply },
	// constraint is a placeholder ,in Rule.Apply it will return a *Constraint (possibly nil)
	"constraint": func() *korrel8.Constraint { panic("placeholder") },
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
func (r *Rule) Apply(start korrel8.Object, c *korrel8.Constraint) (result korrel8.Query, err error) {
	b := &strings.Builder{}
	err = r.Template.Funcs(map[string]any{"constraint": func() *korrel8.Constraint { return c }}).Execute(b, start)
	if err != nil {
		err = fmt.Errorf("apply %v to %T:  %w (start = %#v):", r, start, err, start)
	}
	return korrel8.Query(b.String()), err
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
