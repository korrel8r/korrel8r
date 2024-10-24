// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package otel

// TODO investigate official OTEL Go packages for similar constants, use them if found.

// Constants for commonly used OTEL attribute names.
// Note: attribute names are an open set, here we define just those we need as literals in korrel8r code.
const (
	AttrServiceName = "service.name"
	AttrTraceName   = "trace.name"
)
