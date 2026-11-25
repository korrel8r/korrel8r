// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"fmt"
	"text/template"

	"maps"

	"github.com/Masterminds/sprig/v3"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rules"
	"github.com/korrel8r/korrel8r/pkg/status"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Builder initializes the state of an engine.
// [Engine] returns the immutable engine instance.
type Builder struct {
	e   *Engine
	err error

	// Named templates from configuration, collected before building the template base.
	templates []config.Template
}

func Build() *Builder {
	e := &Engine{
		domains:          korrel8r.NewDomains(),
		storeHolders:     map[korrel8r.Domain]*storeHolders{},
		storeMetricAttrs: map[string][2]metric.MeasurementOption{},
	}
	// Add template functions that are always available.
	e.templateFuncs = e.TemplateFuncs()
	maps.Copy(e.templateFuncs, sprig.TxtFuncMap())
	e.templateBase = template.New("").Funcs(e.templateFuncs).Option("missingkey=error")

	return &Builder{e: e}
}

// GetDomains returns the domain registry used by the engine.
// Domains are collected here before the engine is built.
func (b *Builder) GetDomains() *korrel8r.Domains {
	ds := b.e.domains
	if ds.Empty() {
		panic("Builder: GetDomains called before any domains were added. ")
	}
	return ds
}

