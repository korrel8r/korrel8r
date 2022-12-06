// package korrel8 generic interfaces and algorithms to correlate objects between different domains.
//
// Each domain needs an implementation of the interfaces here.
package korrel8

import (
	"context"
	"net/url"
	"path"
	"time"
)

// Object represents an instance of a signal.
//
// Object has no methods to avoid clashes with fields or method names of the underlying object.
// The Class type provides some methods for inspecting objects.
// Object implementations MUST be pointers and MUST support JSON marshal/unmarshal.
type Object any

// Domain is a collection of classes describing signals in the same family.
//
// Domain implementations must be comparable.
type Domain interface {
	String() string     // Name of the domain
	Class(string) Class // Find a class by name, return nil if not found.
	Classes() []Class   // List of known classes in the Domain
}

// Class identifies a subset of objects from the same domain with the same schema.
//
// For example Pod is a class in the k8s domain.
// Class implementations must be comparable.
type Class interface {
	Domain() Domain // Domain of this class.
	New() Object    // Return a new instance of the class, can be unmarshaled from JSON.
	Key(Object) any // Comparable key for de-duplication or nil if object is not in this class.
	String() string // Name of the class within the domain, e.g "Pod". See FullName()
}

// FullName is the qualified domain/name of a class, e.g. "k8s/Pod"
func FullName(c Class) string { return path.Join(c.Domain().String(), c.String()) }

// Store is a source of signals belonging to a single domain.
type Store interface {
	// Get the objects selected by query in this store.
	// Appends resulting objects to Result.
	Get(ctx context.Context, query *Query, result Result) error

	// URL resolves a relative Query URI to a full URL for this store.
	URL(query *Query) *url.URL
}

// Query is a relative URI - a URL with only path and query.
// A Store will combine it with its base URL to get a full REST URL.
type Query = url.URL

// Result gathers results from Store.Get calls.
// See ListResult and SetResult.
type Result interface{ Append(...Object) }

// Rule for finding correlated objects.
// Rule implementations must be comparable.
type Rule interface {
	// Class of start object. If nil, this is a "wildcard" rule that can start from any class it applies to.
	Start() Class
	// Class of desired result object(s), must not be nil.
	Goal() Class
	// Name of the rule
	String() string
	// Apply the rule to a start Object, return a Query for results.
	// Optional Constraint (if non-nil) is included in the Query.
	//
	// FIXME: May optionally return a Constraint to be used by the next rule in the chain.
	Apply(start Object, constraint *Constraint) (*Query, error)
}

// Constraint included in a query to restrict the resulting objects.
type Constraint struct {
	Limit *uint      `json:"limit,omitempty"` // Max number of entries to return
	Start *time.Time `json:"start,omitempty"` // Include only results timestamped after this time.
	End   *time.Time `json:"end,omitempty"`   // Include only results timestamped before this time.
}

// Combine this constraint with a new constraint.
// Values that are not set in the original constraint
func (c *Constraint) Combine(other *Constraint) {
	panic("FIXME constraint")
}
