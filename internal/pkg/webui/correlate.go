package webui

import (
	"context"
	"net/http"
	"net/url"

	"github.com/korrel8/korrel8/internal/pkg/openshift/console"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/unique"
	"github.com/korrel8/korrel8/pkg/uri"
	"go.uber.org/multierr"
)

type correlateValues struct {
	Start, Goal           korrel8.Class
	StartObjects          []korrel8.Object
	Ref                   uri.Reference
	StartStore, GoalStore korrel8.Store
	GoalURLs              []*url.URL
	Diagram               string
	Err                   error
}

type correlateHandler struct {
	// Constants
	UI *WebUI

	// Request parameters
	Params url.Values
	correlateValues
}

func (h *correlateHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	h.update(req)
	t := h.UI.Page("correlate")
	must(t.Parse(`
{{define "body" -}}
<form>
  <label for="ref">URI Reference:</label>
  <input type="text" id="ref" name="ref" value="{{.Params.Get "ref" }}">
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
<p>Corrrelated {{$.Goal.Domain}}/{{$.Goal}} references:
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
{{with .Err}}<p>Errors</p><pre>{{printf "%+v" .}}</pre>{{end -}}
{{end -}}
`))
	check(t.Execute(w, h))
}

func (h *correlateHandler) update(req *http.Request) {
	// Reset
	h.Params = req.URL.Query()
	h.correlateValues = correlateValues{}
	h.Err = nil

	addErr := func(err error) bool { h.Err = multierr.Append(h.Err, err); return h.Err != nil }

	u, err := url.Parse(h.Params.Get("ref"))
	if !addErr(err) {
		h.Ref = uri.Extract(u)
	}
	h.Start, err = h.UI.Engine.ParseClass(h.Params.Get("start"))
	addErr(err)
	h.Goal, err = h.UI.Engine.ParseClass(h.Params.Get("goal"))
	addErr(err)
	if h.Err != nil {
		return
	}

	pathFunc := h.UI.Engine.Graph().AllPaths

	h.StartStore, err = h.UI.Engine.Store(h.Start.Domain().String())
	addErr(err)

	h.GoalStore, err = h.UI.Engine.Store(h.Goal.Domain().String())
	addErr(err)

	paths := must(pathFunc(h.Start, h.Goal))
	starters := korrel8.NewSetResult(h.Start)
	addErr(h.StartStore.Get(context.Background(), h.Ref, starters))
	h.StartObjects = starters.List()
	refs := unique.NewList[uri.Reference]()
	for _, path := range paths {
		qs, err := h.UI.Engine.Follow(context.Background(), starters.List(), nil, path)
		addErr(err)
		refs.Append(qs...)
	}
	h.GoalURLs = nil
	for _, ref := range refs.List {
		u, err := console.FormatURL(h.UI.ConsoleURL, h.Goal, ref)
		addErr(err)
		h.GoalURLs = append(h.GoalURLs, u)
	}
	var rules []korrel8.Rule
	for _, m := range paths {
		for _, r := range m {
			rules = append(rules, r...)
		}
	}
	if rules != nil {
		h.UI.Diagram("paths", rules)
		h.Diagram = "../files/paths.png"
	}
}
