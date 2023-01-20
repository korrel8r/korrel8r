package webui

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/uri"
	"go.uber.org/multierr"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/topo"
)

// correlateFields are fields used by the page template.
type correlateFields struct {
	// Input fields from query parameters
	Start string // Starting point, console URL
	Goal  string // Starting class full name.

	// Computed fields for display
	StartClass, GoalClass korrel8r.Class
	StartRef              uri.Reference
	StartObjects          []korrel8r.Object
	Results               *engine.Results
	Diagram               string
	Topo                  []korrel8r.Class
	Err                   error
	FollowErr             error
}

// addErr adds an error to the list. Display as much information as possible even with errors.
func (f *correlateFields) addErr(err error, msg ...any) bool {
	if len(msg) > 0 && err != nil {
		err = fmt.Errorf(msg[0].(string), msg[1:])
	}
	return multierr.AppendInto(&f.Err, err)
}

// Reset the fields to contain only URL query parameters
func (f *correlateFields) Reset(params url.Values) {
	*f = correlateFields{
		Start: params.Get("start"),
		Goal:  params.Get("goal"),
	}
}

// Params returns the URL query parameters from form fields
func (f *correlateFields) Params() url.Values {
	v := url.Values{}
	v.Set("start", f.Start)
	v.Set("goal", f.Goal)
	return v
}

type correlateHandler struct {
	ui *WebUI
	f  correlateFields
}

func (h *correlateHandler) refToConsole(c korrel8r.Class, r uri.Reference) *url.URL {
	cref, err := h.ui.Rewriter.RefStoreToConsoleURL(c, r)
	if err != nil {
		cref = &url.URL{Path: "/error", RawQuery: uri.Values{"err": []string{err.Error()}}.Encode()}
	}
	return cref
}

func (h *correlateHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.V(4).Info("correlation page")
	defer req.Body.Close()
	h.update(req)
	if h.f.Err != nil {
		log.Error(h.f.Err, "page errors")
	}
	t, err := h.ui.Page("correlate").Funcs(map[string]any{"refToConsole": h.refToConsole}).Parse(correlateTemplate)
	if !httpError(w, err, http.StatusInternalServerError) {
		httpError(w, t.Execute(w, h.f), http.StatusInternalServerError)
	}
}

func (h *correlateHandler) update(req *http.Request) {
	h.f.Reset(req.URL.Query())
	addErr := h.f.addErr

	start, err := uri.Parse(h.f.Start) // Console URL
	if !addErr(err) {
		h.f.StartClass, h.f.StartRef, err = h.ui.Rewriter.RefConsoleToStore(start)
		if !addErr(err, "start: %v", err) {
			// Get start objects
			startStore, err := h.ui.Engine.Store(h.f.StartClass.Domain().String())
			if !addErr(err) {
				result := korrel8r.NewResult(h.f.StartClass)
				log.V(3).Info("get start", "url", startStore.Resolve(h.f.StartRef))
				addErr(startStore.Get(context.Background(), h.f.StartRef, result))
				h.f.StartObjects = result.List()
			}
		}
	}
	// Get correlated goal references, need start and goal
	h.f.GoalClass, err = h.ui.Engine.ParseClass(h.f.Goal)
	if addErr(err, "goal: %w", err) {
		return
	}

	paths, err := h.ui.Engine.Graph().ShortestPaths(h.f.StartClass, h.f.GoalClass)
	if addErr(err) {
		return
	}

	// Collect results, including start and goal results
	results := engine.NewResults()
	first := results.Get(h.f.StartClass)
	first.References.Append(h.f.StartRef)
	first.Objects.Append(h.f.StartObjects...)
	h.f.FollowErr = h.ui.Engine.FollowAll(context.Background(), h.f.StartObjects, nil, paths, results)
	multierr.AppendInto(&h.f.FollowErr, h.ui.Engine.GetLast(context.Background(), results))
	h.f.Results = results.Prune() // Keep only results with objects

	// Generate a rule diagram
	h.diagram(paths)
}

// diagram the set of rules used in the given paths.
func (h *correlateHandler) diagram(multipaths []graph.MultiPath) {
	addErr := h.f.addErr
	var rules []korrel8r.Rule
	for _, m := range multipaths {
		for _, r := range m {
			rules = append(rules, r...)
		}
	}
	if rules == nil {
		return
	}
	name := "rule_graph"
	g := graph.New(name, rules, nil)

	// Decorate the graph to show results
	for _, result := range h.f.Results.List {
		attrs := g.NodeForClass(result.Class).Attrs
		attrs["fillcolor"] = "green"
		if len(result.References.List) > 0 {
			// TODO Pop-up for multiple refs?
			attrs["URL"] = h.refToConsole(result.Class, result.References.List[0]).String()
			attrs["target"] = "_blank"
		}
		attrs["xlabel"] = fmt.Sprintf("%v", len(result.Objects.List()))
	}

	// Write the graphViz dot file
	gv, err := dot.MarshalMulti(g, "", "", "  ")
	if addErr(err) {
		return
	}
	baseName := filepath.Join(h.ui.dir, "files", name)
	gvFile := baseName + ".gv"
	if addErr(os.WriteFile(gvFile, gv, 0664)) {
		return
	}
	// Render and write the graph image
	imageFile := baseName + ".svg"
	cmd := exec.Command("dot", "-x", "-Tsvg", "-o", imageFile, gvFile)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if addErr(cmd.Run()) {
		return
	}
	h.f.Diagram = path.Join("files", filepath.Base(imageFile))

	// TODO Experimental topological sort.
	order, err := topo.Sort(g)
	if !addErr(err, "topological sort: %v", err) {
		for _, n := range order {
			h.f.Topo = append(h.f.Topo, n.(*graph.Node).Class)
		}
	}
}
