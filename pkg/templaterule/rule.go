// package templaterule implements korrel8.Rule using Go templates.
package templaterule

import (
	"fmt"

	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/korrel8"
)

// Rule is a template rule specification that can be serialized as JSON.
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
	// If both Classes and Matches are omitted, then all classes in the domain are selected.
	Domain string `json:"domain"`

	// Classes is a list of class names to be selected from the domain.
	Classes []string `json:"classes,omitempty"`

	// Matches is a list of templates to select classes from the domain.
	// Each template is executed with an empty instance of each class in the domain.
	// If the template executes without error, the class is selected.
	// The result of the template is ignored.
	Matches []string `json:"matches,omitempty"`
}

// ResultSpec contains result templates.
type ResultSpec struct {
	// URI template generates a query URI (normally a relative URI reference) that can be resolved
	// by a Store to retrieve objects.
	URI string `json:"uri"`

	// Class template generates the qualified class name for objects referenced by the URI.
	// Must be a class name selected by the Goal field.
	// If the Goal field contains a single, non-wildcard class, this field is optional.
	Class string `json:"class,omitempty"`

	// Constraint template is optional, it generates a korrel8.Constraint in JSON form.
	// This constraint is combined with the constraint already in force, if there is one.
	// See Constraint.Combine
	Constraint string `json:"constraint,omitempty"`
}

// Rules generates one or more korrel8.Rule from the template Rule.
func (r *Rule) Rules(e *engine.Engine) (rules []korrel8.Rule, err error) {
	rb, err := newRuleBuilder(r, e)
	if err != nil {
		return nil, err
	}
	return rb.rules()
}

func (c ClassSpec) single() bool { return len(c.Matches) == 0 && len(c.Classes) == 1 }

func (c ClassSpec) String() string {
	if c.single() {
		return fmt.Sprintf("%v/%v", c.Domain, c.Classes[0])
	} else {
		return fmt.Sprintf("%v/%v%v", c.Domain, c.Classes, c.Matches)
	}
}
