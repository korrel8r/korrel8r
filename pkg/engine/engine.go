// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package engine implements generic correlation logic to correlate across domains.
package engine

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/status"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var log = logging.Log()

// Engine manages a set of rules and stores to perform correlation.
// Searches use an immutable rule graph snapshot; rule updates are published atomically.
type Engine struct {
	// Set during construction; unchanged after the engine is built.
	domains             *korrel8r.Domains
	storeHolders        map[korrel8r.Domain]*storeHolders
	templateFuncs       template.FuncMap
	templateBase        *template.Template // Base template set with named templates from configuration.
	configs             config.Configs
	ruleFactories       []func(*korrel8r.Domains) []korrel8r.Rule
	statusRuleFactories []func(*korrel8r.Domains) []status.Rule
	growing             []korrel8r.GrowingDomain // Domains that can add classes.
	storeMetricAttrs    map[string][2]metric.MeasurementOption
	searchGuard         SearchGuard
	Tuning              config.Tuning

	// Rules are rebuilt when domains add classes
	mu           sync.Mutex // Serializes rebuilds.
	ruleState    atomic.Pointer[ruleState]
	classVersion atomic.Int64 // Class version used to build ruleState, see classVersionNow.
}

// SearchGuard registers active searches for cancellation by a process-wide guard.
type SearchGuard interface {
	Register(cancel func()) (unregister func())
}

// RegisterSearch registers a search with the configured guard.
func (e *Engine) RegisterSearch(cancel func()) func() {
	if e.searchGuard == nil {
		return func() {}
	}
	return e.searchGuard.Register(cancel)
}

// MemoryPressureLimitValue returns the configured value reported in a memory-pressure limit error.
func (e *Engine) MemoryPressureLimitValue() int {
	if e.Tuning.MemoryLimit > 0 {
		return int(e.Tuning.MemoryLimit)
	}
	return e.Tuning.GetMemoryPressureLimit()
}

func (e *Engine) Domain(name string) (korrel8r.Domain, error) { return e.domains.Domain(name) }
func (e *Engine) Domains() []korrel8r.Domain                  { return e.domains.List() }

// ClassMetricAttrs returns the immutable metric attributes for a class.
func (e *Engine) ClassMetricAttrs(c korrel8r.Class) metric.MeasurementOption {
	if a, ok := e.ruleState.Load().classMetrics[c]; ok {
		return a
	}
	return classMetricAttrs(c) // Not in the current rules, e.g. used by a search on old rules.
}

// RuleMetricAttrs returns the immutable metric attributes for a rule.
func (e *Engine) RuleMetricAttrs(r korrel8r.Rule) metric.MeasurementOption {
	if a, ok := e.ruleState.Load().ruleMetrics[r]; ok {
		return a
	}
	return ruleMetricAttrs(r) // Not in the current rules, e.g. used by a search on old rules.
}

func classMetricAttrs(c korrel8r.Class) metric.MeasurementOption {
	return metric.WithAttributes(attribute.String("domain", c.Domain().Name()), attribute.String("class", c.Name()))
}

func ruleMetricAttrs(r korrel8r.Rule) metric.MeasurementOption {
	return metric.WithAttributes(attribute.String("rule", r.Name()))
}

// StoreFor returns the aggregated store for a domain, nil if there is none.
func (e *Engine) StoreFor(d korrel8r.Domain) korrel8r.Store {
	s := e.storeHolders[d]
	if s == nil || len(s.stores) == 0 {
		return nil
	}
	return s
}

// StoresFor returns the list of individual stores for a domain.
func (e *Engine) StoresFor(d korrel8r.Domain) []korrel8r.Store {
	if ss := e.storeHolders[d]; ss != nil {
		return ss.Ensure()
	}
	return nil
}

