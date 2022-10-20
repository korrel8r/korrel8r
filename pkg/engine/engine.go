// package engine implements generic correlation logic to correlate across domains.
package engine

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/korrel8/korrel8/internal/pkg/decoder"
	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/korrel8/korrel8/pkg/graph"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/templaterule"
	"github.com/korrel8/korrel8/pkg/unique"
	"go.uber.org/multierr"
)

var (
	log   = logging.Log
	debug = log.V(2)
	warn  = log.V(1)
)

// Engine combines a set of domains and a set of rules, so it can perform correlation.
type Engine struct {
	Stores  map[string]korrel8.Store
	Domains map[string]korrel8.Domain
	Graph   *graph.Graph // Rules forms a directed graph, with korrel8.Class nodes and korrel8.Rule edges.
}

func New() *Engine {
	return &Engine{Stores: map[string]korrel8.Store{}, Domains: map[string]korrel8.Domain{}}
}

func (e *Engine) ParseClass(name string) (korrel8.Class, error) {
	parts := strings.SplitN(name, "/", 2)
	domain := e.Domains[parts[0]]
	if domain == nil {
		return nil, fmt.Errorf("unknown domain: %q", parts[0])
	}
	var cname string
	if len(parts) == 2 {
		cname = parts[1]
	}
	class := domain.Class(cname)
	if class == nil {
		return nil, fmt.Errorf("unknown class: %q", parts[1])
	}
	return class, nil
}

// AddDomain domain and corresponding store, s may be nil.
func (e *Engine) AddDomain(d korrel8.Domain, s korrel8.Store) {
	e.Domains[d.String()] = d
	e.Stores[d.String()] = s
}

// Follow rules in a path.
// Returns multiple queries if some rules in the path return multiple objects.
// May return queries and a multierr if there are some errors.
func (e Engine) Follow(ctx context.Context, start korrel8.Object, c *korrel8.Constraint, path []korrel8.Rule) (queries []korrel8.Query, err error) {
	// TODO multi-path following needs thought, reduce duplication.
	debug.Info("following path", "path", path)
	if err := e.Validate(path); err != nil {
		return nil, err
	}
	starters := []korrel8.Object{start}
	for i, rule := range path {
		debug.Info("following rule", "rule", rule, "starters", starters)
		queries, err = e.followEach(rule, starters, c)
		if err != nil {
			warn.Error(err, "ignored")
			continue
		}
		debug.Info("queries", "queries", queries)
		if i == len(path)-1 {
			break
		}
		d := rule.Goal().Domain()
		store := e.Stores[d.String()]
		if store == nil {
			warn.Info("no store", "domain", d)
			continue
		}
		var result korrel8.ListResult
		for _, q := range queries {
			if err := store.Get(ctx, q, &result); err != nil {
				warn.Error(err, "ignored")
			}
		}
		starters = result.List()
	}
	return unique.InPlace(queries, unique.Same[korrel8.Query]), err
}

// Validate checks that the Goal() of each rule matches the Start() of the next,
// and that the engine has all the stores needed to follow the path.
func (e Engine) Validate(path []korrel8.Rule) error {
	for i, r := range path {
		if i < len(path)-1 {
			if r.Goal() != path[i+1].Start() {
				return fmt.Errorf("invalid path, mismatched rues: %v, %v", r, path[i+1])
			}
			d := r.Goal().Domain()
			if _, ok := e.Stores[d.String()]; !ok {
				return fmt.Errorf("no store available for %v", d)
			}
		}
	}
	return nil
}

// FollowEach calls r.Apply() for each start object and collects the resulting queries.
// May return queries and a multierr if some rules fail to apply.
func (f Engine) followEach(rule korrel8.Rule, start []korrel8.Object, c *korrel8.Constraint) ([]korrel8.Query, error) {
	var (
		queries []korrel8.Query
		merr    error
	)
	for _, s := range start {
		q, err := rule.Apply(s, c)
		if err == nil && q != "" {
			queries = append(queries, q)
		}
		merr = multierr.Append(merr, err)
	}
	return unique.InPlace(queries, unique.Same[korrel8.Query]), merr
}

// Load rules from a file or walk a directory to find files.
func (e Engine) LoadRules(root string) error {
	var rules []korrel8.Rule
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) (reterr error) {
		defer func() {
			if reterr != nil { // Add file name to error
				reterr = fmt.Errorf("%v: %w", path, reterr)
			}
		}()
		ext := filepath.Ext(path)
		if err != nil || !d.Type().IsRegular() || (ext != ".yaml" && ext != ".yml" && ext != ".json") {
			return nil
		}
		debug.Info("loading rules", "path", path)
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		decoder := decoder.New(f)
		for {
			rule, err := templaterule.Decode(decoder, e.ParseClass)
			switch err {
			case nil:
				debug.Info("loaded rule", "rule", rule)
				rules = append(rules, rule)
			case io.EOF:
				return nil
			default:
				// Estimate number of lines
				return fmt.Errorf("line %v: %w", decoder.Line(), err)
			}
		}
	})
	e.Graph = graph.New(rules)
	return err
}
