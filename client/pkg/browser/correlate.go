// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package browser

import (
	_ "embed"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/client/pkg/swagger/client/operations"
	"github.com/korrel8r/korrel8r/client/pkg/swagger/models"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

// correlate web page handler.
type correlate struct {
	URL *url.URL
	// URL Query parameter fields
	Start string // Start query
	Goal  string // Goal class or neighbourhood depth

	// Computed fields used by page template.
	Depth                           int
	Graph                           *Graph
	Diagram, DiagramTxt, DiagramImg string
	ConsoleURL                      *url.URL
	UpdateTime                      time.Duration

	// Other context
	Err     error // Accumulated errors from template.
	Browser *Browser
}

// reset the fields to contain only URL query parameters
func (c *correlate) reset(url *url.URL) {
	params := url.Query()
	app := c.Browser // Save
	*c = correlate{  // Overwrite
		URL:     url,
		Start:   params.Get("start"),
		Goal:    params.Get("goal"),
		Browser: app,
		Graph:   NewGraph(nil),
	}
}

func (c *correlate) HTML(gc *gin.Context) {
	c.update(gc.Request)
	if c.Err != nil {
		c.Graph = NewGraph(nil)
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
	c.Err = errors.Join(c.Err, err)
	return true
}

// checkURL if err != nil generates a URL that will display an error when clicked.
func (c *correlate) checkURL(u *url.URL, err error) *url.URL {
	if c.addErr(err) {
		return &url.URL{Path: "/error", RawQuery: url.Values{"err": []string{err.Error()}}.Encode()}
	}
	return u
}

func (c *correlate) update(req *http.Request) {
	c.reset(req.URL)
	start := models.Start{Queries: []string{c.Start}}
	if c.Goal == "" {
		c.addErr(errors.New("Must provide a goal class or neighbourhood search depth."))
		return
	}
	var err error
	c.Depth, err = strconv.Atoi(c.Goal)
	if err == nil {
		ok, err := c.Browser.client.Operations.PostGraphsNeighbours(
			&operations.PostGraphsNeighboursParams{
				Request: &models.Neighbours{
					Start: &start,
					Depth: int64(c.Depth),
				},
			})
		if !c.addErr(err) {
			c.Graph = NewGraph(ok.Payload)
		}
	} else {
		ok, err := c.Browser.client.Operations.PostGraphsGoals(
			&operations.PostGraphsGoalsParams{
				Request: &models.Goals{
					Start: &start,
					Goals: []string{c.Goal},
				},
			})
		if !c.addErr(err) {
			c.Graph = NewGraph(ok.Payload)
		}
	}
	c.updateDiagram()
}

var domainAttrs = map[string]graph.Attrs{
	"k8s":       {"shape": "octagon", "fillcolor": "#326CE5", "fontcolor": "white"},
	"log":       {"shape": "note", "fillcolor": "goldenrod", "fontname": "Courier"},
	"alert":     {"shape": "triangle", "fillcolor": "yellow"},
	"metric":    {"shape": "oval", "fillcolor": "wheat"},
	"netobserv": {"shape": "rectangle", "fillcolor": "brick"},
}

// updateDiagram generates an SVG diagram via graphviz.
func (c *correlate) updateDiagram() {
	if c.Depth > 0 {
		c.Graph.GraphAttrs["layout"] = "twopi"
	}
	nodes := c.Graph.Nodes()
	for nodes.Next() {
		n := nodes.Node().(*Node)
		count := n.Model.Count
		a := n.Attrs
		a["style"] += "filled"
		a["label"] = fmt.Sprintf("%v\n%v", n.Model.Class, count)
		a["tooltip"] = fmt.Sprintf("%#+v", n.Model.Queries)
		maps.Copy(a, domainAttrs[strings.SplitN(n.Model.Class, ":", 2)[0]])
	}
	// Write the graph files
	baseName := filepath.Join(c.Browser.dir, "files", "korrel8r")
	if gv, err := dot.MarshalMulti(c.Graph, "", "", "  "); !c.addErr(err) {
		gvFile := baseName + ".txt"
		if !c.addErr(os.WriteFile(gvFile, gv, 0664)) {
			// Render and write the graph image
			svgFile := baseName + ".svg"
			if !c.addErr(runDot("dot", "-v", "-Tsvg", "-o", svgFile, gvFile)) {
				c.Diagram, _ = filepath.Rel(c.Browser.dir, svgFile)
				c.DiagramTxt, _ = filepath.Rel(c.Browser.dir, gvFile)
				pngFile := baseName + ".png"
				if !c.addErr(runDot("dot", "-v", "-Tpng", "-o", pngFile, gvFile)) {
					c.DiagramImg, _ = filepath.Rel(c.Browser.dir, pngFile)
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
