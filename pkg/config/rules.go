// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package config

import (
	"fmt"
	"text/template"

	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rules"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

func newRule(b *engine.Builder, r *Rule) (rule korrel8r.Rule, err error) {
	start, err := classes(b, &r.Start)
	if err != nil {
		return nil, err
	}
	goal, err := classes(b, &r.Goal)
	if err != nil {
		return nil, err
	}
	if len(start) == 0 || len(goal) == 0 || r.Name == "" {
		return nil, fmt.Errorf("invalid: %#v", r)
	}
	tmpl, err := template.New(r.Name).Funcs(b.GetTemplateFuncs()).Parse(r.Result.Query)
	if err != nil {
		return nil, err
	}
	return rules.NewTemplateRule(start, goal, tmpl), nil
}

func classes(b *engine.Builder, spec *ClassSpec) ([]korrel8r.Class, error) {
	d, err := b.GetDomain(spec.Domain)
	if err != nil {
		return nil, err
	}
	list := unique.NewList[korrel8r.Class]()
	if len(spec.Classes) == 0 {
		list.Append(d.Classes()...) // Missing class list means all classes in domain.
	} else {
		for _, class := range spec.Classes {
			c := d.Class(class)
			if c == nil {
				return nil, korrel8r.ClassNotFoundErr{Class: class, Domain: d}
			}
			list.Append(c)
		}
	}
	return list.List, nil
}
