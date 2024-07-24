// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package config

// Config defines the configuration for an instance of korrel8r.
// Configuration files may be JSON or YAML.
type Config struct {
	// Rules define the relationships that korrel8r will follow.
	Rules []Rule `json:"rules,omitempty"`

	// Aliases defines short names for groups of related classes.
	Aliases []Class `json:"aliases,omitempty"`

	// Stores is a list of store configurations.
	Stores []Store `json:"stores,omitempty"`

	// Include lists additional configuration files or URLs to include.
	Include []string `json:"include,omitempty"`
}

// Store is a map of name:value attributes used to connect to a store.
// The names and values depend on the type of store.
type Store map[string]string

// Store keys that may be used by any stores.
const (
	StoreKeyDomain     = "domain"               // Required domain name
	StoreKeyError      = "error"                // Error message if store failed to load.
	StoreKeyErrorCount = "errorCount"           // Count of errors on a store.
	StoreKeyMock       = "mockData"             // Store loads mock data from a file.
	StoreKeyCA         = "certificateAuthority" // Path to CA certificate.
)

// Rule configures a template rule.
//
// The rule template is applied to a instance of the start object.
// It should generate one of the following:
// - a goal query string of the form DOAMAIN:CLASS:QUERY_DATA.
// - a blank (whitespace-only) string if the rule does not apply to the given object.
// - an error if something unexpected goes wrong.
//
// If a rule returns an invalid query this will be logged as an error but will not prevent the
// progress on other rules. For expected conditions, returning blank generates less noise than an
// error.
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
