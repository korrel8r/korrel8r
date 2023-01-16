// package webui is an experimental HTML server for browsers.
package webui

import (
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"context"

	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/korrel8/korrel8/internal/pkg/must"
	"github.com/korrel8/korrel8/internal/pkg/openshift"
	"github.com/korrel8/korrel8/internal/pkg/rewrite"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/templaterule"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = logging.Log()

type WebUI struct {
	Engine   *engine.Engine
	Rewriter *rewrite.Rewriter
	Mux      *http.ServeMux
	dir      string
}

func New(e *engine.Engine, cfg *rest.Config, c client.Client) (*WebUI, error) {
	ui := &WebUI{Engine: e}
	var err error
	if ui.dir, err = os.MkdirTemp("", "korrel8"); err != nil {
		return nil, err
	}
	if err := os.Mkdir(filepath.Join(ui.dir, "files"), 0700); err != nil {
		return nil, err
	}
	log.Info("working directory", "dir", ui.dir)
	consoleURL, err := openshift.ConsoleURL(context.Background(), c)
	if err != nil {
		return nil, err
	}
	ui.Rewriter = rewrite.New(consoleURL, e)
	ui.Mux = http.NewServeMux()
	ui.Mux.Handle("/", &correlateHandler{ui: ui})
	ui.Mux.Handle("/files/", http.FileServer(http.Dir(ui.dir)))
	ui.Mux.Handle("/stores/", &storeHandler{ui: ui})
	ui.Mux.HandleFunc("/error/", func(w http.ResponseWriter, req *http.Request) {
		// So links that can't be generated can link to an error message.
		msg := req.URL.Query().Get("err")
		http.Error(w, "error: "+msg, http.StatusInternalServerError)
	})
	return ui, nil
}

func (ui *WebUI) Page(name string) *template.Template {
	return must.Must1(template.New(name).
		Funcs(templaterule.Funcs).
		Funcs(ui.Engine.TemplateFuncs()).
		Parse(`<!DOCTYPE html PUBLIC " - //W3C//DTD xhtml 1.0 Strict//EN"	"http://www.w3.org/1999/xhtml">
<head>
<style>
html, body {
    height: 100%;
    margin: 0;
    padding: 0;
}

img {
    padding: 0;
    display: block;
    margin: 0 auto;
    max-height: 100%;
    max-width: 100%;
}
</style>
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
