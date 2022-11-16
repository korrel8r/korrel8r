package webui

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/korrel8/korrel8/internal/pkg/openshift/console"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/unique"
	"go.uber.org/multierr"
)

type correlateValues struct {
	Start, Goal           korrel8.Class
	StartObjects          []korrel8.Object
	Query                 *korrel8.Query
	StartStore, GoalStore korrel8.Store
	GoalURLs              []*url.URL
	Diagram               string
	Err                   error
}

type correlateHandler struct {
	// Constants
	UI   *WebUI
	Page func() *template.Template

	// Request parameters
	Params url.Values
	correlateValues
}

func (h correlateHandler) Errors() []error {
	return multierr.Errors(h.Err)
}

func (h *correlateHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	h.update(req)
	page, err := h.Page().Parse(`
{{define "content" -}}
<form>
  <label for="query">Query URL:</label>
  <input type="text" id="query" name="query" value="{{.Params.Get "query" }}">
  <br>
  <label for="start">Start Class:</label>
  <input type="text" id="start" name="start" value="{{.Params.Get "start" }}">
  <br>
  <label for="goal">Goal Class:</label>
  <input type="text" id="goal" name="goal" value="{{.Params.Get "goal" }}">
  <br>
  <input type="submit" value="Submit">
</form>

{{with .GoalURLs -}}
<p>Corrrelated {{$.Goal.Domain}}/{{$.Goal}} queries:
<ul>
  {{range .}}<li><a href={{.}}>{{.}}</a></li>{{end}}
</ul>
{{end -}}

{{with .Diagram}}
<img src="{{.}}">
{{end}}

{{with .StartObjects -}}
<p>Start objects:</p>
{{range . -}}
<p><pre>
---
{{toYAML .}}
</pre></p>
{{end -}}
{{end -}}

{{with .Errors -}}
<p>Errors:
<ul>
  {{range . -}}
  <li>{{.}}</li>
  {{end -}}
</ul>
</p>
{{end -}}

{{end -}}
`)
	if err == nil {
		err = page.Execute(w, h)
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (h *correlateHandler) update(req *http.Request) {
	h.Params = req.URL.Query()
	h.correlateValues = correlateValues{} // Reset

	var err error
	h.Err = nil

	query, err := url.Parse(h.Params.Get("query"))
	h.Err = multierr.Append(h.Err, err)

	pathFunc := h.UI.Engine.Graph().ShortestPaths
	// FIXME brutal hack for demo, need URL recognizers and class from query. Sort out URL rewrite story.
	if strings.HasPrefix(query.Host, "console") {
		h.Start, h.Query, err = console.ParseURL(h.Params.Get("query"))
	} else {
		// FIXME hack, hack, hack
		h.Query = query
		h.Start, err = h.UI.Engine.ParseClass(h.Params.Get("start"))
		pathFunc = h.UI.Engine.Graph().AllPaths
	}
	h.Err = multierr.Append(h.Err, err)

	h.Goal, err = h.UI.Engine.ParseClass(h.Params.Get("goal"))
	h.Err = multierr.Append(h.Err, err)

	if h.Err == nil {
		h.StartStore = h.UI.Engine.Store(h.Start.Domain())
		if h.StartStore == nil {
			h.Err = multierr.Append(h.Err, fmt.Errorf("no store for %v", h.Start.Domain()))
		}
		h.GoalStore = h.UI.Engine.Store(h.Goal.Domain())
		if h.GoalStore == nil {
			h.Err = multierr.Append(h.Err, fmt.Errorf("no store for %v", h.Goal.Domain()))
		}
	}
	if h.Err == nil {
		paths := must(pathFunc(h.Start, h.Goal))
		starters := korrel8.NewSetResult(h.Start)
		h.StartStore.Get(context.Background(), h.Query, starters)
		h.StartObjects = starters.List()
		queries := unique.NewList[url.URL]()
		for _, path := range paths {
			qs, err := h.UI.Engine.Follow(context.Background(), starters.List(), nil, path)
			h.Err = multierr.Append(h.Err, err)
			queries.Append(qs...)
		}
		h.GoalURLs = nil
		for _, q := range queries.List {
			u, err := console.FormatURL(h.UI.ConsoleURL, h.Goal, &q)
			h.Err = multierr.Append(h.Err, err)
			h.GoalURLs = append(h.GoalURLs, u)
		}
		var rules []korrel8.Rule
		for _, m := range paths {
			for _, r := range m {
				rules = append(rules, r...)
			}
		}
		if rules != nil {
			// FIXME draw from start to goal?
			h.Diagram = h.UI.Diagram("paths", rules)
		}
	}
}
