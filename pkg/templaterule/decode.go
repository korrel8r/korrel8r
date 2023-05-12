// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package templaterule

import (
	"io"
	"text/template"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type Group struct {
	Name    string   `json:"name"`
	Classes []string `json:"classes"`
}

type Groups map[string][]string

func NewGroups(gs []Group) Groups {
	groups := Groups{}
	for _, g := range gs {
		groups[g.Name] = groups.Expand(g.Classes)
	}
	return groups
}

func (g Groups) Expand(names []string) []string {
	var expanded []string
	for _, name := range names {
		if classes, ok := g[name]; ok {
			expanded = append(expanded, classes...)
		} else {
			expanded = append(expanded, name)
		}
	}
	return expanded
}

type RuleFile struct {
	Groups []Group
	Rules  []Rule
}

// Decode and expand template rules from a file.
func Decode(r io.Reader, domains map[string]korrel8r.Domain, funcs template.FuncMap) ([]korrel8r.Rule, error) {
	d := yaml.NewYAMLOrJSONDecoder(r, 1024)
	var rf RuleFile
	if err := d.Decode(&rf); err != nil {
		return nil, err
	}
	rules := make([]korrel8r.Rule, 0, len(rf.Rules))
	groups := NewGroups(rf.Groups)
	for _, tr := range rf.Rules {
		tr.Start.Classes = groups.Expand(tr.Start.Classes)
		tr.Goal.Classes = groups.Expand(tr.Goal.Classes)
		krs, err := tr.Rules(domains, funcs)
		if err != nil {
			return nil, err
		}
		rules = append(rules, krs...)
	}
	return rules, nil
}
