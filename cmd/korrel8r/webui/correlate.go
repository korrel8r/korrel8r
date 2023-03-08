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
	Results               *engine.Results
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
		Results:     engine.NewResults(),
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
	var paths *graph.Graph
	if c.All {
		paths = full.AllPaths(c.StartClass, c.GoalClass)
	} else {
		paths = full.ShortestPaths(c.StartClass, c.GoalClass)
	}
	// Traverse and collect results
	if !c.addErr(c.ui.Engine.Traverse(context.Background(), starters.List(), nil, paths, c.Results)) {
		// Only keep rules that yielded results.
		paths = paths.SubGraph(func(l *graph.Line) bool {
			qr, ok := c.Results.Get(l)
			return ok && len(qr.Result.List()) > 0
			// FIXME combine traverse and subgraph for efficiency?
		})
	}
	c.diagram(paths, startQuery, starters.List())
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

func (c *correlate) queryURL(a graph.Attrs, q korrel8r.Query) {
	a["URL"] = c.checkURL(c.ui.Console.QueryToConsoleURL(q)).String()
	a["target"] = "_blank"
}

// diagram the set of rules used in the given paths.
func (c *correlate) diagram(g *graph.Graph, startQuery korrel8r.Query, starters []korrel8r.Object) {
	// Decorate the graph to show results
	a := g.NodeFor(c.StartClass).Attrs
	a["rank"] = "first"
	a["shape"] = "oval"
	c.queryURL(a, startQuery)

	a = g.NodeFor(c.GoalClass).Attrs
	a["rank"] = "last"
	a["shape"] = "oval"

	g.EachLine(func(l *graph.Line) {
		a := l.Attrs
		a["xlabel"] = l.Rule.String()
		if qr, ok := c.Results.Get(l); ok {
			if count := len(qr.Result.List()); count > 0 {
				// Decorate the rule arrow
				a["tooltip"] = fmt.Sprintf("%v : %v", korrel8r.ClassName(l.Rule.Goal()), count)
				a["color"] = "green"
				c.queryURL(a, qr.Query)
				a["target"] = "_blank"

				// Decorate the goal node
				n := l.To().(*graph.Node)
				a := n.Attrs
				// TODO handle multiple incoming lines to a single node.
				a["xlabel"] = fmt.Sprintf("%v", count)
				a["fillcolor"] = "cyan"
				c.queryURL(a, qr.Query)
			}
		}
	})

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
