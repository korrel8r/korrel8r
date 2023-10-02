// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package korrel8r contains interfaces and algorithms to correlate objects between different domains.
//
// A 'domain' is a package that implements the interfaces [Domain], [Class], [Store], [Query], [Object]
// There are some optional interfaces that can also be implemented if they are relevant to the domain.
//
// Rules for a domain should be added to https://github.com/korrel8r/korrel8r/blob/etc/korrel8/rules.
// Rules can express relationships within the new domain and link the new domain to existing domains and vice-versa.
//
// Once a domain and rules are available, korrel8r can:
// - apply rules linking the new domain with other domains.
// - execute queries in the new domain while traversing search graphs.
// - display correlated results in the new domain.
package korrel8r

import (
	"context"
	"time"
)

// Domain is a collection of classes describing signals in the same family.
//
// Must be implemented by a korrel8r domain.
type Domain interface {
	Class(string) Class               // Class finds a class by name, returns nil if not found.
	Classes() []Class                 // Classes returns a list of known classes in the Domain.
	Name() string                     // Name of the domain
	Description() string              // Description for human-readable documentation.
	Query(string) (Query, error)      // Query converts a query string to a Query object.
	Store(StoreConfig) (Store, error) // Store creates a new store for this domain.
}

// Class identifies a subset of objects with the same schema.
//
// For example, in the k8s domain Pod and Deployment are two separate classes.
// Some domains don't need multiple classes, for  example the metric domain has a single class 'metric'.
//
// The motivation for separate classes is separate schema to decode objects.
// For example, all metrics can be decoded into the same data structure, so there is only one metric class.
// In the k8s domain there is a separate class for each kind of resource, because there is a separate Go type
// to decode each kind of resource (Pod, Deployment etc.)
//
// Must be implemented by a korrel8r domain.
type Class interface {
	Domain() Domain      // Domain of this class.
	Name() string        // String name of the class within the domain.
	Description() string // Description for human-readable documentation.
	New() Object         // Return a blank instance of the class, can be unmarshaled from JSON.
}

// Store is a source of signal data that can be queried.
//
// Must be implemented by a korrel8r domain.
type Store interface {
	// Domain of the Store
	Domain() Domain

	// Get requests objects selected by the Query.
	// Collected objects are appended to the Appender.
	Get(context.Context, Query, Appender) error
}

// Query is a request that selects some subset of Objects from a Store.
//
// A query can only be used with a Store for the same domain as its class.
//
// Must be implemented by a korrel8r domain.
type Query interface {
	// Class returned by this query.
	Class() Class
	// String form of the Query
	String() string
}

// Object represents an instance of a signal.
//
// Object can be any Go type that supports JSON marshal/unmarshal.
// It does not require any special methods.
// It can be a simple string, a struct, or some more complicated API data structure.
// The goal is to allow values from some underlying toolkit to be used directly as Object.
//
// [Class] provides some methods for inspecting objects.
type Object any

// IDer is optionally implemented by Class implementations that have a meaningful unique identifier.
//
// Classes that implement IDer can be de-duplicated when collected in a Result.map1
type IDer interface {
	ID(Object) any // ID must be comparable for de-duplication.
}

// Previewer is optionally implemented by Class implementations to show a short "preview" string from the object.
//
// The preview could be a name, a message, or some other one-liner suitable for a human trying to preview the data.
// For example it might be shown in a pop-up box on a UI display.
type Previewer interface {
	Preview(Object) string
}

// StoreConfig is a map of name:value attributes used to connect to a store.
// The names and values depend on the type of store.
type StoreConfig map[string]string

// Store keys that may be used by any stores.
const (
	StoreKeyDomain = "domain" // Required domain name
	StoreKeyError  = "error"  // Error message if store failed to load.
)

// Constraint included in a reference to restrict the resulting objects.
type Constraint struct {
	Limit *uint      `json:"limit,omitempty"` // Max number of entries to return
	Start *time.Time `json:"start,omitempty"` // Include only results timestamped after this time.
	End   *time.Time `json:"end,omitempty"`   // Include only results timestamped before this time.
}

// Appender gathers results from Store.Get calls.
//
// Not required for a domain implementations: implemented by [Result]
type Appender interface{ Append(...Object) }

// Rule describes a relationship for finding correlated objects.
//
// Not required for a domain implementations: implemented by [github.com/korrel8r/korrel8r/pkg/rules]
type Rule interface {
	// Apply the rule to a start Object, return a Query for results.
	// Optional Constraint may be included in the Query.
	Apply(start Object, constraint *Constraint) (Query, error)
	// Class of start object. If nil, this is a "wildcard" rule that can start from any class it applies to.
	Start() Class
	// Class of desired result object(s), must not be nil.
	Goal() Class
	// Name of the rule
	Name() string
}
