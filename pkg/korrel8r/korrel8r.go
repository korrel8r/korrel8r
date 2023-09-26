// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package korrel8r generic interfaces and algorithms to correlate objects between different domains.
//
// Each domain needs an implementation of the interfaces here.
package korrel8r

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"strings"

	"sigs.k8s.io/yaml"
)

// Object represents an instance of a signal.
//
// Object has no methods to avoid clashes with fields or method names of the underlying object.
// The Class type provides some methods for inspecting objects.
// Object implementations must support JSON marshal and unmarshal.
type Object any

// Domain is a collection of classes describing signals in the same family.
type Domain interface {
	// Class finds  a class by name, return nil if not found.
	Class(string) Class
	// Classes returns a list of known classes in the Domain.
	Classes() []Class
	// Name returns the name of the domain
	Name() string
	// Query converts a query string to a Query object.
	Query(string) (Query, error)
	// Store returns a new store for this domain.
	Store(StoreConfig) (Store, error)
}

// StoreConfig name:value attributes to connect to a store.
type StoreConfig map[string]string

// Store keys that are used by all stores.
const (
	StoreKeyDomain = "domain" // Required domain name
	StoreKeyError  = "error"  // Provides error message if store failed to load.
)

// Class identifies a subset of objects from the same domain with the same schema.
// For example Pod is a class in the k8s domain.
//
// Class implementations must be comparable.
type Class interface {
	Domain() Domain // Domain of this class.
	New() Object    // Return a new instance of the class, can be unmarshaled from JSON.
	Name() string   // String name of the class within the domain.
}

// IDer is implemented by classes that have a meaningful identifier.
// Classes that implement IDer can be de-duplicated when collected in a Result.
type IDer interface {
	ID(Object) any // Comparable ID for de-duplication.
}

// Previewer is implemented by classes that can show a short "preview" string from the object.
// Could be a name or a message.
type Previewer interface {
	Preview(Object) string
}

// ClassName returns the fully qualified 'class.domain' name of a class, e.g. "Pod.v1.k8s"
func ClassName(c Class) string {
	if c == nil {
		return "<nil>"
	}
	name, domain := c.Name(), c.Domain().Name()
	if strings.HasSuffix(name, ".") {
		return name + domain
	}
	return fmt.Sprintf("%v.%v", name, domain)
}

// SplitClassName splits a fully qualified 'class.domain' name into class and domain.
// If there is no '.' treat the string as a domain name.
func SplitClassName(fullname string) (class, domain string) {
	i := strings.LastIndex(fullname, ".")
	if i < 0 {
		return "", fullname
	} else {
		return fullname[:i], fullname[i+1:]
	}
}

// Query is query for a subset of Objects in a Domain.
// A Store for the same domain can process the query.
type Query interface {
	// Class returned by this query.
	Class() Class
	// String form of the Query
	String() string
}

// Store is a source of signal Objects belonging to a single domain.
type Store interface {
	// Domain of the Store
	Domain() Domain

	// Get requests objects selected by the Query.
	// Collected objects are appended to the Appender.
	Get(context.Context, Query, Appender) error
}

// Constraint included in a reference to restrict the resulting objects.
type Constraint struct {
	Limit *uint      `json:"limit,omitempty"` // Max number of entries to return
	Start *time.Time `json:"start,omitempty"` // Include only results timestamped after this time.
	End   *time.Time `json:"end,omitempty"`   // Include only results timestamped before this time.
}

// Appender gathers results from Store.Get calls. See also Result.
type Appender interface{ Append(...Object) }

// Rule describes a relationship for finding correlated objects.
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

// RuleName returns a string including the rule name with full start and goal class names.
func RuleName(r Rule) string {
	return fmt.Sprintf("%v [%v]->[%v]", r.Name(), ClassName(r.Start()), ClassName(r.Goal()))
}

// JSONString returns the JSON marshaled string from v, or the error message if marshal fails
func JSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%q", err.Error())
	}
	return string(b)
}

func YAMLString(v any) string {
	b, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%q", err.Error())
	}
	return string(b)
}
