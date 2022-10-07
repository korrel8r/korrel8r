// package korrel8 generic interfaces and algorithms to correlate objects between different domains.
//
// Each domain needs an implementation of the interfaces here.
package korrel8

import (
	"context"
	"time"

	"github.com/alanconway/korrel8/internal/pkg/logging"
)

var log = logging.Log

// Object represents a signal instance. Values must support JSON marshal.
type Object any

// Domain is a collection of classes describing signals in the same family.
type Domain interface {
	String() string        // Name of the domain
	Class(string) Class    // Find a class by name, return nil if not found.
	KnownClasses() []Class // List of known classes in the Domain
}

// Class identifies a subset of objects from the same domain with the same schema.
// For example Pod is a class in the k8s domain.
//
// Class implementations must be comparable.
type Class interface {
	Contains(Object) bool          // True if object is in this class
	Domain() Domain                // Domain of this class.
	New() Object                   // Return a new instance of the class, for decoding from JSON.
	NewDeduplicator() Deduplicator // Deduplicates objects of this class.
	String() string                // Name of the class
}

// Deduplicator identifies duplicate values.
type Deduplicator interface {
	Unique(Object) bool // True if object is unique so far, remembers Object.
}

// Result gathers results from Store.Get calls.
type Result interface {
	Append(...Object)
}

// Store is a source of signals belonging to a single domain.
type Store interface {
	// Get executes one or more a Queries and appends objects to Result the resulting objects.
	Get(ctx context.Context, query string, r Result) error
}

// Rule encapsulates logic to find correlated goal objects from a start object.
//
type Rule interface {
	Start() Class   // Class of start object
	Goal() Class    // Class of desired result object(s)
	String() string // Name of the rule

	// Apply the rule to start Object.
	// Return a list of queries for correlated objects in the Goal() domain.
	// The queries include the contraint (which can be nil)
	Apply(Object, *Constraint) ([]string, error)
}

// Constraint to apply to the result of following a rule.
type Constraint struct {
	Start *time.Time // Include only results timestamped after this time.
	End   *time.Time // Include only results timestamped before this time.
}

// Path is a list of rules where the Goal() of each rule is the Start() of the next.
type Path []Rule

// Goal returns the goal of the last rule in the path, nil if the path is empty
func (p Path) Goal() Class {
	if len(p) == 0 {
		return nil
	}
	return p[len(p)-1].Goal()
}
