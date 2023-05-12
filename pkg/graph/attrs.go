// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import "gonum.org/v1/gonum/graph/encoding"

// Attributes for nodes and lines rendered by Graphviz.
type Attrs map[string]string

var (
	_ encoding.Attributer = Attrs{}
	_ encoding.Attributer = Node{}
	_ encoding.Attributer = Line{}
)

func (a Attrs) Attributes() (enc []encoding.Attribute) {
	for k, v := range a {
		enc = append(enc, encoding.Attribute{Key: k, Value: v})
	}
	return enc
}
