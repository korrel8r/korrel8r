// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"text/template"

	sprig "github.com/go-task/slim-sprig"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

//  FIXME config stuff here from config package.

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

// Err returns a non-nil error if anything goes wrong during building.
func (b *Builder) Err() error { return b.err }

func (b *Builder) error(err error) bool {
	b.err = errors.Join(b.err, err)
	return b.err != nil
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
			b.error(fmt.Errorf("Duplicate domain name: %v", d.Name()))
			return b
		}
	}
	return b
}

func (b *Builder) Stores(stores ...korrel8r.Store) *Builder {
	for _, s := range stores {
		d := s.Domain()
		b.Domains(d)
		if b.Err() != nil {
			return b
		}
		b.e.stores[d] = append(b.e.stores[d], &store{Store: s})
	}
	return b
}

func (b *Builder) StoreConfigs(storeConfigs ...korrel8r.StoreConfig) *Builder {
	for _, sc := range storeConfigs {
		d, err := b.GetDomain(sc[korrel8r.StoreKeyDomain])
		if b.error(err) {
			continue
		}
		ss := b.e.stores[d]
		if slices.IndexFunc(ss, func(s *store) bool { return reflect.DeepEqual(sc, s.Original) }) >= 0 {
			continue // Already present
		}
		b.e.stores[d] = append(ss, &store{Original: sc})
	}
	return b
}

func (b *Builder) Rules(rules ...korrel8r.Rule) *Builder {
	for _, r := range rules {
		r2 := b.e.rulesByName[r.Name()]
		switch {
		case r2 == nil:
			b.Domains(r.Start()[0].Domain(), r.Goal()[0].Domain())
			b.e.rulesByName[r.Name()] = r
			b.e.rules = append(b.e.rules, r)
		case reflect.DeepEqual(r, r2):
			// Rule is already present, ignore
		default:
			_ = b.error(fmt.Errorf("Duplicate rule name: %v", r.Name()))
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
	// Create all the stores so we have status if there are any problems.
	for d, ss := range e.stores {
		for _, s := range ss {
			// Not an error if create fails, will be registered in stores.
			_ = s.Ensure(d, func(s string) (string, error) { return e.execTemplate(s, nil) })
		}
	}
	return e, b.Err()
}

type Applier interface{ Apply(*Builder) error }

func (b *Builder) Apply(a Applier) *Builder { b.err = errors.Join(b.err, a.Apply(b)); return b }

func (b *Builder) GetDomain(name string) (korrel8r.Domain, error) { return b.e.DomainErr(name) }
func (b *Builder) GetTemplateFuncs() map[string]any               { return b.e.TemplateFuncs() }
