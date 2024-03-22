// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

//+kubebuilder:object:generate=true

// Package config contains configuration types for Korrel8r.
// These types can be loaded from YAML configuration files and/or used in a Kubernetes resource spec.
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
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/yaml"
)

var log = logging.Log()

// Configs is a map of config files by their source file/url.
type Configs map[string]*Config

// Load loads all configurations from a file or URL.
// If a configuration has an Include section, also loads all referenced configurations.
// Relative paths in Include are relative to the location of file containing them.
func Load(fileOrURL string) (Configs, error) {
	configs := Configs{}
	return configs, load(fileOrURL, configs)
}

func load(source string, configs Configs) (err error) {
	if _, ok := configs[source]; ok {
		return nil // Already loaded
	}
	log.V(2).Info("Loading configuration", "config", source)
	b, err := readFileOrURL(source)
	if err != nil {
		return fmt.Errorf("%v: %w", source, err)
	}
	c := &Config{}
	if err := yaml.Unmarshal(b, c); err != nil {
		return fmt.Errorf("%v: %w", source, err)
	}
	configs[source] = c
	for _, s := range c.Include {
		ref := resolve(source, s)
		if err := load(ref, configs); err != nil {
			return err
		}
	}
	return nil
}

// Apply normalized configurations to an engine.
func (configs Configs) Apply(e *engine.Engine) error {
	sources := maps.Keys(configs)
	slices.Sort(sources) // Predictable order
	groupMap := groupMap{}
	// Gather groupMap first, before interpreting rules.
	for _, source := range sources {
		c := configs[source]
		for _, g := range c.Aliases {
			if _, err := e.DomainErr(g.Domain); err != nil {
				return fmt.Errorf("%v: group %q: %w", source, g.Name, err)
			}
			if len(g.Classes) == 0 {
				return fmt.Errorf("%v: group %q: no classes", source, g.Name)
			}
			if !groupMap.Add(g) {
				return fmt.Errorf("%v: group %q: duplicate name", source, g.Name)
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
	for _, source := range sources {
		c := configs[source]
		for _, r := range c.Rules {
			r.Start.Classes = groupMap.Expand(r.Start.Domain, r.Start.Classes)
			r.Goal.Classes = groupMap.Expand(r.Goal.Domain, r.Goal.Classes)
			if err := addRules(e, r); err != nil {
				return fmt.Errorf("%v: %w", source, err)
			}
		}
		for _, sc := range c.Stores {
			sc = maps.Clone(sc)
			if err := e.AddStoreConfig(sc); err != nil {
				log.V(1).Error(err, "error configuring store", "config", source, "store", logging.JSON(sc))
			} else {
				log.V(1).Info("configured store", "config", source, "store", logging.JSON(sc))
			}
		}
	}
	return nil
}

// map of domain names to group names with class name lists
type groupMap map[string]map[string][]string

func (gm groupMap) Add(g Class) bool {
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
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("%v", http.StatusText(resp.StatusCode))
		}
		return b, nil
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