// Domains adds domains that will be recognized by the engine.
func (b *Builder) Domains(domains ...korrel8r.Domain) *Builder {
	for _, d := range domains {
		switch b.e.domains.Get(d.Name()) {
		case d: // Already present
		case nil:
			b.e.domains.Add(d)
			b.e.storeHolders[d] = newStoreHolders(d)
			name := d.Name()
			b.e.storeMetricAttrs[name] = [2]metric.MeasurementOption{
				metric.WithAttributes(attribute.String("domain", name), attribute.String("status", "ok")),
				metric.WithAttributes(attribute.String("domain", name), attribute.String("status", "error")),
			}
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

// Stores add ready-to-use stores to the engine. Compare with StoreConfigs.
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

// StoreConfigs adds store configuration to the engine.
// The stores are (re)created on demand or after an error.
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

// store adds a store wrapper with configuration or a working Store.
// Exactly one of sc and s must be non-nil.
func (b *Builder) store(sc config.Store, s korrel8r.Store) *storeHolder {
	var wrapper *storeHolder
	wrapper, b.err = wrap(b.e, sc, s)
	if b.err != nil {
		return nil
	}
	b.e.storeHolders[wrapper.Domain()].Add(wrapper)
	// Store errors don't prevent startup, but check and log a warning.
	var err error
	_, err = wrapper.Ensure()
	if err != nil {
		log.Info("cannot connect to store", "domain", wrapper.Domain().Name(), "error", err)
	}
	return wrapper
}

// Rules adds pre-built rules to the engine, and adds their domains if necessary.
func (b *Builder) Rules(rules ...korrel8r.Rule) *Builder {
	for _, r := range rules {
		b.Domains(r.Start()[0].Domain(), r.Goal()[0].Domain())
	}
	return b.RuleFactory(func(*korrel8r.Domains) []korrel8r.Rule { return rules })
}

// StatusRules adds pre-built status rules to the engine.
func (b *Builder) StatusRules(rules ...status.Rule) *Builder {
	return b.StatusRuleFactory(func(*korrel8r.Domains) []status.Rule { return rules })
}

// Config an engine.Builder.
func (b *Builder) Config(configs config.Configs) *Builder {
	if b.err != nil {
		return b
	}
	b.e.configs = append(b.e.configs, configs...)
	for source, c := range configs {
		b.config(&c)
		if b.err != nil {
			b.err = fmt.Errorf("%v: %w", source, b.err)
			return b
		}
		b.Tuning(c.Tuning)
	}
	return b
}

func (b *Builder) Tuning(t *config.Tuning) *Builder {
	if t != nil {
		b.e.Tuning = *t
	}
	return b
}

// RuleFactory adds rules produced from the engine domains.
// The factory is called again when a [korrel8r.GrowingDomain] adds classes.
func (b *Builder) RuleFactory(factory func(*korrel8r.Domains) []korrel8r.Rule) *Builder {
	b.e.ruleFactories = append(b.e.ruleFactories, factory)
	return b
}

// StatusRuleFactory adds status rules produced from the engine domains.
// The factory is called again when a [korrel8r.GrowingDomain] adds classes.
func (b *Builder) StatusRuleFactory(factory func(*korrel8r.Domains) []status.Rule) *Builder {
	b.e.statusRuleFactories = append(b.e.statusRuleFactories, factory)
	return b
}

// SearchGuard sets the process-wide guard used to cancel active searches.
func (b *Builder) SearchGuard(guard SearchGuard) *Builder {
	b.e.searchGuard = guard
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

// Engine returns the final engine.
// Rules are rebuilt by [Engine.RefreshNewClasses] when domains add classes.
// The [Builder] is reset to the initial state returned by [Build].
func (b *Builder) Engine() (*Engine, error) {
	// Build the base template set with named templates before processing rules.
	b.buildTemplateBase()
	e := b.e
	for _, d := range e.Domains() {
		if g, ok := d.(korrel8r.GrowingDomain); ok {
			e.growing = append(e.growing, g)
		}
	}
	e.ruleState.Store(newRuleState(nil, nil, nil)) // Never nil, even on error.
	if b.err == nil {
		b.err = e.rebuildRules()
	}
	err := b.err
	*b = *Build() // Reset the builder.
	return e, err
}

func (b *Builder) config(c *config.Config) {
	if b.err != nil {
		return
	}
	b.StoreConfigs(c.Stores...)
	b.templates = append(b.templates, c.Templates...)
}

// buildTemplateBase adds named templates from configuration to the base template set.
func (b *Builder) buildTemplateBase() {
	if b.err != nil {
		return
	}
	// Update funcs to include domain-specific functions added after Build().
	b.e.templateBase.Funcs(b.e.templateFuncs)
	for _, t := range b.templates {
		if _, err := b.e.templateBase.New(t.Name).Parse(t.Template); err != nil {
			b.err = fmt.Errorf("invalid template %q: %w", t.Name, err)
			return
		}
	}
}

// ruleBuilder resolves the engine's rules against the current domain classes.
type ruleBuilder struct {
	e        *Engine
	base     *template.Template
	err      error
	byName   map[string]korrel8r.Rule
	rules    []korrel8r.Rule
	statuses map[string][]status.Rule // Keyed by class.String().
	missing  rules.MissingClasses     // Report of rules affected by unknown classes.
}

// buildRules builds a rule state from configuration and rule factories.
func (e *Engine) buildRules() (*ruleState, error) {
	// Rule templates are associated with a clone, so a rebuild doesn't modify templates in use.
	base, err := e.templateBase.Clone()
	if err != nil {
		return nil, err
	}
	b := &ruleBuilder{e: e, base: base, byName: map[string]korrel8r.Rule{},
		statuses: map[string][]status.Rule{}}
	for _, c := range e.configs {
		for _, r := range c.Rules {
			b.configRule(r)
		}
		for _, r := range c.StatusRules {
			b.configStatusRule(r)
		}
	}
	for _, f := range e.ruleFactories {
		b.addRules(f(e.domains)...)
	}
	for _, f := range e.statusRuleFactories {
		b.addStatusRules(f(e.domains)...)
	}
	if b.err != nil {
		return nil, b.err
	}
	b.missing.Log(log)
	return newRuleState(b.rules, b.byName, b.statuses), nil
}

func (b *ruleBuilder) addRules(rules ...korrel8r.Rule) {
	for _, r := range rules {
		if b.err != nil {
			return
		}
		if b.byName[r.Name()] != nil {
			b.err = fmt.Errorf("duplicate rule name: %v", r.Name())
			return
		}
		for _, d := range []korrel8r.Domain{r.Start()[0].Domain(), r.Goal()[0].Domain()} {
			if b.e.domains.Get(d.Name()) == nil {
				b.err = fmt.Errorf("rule %v: unknown domain %v", r.Name(), d.Name())
				return
			}
		}
		b.byName[r.Name()] = r
		b.rules = append(b.rules, r)
	}
}

func (b *ruleBuilder) addStatusRules(rules ...status.Rule) {
	for _, r := range rules {
		for _, c := range r.Start() {
			key := c.String()
			b.statuses[key] = append(b.statuses[key], r)
		}
	}
}

func (b *ruleBuilder) configRule(r config.Rule) {
	if b.err != nil {
		return
	}
	defer func() {
		if b.err != nil {
			b.err = fmt.Errorf("invalid rule %v: %w", r.Name, b.err)
		}
	}()
	start := b.classes(r.Name, &r.Start)
	goal := b.classes(r.Name, &r.Goal)
	if b.err != nil {
		return
	}
	if len(start) == 0 || len(goal) == 0 {
		b.missing.DropRule(r.Name)
		return
	}
	var kr korrel8r.Rule
	kr, b.err = rules.NewTemplateRule(start, goal, b.base.New(r.Name), r.Result.Query, b.e.domains)
	if b.err == nil {
		b.addRules(kr)
	}
}

func (b *ruleBuilder) configStatusRule(r config.StatusRule) {
	if b.err != nil {
		return
	}
	defer func() {
		if b.err != nil {
			b.err = fmt.Errorf("invalid label rule %v: %w", r.Name, b.err)
		}
	}()
	start := b.classes(r.Name, &r.Start)
	if b.err != nil {
		return
	}
	if len(start) == 0 {
		b.missing.DropStatusRule(r.Name)
		return
	}
	var tmpl *template.Template
	tmpl, b.err = b.base.New(r.Name).Parse(r.Status)
	if b.err != nil {
		return
	}
	b.addStatusRules(status.New(start, tmpl))
}

func (b *ruleBuilder) classes(name string, spec *config.ClassSpec) []korrel8r.Class {
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
				// A missing class is not a fatal error, because some domains (k8s)
				// may have different sets of classes (custom resources) for different clusters
				// Collect and log a report at end of configuration.
				b.missing.Add(name, spec.Domain+":"+class)
			} else {
				list.Append(c)
			}
		}
		return list.List
	} else { // Wildcard
		return d.Classes()
	}
}
