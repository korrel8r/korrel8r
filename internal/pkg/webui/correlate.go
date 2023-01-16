package webui

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"text/template"

	"github.com/korrel8/korrel8/pkg/graph"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/unique"
	"github.com/korrel8/korrel8/pkg/uri"
	"go.uber.org/multierr"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

// correlateFields are fields used by the page template.
type correlateFields struct {
	// Input fields from query parameters
	Start string // Starting point, console URL
	Goal  string // Starting class full name.

	// Computed fields for display
	StartClass, GoalClass korrel8.Class
	StartRef              uri.Reference
	StartObjects          []korrel8.Object
	GoalRefs              []uri.Reference
	GoalObjects           []korrel8.Object
	Diagram               string
	Err                   error
}

// addErr adds an error to the list. Display as much information as possible even with errors.
func (f *correlateFields) addErr(err error, msg ...any) bool {
	if len(msg) > 0 && err != nil {
		err = fmt.Errorf(msg[0].(string), msg[1:])
	}
	return multierr.AppendInto(&f.Err, err)
}

const correlateTemplate = `
{{define "body"}}
    <form>
        <input type="text" id="start" name="start" value="{{.Start}}">
        <label for="start">Start Console URL</label>
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
        Start Objects: {{.StartClass}}
        {{if .StartObjects}}
            <a href={{.Start}} target="_blank">Console</a>,
            <a href="/stores/{{fullname .StartClass}}/{{.StartRef}}" target="_blank">Raw</a> ({{len .StartObjects}})
        {{end}}
    </div>

    <hr>
    <div>
        Goal References: {{.GoalClass}} ({{len .GoalRefs}})
        {{if .GoalRefs}}
            <ul>
                {{range .GoalRefs}}
                    <li>
                        <a href="{{refToConsole $.GoalClass .}}" target="_blank">Console</a>,
                        <a href="/stores/{{fullname $.GoalClass}}/{{.}}" target="_blank">Raw</a>
                {{end}}
            </ul>
        {{end}}
    </div>

    {{with .Diagram}}
        <hr>
        <div>
           <img src="{{.}}">
        </div>
     {{end}}

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
	log.V(4).Info("correlation page")
	defer req.Body.Close()
	h.update(req)
	t := h.ui.Page("correlate").Funcs(map[string]any{
		"refToConsole": func(c korrel8.Class, r uri.Reference) *url.URL {
			cref, err := h.ui.Rewriter.RefStoreToConsoleURL(c, r)
			if err != nil {
				cref = &url.URL{Path: "/error", RawQuery: uri.Values{"err": []string{err.Error()}}.Encode()}
			}
			return cref
		}})
	if err := template.Must(t.Parse(correlateTemplate)).Execute(w, h.f); err != nil {
		http.Error(w, fmt.Sprintf("can't generate page: %v", err), 505)
	}
}

func (h *correlateHandler) update(req *http.Request) {
	h.f.Reset(req.URL.Query())
	addErr := h.f.addErr

	start, err := uri.Parse(h.f.Start) // Console URL
	if !addErr(err) {
		h.f.StartClass, h.f.StartRef, err = h.ui.Rewriter.RefConsoleToStore(start)
		if !addErr(err, "start: %v", err) {
			// Get start objects
			startStore, err := h.ui.Engine.Store(h.f.StartClass.Domain().String())
			if !addErr(err) {
				result := korrel8.NewResult(h.f.StartClass)
				log.V(3).Info("get start", "ref", h.f.StartRef)
				addErr(startStore.Get(context.Background(), h.f.StartRef, result))
				h.f.StartObjects = result.List()
			}
		}
	}
	// Get correlated goal references, need start and goal
	h.f.GoalClass, err = h.ui.Engine.ParseClass(h.f.Goal)
	if addErr(err, "goal: %v", err) {
		return
	}

	pathFunc := h.ui.Engine.Graph().ShortestPaths // FIXME - common settings with CLI
	paths, err := pathFunc(h.f.StartClass, h.f.GoalClass)
	if addErr(err) {
		return
	}
	goalRefs := unique.NewList[uri.Reference]()
	for _, path := range paths {
		refs, err := h.ui.Engine.Follow(context.Background(), h.f.StartObjects, nil, path)
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
			addErr(goalStore.Get(context.Background(), ref, result))
		}
		h.f.GoalObjects = result.List()
	}
	// Generate a rule diagram
	h.diagram(paths)
}

// diagram the set of rules used in the given paths.
func (h *correlateHandler) diagram(multipaths []graph.MultiPath) {
	addErr := h.f.addErr
	var rules []korrel8.Rule
	for _, m := range multipaths {
		for _, r := range m {
			rules = append(rules, r...)
		}
	}
	if rules == nil {
		return
	}
	name := "rule_graph"
	g := graph.New(name, rules, nil)
	// Write the graphViz dot file
	gv, err := dot.MarshalMulti(g, "", "", "  ")
	if addErr(err) {
		return
	}
	baseName := filepath.Join(h.ui.dir, "files", name)
	gvFile := baseName + ".gv"
	if addErr(os.WriteFile(gvFile, gv, 0664)) {
		return
	}
	// Render and write the graph image
	imageFile := baseName + ".svg"
	cmd := exec.Command("dot", "-x", "-Tsvg", "-o", imageFile, gvFile)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if addErr(cmd.Run()) {
		return
	}
	h.f.Diagram = path.Join("files", filepath.Base(imageFile))
}
