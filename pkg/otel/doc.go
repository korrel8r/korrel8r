// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package otel implements korrel8r-friendly OTEL attribute maps and attribute types.
//
// Korrel8r does not directly use/expose packages from [go.opentelemetry.io],
// the are not well suited to writing korrel8r template rules.
//
// Internally this package should use official [go.opentelemetry.io] packages where possible to
// better comply with the spec.
package otel