// StoreConfigsFor returns the expanded store configurations and status.
func (e *Engine) StoreConfigsFor(d korrel8r.Domain) []config.Store {
	if ss, ok := e.storeHolders[d]; ok {
		return ss.Configs()
	}
	return nil
}

// Class parses a full class name and returns the [korrel8r.Class]
func (e *Engine) Class(fullname string) (korrel8r.Class, error) {
	d, c, err := korrel8r.ClassSplit(fullname)
	if err != nil {
		return nil, err
	}

	if c == "" {
		return nil, fmt.Errorf("invalid class name: %v", fullname)
	} else {
		return e.DomainClass(d, c)
	}
}

func (e *Engine) Classes(fullnames []string) ([]korrel8r.Class, error) {
	var classes []korrel8r.Class
	for _, name := range fullnames {
		c, err := e.Class(name)
		if err != nil {
			return nil, err
		}
		classes = append(classes, c)
	}
	return classes, nil
}

func (e *Engine) DomainClass(domain, class string) (korrel8r.Class, error) {
	d, err := e.Domain(domain)
	if err != nil {
		return nil, err
	}
	c := d.Class(class)
	if c == nil {
		return nil, korrel8r.NewClassNotFoundError(domain, class)
	}
	return c, nil
}

// Query parses a query string to a query object.
func (e *Engine) Query(query string) (korrel8r.Query, error) { return e.domains.Query(query) }

// Queries parses a slice of query strings and returns a slice of query objects.
func (e *Engine) Queries(queryStrings []string) ([]korrel8r.Query, error) {
	var queries []korrel8r.Query
	for _, q := range queryStrings {
		query, err := e.Query(q)
		if err != nil {
			return nil, err
		}
		queries = append(queries, query)
	}
	return queries, nil
}

// Rules returns the current rules, see [Engine.RefreshNewClasses].
func (e *Engine) Rules() []korrel8r.Rule { return slices.Clone(e.currentRules().rules) }

// Rule returns the named rule from the current rules, nil if not found.
func (e *Engine) Rule(name string) korrel8r.Rule { return e.currentRules().byName[name] }

// StatusRulesFor returns the status rules for the given class.
func (e *Engine) StatusRulesFor(c korrel8r.Class) []status.Rule {
	return e.currentRules().statuses[c.String()]
}

// GraphData returns the immutable graph data for the current rules.
// A rule refresh replaces the graph data, callers should hold on to the returned value
// for a consistent view of the rules.
func (e *Engine) GraphData() *graph.Data { return e.currentRules().data }

// Graph returns a read-only Gonum view directly over the immutable topology.
// It does not allocate or cache a mutable graph. Use GraphData for rule/line IDs.
func (e *Engine) Graph() graph.View { return e.GraphData().Graph() }

type ruleState struct {
	rules        []korrel8r.Rule
	byName       map[string]korrel8r.Rule
	data         *graph.Data
	classMetrics map[korrel8r.Class]metric.MeasurementOption
	ruleMetrics  map[korrel8r.Rule]metric.MeasurementOption
	statuses     map[string][]status.Rule // Keyed by class.String()
}

func newRuleState(rules []korrel8r.Rule, byName map[string]korrel8r.Rule, statuses map[string][]status.Rule) *ruleState {
	s := &ruleState{rules: rules, byName: byName, statuses: statuses, data: graph.NewData(rules...),
		ruleMetrics: make(map[korrel8r.Rule]metric.MeasurementOption, len(rules))}
	s.classMetrics = make(map[korrel8r.Class]metric.MeasurementOption, s.data.NodeCount())
	for id := range s.data.NodeCount() {
		c := s.data.Class(int64(id))
		s.classMetrics[c] = classMetricAttrs(c)
	}
	for _, r := range rules {
		s.ruleMetrics[r] = ruleMetricAttrs(r)
	}
	return s
}

