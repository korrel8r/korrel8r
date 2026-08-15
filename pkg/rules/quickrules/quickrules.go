// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

//go:generate qtc --dir .

package quickrules

import (
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/yaml"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rules"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

var logger = logging.Log()

//go:embed *.qtpl
var qtplFiles embed.FS

// applyFuncs maps rule names to the generated stream functions that render them.
// It is generated from the *.qtpl {% func %} declarations by hack/gen-applyfuncs.sh,
// see applyfuncs.go.

// ruleSpec is the registry metadata of one compiled rule, from its annotation in *.qtpl.
// It uses the same schema as a [github.com/korrel8r/korrel8r/pkg/config] rule entry.
type ruleSpec struct {
	Name  string    `json:"name"`
	Start classSpec `json:"start"`
	Goal  classSpec `json:"goal"`
}

// classSpec specifies one or more classes, as in a configuration-file rule.
type classSpec struct {
	Domain  string   `json:"domain"`
	Classes []string `json:"classes"`
}

// Rules returns the pre-compiled rules, using d to resolve classes and parse generated queries.
// The rule list and its metadata come from the annotations in the embedded *.qtpl source;
// see the package documentation.
func Rules(d *korrel8r.Domains) []korrel8r.Rule {
	var result []korrel8r.Rule
	entries, err := fs.ReadDir(qtplFiles, ".")
	if err != nil {
		panic(fmt.Errorf("compiled rules: %w", err))
	}

	for _, entry := range entries {
		qtpl, err := fs.ReadFile(qtplFiles, entry.Name())
		if err != nil {
			panic(fmt.Errorf("compiled rules: file %q: %w", entry.Name(), err))
		}
		specs, err := parseRuleAnnotations(string(qtpl))
		if err != nil {
			panic(fmt.Errorf("compiled rules: file %q: %w", entry.Name(), err))
		}
		for _, spec := range specs {
			r, err := newRule(d, spec)
			if err != nil {
				logger.V(1).Info("skipped compiled rule", "rule", spec.Name, "error", err)
				continue
			}
			if r != nil {
				result = append(result, r)
			}
		}
	}
	return result
}

// funcDecl matches a quicktemplate func declaration such as `{% func MetricToPod(o interface{}) %}`.
var funcDecl = regexp.MustCompile(`\{\%\s*func\s+(\w+)\s*\(`)

// parseRuleAnnotations scans a quicktemplate source file for rule metadata and returns it in
// source order. The bare text between two template-tag blocks is a single YAML rule descriptor:
// each {% func %} is preceded by the YAML that describes it. It also verifies that every rule
// annotation matches its {% func %} name.
func parseRuleAnnotations(src string) ([]*ruleSpec, error) {
	var (
		specs  []*ruleSpec
		seen   = map[string]bool{}
		buf    []string
		inFunc bool
	)
	add := func(spec *ruleSpec) error {
		if seen[spec.Name] {
			return fmt.Errorf("duplicate rule annotation %q", spec.Name)
		}
		seen[spec.Name] = true
		specs = append(specs, spec)
		return nil
	}
	for line := range strings.SplitSeq(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "{%") {
			if m := funcDecl.FindStringSubmatch(line); m != nil {
				name := m[1]
				spec, err := parseRule(buf)
				if err != nil {
					return nil, err
				}
				if spec == nil {
					return nil, fmt.Errorf("rule %q: no rule annotation, add YAML above its function", name)
				}
				if spec.Name != name {
					return nil, fmt.Errorf("rule annotation %q does not match function name %q", spec.Name, name)
				}
				if err := add(spec); err != nil {
					return nil, err
				}
			}
			if strings.Contains(line, "{% func") {
				inFunc = true
			}
			if strings.Contains(line, "{% endfunc") {
				inFunc = false
			}
			buf = nil // Start a new YAML descriptor after this template block.
			continue
		}
		if !inFunc {
			buf = append(buf, line)
		}
	}
	// Bare text after the last template block describes a rule with no function to implement it.
	if spec, err := parseRule(buf); err != nil {
		return nil, err
	} else if spec != nil {
		return nil, fmt.Errorf("rule %q: annotation has no matching function declaration", spec.Name)
	}
	return specs, nil
}

// parseRule parses the YAML text preceding a rule function into a ruleSpec.
// It returns nil if the text is blank (no annotation present).
func parseRule(buf []string) (*ruleSpec, error) {
	text := strings.Join(buf, "\n")
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	var spec ruleSpec
	if err := yaml.Unmarshal([]byte(text), &spec); err != nil {
		return nil, fmt.Errorf("invalid rule annotation: %w", err)
	}
	if spec.Name == "" {
		return nil, fmt.Errorf("rule annotation has no name")
	}
	if spec.Start.Domain == "" || spec.Goal.Domain == "" {
		return nil, fmt.Errorf("rule %q: annotation must specify start and goal domains", spec.Name)
	}
	return &spec, nil
}

// newRule constructs a korrel8r.Rule for a compiled rule, resolving its classes and applying the
// generated stream function registered for it in applyFuncs.
// It returns (nil, nil) if the rule has no usable start or goal classes, so the rule is skipped.
func newRule(d *korrel8r.Domains, spec *ruleSpec) (korrel8r.Rule, error) {
	start, missingStart := classes(d, spec.Name, &spec.Start)
	goal, missingGoal := classes(d, spec.Name, &spec.Goal)
	if len(start) == 0 || len(goal) == 0 {
		logger.V(1).Info("skipped compiled rule: no known start or goal classes", "rule", spec.Name,
			"missingClasses", append(missingStart, missingGoal...))
		return nil, nil
	}
	apply, ok := applyFuncs[spec.Name]
	if !ok {
		return nil, fmt.Errorf("no apply function registered for rule %q", spec.Name)
	}
	return rules.NewQuickRule(spec.Name, start, goal, apply, d), nil
}

// classes resolves the classes of a classSpec to korrel8r.Class.
// If no classes are listed, all classes in the domain are used.
// Unknown classes are skipped and returned in the second result, so callers can report them;
// an unknown domain reports all classes as missing.
func classes(d *korrel8r.Domains, name string, spec *classSpec) (found []korrel8r.Class, missing []string) {
	if len(spec.Classes) == 0 {
		dom, err := d.Domain(spec.Domain)
		if err != nil {
			logger.V(1).Info("skipped class: unknown domain", "rule", name, "domain", spec.Domain)
			return nil, nil
		}
		return dom.Classes(), nil
	}
	result := unique.Set[korrel8r.Class]{}
	for _, c := range spec.Classes {
		k, err := d.DomainClass(spec.Domain, c)
		if err != nil {
			missing = append(missing, spec.Domain+":"+c)
			continue
		}
		result.Add(k)
	}
	return result.List(), missing
}
