// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package templaterule implements korrel8r.Rule using Go templates.
package templaterule

import (
	"fmt"
	"text/template"

	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

// Factory for creating korrel8r.Rule s from api.Rule configuration.
type Factory struct {
	domains map[string]korrel8r.Domain
	funcs   template.FuncMap
}

func NewFactory(domains map[string]korrel8r.Domain, funcs template.FuncMap) *Factory {
	return &Factory{domains: domains, funcs: funcs}
}

// Rules generates korrel8r.Rules from an api.Rule configuration.
func (f *Factory) Rules(r api.Rule) (rules []korrel8r.Rule, err error) {
	name := r.Name
	if name == "" {
		name = fmt.Sprintf("%v->%v", r.Start, r.Goal)
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("rule %v: %w", name, err)
		}
	}()
	newTemplate := func(name, text string) (*template.Template, error) {
		return template.New(name).Option("missingkey=error").Funcs(Funcs).Funcs(f.funcs).Parse(text)
	}
	query, err := newTemplate(name, r.Result.Query)
	if err != nil {
		return nil, fmt.Errorf("error in query %q: %w", r.Result.Query, err)
	}
	constraint, err := newTemplate(name+"-constraint", r.Result.Constraint)
	if err != nil {
		return nil, fmt.Errorf("error in constraint %q: %w", r.Result.Constraint, err)
	}
	starts, err := f.classes(&r.Start, "start")
	if err != nil {
		return nil, fmt.Errorf("start %#v: %w", r.Start, err)
	}
	goals, err := f.classes(&r.Goal, "goal")
	if err != nil {
		return nil, fmt.Errorf("goal %#v: %w", r.Goal, err)
	}
	// Generate rules for each start/goal pair
	for _, start := range starts {
		for _, goal := range goals {
			rules = append(rules, &rule{
				start:      start,
				goal:       goal,
				query:      query,
				constraint: constraint,
			})
		}
	}
	return rules, nil
}

func (f *Factory) classes(spec *api.ClassSpec, what string) ([]korrel8r.Class, error) {
	domain := f.domains[spec.Domain]
	if domain == nil {
		return nil, korrel8r.DomainNotFoundErr{Domain: spec.Domain}
	}
	list := unique.NewList[korrel8r.Class]()
	if len(spec.Classes) == 0 {
		list.Append(domain.Classes()...)
	} else {
		for _, name := range spec.Classes {
			c := domain.Class(name)
			if c == nil {
				return nil, korrel8r.ClassNotFoundErr{Class: name, Domain: domain}
			}
			list.Append(c)
		}
	}
	if len(list.List) == 0 {
		return nil, fmt.Errorf("no classes for %v", what)
	}
	return list.List, nil
}
