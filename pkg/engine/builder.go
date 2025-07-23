// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// # Template Functions
//
//	query
//	    Executes its argument as a korrel8r query, returns []any.
//	    May return an error.
package engine

import (
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
			b.err = fmt.Errorf("duplicate domain name: %v", d.Name())
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

// store adds a store wrapper.
// Exactly one of sc and s must be non-nil.
func (b *Builder) store(sc config.Store, s korrel8r.Store) *storeHolder {
	var wrapper *storeHolder
	wrapper, b.err = wrap(b.e, sc, s)
	if b.err != nil {
		return nil
	}
	b.e.stores[wrapper.Domain()].Add(wrapper)
	// Store errors don't prevent startup, but check and log a warning.
	var err error
	_, err = wrapper.Ensure()
	if err != nil {
		log.Info("warning: cannot connect to store", "domain", wrapper.Domain().Name(), "error", err)
	}
	return wrapper
}

// store adds a wrapper store created by [Store] or [StoreConfigs]

func (b *Builder) Rules(rules ...korrel8r.Rule) *Builder {
	// Delay adding rules after domains and stores are configured.
	b.finally = append(b.finally, func() { b.rules(rules...) })
	return b
}

func (b *Builder) rules(rules ...korrel8r.Rule) {
	for _, r := range rules {
		if b.err != nil {
			return
		}
		r2 := b.e.rulesByName[r.Name()]
		if r2 != nil {
			b.err = fmt.Errorf("duplicate rule name: %v", r.Name())
			return
		}
		b.Domains(r.Start()[0].Domain(), r.Goal()[0].Domain())
		b.e.rulesByName[r.Name()] = r
		b.e.rules = append(b.e.rules, r)
	}
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
		// Defer adding rules until domains and stores are configured.
		b.finally = append(b.finally, func() { b.configRule(r) })
	}
}

func (b *Builder) configRule(r config.Rule) {
	if b.err != nil {
		return
	}
	defer func() {
		if b.err != nil {
			b.err = fmt.Errorf("invalid rule %v: %w", r.Name, b.err)
		}
	}()
	start := b.classes(r, &r.Start)
	goal := b.classes(r, &r.Goal)
	if len(start) == 0 || len(goal) == 0 {
		return
	}
	var tmpl *template.Template
	tmpl, b.err = b.e.NewTemplate(r.Name).Parse(r.Result.Query)
	if b.err != nil {
		return
	}
	b.rules(rules.NewTemplateRule(start, goal, tmpl))
}

func (b *Builder) classes(r config.Rule, spec *config.ClassSpec) []korrel8r.Class {
	var d korrel8r.Domain
	d, b.err = b.e.Domain(spec.Domain)
	if b.err != nil {
		return nil
	}
	if len(spec.Classes) > 0 {
		list := unique.NewList[korrel8r.Class]()
		for _, class := range spec.Classes {
			c := d.Class(class)
			if c == nil {
				b.err = korrel8r.ClassNotFoundError(class)
				return nil
			}
			list.Append(c)
		}
		return list.List
	} else { // Wildcard
		classes, err := b.e.ClassesFor(d)
		if err != nil {
			// Log a message but continue
			log.Error(err, "Skip rule", "rule", r)
		}
		return classes
	}
}
