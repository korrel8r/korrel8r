// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package traverse

import (
	"context"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// Traverser traverses a graph, filling in [Node.Result], [Node.Queries] and [Line.Queries].
// A korrel8r.Constraint can be set on the context if needed.
type Traverser interface {
	// Goals traverses all paths from start objects to all goal classes.
	Goals(ctx context.Context, start Start, goals []korrel8r.Class) (*graph.Graph, error)
	// Neighbours traverses to all neighbours of the start objects, traversing links up to the given depth.
	Neighbours(ctx context.Context, start Start, depth int) (*graph.Graph, error)
}

// New creates a default traverser, starting from objects/queries of class.
var New func(*engine.Engine, *graph.Graph) Traverser = NewAsync

// Start point information for graph traversal.
type Start struct {
	Class   korrel8r.Class    // Start class.
	Objects []korrel8r.Object // Start objects, must be of Start class.
	Queries []korrel8r.Query  // Queries for start objects, must be of Start class.
}

var log = logging.Log()
