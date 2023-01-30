package webui

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"go.uber.org/multierr"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

// correlate web page handler.
type correlate struct {
	// URL Query parameter fields
	Start string // Starting point, console URL
	Goal  string // Starting class full name.

	// Computed fields used by page template.
	Time                  time.Time
	StartClass, GoalClass korrel8r.Class
	StartQuery            korrel8r.Query
	StartObjects          []korrel8r.Object
	Results               engine.Results
	Diagram               string
	Topo                  []korrel8r.Class

	// Accumulated errors displayed on page
	Err error

	// Parent
	ui *WebUI
}

func (c *correlate) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	c.update(req)
	if c.Err != nil {
		log.Error(c.Err, "page errors")
	}
	t := c.ui.Page("correlate").Funcs(map[string]any{
		"queryToConsole": func(q korrel8r.Query) *url.URL { return c.checkURL(c.ui.Console.QueryToConsoleURL(q)) },
	})
	serveTemplate(w, t, correlateHTML, c)
}

// reset the fields to contain only URL query parameters
func (c *correlate) reset(params url.Values) {
	ui := c.ui      // Save
	*c = correlate{ // Overwrite
		Start: params.Get("start"),
		Goal:  params.Get("goal"),
		Time:  time.Now(),
	}
	c.ui = ui
}

// addErr adds an error to be displayed on the page.
func (c *correlate) addErr(err error, msg ...any) bool {
	if len(msg) > 0 && err != nil {
		err = fmt.Errorf(msg[0].(string), msg[1:])
	}
	return multierr.AppendInto(&c.Err, err)
}

// checkURL if err != nil generates a URL that will display an error when clicked.
func (c *correlate) checkURL(u *url.URL, err error) *url.URL {
	if c.addErr(err) {
		return &url.URL{Path: "/error", RawQuery: url.Values{"err": []string{err.Error()}}.Encode()}
	}
	return u
}

func (c *correlate) update(req *http.Request) {
	c.reset(req.URL.Query())

	start, err := url.Parse(c.Start)
	if !c.addErr(err, "start: %v", err) {
		if c.StartQuery, err = c.ui.Console.ConsoleURLToQuery(start); !c.addErr(err) {
			c.StartClass = c.StartQuery.Class()
			store, err := c.ui.Engine.StoreErr(c.StartClass.Domain().String())
			if !c.addErr(err) {
				result := korrel8r.NewResult(c.StartClass)
				log.V(3).Info("get start", "query", c.StartQuery)
				c.addErr(store.Get(context.Background(), c.StartQuery, result))
				c.StartObjects = result.List()
			}
			// Include the start queries in the result for display
			first := c.Results.Get(c.StartClass)
			first.Queries.Append(c.StartQuery)
			first.Objects.Append(c.StartObjects...)
		}
	}

	c.GoalClass, err = c.ui.Engine.Class(c.Goal)
	if !c.addErr(err, "goal: %v", err) {
		paths, err := c.ui.Engine.Graph().ShortestPaths(c.StartClass, c.GoalClass)
		if !c.addErr(err, "finding paths: %v", err) {
			if !c.addErr(c.ui.Engine.FollowAll(context.Background(), c.StartObjects, nil, paths, &c.Results)) {
				c.addErr(c.ui.Engine.GetLast(context.Background(), &c.Results))
			}
		}
		c.diagram(paths, &c.Results)
	}
}

// diagram the set of rules used in the given paths.
func (c *correlate) diagram(multipaths []graph.MultiPath, results *engine.Results) {
	var rules []korrel8r.Rule
	for _, m := range multipaths {
		m.Sort() // Predictable order
		for _, r := range m {
			rules = append(rules, r...)
		}
	}
	g := graph.New("rule_graph", rules, nil)

	// Decorate the graph to show results
	for _, result := range *results {
		attrs := g.NodeForClass(result.Class).Attrs
		if result.Objects != nil && len(result.Objects.List()) > 0 {
			attrs["fillcolor"] = "green"
			attrs["xlabel"] = fmt.Sprintf("%v", len(result.Objects.List()))
		}
		if len(result.Queries.List) > 0 {
			q := result.Queries.List[0] // TODO handle multiple queries
			attrs["URL"] = c.checkURL(c.ui.Console.QueryToConsoleURL(q)).String()
			attrs["target"] = "_blank"
			attrs["tooltip"] = fmt.Sprintf("first of %v queries: %v", len(result.Queries.List), korrel8r.JSONString(q))
		}
	}

	// Write the graph files
	baseName := filepath.Join(c.ui.dir, "files", g.Name())
	if gv, err := dot.MarshalMulti(g, "", "", "  "); !c.addErr(err) {
		gvFile := baseName + ".gv"
		if !c.addErr(os.WriteFile(gvFile, gv, 0664)) {
			// Render and write the graph image
			imageFile := baseName + ".svg"
			cmd := exec.Command("dot", "-x", "-Tsvg", "-o", imageFile, gvFile)
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			if !c.addErr(cmd.Run()) {
				c.Diagram, err = filepath.Rel(c.ui.dir, imageFile) // URL path is relative to root
				c.addErr(err)
			}
		}
	}
}
