// package templaterule implements korrel8r.Rule using Go templates.
package templaterule

import (
	"text/template"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// Rule is a template rule specification that can be serialized as JSON.
// It generates one or more korrel8r.Rule.
type Rule struct {
	// Name is a short, descriptive name.
	// If omitted, a name is generated from Start and Goal.
	Name string `json:"name,omitempty"`

	// Start specifies the set of classes that this rule can apply to.
	Start ClassSpec `json:"start"`

	// Goal specifies the set of classes that this rule can produce.
	Goal ClassSpec `json:"goal"`

	// Result contains templates to generate the result of applying this rule.
	// Each template is applied to an object from one of the `start` classes.
	// If any template yields a blank string or an error, the rule does not apply.
	Result ResultSpec
}

// ClassSpec specifies one or more classes.
type ClassSpec struct {
	// Domain is the domain for selected classes.
	Domain string `json:"domain"`

	// Classes is a list of class names to be selected from the domain.
	// If absent, all classes in the domain are selected.
	Classes []string `json:"classes,omitempty"`
}

// ResultSpec contains result templates.
type ResultSpec struct {
	// Query template generates a query object suitable for the goal store.
	Query string `json:"query"`

	// Constraint template is optional, it generates a korrel8r.Constraint in JSON form.
	// This constraint is combined with the constraint already in force, if there is one.
	// See Constraint.Combine
	Constraint string `json:"constraint,omitempty"`
}

// Rules generates one or more korrel8r.Rule from the template Rule.
func (r *Rule) Rules(domains map[string]korrel8r.Domain, funcs template.FuncMap) (rules []korrel8r.Rule, err error) {
	rb, err := newRuleBuilder(r, domains, funcs)
	if err != nil {
		return nil, err
	}
	return rb.rules()
}
