package webui

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/korrel8/korrel8/internal/pkg/must"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/unique"
	"github.com/korrel8/korrel8/pkg/uri"
	"go.uber.org/multierr"
)

// correlateFields are fields used by the page template.
type correlateFields struct {
	// Input fields from query parameters
	Start, Goal string

	// Computed fields for display
	StartClass, GoalClass korrel8.Class
	StartRef              uri.Reference
	StartObjects          []korrel8.Object
	GoalRefs              []uri.Reference
	GoalObjects           []korrel8.Object
	Diagram               string
	Err                   error
}

const correlateTemplate = `
{{define "body"}}
    <form>
        <input type="text" id="start" name="start" value="{{.Start}}">
        <label for="start">Start reference or console URL</label>
        <br>
        <input type="text" id="goal" name="goal" value="{{.Goal}}">
        <label for="goal">Goal Class</label>
        <br>
        <input type="submit" value="Submit">
    </form>

    {{with .Err}}
        <hr>
        <div>
            Errors:<br>
            <pre>{{printf "%+v" .}}</pre>
        </div>
    {{end}}

    <hr>
    <div>
        Start: {{.StartClass}}
        {{if and .StartClass .StartRef}}
            <a href={{refToConsole .StartRef}}>Console</a>,
            <a href="/stores/{{.StartClass.Domain}}/{{.StartRef}}">Raw</a> ({{len .StartObjects}})
        {{end}}
    </div>

    <hr>
    <div>
        Goal: {{.GoalClass}}
        {{if and .GoalClass .GoalRefs}}
            <ul>
                {{range .GoalRefs}}
                    <li>
                        {{$.GoalClass}}
                        <a href="{{refToConsole .}}">Console</a>,
                        <a href="/stores/{{$.GoalClass.Domain}}/{{.}}">Raw</a>
                {{end}}
            </ul>
        {{end}}

        {{with .Diagram}}
            <img  src="{{.}}">
        {{end}}
    </div>

{{end}}
`

func (f *correlateFields) Reset(params url.Values) {
	*f = correlateFields{
		Start: params.Get("start"),
		Goal:  params.Get("goal"),
	}
}

func (f *correlateFields) Params() url.Values {
	v := url.Values{}
	v.Set("start", f.Start)
	v.Set("goal", f.Goal)
	return v
}

type correlateHandler struct {
	ui *WebUI
	f  correlateFields
}

func (h *correlateHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	var err error
	defer func() { HTTPError(w, err) }()

	if err = h.update(req); err != nil {
		return
	}
	t := h.ui.Page("correlate").Funcs(map[string]any{
		"refToConsole": h.ui.Rewriter.RefStoreToConsoleURL,
	})
	must.Must1(t.Parse(correlateTemplate))
	must.Must(t.Execute(w, h.f))
}

func (h *correlateHandler) update(req *http.Request) error {
	addErr := func(err error) bool { return multierr.AppendInto(&h.f.Err, err) }
	h.f.Reset(req.URL.Query())

	// Start may be a store reference or a console URL
	if h.f.Start == "" {
		addErr(errors.New("no start reference"))
		return nil
	}

	start, err := uri.Parse(h.f.Start)
	if addErr(err) {
		return nil
	}

	// First Try as a console reference.
	h.f.StartClass, h.f.StartRef, err = h.ui.Rewriter.RefConsoleToStore(start)
	if err != nil { // Now try as a store reference.
		h.f.StartRef = start
		h.f.StartClass, err = h.ui.Rewriter.RefClass(start)
		if err != nil {
			addErr(fmt.Errorf("can't deduce start class: %w", err))
			return nil
		}
	}

	// Get start objects
	startStore := must.Must1(h.ui.Engine.Store(h.f.StartClass.Domain().String()))
	result := korrel8.NewResult(h.f.StartClass)
	must.Must(startStore.Get(context.Background(), h.f.StartRef, result))
	h.f.StartObjects = result.List()

	// Get correlated goal references
	h.f.GoalClass, err = h.ui.Engine.ParseClass(h.f.Goal)
	if addErr(must.ErrorIf(err, "goal: %v", err)) {
		return nil
	}
	pathFunc := h.ui.Engine.Graph().ShortestPaths // FIXME - common settings with CLI
	paths := must.Must1(pathFunc(h.f.StartClass, h.f.GoalClass))
	goalRefs := unique.NewList[uri.Reference]()
	for _, path := range paths {
		refs, err := h.ui.Engine.Follow(context.Background(), result.List(), nil, path)
		if !addErr(err) {
			goalRefs.Append(refs...)
		}
	}
	h.f.GoalRefs = goalRefs.List

	// Try to get goal objects
	goalStore, err := h.ui.Engine.Store(h.f.GoalClass.Domain().String())
	if !addErr(err) {
		result := korrel8.NewResult(h.f.GoalClass)
		for _, ref := range goalRefs.List {
			must.Must(goalStore.Get(context.Background(), ref, result))
		}
		h.f.GoalObjects = result.List()
	}

	// Build rule diagram
	var rules []korrel8.Rule
	for _, m := range paths {
		for _, r := range m {
			rules = append(rules, r...)
		}
	}
	if rules != nil {
		h.ui.Diagram("paths", rules)
		h.f.Diagram = "/files/paths.png"
	}

	return nil
}
