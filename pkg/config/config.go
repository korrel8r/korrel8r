// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package config

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
	"strings"
)

var log = logging.Log()

// Configs is a map of config files by their source file/url.
type Configs map[string]*Config

// Load loads all configurations from a file or URL.
// If a configuration has a "More" section, also loads all referenced configurations.
// Relative paths in More are relative to the location of file containing them.
func Load(fileOrURL string) (Configs, error) {
	configs := Configs{}
	return configs, load(fileOrURL, configs)
}

func load(source string, configs Configs) (err error) {
	if _, ok := configs[source]; ok {
		return nil // Already loaded
	}
	b, err := readFileOrURL(source)
	if err != nil {
		return fmt.Errorf("%v: %w", source, err)
	}
	c := &Config{}
	if err := yaml.Unmarshal(b, c); err != nil {
		return fmt.Errorf("%v: %w", source, err)
	}
	configs[source] = c
	for _, s := range c.More {
		if err := load(resolve(source, s), configs); err != nil {
			return err
		}
	}
	return nil
}

// StoreError is an error encountered when trying to add a store configuration.
type StoreError struct {
	Store korrel8r.StoreConfig
	Err error
}
func (se StoreError)  Error() string {	return se.Err.Error() }

// Status is the error state after attempting to apply a configuration.
type Status struct {
	Err error
	StoreErrs []StoreError
}

func (s *Status) Error() string {
	if s.Err != nil {
		return s.Err.Error()
	}
	b := &strings.Builder{}
	for _, se := range s.StoreErrs {
		fmt.Fprintln(b, se.Err.Error())
	}
	return b.String()
}

// Apply normalized configurations to an engine.
// Returns a list of errors for stores that could not be configured.
// If the returned error is non-nil, it wil be of type *Status.
func (configs Configs) Apply(e *engine.Engine) (err error) {
	sources := maps.Keys(configs)
	slices.Sort(sources) // Predictable order
	groupMap := groupMap{}
	// Gather groupMap first, before interpreting rules.
	for _, source := range sources {
		c := configs[source]
		for _, g := range c.Groups {
			if _, err := e.DomainErr(g.Domain); err != nil {
				return &Status{Err: fmt.Errorf("%v: group %q: %w", source, g.Name, err)}
			}
			if len(g.Classes) == 0 {
				return &Status{Err:fmt.Errorf("%v: group %q: no classes", source, g.Name)}
			}
			if !groupMap.Add(g) {
				return &Status{Err: fmt.Errorf("%v: group %q: duplicate name", source, g.Name)}
			}
		}
	}
	// Expand the groups themselves.
	for more := true; more; more = false {
		for domain, groups := range groupMap {
			for group, classes := range groups {
				n := len(classes)
				groups[group] = groupMap.Expand(domain, classes)
				more = more || len(groups[group]) > n // Keep going till there are no more expansions
			}
		}
	}
	// Add rules and stores to the engine
	var badStores []StoreError
	for _, source := range sources {
		c := configs[source]
		for _, r := range c.Rules {
			r.Start.Classes = groupMap.Expand(r.Start.Domain, r.Start.Classes)
			r.Goal.Classes = groupMap.Expand(r.Goal.Domain, r.Goal.Classes)
			if err := addRules(e, r); err != nil {
				return &Status{ Err:fmt.Errorf("%v: %w", source, err)}
			}
		}
		for _, sc := range c.Stores {
			if err := e.AddStoreConfig(sc); err != nil {
				log.Error(err, "error configuring store", "store", logging.JSON(sc))
				badStores = append(badStores, StoreError{Store: sc, Err: fmt.Errorf("%v: %w", source, err)})
			} else {
				log.V(2).Info("configured store", "source", source, "store", logging.JSON(sc))
			}
		}
	}
	if badStores != nil {
		return &Status{StoreErrs: badStores}
	}
	return nil
}

// map of domain names to group names with class name lists
type groupMap map[string]map[string][]string

func (gm groupMap) Add(g Group) bool {
	if gm[g.Domain][g.Name] != nil {
		return false // Already present, can't add.
	}
	if gm[g.Domain] == nil { // Create domain map if missing.
		gm[g.Domain] = map[string][]string{}
	}
	gm[g.Domain][g.Name] = g.Classes
	return true
}

func (gm groupMap) Expand(domain string, names []string) []string {
	groups := gm[domain]
	if groups == nil {
		return names
	}
	var result []string
	for _, name := range names {
		if groups[name] != nil {
			result = append(result, groups[name]...)
		} else {
			result = append(result, name)
		}
	}
	return result
}

func readFileOrURL(source string) ([]byte, error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, err
	}
	if u.IsAbs() {
		resp, err := http.Get(u.String())
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	} else {
		return os.ReadFile(u.Path)
	}
}

func resolve(base, ref string) string {
	if filepath.IsAbs(ref) {
		return ref
	}
	if r, err := url.Parse(ref); err == nil {
		if r.IsAbs() {
			return ref
		}
		if b, err := url.Parse(base); err == nil && b.IsAbs() {
			return b.ResolveReference(r).String()
		}
	}
	return filepath.Join(filepath.Dir(base), ref)
}

// Store provides getters/setters for common store config map keys.
type Store korrel8r.StoreConfig

func (m Store) K8sClient() (client.Client, *rest.Config, error) {
	cfg, err := config.GetConfigWithContext(m["context"])
	if err != nil {
		return nil, nil, err
	}
	// Reduce client-side throttling for rapid results.
	cfg.QPS = 100
	cfg.Burst = 1000
	if err != nil {
		return nil, nil, err
	}
	c, err := client.New(cfg, client.Options{})
	return c, cfg, err
}
