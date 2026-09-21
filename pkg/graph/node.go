// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package graph

import (
	"fmt"
	"slices"
	"strings"

	"github.com/korrel8r/korrel8r/internal/pkg/json"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/result"
	"gonum.org/v1/gonum/graph/multi"
)

// Node is a graph-owned object representing a class. A search-owned node holds
// mutable results, queries and display attributes. Read-only topology algorithms
// use View's lightweight node IDs instead of these mutable objects.
// Data stores class identity (node ID and class), not Node objects.
// Node IDs are meaningful within the Data that assigned them.
type Node struct {
	multi.Node
	Attrs
	Class   korrel8r.Class
	Result  result.Result
	Queries Queries // Queries selecting objects in this class.
}

// Copy preserves identity but initializes fresh, empty mutable state.
// It does not copy accumulated results, queries or display attributes.
func (n *Node) Copy() *Node {
	return &Node{
		Node: n.Node, Class: n.Class,
		Attrs: Attrs{}, Result: result.New(n.Class), Queries: Queries{},
	}
}

func (n *Node) String(sorted bool) string {
	if n.Result == nil {
		return n.Class.String()
	}
	var result []string
	for _, o := range n.Result.List() {
		b, _ := json.Marshal(o)
		result = append(result, string(b))
	}
	if len(result) == 0 {
		return n.Class.String()
	}
	if sorted {
		slices.Sort(result)
	}
	return fmt.Sprintf("%v[%v]", n.Class, strings.Join(result, ","))
}
func (n *Node) DOTID() string { return n.Class.String() }
func (n *Node) Empty() bool   { return n.Result == nil || len(n.Result.List()) == 0 }
