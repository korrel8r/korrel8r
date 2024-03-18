// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package config

import "github.com/korrel8r/korrel8r/pkg/korrel8r"

// Config defines the configuration for an instance of korrel8r.
// Configuration files may be JSON or YAML.
type Config struct {
	// Rules define the relationships that korrel8r will follow.
	Rules []Rule `json:"rules,omitempty"`

	// Aliases defines short names for groups of related classes.
	Aliases []Class `json:"aliases,omitempty"`

	// Stores is a list of store configurations.
	Stores []korrel8r.StoreConfig `json:"stores,omitempty"`

	// Include lists additional configuration files or URLs to include.
	Include []string `json:"include,omitempty"`
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
}

// Class defines a shortcut name for a set of existing classes.
type Class struct {
	// Name is the short name for a group of classes.
	Name string `json:"name"`
	// Domain of the classes, all must be in the same domain.
	Domain string `json:"domain"`
	// Classes are the names of classes in this group.
	Classes []string `json:"classes"`
}
