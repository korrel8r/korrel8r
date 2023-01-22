package templaterule

import (
	"fmt"
	"io"
	"text/template"

	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

type ruleBuilder struct {
	name                     string
	starts, goals            []korrel8r.Class
	query, class, constraint *template.Template
	engine                   *engine.Engine
}

func newRuleBuilder(r *Rule, e *engine.Engine) (*ruleBuilder, error) {
	var (
		err error
		rb  = &ruleBuilder{name: r.Name, engine: e}
	)
	if rb.name == "" {
		rb.name = fmt.Sprintf("%v_to_%v", r.Start, r.Goal)
	}
	if rb.starts, err = rb.expand(&r.Start, "start"); err != nil {
		return nil, fmt.Errorf("expanding start of %v: %w", r.Name, err)
	}
	if rb.goals, err = rb.expand(&r.Goal, "goal"); err != nil {
		return nil, fmt.Errorf("expanding goal of %v: %w", r.Name, err)
	}
	if r.Result.Query == "" {
		return nil, fmt.Errorf("template is empty: %v.result.query", rb.name)
	}
	if rb.query, err = rb.newTemplate(r.Result.Query, ""); err != nil {
		return nil, err
	}
	c := r.Result.Class
	if c == "" && r.Goal.single() {
		c = korrel8r.ClassName(rb.goals[0])
	}
	if c == "" {
		return nil, fmt.Errorf("template is empty: %v.result.class ", rb.name)
	}
	if rb.class, err = rb.newTemplate(c, "-class"); err != nil {
		return nil, err
	}
	if rb.constraint, err = rb.newTemplate(r.Result.Constraint, "-constraint"); err != nil {
		return nil, err
	}
	return rb, nil
}

func (rb *ruleBuilder) expand(spec *ClassSpec, what string) (classes []korrel8r.Class, err error) {
	domain, err := rb.engine.DomainErr(spec.Domain)
	if err != nil {
		return nil, err
	}
	if len(spec.Classes) == 0 && len(spec.Matches) == 0 {
		return domain.Classes(), nil // Default to all classes in domain
	}
	list := unique.NewList[korrel8r.Class]()
	for _, name := range spec.Classes {
		c := domain.Class(name)
		if c == nil {
			return nil, fmt.Errorf("unknown class %v in domain %v", name, domain)
		}
		list.Append(c)
	}
	for _, s := range spec.Matches {
		t, err := rb.newTemplate(s, "-matches")
		if err != nil {
			return nil, err
		}
		for _, c := range domain.Classes() {
			if err := t.Execute(io.Discard, c.New()); err == nil {
				list.Append(c)
				log.V(4).Info("match", "rule", rb.name, what, c)
			}
		}
	}
	return list.List, nil
}

func (rb *ruleBuilder) newTemplate(text, suffix string) (*template.Template, error) {
	return template.New(rb.name + suffix).
		Option("missingkey=error").
		Funcs(Funcs).
		Funcs(rb.engine.TemplateFuncs()).
		Parse(text)
}

func (rb *ruleBuilder) rules() (rules []korrel8r.Rule, err error) {
	for _, start := range rb.starts {
		for _, goal := range rb.goals {
			rules = append(rules, &rule{
				start: start,
				goal:  goal,
				query: rb.query, class: rb.class,
				constraint: rb.constraint,
			})
		}
	}
	return rules, nil
}