// classVersionNow returns the sum of [korrel8r.GrowingDomain] class versions.
// Versions only increase, so the sum changes if any domain adds classes.
func (e *Engine) classVersionNow() (v int64) {
	for _, g := range e.growing {
		v += int64(g.ClassVersion())
	}
	return v
}

// currentRules returns the rule state, refreshed first if domains have added classes.
func (e *Engine) currentRules() *ruleState {
	_ = e.RefreshNewClasses() // Errors are logged, the previous rules remain in use.
	return e.ruleState.Load()
}

// rebuildRules rebuilds and publishes the rule state. Searches using the old state can finish on it.
func (e *Engine) rebuildRules() error {
	v := e.classVersionNow() // Before building, so classes added during the build cause another rebuild.
	state, err := e.buildRules()
	if err != nil {
		return err
	}
	e.ruleState.Store(state)
	e.classVersion.Store(v)
	return nil
}

// RefreshNewClasses rebuilds the rules if a [korrel8r.GrowingDomain] has added classes since the last build.
// Classes are never removed, so the new rules are a superset of the old.
// Methods that return rules call it automatically, it is cheap if there are no new classes.
//
// If the rebuild fails, the error is logged and returned, and the previous rules remain in use.
// The rebuild is not retried until more classes are added, so a bad rule doesn't fail every search.
func (e *Engine) RefreshNewClasses() error {
	if e.classVersionNow() == e.classVersion.Load() {
		return nil // Fast path: no new classes.
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	v := e.classVersionNow()
	if v == e.classVersion.Load() {
		return nil // Another goroutine rebuilt the rules.
	}
	log.V(2).Info("rebuilding rules for new domain classes")
	if err := e.rebuildRules(); err != nil {
		e.classVersion.Store(v) // Don't retry until classes change again.
		log.Error(err, "rule rebuild failed, using previous rules")
		return err
	}
	return nil
}

// Get results for query from all stores for the query domain.
func (e *Engine) Get(ctx context.Context, query korrel8r.Query, constraint *korrel8r.Constraint, result korrel8r.Appender) (err error) {
	count := 0
	constraint = constraint.Default()
	domain := query.Class().Domain().Name()
	ss := e.storeHolders[query.Class().Domain()]
	if len(ss.stores) == 0 {
		return fmt.Errorf("no stores found for domain %v", domain)
	}
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		statusIdx := 0
		if err != nil {
			statusIdx = 1
		}
		attrs := e.storeMetricAttrs[domain][statusIdx]
		metricStoreQueries.Add(ctx, 1, attrs)
		metricStoreQueryDuration.Record(ctx, latency.Seconds(), attrs)
		if err != nil {
			log.V(5).Info("Get failed", "error", err, "query", query, "constraint", constraint, "latency", latency)
		} else {
			log.V(5).Info("Get", "count", count, "query", query, "constraint", constraint, "latency", latency)
		}
	}()
	return ss.Get(ctx, query, constraint, korrel8r.AppenderFunc(func(o ...korrel8r.Object) {
		count += len(o)
		result.Append(o...)
	}))
}

// NewTemplate returns a template set up with options, funcs and named templates for this engine.
// See package documentation for more.
func (e *Engine) NewTemplate(name string) *template.Template { return e.templateBase.New(name) }

// execTemplate is a convenience to call NewTemplate, execute the template and stringify the result.
func (e *Engine) execTemplate(name, tmplString string, data any) (string, error) {
	tmpl, err := e.NewTemplate(name).Parse(tmplString)
	if err != nil {
		return "", err
	}
	w := &bytes.Buffer{}
	if err := tmpl.Execute(w, data); err != nil {
		return "", err
	}
	return w.String(), nil
}

// WithTimeout sets timeout on ctx, using configured default value if timeout == 0.
// If timeout == 0 and there is no default value, the returned context has no timeout.
func (e *Engine) WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, func()) {
	if timeout == 0 {
		timeout = time.Duration(e.Tuning.RequestTimeout)
	}
	if timeout == 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}
