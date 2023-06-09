package api

import "github.com/korrel8r/korrel8r/pkg/korrel8r"

// Config defines the configuration for an instance of korrel8r.
// Configuration files may be JSON or YAML.
type Config struct {
	// Rules define the relationships that korrel8r will follow.
	Rules []Rule `json:"rules,omitempty"`

	// Groups defines short names for groups of related classes.
	Groups []Group `json:"groups,omitempty"`

	// Domains is a map of domain names to stores.
	Domains map[string][]korrel8r.StoreConfig `json:"domains,omitempty"`

	// More is a list of file names or URLs for additional configuration files to load.
	More []string `json:"more,omitempty"`
}

// Rule specifies a template rule.
// It generates one or more korrel8r.Rule.
type Rule struct {
	// Name is a short, descriptive name.
	// If omitted, a name is generated from Start and Goal.
	Name string `json:"name,omitempty"`

	// Start specifies the set of classes that this rule can apply to.
	Start ClassSpec `json:"start"`

	// Goal specifies the set of classes that this rule can produce.
	Goal ClassSpec `json:"goal"`

	// TemplateResult contains templates to generate the result of applying this rule.
	// Each template is applied to an object from one of the `start` classes.
	// If any template yields a blank string or an error, the rule does not apply.
	Result ResultSpec `json:"result"`
}

// ClassSpec specifies one or more classes.
type ClassSpec struct {
	// Domain is the domain for selected classes.
	Domain string `json:"domain"`

	// Classes is a list of class names to be selected from the domain.
	// If absent, all classes in the domain are selected.
	Classes []string `json:"classes,omitempty"`
}

// ResultSpec contains templates to generate a result.
type ResultSpec struct {
	// Query template generates a query object suitable for the goal store.
	Query string `json:"query"`

	// Constraint template is optional, it generates a korrel8r.Constraint in JSON form.
	// This constraint is combined with the constraint already in force, if there is one.
	// See Constraint.Combine
	Constraint string `json:"constraint,omitempty"`
}

// Group of similar classes that can be referred to by a short name in a ClassSpec.
type Group struct {
	// Name is the short name for a group of classes.
	Name string `json:"name"`
	// Domain of the classes, all must be in the same domain.
	Domain string `json:"domain"`
	// Classes are the names of classes in this group.
	Classes []string `json:"classes"`
}

// StoreConfig name:value attributes to connect to a store.
type StoreConfig = korrel8r.StoreConfig
