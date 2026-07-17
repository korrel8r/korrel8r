// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package api

import "github.com/getkin/kin-openapi/openapi3"

var (
	Spec     = func() *openapi3.T { s, err := GetSpec(); if err != nil { panic(err) }; return s }()
	BasePath = func() string { p, err := Spec.Servers.BasePath(); if err != nil { panic(err) }; return p }()
)
