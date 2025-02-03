// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// # Template Functions
//
//	query
//	    Executes its argument as a korrel8r query, returns []any.
//	    May return an error.
package engine

import (
	"errors"
	"fmt"
	"text/template"

	"maps"

	"github.com/Masterminds/sprig/v3"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rules"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

// Builder initializes the state of an engine.
// [Engine] returns the immutable engine instance.
type Builder struct {
	e       *Engine
	err     error
	finally []func()
}

func Build() *Builder {
	e := &Engine{
		domains:     map[string]korrel8r.Domain{},
		stores:      map[korrel8r.Domain]*stores{},
		rulesByName: map[string]korrel8r.Rule{},
	}
	// Add template functions that are always available.
	e.templateFuncs = template.FuncMap{"query": e.query}
	maps.Copy(e.templateFuncs, sprig.TxtFuncMap())

	return &Builder{e: e}
}

func (b *Builder) Domains(domains ...korrel8r.Domain) *Builder {
	for _, d := range domains {
		switch b.e.domains[d.Name()] {
		case d: // Already present
		case nil:
			b.e.domains[d.Name()] = d
			b.e.stores[d] = newStores(d)
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
		if b.err != nil {
			return b
		}
		d := s.Domain()
		b.Domains(d)
		if b.err != nil {
			return b
		}
		b.store(nil, s)
	}
	return b
}

func (b *Builder) StoreConfigs(storeConfigs ...config.Store) *Builder {
	for _, sc := range storeConfigs {
		if b.err != nil {
			return b
		}
		if b.err != nil {
			return b
		}
		b.store(maps.Clone(sc), nil)
	}
	return b
}

func (b *Builder) store(sc config.Store, s korrel8r.Store) *store {
	var wrapper *store
	wrapper, b.err = newStore(b.e, sc, s)
	if b.err != nil {
		return nil
	}
	s, b.err = wrapper.Ensure()
	if b.err != nil {
		return nil
	}
	if tf, ok := s.(interface{ TemplateFuncs() map[string]any }); ok {
		maps.Copy(b.e.templateFuncs, tf.TemplateFuncs())
	}
	b.err = b.e.stores[s.Domain()].Add(wrapper)
	return wrapper
}

// store adds a wrapper store created by [Store] or [StoreConfigs]

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

// Config an engine.Builder.
func (b *Builder) Config(configs config.Configs) *Builder {
	if b.err != nil {
		return b
	}
	for source, c := range configs {
		b.config(&c)
		if b.err != nil {
			b.err = fmt.Errorf("%v: %w", source, b.err)
			return b
		}
	}
	return b
}

func (b *Builder) ConfigFile(file string) *Builder {
	cfg, err := config.Load(file)
	if err != nil {
		b.err = err
		return b
	}
	return b.Config(cfg)
}

// Engine returns the final engine, which can not be modified.
// The [Builder] is reset to the initial state returned by [Build].
func (b *Builder) Engine() (*Engine, error) {
	for _, f := range b.finally { // Complete deferred configuration
		f()
		if b.err != nil {
			break
		}
	}
	e, err := b.e, b.err
	*b = *Build() // Reset the builder.
	return e, err
}

func (b *Builder) config(c *config.Config) {
	if b.err != nil {
		return
	}
	b.StoreConfigs(c.Stores...)
	for _, r := range c.Rules {
		if b.err != nil {
			return
		}
		startDomain, start := b.classes(&r.Start)
		if b.err != nil {
			return
		}
		goalDomain, goal := b.classes(&r.Goal)
		if b.err != nil {
			return
		}
		var tmpl *template.Template
		tmpl, b.err = b.e.NewTemplate(r.Name).Parse(r.Result.Query)
		if b.err != nil {
			return
		}
		// Defer creation of rules so wildcards are evaluated after dynamic domains have
		// accumulated all their classes and wildcards can be evaluated.
		b.finally = append(b.finally, func() {
			start := b.wildcard(startDomain, start)
			goal := b.wildcard(goalDomain, goal)
			if b.err != nil {
				b.err = fmt.Errorf("bad rule %v: %w", tmpl.Name(), b.err)
			} else {
				b.Rules(rules.NewTemplateRule(start, goal, tmpl))
			}
		})
	}
}

func (b *Builder) wildcard(d korrel8r.Domain, classes []korrel8r.Class) []korrel8r.Class {
	if len(classes) == 0 {
		classes = d.Classes()
	}
	if len(classes) == 0 {
		b.joinErr(fmt.Errorf("no classes in domain: %v", d))
	}
	return classes
}

func (b *Builder) classes(spec *config.ClassSpec) (korrel8r.Domain, []korrel8r.Class) {
	d := b.getDomain(spec.Domain)
	if b.err != nil {
		return d, nil
	}
	list := unique.NewList[korrel8r.Class]()
	for _, class := range spec.Classes {
		c := d.Class(class)
		if c == nil {
			b.err = korrel8r.ClassNotFoundError{Class: class, Domain: d}
			return d, nil
		}
		list.Append(c)
	}
	return d, list.List
}

func (b *Builder) getDomain(name string) (d korrel8r.Domain) {
	d, b.err = b.e.DomainErr(name)
	return
}

func (b *Builder) joinErr(err error) { b.err = errors.Join(b.err, err) }
