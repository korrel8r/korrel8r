// package templaterule implements korrel8.Rule using Go templates.
package templaterule

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/unique"
)

// Rule is a serializable rule template, that generates one or more korrel8.Rules.
type Rule struct {
	// Name is a short, descriptive name.
	Name string `json:"name"`

	// Start defines the set of classes that this rule can apply to.
	//
	// Start[0] is a domain name.
	//
	// Elements of Start[1:] may be class names in the Start[0] domain or class selection templates.
	//
	// Class selection templates are applied to an empty instance of each class in the domain.
	// If they generate the string "true" the class is selected.
	// Any other result, including errors, means the class is not included as a start class.
	//
	Start []string `json:"start"`

	// Goal indicates the class(es) of goals this rule can produce.
	//
	// Start[0] is a domain name.
	//
	// Start[1] may be a class name in the Start[0] domain or a class generator template.
	//
	// Class generator templates are applied to a starting object for the rule.
	// They should generate the class name of the goal class when applying the rule to that object.
	// Any other result, including errors, means the rule will not be followed for that start object.
	//
	Goal []string `json:"goal"`

	// Query is a template that generates the goal query from a start object.
	//
	// For a given start object, Query should return a query that gets instances of the Goal class.
	Query string `json:"query"`

	// Constraint is an optional template that generates a korrel8.Constraint in JSON form from a start object.
	//
	// If there is already a constraint in force, this constraint is ignored.
	Constraint string `json:"constraint,omitempty"`
}

type ruleBuilder struct {
	Rule
	engine *engine.Engine

	startDomain  korrel8.Domain
	startClasses unique.List[korrel8.Class]

	goalDomain   korrel8.Domain
	goalClass    korrel8.Class
	goalTemplate *template.Template

	query, constraint *template.Template
}

// Rules generates one or more korrel8.Rule from the template Rule.
func (r Rule) Rules(e *engine.Engine) ([]korrel8.Rule, error) {
	rb := ruleBuilder{Rule: r, engine: e, startClasses: unique.NewList[korrel8.Class]()}
	var rules []korrel8.Rule
	if err := rb.setup(); err != nil {
		return nil, err
	}
	for _, start := range rb.startClasses.List {
		switch {
		case rb.goalClass != nil: // Literal goal class, single korrel8.Rule
			rules = append(rules, &rule{Template: rb.query, start: start, goal: rb.goalClass})
		case rb.goalTemplate != nil: // Goal class template test, must try against all goal domain classes.
			for _, g := range rb.goalDomain.Classes() {
				rules = append(rules, &rule{Template: rb.query, start: start, goal: g})
			}
		}
		// FIXME constraint handling - see Apply
	}
	return rules, nil
}

/// FIXME this has gotten too complex.?

func (rb *ruleBuilder) setup() (err error) {
	for _, f := range []func() error{
		rb.setupStart,
		rb.setupGoal,
		// FIXME use nested templates?
		func() error { rb.query, err = rb.newTemplate(rb.Query, ""); return err },
		func() error { rb.constraint, err = rb.newTemplate(rb.Constraint, "-constraint"); return err },
	} {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

func (rb *ruleBuilder) setupStart() (err error) {
	if rb.startDomain, err = rb.getDomain("start", rb.Start); err != nil {
		return err
	}
	knownClasses := rb.startDomain.Classes()
	for _, x := range rb.Start[1:] {
		c, t, err := rb.classOrTemplate(x, rb.startDomain)
		switch {
		case c != nil: // Literal class name
			rb.startClasses.Append(c)
		case t != nil: // Test template, test each class in start domain
			for _, c := range knownClasses {
				if rb.matches(c, t) {
					rb.startClasses.Append(c)
				}
			}
		case err != nil:
			return err
		}
	}
	if rb.startClasses.List == nil {
		return fmt.Errorf("start must match at least one class: %+v", rb.Rule)
	}
	return nil
}

func (rb *ruleBuilder) setupGoal() (err error) {
	if len(rb.Goal) != 2 {
		return fmt.Errorf("goal must contain two elements; [domain, classOrTemplate]")
	}
	if rb.goalDomain, err = rb.getDomain("goal", rb.Goal); err != nil {
		return err
	}
	if rb.goalClass, rb.goalTemplate, err = rb.classOrTemplate(rb.Goal[1], rb.goalDomain); err != nil {
		return err
	}
	return nil
}

func (rb *ruleBuilder) newTemplate(text, suffix string) (*template.Template, error) {
	return template.New(rb.Name + suffix).Funcs(Funcs).Option("missingkey=error").Parse(text)
}

func (rb *ruleBuilder) getDomain(what string, list []string) (korrel8.Domain, error) {
	if len(list) > 0 {
		d := rb.engine.Domain(list[0])
		if d != nil {
			return d, nil
		}
	}
	return nil, fmt.Errorf("%v must start with a domain name: %v", what, list)
}

func (rb *ruleBuilder) classOrTemplate(x string, d korrel8.Domain) (korrel8.Class, *template.Template, error) {
	if c := d.Class(x); c != nil {
		return c, nil, nil
	}
	t, err := rb.newTemplate(x, "-test")
	return nil, t, err
}

func (rb *ruleBuilder) matches(c korrel8.Class, t *template.Template) bool {
	w := &strings.Builder{}
	if err := t.Execute(w, c.New()); err != nil {
		return false
	}
	return w.String() == "true"
}
