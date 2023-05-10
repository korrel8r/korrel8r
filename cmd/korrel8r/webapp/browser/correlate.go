package browser

import (
	"context"
	_ "embed"
	"errors"
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

var (
	//go:embed correlate.html.tmpl
	correlateHTML string
)

// correlate web page handler.
type correlate struct {
	// URL Query parameter fields
	Start       string // Starting point, console URL or query string.
	StartDomain string
	Goal        string
	Other       string
	Neighbours  string

	ShortPaths bool // All paths
	RuleGraph  bool // Rules graph without results
	// Goals to list as radio options, map[value]id
	Goals []struct{ Value, Label string }

	// Computed fields used by page template.
	StartQuery                      korrel8r.Query
	StartClass, GoalClass           korrel8r.Class
	Depth                           int
	Graph                           *graph.Graph
	Diagram, DiagramTxt, DiagramImg string
	ConsoleURL                      *url.URL
	UpdateTime                      time.Duration
	// Accumulated errors displayed on page
	Err error

	// Parent
	app *App
}

// reset the fields to contain only URL query parameters
func (c *correlate) reset(params url.Values) {
	app := c.app    // Save
	*c = correlate{ // Overwrite
		Start:       params.Get("start"),
		StartDomain: params.Get("domain"),
		Goal:        params.Get("goal"),
		Other:       params.Get("other"),
		Neighbours:  params.Get("neighbours"),
		ShortPaths:  params.Get("short") == "true",
		RuleGraph:   params.Get("rules") == "true",
	}
	c.Goals = []struct{ Value, Label string }{
		{"logs/infrastructure", "Logs (infrastructure)"}, // FIXME wildcard for logs, multi-goal.
		{"k8s/Event", "Events"},
		{"metric/metric", "Metrics"},
	}
	c.app = app
	c.ConsoleURL = c.app.Console.BaseURL
	c.Graph = c.app.Engine.Graph()
	// Defaults
	if c.Goal == "" {
		c.Goal = "neighbours"
	}
	if c.Goal == "neighbours" {
		c.Depth, _ = strconv.Atoi(c.Neighbours)
		if c.Depth <= 0 {
			c.Depth = 9 // Invalid use default
		}
		c.Neighbours = strconv.Itoa(c.Depth)
	}
}

func (c *correlate) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	log.V(2).Info("serving correlate page", "uri", req.URL.RequestURI())
	c.update(req)
	if c.Err != nil {
		log.Error(c.Err, "page errors")
	}
	t := c.app.page("correlate").Funcs(map[string]any{
		"queryToConsole": func(q korrel8r.Query) *url.URL { return c.checkURL(c.app.Console.QueryToConsoleURL(q)) },
	})
	serveTemplate(w, t, correlateHTML, c)
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
	updateStart := time.Now()
	defer func() {
		c.UpdateTime = time.Since(updateStart)
		log.V(2).Info("update complete", "duration", c.UpdateTime)
	}()
	c.reset(req.URL.Query())
	if !c.addErr(c.updateStart(), "start") {
		// Prime the start node with initial results
		start := c.Graph.NodeFor(c.StartClass)
		if c.addErr(c.app.Engine.Get(context.Background(), c.StartClass, c.StartQuery, start.Result)) {
			return
		}
		start.QueryCounts.Put(c.StartQuery, len(start.Result.List()))
	}
	c.addErr(c.updateGoal(), "goal")
	if c.Err != nil {
		return
	}
	follower := c.app.Engine.Follower(context.Background())

	if c.GoalClass != nil { // Paths from start to goal.
		if c.ShortPaths {
			c.Graph = c.Graph.ShortestPaths(c.StartClass, c.GoalClass)
		} else {
			c.Graph = c.Graph.AllPaths(c.StartClass, c.GoalClass)
		}
	} else {
		// Find Neighbours
		traverse := follower.Traverse
		if c.RuleGraph {
			traverse = func(l *graph.Line) {}
		}
		c.Graph = c.Graph.Neighbours(c.StartClass, c.Depth, traverse)
	}
	if !c.RuleGraph {
		c.addErr(c.Graph.Traverse(follower.Traverse))
		c.Graph = c.Graph.Select(func(l *graph.Line) bool { // Remove lines with no queries
			return l.QueryCounts.Total() > 0
		})
		if c.GoalClass != nil {
			// Only include start->goal paths, remove dead-ends.
			c.Graph = c.Graph.AllPaths(c.StartClass, c.GoalClass)
		}
	}
	// Add start/goal nodes even if empty.
	c.addClassNode(c.StartClass)
	c.addClassNode(c.GoalClass)

	c.updateDiagram()
}

