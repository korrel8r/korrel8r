// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

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

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"go.uber.org/multierr"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

// correlate web page handler.
type correlate struct {
	// URL Query parameter fields
	Start       string // Starting point, console URL or query string.
	StartDomain string
	Goal        string
	Other       string
	Neighbours  string

	ShortPaths bool     // All paths
	RuleGraph  bool     // Rules graph without results
	Goals      []string // Goals for radio option

	// Computed fields used by page template.
	StartQuery                      korrel8r.Query
	StartClass                      korrel8r.Class
	GoalClasses                     []korrel8r.Class
	Depth                           int
	Graph                           *graph.Graph
	Diagram, DiagramTxt, DiagramImg string
	ConsoleURL                      *url.URL
	UpdateTime                      time.Duration
	// Accumulated errors displayed on page
	Err error

	// Parent
	browser *Browser
}

// reset the fields to contain only URL query parameters
func (c *correlate) reset(params url.Values) {
	app := c.browser // Save
	*c = correlate{  // Overwrite
		Start:       params.Get("start"),
		StartDomain: params.Get("domain"),
		Goal:        params.Get("goal"),
		Other:       params.Get("other"),
		Neighbours:  params.Get("neighbours"),
		ShortPaths:  params.Get("short") == "true",
		RuleGraph:   params.Get("rules") == "true",
	}
	c.Goals = []string{"log", "k8s/Event", "metric/metric"}
	c.browser = app
	c.ConsoleURL = c.browser.console.BaseURL
	c.Graph = c.browser.engine.Graph()
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

func (c *correlate) HTML(gc *gin.Context) {
	c.update(gc.Request)
	if c.Err != nil {
		log.Error(c.Err, "page errors")
	}
	gc.HTML(http.StatusOK, "correlate.html.tmpl", c)
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
		if c.addErr(c.browser.engine.Get(context.Background(), c.StartClass, c.StartQuery, start.Result)) {
			return
		}
		start.QueryCounts.Put(c.StartQuery, len(start.Result.List()))
	}
	c.addErr(c.updateGoal(), "goal")
	if c.Err != nil {
		return
	}
	follower := c.browser.engine.Follower(context.Background())

	if c.GoalClasses != nil { // Paths from start to goal.
		if c.ShortPaths {
			c.Graph = c.Graph.ShortestPaths(c.StartClass, c.GoalClasses...)
		} else {
			c.Graph = c.Graph.AllPaths(c.StartClass, c.GoalClasses...)
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
		if c.GoalClasses != nil {
			// Only include start->goal paths, remove dead-ends.
			c.Graph = c.Graph.AllPaths(c.StartClass, c.GoalClasses...)
		}
	}
	// Add start/goal nodes even if empty.
	c.addClassNode(c.StartClass)
	for _, goal := range c.GoalClasses {
		c.addClassNode(goal)
	}

	c.updateDiagram()
}

func (c *correlate) updateStart() (err error) {
	if c.Start == "" {
		return errors.New("empty")
	}
	if c.StartClass, err = c.browser.engine.Class(c.Start); err == nil {
		return nil
	}
	if u, err := url.Parse(c.Start); err == nil {
		if c.StartQuery, err = c.browser.console.ConsoleURLToQuery(u); err != nil {
			return err
		}
		c.StartClass = c.StartQuery.Class()
		return nil
	}
	domain, err := c.browser.engine.DomainErr(c.StartDomain)
	if err != nil {
		return err
	}
	if c.StartQuery, err = domain.Query(c.Start); err != nil {
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
		c.GoalClasses, err = c.goalClasses(c.Other)
	default:
		c.GoalClasses, err = c.goalClasses(c.Goal)
	}
	return err
}

func (c *correlate) goalClasses(domainOrClass string) ([]korrel8r.Class, error) {
	if d, err := c.browser.engine.DomainErr(domainOrClass); err == nil {
		return d.Classes(), nil // all in domain
	}
	if c, err := c.browser.engine.Class(domainOrClass); err == nil {
		return []korrel8r.Class{c}, nil
	} else {
		return nil, err
	}
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
		a["URL"] = c.checkURL(c.browser.console.QueryToConsoleURL(qcs.Sort()[0].Query)).String()
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
	if c.Goal == "neighbours" {
		c.Graph.GraphAttrs["layout"] = "twopi"
	}
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

	for _, class := range c.GoalClasses {
		goal := g.NodeFor(class)
		a := goal.Attrs
		a["shape"] = "diamond"
		a["fillcolor"] = goalColor
		if len(goal.Result.List()) == 0 {
			a["fillcolor"] = emptyColor
		}
	}

	// Write the graph files
	baseName := filepath.Join(c.browser.dir, "files", "korrel8r")
	if gv, err := dot.MarshalMulti(g, "", "", "  "); !c.addErr(err) {
		gvFile := baseName + ".txt"
		if !c.addErr(os.WriteFile(gvFile, gv, 0664)) {
			// Render and write the graph image
			svgFile := baseName + ".svg"
			if !c.addErr(runDot("dot", "-v", "-Tsvg", "-o", svgFile, gvFile)) {
				c.Diagram, _ = filepath.Rel(c.browser.dir, svgFile)
				c.DiagramTxt, _ = filepath.Rel(c.browser.dir, gvFile)
			}
			pngFile := baseName + ".png"
			if !c.addErr(runDot("dot", "-v", "-Tpng", "-o", pngFile, gvFile)) {
				c.DiagramImg, _ = filepath.Rel(c.browser.dir, pngFile)
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
