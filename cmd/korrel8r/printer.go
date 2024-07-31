// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"sigs.k8s.io/yaml"
)

// printer prints in the format requested by --output
type printer struct{ Print func(any) }

func newPrinter(w io.Writer) printer {
	switch *outputFlag {

	case "json":
		return printer{Print: func(v any) {
			if b, err := json.Marshal(v); err != nil {
				fmt.Fprintf(w, "%v\n", err)
			} else {
				fmt.Fprintf(w, "%v\n", string(b))
			}
		}}

	case "json-pretty":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return printer{Print: func(v any) { must.Must(encoder.Encode(v)) }}

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
