// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import "github.com/korrel8r/korrel8r/pkg/korrel8r"

// Compile time assertions: these will fail to compile if the argument type does not meet requirements.

type nothing *struct{}

// AssertDomain will fail to compile if its argument is not a valid korrel8r.Domain.
func AssertDomain(korrel8r.Domain) nothing { return nil }

// AssertClass will fail to compile if its argument is not a valid korrel8r.Class.
func AssertClass[T interface {
	korrel8r.Class
	comparable
}](T) nothing {
	return nil
}

// AssertQuery will fail to compile if its argument is not a valid korrel8r.Query.
func AssertQuery[T interface {
	korrel8r.Query
	comparable
}](T) nothing {
	return nil
}

// AssertRule will fail to compile if its argument is not a valid korrel8r.Rule.
func AssertRule[T interface {
	korrel8r.Rule
	comparable
}](T) nothing {
	return nil
}

// AssertStore will fail to compile if its argument is not a valid korrel8r.Store.
func StoreOK(korrel8r.Store) nothing { return nil }
