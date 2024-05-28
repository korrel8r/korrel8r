// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"sigs.k8s.io/yaml"
)

// NewPrinter returns a function that prints in the chose format to writer.
func NewPrinter(format string, w io.Writer) func(any) {
	switch format {

	case "json":
		return func(v any) {
			if b, err := json.Marshal(v); err != nil {
				fmt.Fprintf(w, "%v\n", err)
			} else {
				fmt.Fprintf(w, "%v\n", string(b))
			}
		}

	case "json-pretty":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return func(v any) { must.Must(encoder.Encode(v)) }

	case "yaml":
		return func(v any) { fmt.Fprintf(w, "%s", must.Must1(yaml.Marshal(v))) }

	default:
		return func(v any) { fmt.Fprintf(w, "%v", v) }
	}
}
