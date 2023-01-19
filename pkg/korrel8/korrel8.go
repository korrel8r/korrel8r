// package korrel8 generic interfaces and algorithms to correlate objects between different domains.
//
// Each domain needs an implementation of the interfaces here.
package korrel8

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"time"

	"github.com/korrel8/korrel8/pkg/uri"
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
	String() string // Name of the class within the domain, e.g "Pod". See FullName()
}

// IDer is implemented by classes that have a meaningful identifier.
// If a Class implements IDer then SetResult will eliminate the second and subsequent objects with same ID.
type IDer interface {
	ID(Object) any // Comparable ID for de-duplication.
}

// ClassName is the qualified domain/name of a class, e.g. "k8s/Pod"
func ClassName(c Class) string { return path.Join(c.Domain().String(), c.String()) }

// Store is a source of signals belonging to a single domain.
type Store interface {
	// Get the objects selected by reference in this store.
	// Appends resulting objects to Appender.
	Get(context.Context, uri.Reference, Appender) error
	// Resolve a URI reference relative to the store's base URL.
	Resolve(uri.Reference) *url.URL
}

// Appender gathers results from Store.Get calls.
type Appender interface{ Append(...Object) }

// Rule for finding correlated objects.
// Rule implementations must be comparable.
type Rule interface {
	// Class of start object. If nil, this is a "wildcard" rule that can start from any class it applies to.
	Start() Class
	// Class of desired result object(s), must not be nil.
	Goal() Class
	// Name of the rule
	String() string
	// Apply the rule to a start Object, return a Reference for results.
	// Optional Constraint (if non-nil) is included in the Reference.
	Apply(start Object, constraint *Constraint) (uri.Reference, error)
}

// RuleName is the full rule name including start and goal classes.
func RuleName(r Rule) string {
	return fmt.Sprintf("%v(%v->%v)", r, ClassName(r.Start()), ClassName(r.Goal()))
}

// Constraint included in a reference to restrict the resulting objects.
type Constraint struct {
	Limit *uint      `json:"limit,omitempty"` // Max number of entries to return
	Start *time.Time `json:"start,omitempty"` // Include only results timestamped after this time.
	End   *time.Time `json:"end,omitempty"`   // Include only results timestamped before this time.
}

// TemplateFuncser may be implemented by Domain or Store to provide template helper functions.
// See text/template.Template.Funcs
type TemplateFuncser interface{ TemplateFuncs() map[string]any }

// RefConverter may be implemented by Domain or Store implementations that support console URLs.
type RefConverter interface {
	// RefConsoleToStore converts a console reference to an equivalent store reference.
	RefConsoleToStore(uri.Reference) (Class, uri.Reference, error)
	// RefStoreToConsole converts a store reference to an equivalent console reference.
	RefStoreToConsole(Class, uri.Reference) (uri.Reference, error)
}

// RefClasser may be implemented by Domain or Store implementations that can deduce the class from a reference.
type RefClasser interface {
	// RefClass gets the class of a URI reference.
	RefClass(uri.Reference) (Class, error)
}