func (c *correlate) updateStart() (err error) {
	if c.Start == "" {
		return errors.New("empty")
	}
	if c.StartClass, err = c.app.Engine.Class(c.Start); err == nil {
		return nil
	}
	if u, err := url.Parse(c.Start); err == nil {
		if c.StartQuery, err = c.app.Console.ConsoleURLToQuery(u); err != nil {
			return err
		}
		c.StartClass = c.StartQuery.Class()
		return nil
	}
	domain, err := c.app.Engine.DomainErr(c.StartDomain)
	if err != nil {
		return err
	}
	if c.StartQuery, err = domain.UnmarshalQuery([]byte(c.Start)); err != nil {
		return err
	}
	c.StartClass = c.StartQuery.Class()
	return nil
}

func (c *correlate) updateGoal() (err error) {
	switch c.Goal {
	case "neighbours":
		return nil // No goal, depth is set.
	case "other":
		c.GoalClass, err = c.app.Engine.Class(c.Other)
	default:
		c.GoalClass, err = c.app.Engine.Class(c.Goal)
	}
	return err
}

func (c *correlate) addClassNode(class korrel8r.Class) {
	if class != nil {
		n := c.Graph.NodeFor(class)
		if c.Graph.Node(n.ID()) == nil {
			c.Graph.AddNode(n)
		}
	}
}

func (c *correlate) queryURLAttrs(a graph.Attrs, qcs graph.QueryCounts) {
	if len(qcs) > 0 {
		a["URL"] = c.checkURL(c.app.Console.QueryToConsoleURL(qcs.Sort()[0].Query)).String()
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
		a["tooltip"] = fmt.Sprintf("%v (%v)", korrel8r.ClassName(n.Class), len(n.Result.List()))
		a["style"] = "rounded,filled"
		result := n.Result.List()
		if len(result) == 0 {
			a["fillcolor"] = emptyColor
		} else {
			a["label"] = fmt.Sprintf("%v\n(%v)", a["label"], len(result))
			a["fillcolor"] = fullColor
			a["style"] = strings.Join([]string{a["style"], "bold"}, ",")
			c.queryURLAttrs(a, n.QueryCounts)
			previewer, _ := n.Class.(korrel8r.Previewer)
			if previewer != nil {
				const limit = 10
				for i, o := range result {
					a["tooltip"] = fmt.Sprintf("%v\n- %v", a["tooltip"], previewer.Preview(o))
					if i == limit {
						a["tooltip"] = fmt.Sprintf("%v\n%v", a["tooltip"], "...")
						break
					}
				}
			}
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

	if c.GoalClass != nil {
		goal := g.NodeFor(c.GoalClass)
		a := goal.Attrs
		a["shape"] = "diamond"
		a["fillcolor"] = goalColor
		if len(goal.Result.List()) == 0 {
			a["fillcolor"] = emptyColor
		}
	}

	// Write the graph files
	baseName := filepath.Join(c.app.dir, "files", "korrel8r")
	if gv, err := dot.MarshalMulti(g, "", "", "  "); !c.addErr(err) {
		gvFile := baseName + ".txt"
		if !c.addErr(os.WriteFile(gvFile, gv, 0664)) {
			// Render and write the graph image
			svgFile := baseName + ".svg"
			if !c.addErr(runDot("dot", "-v", "-Tsvg", "-o", svgFile, gvFile)) {
				c.Diagram, _ = filepath.Rel(c.app.dir, svgFile)
				c.DiagramTxt, _ = filepath.Rel(c.app.dir, gvFile)
			}
			pngFile := baseName + ".png"
			if !c.addErr(runDot("dot", "-v", "-Tpng", "-o", pngFile, gvFile)) {
				c.DiagramImg, _ = filepath.Rel(c.app.dir, pngFile)
			}
		}
	}
}

func runDot(cmdName string, args ...string) error {
	cmd := exec.Command(cmdName, args[1:]...)
	log.V(1).Info("run", "cmd", cmdName, "args", args)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}
