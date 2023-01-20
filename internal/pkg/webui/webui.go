// package webui is an experimental HTML server for browsers.
package webui

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"context"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/openshift"
	"github.com/korrel8r/korrel8r/internal/pkg/rewrite"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/templaterule"
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
	if ui.dir, err = os.MkdirTemp("", "korrel8r"); err != nil {
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
		httpError(w, errors.New(req.URL.Query().Get("err")), http.StatusInternalServerError)
	})
	return ui, nil
}

func (ui *WebUI) Page(name string) *template.Template {
	return template.Must(
		template.New(name).
			Funcs(templaterule.Funcs).
			Funcs(ui.Engine.TemplateFuncs()).
			Parse(basePageHTML))
}

func (ui *WebUI) Close() {
	if err := os.RemoveAll(ui.dir); err != nil {
		log.Error(err, "closing")
	}
}

func httpError(w http.ResponseWriter, err error, status int) bool {
	if err != nil {
		log.Error(err, "http error")
		http.Error(w, err.Error(), status)
	}
	return err != nil
}
