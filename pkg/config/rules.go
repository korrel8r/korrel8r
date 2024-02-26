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

func addRules(e *engine.Engine, r Rule) (err error) {
	if r.Name == "" {
		r.Name = fmt.Sprintf("%v=%v", r.Start, r.Goal)
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("rule %v: %w", r.Name, err)
		}
	}()
	newTemplate := func(name, text string) (*template.Template, error) {
		return template.New(name).Option("missingkey=error").Funcs(rules.Funcs).Funcs(e.TemplateFuncs()).Parse(text)
	}
	query, err := newTemplate(r.Name, r.Result.Query)
	if err != nil {
		return fmt.Errorf("error in query %q: %w", r.Result.Query, err)
	}
	return eachClass(e, &r.Start, func(start korrel8r.Class) error {
		return eachClass(e, &r.Goal, func(goal korrel8r.Class) error {
			e.AddRules(rules.NewTemplateRule(start, goal, query))
			return nil
		})
	})
}

func eachClass(e *engine.Engine, spec *ClassSpec, f func(korrel8r.Class) error) error {
	d, err := e.DomainErr(spec.Domain)
	if err != nil {
		return err
	}
	list := unique.NewList[korrel8r.Class]()
	if len(spec.Classes) == 0 {
		list.Append(d.Classes()...) // Missing class list means all classes in domain.
	} else {
		for _, class := range spec.Classes {
			c := d.Class(class)
			if c == nil {
				return korrel8r.ClassNotFoundErr{Class: class, Domain: d}
			}
			list.Append(c)
		}
	}
	for _, c := range list.List {
		if err := f(c); err != nil {
			return err
		}
	}
	return nil
}
