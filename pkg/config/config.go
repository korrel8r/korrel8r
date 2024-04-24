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
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
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

// Apply configuration to an engine.Builder.
func (configs Configs) Apply(b *engine.Builder) error {
	sources := maps.Keys(configs)
	slices.Sort(sources) // Predictable order
	aliasMap := aliasMap{}

	// Gather aliasMap first, before interpreting rules.
	for _, source := range sources {
		c := configs[source]
		for _, a := range c.Aliases {
			if _, err := b.GetDomain(a.Domain); err != nil {
				return fmt.Errorf("%v: alias %q: %w", source, a.Name, err)
			}
			if len(a.Classes) == 0 {
				return fmt.Errorf("%v: alias %q: no classes", source, a.Name)
			}
			if !aliasMap.Add(a) {
				return fmt.Errorf("%v: alias %q: duplicate name", source, a.Name)
			}
		}
	}
	// Expand the aliases themselves.
	for more := true; more; more = false {
		for domain, aliases := range aliasMap {
			for alias, classes := range aliases {
				n := len(classes)
				aliases[alias] = aliasMap.Expand(domain, classes)
				more = more || len(aliases[alias]) > n // Keep going till there are no more expansions
			}
		}
	}
	// Add stores
	for _, source := range sources {
		c := configs[source]
		for _, sc := range c.Stores {
			sc = maps.Clone(sc)
			b.StoreConfigs(sc)
			if b.Err() != nil {
				log.V(1).Error(b.Err(), "Error configuring store", "config", source, "domain", sc[korrel8r.StoreKeyDomain])
			} else {
				log.V(1).Info("configured store", "config", source, "store", logging.JSON(sc))
			}
		}
	}
	// Add rules
	for _, source := range sources {
		c := configs[source]
		for _, r := range c.Rules {
			r.Start.Classes = aliasMap.Expand(r.Start.Domain, r.Start.Classes)
			r.Goal.Classes = aliasMap.Expand(r.Goal.Domain, r.Goal.Classes)
			rule, err := newRule(b, &r)
			if err != nil {
				return fmt.Errorf("%v: rule %v: %w", source, r.Name, err)
			}
			b.Rules(rule)
		}
	}
	return nil
}

// map of domain names to alias names with class name lists
type aliasMap map[string]map[string][]string

func (gm aliasMap) Add(g Class) bool {
	if gm[g.Domain][g.Name] != nil {
		return false // Already present, can't add.
	}
	if gm[g.Domain] == nil { // Create domain map if missing.
		gm[g.Domain] = map[string][]string{}
	}
	gm[g.Domain][g.Name] = g.Classes
	return true
}

func (gm aliasMap) Expand(domain string, names []string) []string {
	aliases := gm[domain]
	if aliases == nil {
		return names
	}
	var result []string
	for _, name := range names {
		if aliases[name] != nil {
			result = append(result, aliases[name]...)
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
