// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules

import (
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// FIXME doc for domain-specific functions

// Funcs available for use in rule templates.
//
// As well as the functions listed below, rule templates can use the [slim-sprig] functions.
//
// Additional domain-specific functions are added by the [engine.Engine] for each domain loaded.
// Domain functions are prefixed with the domain name (e.g. k8sLogType), and are documented with the domain.
//
//	rule
//	  Returns the korrel8r.Rule being applied.
//	constraint
//	  Returns the korrel8r.Constraint in force when applying a rule. May be nil.
//	className CLASS
//	  Returns the fully qualified name of CLASS, with domain prefix.
//	ruleName RULE
//	  Returns the fully qualified name of RULE, with domain prefix.
//
// [Sprig]: https://go-task.github.io/slim-sprig/
var Funcs map[string]any

// FIXME functions moved to engine. need better doc.

func init() {
	Funcs = map[string]any{
		"rule":       func() korrel8r.Rule { return nil },
		"constraint": func() *korrel8r.Constraint { return nil },
	}
}
