// package webui is an experimental HTML server for browsers.
package webui

import (
	"net/http"
	"os"
	"text/template"

	"context"

	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/korrel8/korrel8/internal/pkg/openshift"
	"github.com/korrel8/korrel8/internal/pkg/openshift/console"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/templaterule"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = logging.Log().WithName("webui")

type WebUI struct {
	Engine   *engine.Engine
	Console  *console.Console
	dir      string
	handlers map[string]http.Handler
}

func New(e *engine.Engine, cfg *rest.Config, c client.Client) (*WebUI, error) {
	ui := &WebUI{Engine: e}
	var err error
	if ui.dir, err = os.MkdirTemp("", "korrel8"); err != nil {
		return nil, err
	}
	log.Info("working directory", "dir", ui.dir)
	consoleURL, err := openshift.ConsoleURL(context.Background(), c)
	if err != nil {
		return nil, err
	}
	ui.Console = console.New(consoleURL, e)
	ui.handlers = map[string]http.Handler{
		"/":           http.HandlerFunc(ui.root),
		"/correlate/": &correlateHandler{UI: ui},
		"/files/":     http.FileServer(http.Dir(ui.dir)),
	}
	return ui, nil
}

func (ui *WebUI) Page(name string) *template.Template {
	return must(template.New(name).
		Funcs(templaterule.Funcs).
		Funcs(ui.Engine.TemplateFuncs()).
		Parse(`<!DOCTYPE html PUBLIC " - //W3C//DTD xhtml 1.0 Strict//EN"	"http://www.w3.org/1999/xhtml">
<head>
{{block "head" . -}}
  <title>Korrel8 Web UI</title>
{{end -}}
</head>
<body>
{{block "body" . -}}
Nothing to see here.
{{end -}}
</body>
`))
}

func (ui *WebUI) Close() {
	if err := os.RemoveAll(ui.dir); err != nil {
		log.Error(err, "closing")
	}
}

func (ui *WebUI) HandlerFuncs(mux *http.ServeMux) {
	for p, f := range ui.handlers {
		mux.Handle(p, f)
	}
}

func (ui *WebUI) root(w http.ResponseWriter, req *http.Request) {
	t := ui.Page("root")
	must(t.Parse(`
{{- define "body" -}}
Available Endpoints:
<ul>
{{ range $k,$v := . -}}
<li><a href={{$k}}>{{$k}}</li>
{{end -}}
</ul>
{{- end}}
`))
	check(t.Execute(w, ui.handlers))
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func must[T any](v T, err error) T {
	check(err)
	return v
}
