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

	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"go.uber.org/multierr"
	ggraph "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/multi"
)

// correlate web page handler.
type correlate struct {
	// URL Query parameter fields
	Start       string // Starting point, console URL or query string.
	StartIs     string // "Openshift Console"or "korrel8r Query"
	StartDomain string
	Goal        string // Goal class full name.
	Goals       string // Goal radio choice
	Full        bool   // Full diagram
	All         bool   // All paths

	// Computed fields used by page template.
	Time                  time.Time
	StartClass, GoalClass korrel8r.Class
	Diagram               string
	Results               []*graph.Result
	ConsoleURL            *url.URL
	// Accumulated errors displayed on page
	Err error

	// Parent
	ui *WebUI
}

func (c *correlate) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	log.V(2).Info("serving correlate page", "uri", req.URL.RequestURI())
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
		Start:       params.Get("start"),
		StartIs:     params.Get("startis"),
		StartDomain: params.Get("domain"),
		Goal:        params.Get("goal"),
		Goals:       params.Get("goals"),
		Full:        params.Get("full") == "true",
		All:         params.Get("all") == "true",
		Time:        time.Now(),
	}
	// Default missing values
	if c.StartIs == "" {
		c.StartIs = "Openshift Console"
	}
	if c.Goals == "" {
		c.Goals = "logs"
	}
	c.ui = ui
	c.ConsoleURL = c.ui.Console.BaseURL
}

// addErr adds an error to be displayed on the page.
func (c *correlate) addErr(err error, msg ...any) bool {
	if err == nil {
		return false
	}
	switch len(msg) {
	case 0: // Use err unmodified
	case 1: // Use bare msg string as prefix
		err = fmt.Errorf("%v: %w", msg[0], err)
	default: // Treat msg as printf format
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
	starters := korrel8r.NewResult(c.StartClass)
	startQuery, err := c.updateStart(starters)
	c.addErr(err, "start")
	c.addErr(c.updateGoal(), "goal")
	if c.StartClass == nil || c.GoalClass == nil || len(starters.List()) == 0 {
		return
	}
	full := c.ui.Engine.Graph()
	var paths [][]ggraph.Node
	if c.All {
		paths = graph.AllPaths(full, full.NodeForClass(c.StartClass).ID(), full.NodeForClass(c.GoalClass).ID())
	} else {
		paths = full.ShortestPaths(c.StartClass, c.GoalClass)
	}
	pathGraph := full.PathGraph(paths)
	if !c.addErr(c.ui.Engine.Traverse(context.Background(), starters.List(), nil, pathGraph)) {
		nodes := pathGraph.Nodes()
		for nodes.Next() {
			n := nodes.Node().(*graph.Node)
			r := n.Result
			c.Results = append(c.Results, r)
			if len(r.Queries.List) == 0 && n.Class != c.StartClass && n.Class != c.GoalClass && !c.Full {
				pathGraph.RemoveNode(n.ID())
			}
		}
	}
	c.diagram(pathGraph, startQuery, starters.List())
}

func (c *correlate) updateStart(result korrel8r.Result) (korrel8r.Query, error) {
	var query korrel8r.Query
	switch {
	case c.Start == "":
		return nil, fmt.Errorf("missing")

	case c.StartIs == "query":
		domain, err := c.ui.Engine.DomainErr(c.StartDomain)
		if err != nil {
			return nil, err
		}
		query, err = domain.UnmarshalQuery([]byte(c.Start))
		if err != nil {
			return nil, err
		}

	default: // Console URL
		u, err := url.Parse(c.Start)
		if err != nil {
			return nil, err
		}
		query, err = c.ui.Console.ConsoleURLToQuery(u)
		if err != nil {
			return nil, err
		}
	}
	c.StartClass = query.Class()
	// Get start objects, save in c.Results
	return query, c.ui.Engine.Get(c.StartClass, context.Background(), query, result)
}

func (c *correlate) updateGoal() error {
	switch c.Goals {
	case "logs":
		c.Goal = "logs/infrastructure"
	case "metrics":
		c.Goal = "metric/metric"
	case "events":
		c.Goal = "k8s/Event"
	}
	if c.Goal == "" {
		return fmt.Errorf("missing")
	}
	var err error
	c.GoalClass, err = c.ui.Engine.Class(c.Goal)
	return err
}

// diagram the set of rules used in the given paths.
func (c *correlate) diagram(g *graph.Graph, startQuery korrel8r.Query, starters []korrel8r.Object) {
	// Decorate the graph to show results
	start := g.NodeForClass(c.StartClass)
	start.Result.Queries.Add(startQuery)
	start.Result.Objects = len(starters)
	start.Attrs["rank"] = "first"
	start.Attrs["shape"] = "oval"

	goal := g.NodeForClass(c.GoalClass)
	goal.Attrs["rank"] = "last"
	goal.Attrs["shape"] = "oval"

	edges := g.Edges()
	for edges.Next() {
		lines := edges.Edge().(multi.Edge)
		for lines.Next() {
			l := lines.Line().(*graph.Line)
			l.Attrs["xlabel"] = l.Rule.String()
			c.resultAttrs(l.Result, l.Attrs)
		}
	}
	nodes := g.Nodes()
	for nodes.Next() {
		node := nodes.Node().(*graph.Node)
		r, a := node.Result, node.Attrs
		if r.Objects > 0 {
			a["xlabel"] = fmt.Sprintf("%v/%v", len(r.Queries.List), r.Objects)
		}
		c.resultAttrs(r, a)
	}

	// Write the graph files
	baseName := filepath.Join(c.ui.dir, "files", "korrel8r")
	if gv, err := dot.MarshalMulti(g, "", "", "  "); !c.addErr(err) {
		gvFile := baseName + ".txt"
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

func (c *correlate) resultAttrs(r *graph.Result, a graph.Attrs) {
	a["tooltip"] = r.String()
	if r.Objects > 0 {
		a["fillcolor"] = "cyan"
		a["color"] = "green"
		a["URL"] = c.checkURL(c.ui.Console.QueryToConsoleURL(r.Queries.List[0])).String()
		a["target"] = "_blank"
	}
}
