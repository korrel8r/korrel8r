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
	"golang.org/x/exp/maps"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

// correlate web page handler.
type correlate struct {
	URL *url.URL
	// URL Query parameter fields
	Start       string // Starting point, console URL or query string.
	StartDomain string
	Goal        string
	Other       string
	Neighbours  string
	Goals       []string // Goals for radio option

	// Computed fields used by page template.
	StartQuery                      korrel8r.Query
	StartClass                      korrel8r.Class
	GoalClasses                     []korrel8r.Class
	Depth                           int
	Graph                           *graph.Graph
	Diagram, DiagramTxt, DiagramImg string
	ConsoleURL                      *url.URL
	UpdateTime                      time.Duration

	// Other context
	Version string
	Err     error // Accumulated errors from template.

	browser *Browser
}

// reset the fields to contain only URL query parameters
func (c *correlate) reset(url *url.URL) {
	params := url.Query()
	app := c.browser // Save
	*c = correlate{  // Overwrite
		URL:         url,
		Start:       params.Get("start"),
		StartDomain: params.Get("domain"),
		Goal:        params.Get("goal"),
		Other:       params.Get("other"),
		Neighbours:  params.Get("neighbours"),
		Version:     app.version,
		browser:     app,
	}
	c.Goals = []string{"log", "k8s:Event", "metric:metric"}
	c.ConsoleURL = c.browser.console.BaseURL
	c.Graph = c.browser.engine.Graph()
	// Defaults
	if c.Goal == "" {
		c.Goal = "neighbours"
	}
	if c.Goal == "neighbours" {
		c.Depth, _ = strconv.Atoi(c.Neighbours)
		if c.Depth <= 0 {
			c.Depth = 1 // Invalid use default
		}
		c.Neighbours = strconv.Itoa(c.Depth)
	}
}

func (c *correlate) HTML(gc *gin.Context) {
	c.update(gc.Request)
	if c.Err != nil {
		log.Error(c.Err, "Page errors")
		c.Graph = graph.New(nil) // Don't show empty graph on error
	}
	gc.HTML(http.StatusOK, "correlate.html.tmpl", c)
}

func (c *correlate) NewStartURL(query string) *url.URL {
	values := c.URL.Query()
	values.Set("start", query) // Replace start query
	u := url.URL(*c.URL)       // Copy
	u.RawQuery = values.Encode()
	return &u
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
	start := time.Now()
	defer func() {
		c.UpdateTime = time.Since(start)
		log.V(2).Info("update complete", "duration", c.UpdateTime)
	}()
	c.reset(req.URL)
	var constraint *korrel8r.Constraint
	if !c.addErr(c.updateStart(), "start") {
		// Prime the start node with initial results
		start := c.Graph.NodeFor(c.StartClass)
		if c.addErr(c.browser.engine.Get(context.Background(), c.StartQuery, constraint, start.Result)) {
			return
		}
		start.Queries.Set(c.StartQuery, len(start.Result.List()))
	}
	c.addErr(c.updateGoal(), "goal")
	if c.Err != nil {
		return
	}
	follower := c.browser.engine.Follower(context.Background(), constraint)

	if c.GoalClasses != nil { // Find paths from start to goals.
		c.Graph = c.Graph.Traverse(c.StartClass, c.GoalClasses, follower.Traverse)
		// Only include paths to a goal, remove dead-ends.
		c.Graph = c.Graph.AllPaths(c.StartClass, c.GoalClasses...)
	} else { // Find Neighbours
		c.Graph = c.Graph.Neighbours(c.StartClass, c.Depth, follower.Traverse)
	}
	// Add start node even if empty.
	c.addClassNode(c.StartClass)
	c.updateDiagram()
}

