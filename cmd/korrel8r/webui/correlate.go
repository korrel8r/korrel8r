package webui

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
	GoalChoice  string // Goal radio choice
	ShortPaths  bool   // All paths
	NoResult    bool   // Rules diagram without results

	// Computed fields used by page template.
	Time                            time.Time
	StartQuery                      korrel8r.Query
	StartClass, GoalClass           korrel8r.Class
	Depth                           int
	Graph                           *graph.Graph
	Diagram, DiagramTxt, DiagramImg string
	ConsoleURL                      *url.URL
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
		GoalChoice:  params.Get("goalChoice"),
		ShortPaths:  params.Get("short") == "true",
		NoResult:    params.Get("noresult") == "true",
		Time:        time.Now(),
	}
	// Default missing values
	if c.StartIs == "" {
		c.StartIs = "Openshift Console"
	}
	if c.GoalChoice == "" {
		c.GoalChoice = "neighbours"
	}
	c.ui = ui
	c.ConsoleURL = c.ui.Console.BaseURL
	c.Graph = c.ui.Engine.Graph()
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
	c.addErr(c.updateStart(), "start")
	c.addErr(c.updateGoal(), "goal")
	if c.Err != nil {
		return
	}
	follower := c.ui.Engine.Follower(context.Background())

	if c.GoalClass != nil { // Paths from start to goal.
		if c.ShortPaths {
			c.Graph = c.Graph.ShortestPaths(c.StartClass, c.GoalClass)
		} else { // Follow shortest paths
			c.Graph = c.Graph.AllPaths(c.StartClass, c.GoalClass)
		}
	} else { // Find Neighbours
		traverse := follower.Traverse
		if c.NoResult {
			traverse = nil
		}
		c.Graph = c.Graph.Neighbours(c.StartClass, c.Depth, traverse)
	}
	if !c.NoResult {
		c.addErr(c.Graph.Traverse(follower.Traverse))
		c.Graph = c.Graph.Select(func(l *graph.Line) bool { // Remove lines with no queries
			return len(l.QueryCounts) > 0 || l.Rule.Goal() == c.GoalClass
		})
		if c.GoalClass != nil {
			// Only include start->goal paths, remove dead-ends.
			c.Graph = c.Graph.AllPaths(c.StartClass, c.GoalClass)
		}
	}
	c.updateDiagram()
}

func (c *correlate) updateStart() error {
	switch {
	case c.Start == "":
		return fmt.Errorf("missing")

	case c.StartIs == "query":
		domain, err := c.ui.Engine.DomainErr(c.StartDomain)
		if err != nil {
			return err
		}
		c.StartQuery, err = domain.UnmarshalQuery([]byte(c.Start))
		if err != nil {
			return err
		}

	default: // Console URL
		u, err := url.Parse(c.Start)
		if err != nil {
			return err
		}
		c.StartQuery, err = c.ui.Console.ConsoleURLToQuery(u)
		if err != nil {
			return err
		}
	}
	c.StartClass = c.StartQuery.Class()

	// Prime the start node with initial results
	start := c.Graph.NodeFor(c.StartClass)
	if err := c.ui.Engine.Get(context.Background(), c.StartClass, c.StartQuery, start.Result); err != nil {
		return err
	}
	start.QueryCounts.Put(c.StartQuery, len(start.Result.List()))
	return nil
}

func (c *correlate) updateGoal() error {
	switch c.GoalChoice {
	case "neighbours":
		c.Depth, _ = strconv.Atoi(c.Goal)
		if c.Depth == 0 {
			c.Depth = 1
		}
		return nil // Nil goal means neighbours
	case "other":
		// c.Goal field was filled in by user
	default:
		// One of the pre-defined choices
		c.Goal = c.GoalChoice
	}
	var err error
	c.GoalClass, err = c.ui.Engine.Class(c.Goal)
	return err
}

func (c *correlate) queryURLAttrs(a graph.Attrs, qcs graph.QueryCounts) {
	if len(qcs) > 0 {
		a["URL"] = c.checkURL(c.ui.Console.QueryToConsoleURL(qcs.Sort()[0].Query)).String()
		a["target"] = "_blank"
	}
}

const (
	startColor = "green2"
	goalColor  = "pink"
	fullColor  = "wheat"
	emptyColor = "white"
)

// updateDiagram generates an SVG diagram via graphviz.
func (c *correlate) updateDiagram() {
	g := c.Graph
	g.EachNode(func(n *graph.Node) {
		a := n.Attrs
		a["label"] = fmt.Sprintf("%v/%v", n.Class.Domain(), korrel8r.ShortString(n.Class))
		a["tooltip"] = fmt.Sprintf("%v (%v)\n", korrel8r.ClassName(n.Class), len(n.Result.List()))
		a["style"] = "rounded,filled"
		if count := len(n.Result.List()); count > 0 {
			a["label"] = fmt.Sprintf("%v\n(%v)", a["label"], count)
			a["style"] = strings.Join([]string{a["style"], "bold"}, ",")
			a["fillcolor"] = fullColor
			c.queryURLAttrs(a, n.QueryCounts)
		} else {
			a["fillcolor"] = emptyColor
		}
	})

	g.EachLine(func(l *graph.Line) {
		a := l.Attrs
		a["tooltip"] = fmt.Sprintf("%v (%v)\n", korrel8r.RuleName(l.Rule), l.QueryCounts.Total())
		if count := l.QueryCounts.Total(); count > 0 {
			a["arrowsize"] = fmt.Sprintf("%v", math.Min(0.3+float64(count)*0.05, 1))
			a["style"] = "bold"
			c.queryURLAttrs(a, l.QueryCounts)
		} else {
			a["style"] = "dashed"
			a["color"] = "gray"
		}
	})

	if c.StartClass != nil {
		a := g.NodeFor(c.StartClass).Attrs
		a["shape"] = "oval"
		a["fillcolor"] = startColor
		a["root"] = "true"
	}

	var layout string
	if c.GoalClass != nil {
		a := g.NodeFor(c.GoalClass).Attrs
		a["shape"] = "diamond"
		a["fillcolor"] = goalColor
		layout = "dot"
	} else {
		layout = "twopi"
	}

	// Write the graph files
	baseName := filepath.Join(c.ui.dir, "files", "korrel8r")
	if gv, err := dot.MarshalMulti(g, "", "", "  "); !c.addErr(err) {
		gvFile := baseName + ".txt"
		if !c.addErr(os.WriteFile(gvFile, gv, 0664)) {
			// Render and write the graph image
			svgFile := baseName + ".svg"
			if !c.addErr(runDot("dot", "-v", "-K", layout, "-Tsvg", "-o", svgFile, gvFile)) {
				c.Diagram, _ = filepath.Rel(c.ui.dir, svgFile)
				c.DiagramTxt, _ = filepath.Rel(c.ui.dir, gvFile)
			}
			pngFile := baseName + ".png"
			if !c.addErr(runDot("dot", "-v", "-K", layout, "-Tpng", "-o", pngFile, gvFile)) {
				c.DiagramImg, _ = filepath.Rel(c.ui.dir, pngFile)
			}
		}
	}
}

func runDot(cmdName string, args ...string) error {
	cmd := exec.Command(cmdName, args[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	log.V(2).Info("generate diagram", "cmd", cmd.String())
	return cmd.Run()
}
