// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package korrel8r contains interfaces and algorithms to correlate objects between different domains.
//
// A 'domain' is a package that implements the interfaces [Domain], [Class], [Store], [Query], [Object]
// There are some optional interfaces that can also be implemented if they are relevant to the domain.
//
// Rules can express relationships within or between domains.
//
// Once a domain and rules are available, korrel8r can:
//
//   - apply Rules linking the new domain with other domains.
//   - execute Queries in the new domain while traversing search graphs.
//   - display or return correlated results in the new domain.
package korrel8r

import (
	"context"
	"fmt"
)

// Domain is the entry-point to a package implementing a korrel8r domain.
// A korrel8r domain package must export a value called 'Domain' that implements this interface.
//
// Must be implemented by a korrel8r domain.
type Domain interface {
	Name() string                // Name of the domain. Domain names must not contain the character ':'.
	Description() string         // Description for human-readable documentation.
	Class(string) Class          // Class finds a class by name, returns nil if not found.
	Query(string) (Query, error) // Query parses a query string to a [Query] object.

	// Store creates a new store for this domain from a store configuration value.
	// Each domain defines its own store configuration.
	Store(config any) (Store, error)

	// Classes returns a list of classes in the Domain.
	//
	// Some domains have a variable set classes - the classes available may vary by store.
	// For example different k8s clusters can have different types of custom resource installed.
	// For such domains, [Classes] returns the list of classes that have been referenced so far -
	// for example by rules in a configuration file.
	Classes() []Class
}

// Class identifies a subset of objects with the same schema.
//
// For example, in the k8s domain Pod and Deployment are two separate classes.
// Some domains don't need multiple classes, for  example the metric domain has a single class 'metric'.
//
// The motivation for separate classes is separate schema to decode objects.
// For example, all metrics can be decoded into the same data structure, so there is only one metric class.
// In the k8s domain there is a separate class for each Kind of resource,
// because there is a separate Go type to decode each kind of resource (Pod, Deployment etc.)
//
// Must be implemented by a korrel8r domain.
type Class interface {
	Domain() Domain                   // Domain of this class.
	Name() string                     // Name of the class within the domain. Class names must not contain the character ':'.
	String() string                   // Fully qualified domain:class name
	Unmarshal([]byte) (Object, error) // Unmarshal a JSON-encoded object of this class.
}

// Store is a source of signal data that can be queried.
//
// Must be implemented by a korrel8r domain.
type Store interface {
	// Domain of the Store
	Domain() Domain

	// Get objects selected by the Query and append to the Appender.
	// If Constraint is non-nil, only objects satisfying the constraint are returned.
	// Note: a "not found" condition should give an empty result, it should not be reported as an error.
	Get(context.Context, Query, *Constraint, Appender) error
}

// ClassChecker may optionally be implemented by a [Store].
//
// Implemented when different store instances in the same domain can contain different sets of classes.
// Returns nil if the [Class] is recognized by the store, [ClassNotFoundError] if not.
// May return other errors if there are problems communicating with the store.
type ClassChecker interface{ ClassCheck(Class) error }

// ClassCheck calls [ClassChecker.ClassCheck] if s implements [ClassChecker], nil otherwise.
func ClassCheck(s Store, c Class) error {
	if r, ok := s.(ClassChecker); ok {
		return r.ClassCheck(c)
	}
	return nil
}

// Query is a request that selects some subset of Objects from a Store.
//
// A query can only be used with a Store for the same domain as its class.
//
// Must be implemented by a korrel8r domain.
type Query interface {
	// Class returned by this query.
	Class() Class
	// Data is the query data without the "DOMAIN:CLASS" prefix. Format depends on the domain.
	// Note the data part may contain any characters, including spaces.
	Data() string
	// String fully qualified DOMAIN:CLASS:DATA query string.
	String() string
}

// IDer is optionally implemented by Class implementations that have a meaningful unique identifier.
//
// Classes that implement IDer can be de-duplicated when collected in a Result.map
type IDer interface {
	ID(Object) any // ID must be a comparable type.
}

// GetID returns the object ID using if class is an IDer, "" otherwise.
func GetID(class Class, object Object) string {
	if ider, _ := class.(IDer); ider != nil {
		return fmt.Sprintf("%v", ider.ID(object))
	}
	return ""
}

// Object represents an instance of a signal.
//
// Object can be any Go type that supports JSON marshal/unmarshal.
// It does not have any special methods.
// It can be a simple string, a struct, or some more complicated API data structure.
// The goal is to allow values from some underlying toolkit to be used directly as Object.
//
// [Class] provides some methods for inspecting objects.
//
// Must be implemented by a korrel8r domain.
type Object = any

// Previewer is optionally implemented by Class implementations to show a short "preview" string from the object.
//
// The preview could be a name, a message, or some other one-liner suitable for a human trying to preview the data.
// For example it might be shown in a pop-up box on a UI display.
type Previewer interface {
	Preview(Object) string
}

// Appender gathers results from Store.Get calls.
//
// Not required for a domain implementations: implemented by [Result]
type Appender interface{ Append(...Object) }

// AppenderFunc adapts a function as an Appender. AppenderFunc(f).Append(object...) calls f for each object.
type AppenderFunc func(Object)

func (f AppenderFunc) Append(objects ...Object) {
	for _, o := range objects {
		f(o)
	}
}

var _ Appender = AppenderFunc(nil) // AppenderFunc implements appender

// Rule describes a relationship for finding correlated objects.
// Rule.Apply() generates correlated queries from start objects.
//
// Not required for a domain implementations.
// Template-based rules are implemented by [github.com/korrel8r/korrel8r/pkg/rules]
// Rules types must be comparable.
type Rule interface {
	// Apply the rule to a start Object, return a Query for results.
	Apply(start Object) (Query, error)
	// Start returns a list of start classes that the rule can apply to, all in the same start domain.
	Start() []Class
	// Goal returns a list of goal classes that may result from this rule, all in the same goal domain.
	Goal() []Class
	// Name is a short, unique, human-readable name to identify the rule.
	Name() string
}

// NameSeparator used in DOMAIN:CLASS and DOMAIN:CLASS:QUERY strings.
const NameSeparator = ":"
