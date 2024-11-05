// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"sigs.k8s.io/yaml"
)

// printer prints in the format requested by --output
type printer struct{ Print func(any) }

func newPrinter(w io.Writer) printer {
	switch *outputFlag {

	case "json":
		encoder := json.NewEncoder(w)
		return printer{Print: func(v any) { must.Must(encoder.Encode(v)) }}

	case "json-pretty":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return printer{Print: func(v any) { must.Must(encoder.Encode(v)) }}

	case "ndjson":
		encoder := json.NewEncoder(w)
		return printer{Print: func(v any) {
			r := reflect.ValueOf(v)
			switch r.Kind() {
			case reflect.Array, reflect.Slice:
				for i := 0; i < r.Len(); i++ {
					must.Must(encoder.Encode(r.Index(i).Interface()))
					fmt.Fprintln(w, "")
				}
			default:
				must.Must(encoder.Encode(v))
			}
		}}

	case "yaml":
		return printer{Print: func(v any) { fmt.Fprintf(w, "---\n%s", must.Must1(yaml.Marshal(v))) }}

	default:
		must.Must(fmt.Errorf("invalid output type: %v", *outputFlag))
		return printer{}
	}
}

func (p printer) Append(objects ...korrel8r.Object) {
	for _, o := range objects {
		p.Print(o)
	}
}
