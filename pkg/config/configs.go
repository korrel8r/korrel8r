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

	"github.com/korrel8r/korrel8r/pkg/unique"
	"sigs.k8s.io/yaml"
)

// Load loads all configurations from a file or URL.
//
// If a configuration has an Include section, also loads all referenced configurations.
// Relative paths in Include are relative to the location of file containing them.
func Load(fileOrURL string) (Configs, error) {
	l := loader{loaded: unique.NewSet[string]()}
	if err := l.load(fileOrURL); err != nil {
		return nil, err
	}
	if err := expand(l.configs); err != nil {
		return nil, err
	}
	return l.configs, nil
}

type loader struct {
	loaded  unique.Set[string]
	configs Configs
}

// Expand aliases in all rules.
func expand(configs Configs) error {
	// Gather am first.
	am := aliasMap{}
	for i := range configs {
		c := &configs[i]
		for _, a := range c.Aliases {
			if len(a.Domain) == 0 {
				return fmt.Errorf("%v: alias %q: no domain", c.Source, a.Name)
			}
			if len(a.Classes) == 0 {
				return fmt.Errorf("%v: alias %q: no classes", c.Source, a.Name)
			}
			if !am.Add(a) {
				return fmt.Errorf("%v: alias %q: duplicate alias name", c.Source, a.Name)
			}
		}
		c.Aliases = nil // Erase unused aliases
	}
	// Expand aliases within aliases.
	for more := true; more; more = false {
		for domain, aliases := range am {
			for alias, classes := range aliases {
				n := len(classes)
				aliases[alias] = am.Expand(domain, classes)
				more = more || len(aliases[alias]) > n // Keep going till there are no more expansions
			}
		}
	}
	// Expand aliases in rules
	for _, c := range configs {
		for i := range c.Rules {
			r := &c.Rules[i]
			if r.Name == "" {
				return fmt.Errorf("rule has no name: %#+v", *r)
			}
			r.Start.Classes = am.Expand(r.Start.Domain, r.Start.Classes)
			r.Goal.Classes = am.Expand(r.Goal.Domain, r.Goal.Classes)
		}
	}

	return nil
}

func (l *loader) load(source string) error {
	if l.loaded.Has(source) {
		return nil // Already loaded
	}
	l.loaded.Add(source)
	b, err := readFileOrURL(source)
	if err != nil {
		return fmt.Errorf("%v: %w", source, err)
	}
	c := Config{Source: source}
	if err := yaml.UnmarshalStrict(b, &c); err != nil {
		return fmt.Errorf("%v: %w", source, err)
	}
	if len(l.configs) > 0 && c.Tuning != nil {
		return fmt.Errorf("unexpected tuning section in included configuration: %v", source)
	}
	l.configs = append(l.configs, c)
	for _, s := range c.Include {
		ref := resolve(source, s)
		if err := l.load(ref); err != nil {
			return err
		}
	}
	return nil
}

// map of domain names to alias names with class name lists
type aliasMap map[string]map[string][]string

func (am aliasMap) Add(c Class) bool {
	if am[c.Domain][c.Name] != nil {
		return false // Already present.
	}
	if am[c.Domain] == nil { // Create domain map if missing.
		am[c.Domain] = map[string][]string{}
	}
	am[c.Domain][c.Name] = c.Classes
	return true
}

func (am aliasMap) Expand(domain string, names []string) []string {
	aliases := am[domain]
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
		defer func() { _ = resp.Body.Close() }()
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
