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

type printer interface {
	korrel8r.Appender
	Print(any) // Print a single item.
	Close()
}

type jsonPrinter struct {
	appender
	*json.Encoder
}

func (p *jsonPrinter) Print(v any) { _ = p.Encode(v) }
func (p *jsonPrinter) Close()      { p.Print(p.appender) }

type ndJSONPrinter struct{ jsonPrinter }

func (p *ndJSONPrinter) Append(objs ...korrel8r.Object) {
	for _, o := range objs {
		p.Print(o)
	}
}
func (p *ndJSONPrinter) Close() {}

type yamlPrinter struct {
	io.Writer
	appender
}

func (p *yamlPrinter) Print(v any) { b, _ := yaml.Marshal(v); _, _ = p.Write(b) }
func (p *yamlPrinter) Close()      { p.Print(p.appender) }

func newPrinter(w io.Writer) printer {
	switch *outputFlag {
	case "json":
		return &jsonPrinter{Encoder: json.NewEncoder(w)}

	case "json-pretty":
		p := &jsonPrinter{Encoder: json.NewEncoder(w)}
		p.SetIndent("", "  ")
		return p

	case "ndjson":
		return &ndJSONPrinter{jsonPrinter{Encoder: json.NewEncoder(w)}}

	case "yaml":
		return &yamlPrinter{Writer: w}

	default:
		must.Must(fmt.Errorf("invalid output type: %v", *outputFlag))
		return nil
	}
}

type appender []korrel8r.Object

func (a *appender) Append(objects ...korrel8r.Object) { *a = append(*a, objects...) }
