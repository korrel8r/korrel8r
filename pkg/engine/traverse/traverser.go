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
type Traverser interface {
	// Goals traverses all paths all start objects to all goal classes.
	Goals(ctx context.Context, goals []korrel8r.Class) (*graph.Graph, error)
	// Neighbours traverses to all neighbours of the start objects, up to a distance of n lines.
	Neighbours(ctx context.Context, n int) (*graph.Graph, error)
}

// New creates a default traverser, starting from objects/queries of class.
var New func(*engine.Engine, *graph.Graph, korrel8r.Class, []korrel8r.Object, []korrel8r.Query) Traverser = NewSync

type traverser struct {
	engine  *engine.Engine
	graph   *graph.Graph
	start   korrel8r.Class
	objects []korrel8r.Object
	queries []korrel8r.Query
}

func newTraverser(e *engine.Engine, g *graph.Graph, start korrel8r.Class, objects []korrel8r.Object, queries []korrel8r.Query) *traverser {
	return &traverser{
		engine:  e,
		graph:   g,
		start:   start,
		objects: objects,
		queries: queries,
	}
}

var log = logging.Log()
