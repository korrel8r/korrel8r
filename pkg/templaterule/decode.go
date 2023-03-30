package templaterule

import (
	"io"

	"github.com/korrel8r/korrel8r/pkg/engine"
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

// Decode template rules and add them to an engine.
func Decode(r io.Reader, e *engine.Engine) error {
	d := yaml.NewYAMLOrJSONDecoder(r, 1024)
	var rf RuleFile
	if err := d.Decode(&rf); err != nil {
		return err
	}
	groups := NewGroups(rf.Groups)
	for _, tr := range rf.Rules {
		tr.Start.Classes = groups.Expand(tr.Start.Classes)
		tr.Goal.Classes = groups.Expand(tr.Goal.Classes)
		krs, err := tr.Rules(e)
		if err != nil {
			return err
		}
		e.AddRules(krs...)
	}
	return nil
}
