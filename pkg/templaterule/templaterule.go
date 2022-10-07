// package templaterule implements korrel8.Rule as a Go template.
package templaterule

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/alanconway/korrel8/pkg/engine"
	"github.com/alanconway/korrel8/pkg/korrel8"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// Rule implements korrel8.Rule as a Go template that generate a query string from the start object.
// The template should return the empty string if the rule does not apply to the start object.
type Rule struct {
	*template.Template
	start, goal korrel8.Class
}

// Funcs that are automatically added to templates created by New.
// Rule.Apply() also adds a "constraint" function.
var funcs = map[string]any{
	// doesnotapply fails template evaluation, call when a rule does not apply to its start object.
	"doesnotapply": func() (int, error) { return 0, errors.New("rule does not apply") },
	// constraint is a placeholder ,in Rule.Apply it will return a *Constraint (possibly nil)
	"constraint": func() *korrel8.Constraint { panic("placeholder") },
}

// New rule using a template to convert the start object to a goal query.
func New(name string, start, goal korrel8.Class, tmpl string) (*Rule, error) {
	t, err := template.New(name).Funcs(funcs).Option("missingkey=error").Parse(tmpl)
	return &Rule{Template: t, start: start, goal: goal}, err
}

func (r *Rule) String() string       { return fmt.Sprintf("%v(%v)->%v", r.Template.Name(), r.start, r.goal) }
func (r *Rule) Start() korrel8.Class { return r.start }
func (r *Rule) Goal() korrel8.Class  { return r.goal }

// Follow the rule by applying the template.
// The template will be executed with start as the "." context object.
// A function "constraint" returns the constraint.
func (r *Rule) Apply(start korrel8.Object, c *korrel8.Constraint) (result []string, err error) {
	b := &strings.Builder{}
	err = r.Template.Funcs(map[string]any{"constraint": func() *korrel8.Constraint { return c }}).Execute(b, start)
	if err != nil {
		err = fmt.Errorf("Appply %T %v: %w", start, start, err)
	}
	return []string{string(b.String())}, err
}

var _ korrel8.Rule = &Rule{}

// Read rules and add them to the engine.
func Read(reader io.Reader, engine *engine.Engine) error {
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 1024)
	sr := struct { // Serialized rule
		Name     string `json:"name"`
		Start    string `json:"start"`
		Goal     string `json:"goal"`
		Template string `json:"template"`
	}{}
	for {
		if err := decoder.Decode(&sr); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if sr.Name == "" || sr.Template == "" {
			return fmt.Errorf("invalid rule: %v", sr)
		}
		start, err := engine.ParseClass(sr.Start)
		if err != nil {
			return err
		}
		goal, err := engine.ParseClass(sr.Goal)
		if err != nil {
			return err
		}
		r, err := New(sr.Name, start, goal, sr.Template)
		if err != nil {
			return err
		}
		engine.Rules.Add(r)
	}
}
