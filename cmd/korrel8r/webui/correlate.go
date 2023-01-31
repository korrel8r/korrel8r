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
	Goal  string // Goal class full name.
	Goals string // Goal radio choice
	Full  bool   // Full diagram

	// Computed fields used by page template.
	Time                  time.Time
	StartClass, GoalClass korrel8r.Class
	Results               engine.Results
	Diagram               string

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
		Goals: params.Get("goals"),
		Full:  params.Get("full") == "true",
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
	if c.Start == "" {
		c.addErr(fmt.Errorf("missing start URL"))
	} else {
		start, err := url.Parse(c.Start)
		if !c.addErr(err, "start: %v", err) {
			if query, err := c.ui.Console.ConsoleURLToQuery(start); !c.addErr(err) {
				c.StartClass = query.Class()
				// Include the start queries in the result for display
				result := c.Results.Get(c.StartClass)
				result.Queries.Append(query)
				store, err := c.ui.Engine.StoreErr(c.StartClass.Domain().String())
				if !c.addErr(err) {
					c.addErr(store.Get(context.Background(), query, result.Objects))
				}
			}
		}
	}
	switch c.Goals {
	case "logs":
		c.Goal = "logs/infrastructure"
	case "metrics":
		c.Goal = "metric/metric"
	case "events":
		c.Goal = "k8s/Event"
	}
	if c.Goal == "" {
		c.addErr(fmt.Errorf("missing goal class"))
	} else {
		var err error
		c.GoalClass, err = c.ui.Engine.Class(c.Goal)
		c.addErr(err, "goal: %v", err)
	}

	if c.StartClass != nil && c.GoalClass != nil {
		paths, err := c.ui.Engine.Graph().AllPaths(c.StartClass, c.GoalClass)
		if !c.addErr(err, "finding paths: %v", err) {
			starters := c.Results.Get(c.StartClass).Objects.List()
			c.ui.Engine.FollowAll(context.Background(), starters, nil, paths, &c.Results)
			c.diagram(paths, &c.Results)
		}
	}
}

// diagram the set of rules used in the given paths.
func (c *correlate) diagram(multipaths []graph.MultiPath, results *engine.Results) {
	var rules []korrel8r.Rule
	if c.Full {
		for _, mp := range multipaths {
			for _, links := range mp {
				rules = append(rules, links...)
			}
		}
	} else {
		for _, result := range *results {
			rules = append(rules, result.Rules...)
		}
	}
	g := graph.New("Korrel8r Path", rules, []korrel8r.Class{c.GoalClass})

	// Decorate the graph to show results
	for i, result := range *results {
		if len(result.Rules) == 0 && i > 0 { // First stage has no rules.
			continue
		}
		g.NodeForClass(c.GoalClass).Attrs["rank"] = "last"
		attrs := g.NodeForClass(result.Class).Attrs
		if len(result.Queries.List) > 0 {
			attrs["xlabel"] = fmt.Sprintf("%v", len(result.Objects.List()))
			switch result.Class {
			case c.StartClass:
				attrs["fillcolor"] = "lightgreen"
			case c.GoalClass:
				attrs["fillcolor"] = "pink"
			default:
				attrs["fillcolor"] = "cyan"
			}
			attrs["URL"] = c.checkURL(c.ui.Console.QueryToConsoleURL(result.Queries.List[0])).String()
			attrs["target"] = "_blank"
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
