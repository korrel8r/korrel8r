// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"fmt"
	"reflect"
	"slices"
	"text/template"

	"maps"

	sprig "github.com/go-task/slim-sprig"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rules"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

// Builder initializes the state of an engine.
// Engine() returns the immutable engine instance.
type Builder struct {
	e   *Engine
	err error
}

func Build() *Builder {
	e := &Engine{
		domains:     map[string]korrel8r.Domain{},
		stores:      map[korrel8r.Domain]stores{},
		rulesByName: map[string]korrel8r.Rule{},
	}
	e.templateFuncs = template.FuncMap{"get": e.get}
	maps.Copy(e.templateFuncs, sprig.FuncMap())
	return &Builder{e: e}
}

func (b *Builder) Domains(domains ...korrel8r.Domain) *Builder {
	for _, d := range domains {
		switch b.e.domains[d.Name()] {
		case d: // Already present
		case nil:
			b.e.domains[d.Name()] = d
			if tf, ok := d.(interface{ TemplateFuncs() map[string]any }); ok {
				maps.Copy(b.e.templateFuncs, tf.TemplateFuncs())
			}
		default:
			b.err = fmt.Errorf("Duplicate domain name: %v", d.Name())
			return b
		}
	}
	return b
}

func (b *Builder) Stores(stores ...korrel8r.Store) *Builder {
	for _, s := range stores {
		d := s.Domain()
		b.Domains(d)
		if b.err != nil {
			return b
		}
		b.e.stores[d] = append(b.e.stores[d], &store{Store: s})
	}
	return b
}

func (b *Builder) StoreConfigs(storeConfigs ...config.Store) *Builder {
	for _, sc := range storeConfigs {
		if b.err != nil {
			return b
		}
		sc = maps.Clone(sc)
		d := b.getDomain(sc[config.StoreKeyDomain])
		if b.err != nil {
			return b
		}
		ss := b.e.stores[d]
		if slices.ContainsFunc(ss, func(s *store) bool { return reflect.DeepEqual(sc, s.Original) }) {
			b.err = fmt.Errorf("duplicate store configuration: %v", sc)
			return b
		}
		b.e.stores[d] = append(ss, &store{Original: sc})
	}
	return b
}

func (b *Builder) Rules(rules ...korrel8r.Rule) *Builder {
	for _, r := range rules {
		if b.err != nil {
			return b
		}
		r2 := b.e.rulesByName[r.Name()]
		if r2 != nil {
			b.err = fmt.Errorf("Duplicate rule name: %v", r.Name())
			return b
		}
		b.Domains(r.Start()[0].Domain(), r.Goal()[0].Domain())
		b.e.rulesByName[r.Name()] = r
		b.e.rules = append(b.e.rules, r)
	}
	return b
}

// Apply an engine.Builder.
func (b *Builder) Apply(configs config.Configs) *Builder {
	if b.err != nil {
		return b
	}
	if b.err = configs.Expand(); b.err != nil {
		return b
	}
	for source, c := range configs {
		b.config(source, c)
		if b.err != nil {
			b.err = fmt.Errorf("%v: %w", source, b.err)
			return b
		}
	}
	return b
}

// Engine returns the final engine, which can no longer be modified.
// The Builder must not be used after calling Engine()
func (b *Builder) Engine() (*Engine, error) {
	e := b.e
	b.e = nil
	// Create all stores to report problems early.
	for d, ss := range e.stores {
		for _, s := range ss {
			// Not an error if create fails, will be registered in stores.
			_ = s.Ensure(d, func(s string) (string, error) { return e.execTemplate(s, nil) })
		}
	}
	return e, b.err
}

func (b *Builder) config(source string, c *config.Config) {
	if b.err != nil {
		return
	}
	b.StoreConfigs(c.Stores...)
	for _, r := range c.Rules {
		if b.err != nil {
			return
		}
		start := b.classes(&r.Start)
		if b.err != nil {
			return
		}
		goal := b.classes(&r.Goal)
		if b.err != nil {
			return
		}
		var tmpl *template.Template
		tmpl, b.err = template.New(r.Name).Funcs(b.e.TemplateFuncs()).Parse(r.Result.Query)
		if b.err != nil {
			return
		}
		b.Rules(rules.NewTemplateRule(start, goal, tmpl))
	}
}

func (b *Builder) classes(spec *config.ClassSpec) []korrel8r.Class {
	d := b.getDomain(spec.Domain)
	if b.err != nil {
		return nil
	}
	list := unique.NewList[korrel8r.Class]()
	if len(spec.Classes) == 0 {
		list.Append(d.Classes()...) // Missing class list means all classes in domain.
	} else {
		for _, class := range spec.Classes {
			c := d.Class(class)
			if c == nil {
				b.err = korrel8r.ClassNotFoundError{Class: class, Domain: d}
				return nil
			}
			list.Append(c)
		}
	}
	if len(list.List) == 0 {
		b.err = fmt.Errorf("invalid class specification: %#+v", *spec)
	}
	return list.List
}

func (b *Builder) getDomain(name string) (d korrel8r.Domain) {
	d, b.err = b.e.DomainErr(name)
	return
}