func (c *correlate) updateStart() (err error) {
	prefix, _, ok := strings.Cut(c.Start, ":") // Guess if its a korrel8r query or a URL
	switch {
	case ok && (prefix == "http" || prefix == "https"): // Looks like URL
		var u *url.URL
		if u, err = url.Parse(c.Start); err == nil {
			c.StartQuery, err = c.browser.console.QueryFromURL(u)
		}
	case ok: // Try as a query
		c.StartQuery, err = c.browser.engine.Query(c.Start)
	case c.Start == "":
		err = errors.New("empty")
	default:
		err = fmt.Errorf("invalid start: %v", c.Start)
	}
	if c.StartQuery != nil {
		c.StartClass = c.StartQuery.Class()
	}
	return err
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

func (c *correlate) queryURLAttrs(a graph.Attrs, qs graph.Queries, d korrel8r.Domain) {
	// TODO find a way to combine multiple queries into a URL?

	// Find the biggest count
	maxS, maxN := "", -1
	for s, qc := range qs {
		if qc.Count > maxN {
			maxS, maxN = s, qc.Count
		}
	}
	if maxS != "" {
		q, err := d.Query(maxS)
		if err != nil {
			a["URL"] = c.checkURL(&url.URL{}, err).String()
		} else {
			a["URL"] = c.checkURL(c.browser.console.URLFromQuery(q)).String()
		}
		a["target"] = "_blank"
	}
}

// TODO make this configurable - map domains to node attrs
var domainAttrs = map[string]graph.Attrs{
	"k8s":    {"shape": "septagon", "fillcolor": "#326CE5", "fontcolor": "white", "fontname": "Ubuntu,Bold"},
	"log":    {"shape": "note", "fillcolor": "goldenrod", "fontname": "Courier"},
	"alert":  {"shape": "triangle", "fillcolor": "yellow", "fontname": "Helvetica", "style": "bold"},
	"metric": {"shape": "oval", "fillcolor": "violet", "style": "rounded"},
}

// updateDiagram generates an SVG diagram via graphviz.
func (c *correlate) updateDiagram() {
	g := c.Graph
	if c.Goal == "neighbours" {
		c.Graph.GraphAttrs["layout"] = "twopi"
	}
	g.EachNode(func(n *graph.Node) {
		a := n.Attrs
		maps.Copy(a, domainAttrs[n.Class.Domain().Name()]) // add in domainAttrs
		result := n.Result.List()
		a["style"] += ",filled"
		a["label"] = fmt.Sprintf("%v\n%v", n.Class.Name(), len(result))
		a["tooltip"] = fmt.Sprintf("%v (%v)", n.Class.String(), len(result))
		c.queryURLAttrs(a, n.Queries, n.Class.Domain())
		if summary := summaryFunc(n.Class); summary != nil && len(result) > 0 {
			if len(result) == 1 {
				a["label"] = fmt.Sprintf("%v\n%v", n.Class.Name(), summary(result[0]))
			}
			n := min(10, len(result)) // Limit max items
			w := &strings.Builder{}
			fmt.Fprintln(w, a["tooltip"])
			for _, o := range result[:n] {
				fmt.Fprintf(w, "- %v\n", summary(o))
			}
			if n < len(result) {
				fmt.Fprintf(w, "\n... %v more\n", len(result)-n)
			}
			a["tooltip"] = w.String()
		}
	})

	g.EachLine(func(l *graph.Line) {
		a := l.Attrs
		a["tooltip"] = fmt.Sprintf("%v (%v)\n", l.Rule, l.Queries.Total())
		if count := l.Queries.Total(); count > 0 {
			a["arrowsize"] = fmt.Sprintf("%v", math.Min(0.3+float64(count)*0.05, 1))
			a["style"] = "bold"
			c.queryURLAttrs(a, l.Queries, l.Rule.Goal()[0].Domain())
		} else {
			a["color"] = "gray"
		}
	})

	if c.StartClass != nil {
		a := g.NodeFor(c.StartClass).Attrs
		a["color"] = "orange"
		a["root"] = "true"
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
				pngFile := baseName + ".png"
				if !c.addErr(runDot("dot", "-v", "-Tpng", "-o", pngFile, gvFile)) {
					c.DiagramImg, _ = filepath.Rel(c.browser.dir, pngFile)
				}
			}
		}
	}
}

func runDot(cmdName string, args ...string) error {
	cmd := exec.Command(cmdName, args[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v %w: %v", cmdName, err, string(out))
	}
	return err
}

func summaryFunc(c korrel8r.Class) func(any) string {
	switch c := c.(type) {
	case korrel8r.Previewer:
		return c.Preview
	case korrel8r.IDer:
		return func(v any) string { return fmt.Sprintf("%v", c.ID(v)) }
	default:
		return nil
	}
}
