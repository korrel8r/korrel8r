// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package quickrules

import "github.com/korrel8r/korrel8r/pkg/rules"

var (
	Empty      = rules.Empty
	Fail       = rules.Fail
	FailErr    = rules.FailErr
	RequireAll = rules.RequireAll
	ToJSON     = rules.ToJSON
)

func Require[T any](v T) T      { return rules.Require(v) }
func Default[T any](dflt, v T) T { return rules.Default(dflt, v) }
