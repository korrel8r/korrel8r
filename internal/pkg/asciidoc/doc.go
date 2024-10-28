// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package asciidoc renders go/doc comments as asciidoc
//
// NOTE: the rest of this package comment is [fake] comment to _self-test_ the doc generator.
//
// This documentation is a fake domain, used to self-test the domain doc generator.
//
// # Class
//
// High class fakery.
//
// # Subclass
//
// This ^ is an implicit heading.
//
// # Query
//
// Queries are JSON-serialized instances of type [Query] and blah foobar
//
// # Object
//
// [Object] is a [regexp.Regexp]
//
// Here is
//
//	some
//	pre-formatted
//	code
//
// [fake]: http://fakery.example
package asciidoc

import "regexp"

// Query for fake signals.
type Query struct {
	Foo Foo // fooable
}

// Store for fake signals.
type Foo int

// Dummy type references.
type Object regexp.Regexp
