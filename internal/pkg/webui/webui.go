// package webui is an experimental HTML server for browsers.
package webui

import (
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"context"

	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/korrel8/korrel8/internal/pkg/openshift"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/templaterule"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = logging.Log.WithName("webui")

type WebUI struct {
	Engine     *engine.Engine
	ConsoleURL *url.URL

	dir      string
	handlers map[string]http.Handler
	page     *template.Template
}

func New(e *engine.Engine, cfg *rest.Config, c client.Client) (*WebUI, error) {
	dir, err := os.MkdirTemp("", "korrel8_webui")
	if err == nil {
		err = os.Mkdir(filepath.Join(dir, "files"), 0777)
	}
	if err != nil {
		return nil, err
	}
	log.Info("working directory", "dir", dir)
	ui := &WebUI{
		dir:        dir,
		Engine:     e,
		ConsoleURL: must(openshift.ConsoleURL(context.Background(), c)),
		// FIXME better organization of template funcs.
		page: template.Must(template.New("page").Funcs(templaterule.Funcs).Parse(`
{{block "header" . -}}
<head>
  <title>Korrel8 Web UI</title>
</head>
<body>
{{end -}}

{{block "content" . -}}{{end -}}

{{block "footer" . -}}
</body>
{{end -}}
`)),
	}
	ui.handlers = map[string]http.Handler{
		"/":          http.HandlerFunc(ui.root),
		"/correlate": &correlateHandler{UI: ui, Page: ui.clonePage},
		"/files/":    http.FileServer(http.Dir(ui.dir)),
	}
	return ui, nil
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

func (ui *WebUI) clonePage() *template.Template { return must(ui.page.Clone()) }
func (ui *WebUI) parseContent(content string) (*template.Template, error) {
	return ui.clonePage().Parse(content)
}

func (ui *WebUI) root(w http.ResponseWriter, req *http.Request) {
	t := must(ui.parseContent(`
{{- define "content" -}}
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
	if err != nil {
		panic(err)
	}
	return v
}
