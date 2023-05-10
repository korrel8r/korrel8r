// package browser is an experimental BROWSER UI server for browsers.
package browser

import (
	"bytes"
	"embed"
	"errors"
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"context"

	_ "net/http/pprof"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/openshift"
	"github.com/korrel8r/korrel8r/pkg/openshift/console"
	"github.com/korrel8r/korrel8r/pkg/templaterule"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log = logging.Log()

	//go:embed images
	static embed.FS

	//go:embed basepage.html.tmpl
	basePageHTML string
)

// App is the HTML browser webapp for human users.
type App struct {
	Engine  *engine.Engine
	Console *console.Console
	dir     string
}

// Register handlers to serve the HTML browser UI with a http.Handler.
// Including a "/" default handler.
// Returns a closer to clean up on exit.
func New(e *engine.Engine, cfg *rest.Config, c client.Client) (*App, error) {
	app := &App{Engine: e}
	var err error
	if app.dir, err = os.MkdirTemp("", "korrel8r"); err != nil {
		return nil, err
	}
	if err := os.Mkdir(filepath.Join(app.dir, "files"), 0700); err != nil {
		return nil, err
	}
	log.Info("working directory", "dir", app.dir)
	consoleURL, err := openshift.ConsoleURL(context.Background(), c)
	if err != nil {
		return nil, err
	}
	app.Console = console.New(consoleURL, e)
	return app, nil
}

// Register handlers with a http.Handler, including a default "/" handler.
func (app *App) Register(mux *http.ServeMux) {
	mux.Handle("/", http.RedirectHandler("/correlate", http.StatusMovedPermanently))
	mux.Handle("/correlate", &correlate{app: app})
	mux.Handle("/files/", http.FileServer(http.Dir(app.dir)))
	mux.Handle("/images/", http.FileServer(http.FS(static)))
	mux.HandleFunc("/stores/", app.stores)
	mux.HandleFunc("/error/", func(w http.ResponseWriter, req *http.Request) {
		// Handler that returns an error message, used when a link can't be generated due to error.
		httpError(w, errors.New(req.URL.Query().Get("err")), http.StatusInternalServerError)
	})
}

// Close should be called on shutdown to clean up external resources (temporary files etc.)
func (app *App) Close() {
	if err := os.RemoveAll(app.dir); err != nil {
		log.Error(err, "closing")
	}
}

// FIXME privatize and clean up

var funcs = map[string]any{
	"asHTML": func(s string) template.HTML { return template.HTML(s) },
}

// page tempate for all pages.
func (app *App) page(name string) *template.Template {
	return template.Must(
		template.New(name).
			Funcs(templaterule.Funcs).
			Funcs(app.Engine.TemplateFuncs()).
			Funcs(funcs).
			Parse(basePageHTML))
}

// httpError if err != nil update the response and return true.
func httpError(w http.ResponseWriter, err error, code int) bool {
	if err != nil {
		http.Error(w, err.Error(), code)
		log.Error(err, "http error")
	}
	return err != nil
}

func serveTemplate(w http.ResponseWriter, t *template.Template, text string, data any) {
	b := bytes.Buffer{}
	const code = http.StatusInternalServerError
	t, err := t.Parse(text)
	if !httpError(w, err, code) && !httpError(w, t.Execute(&b, data), code) {
		_, _ = w.Write(b.Bytes())
	}
}

func (b *App) stores(w http.ResponseWriter, req *http.Request) {
	path := path.Base(req.URL.Path)
	log.V(2).Info("store handler", "path", path, "query", req.URL.RawQuery)

	domain, err := b.Engine.DomainErr(path)
	if httpError(w, err, http.StatusNotFound) {
		return
	}

	store, err := b.Engine.StoreErr(
		domain.String())
	if httpError(w, err, http.StatusNotFound) {
		return
	}

	params := req.URL.Query()
	query, err := domain.UnmarshalQuery([]byte(params.Get("query")))
	if httpError(w, err, http.StatusNotFound) {
		return
	}

	result := korrel8r.NewResult(query.Class())
	err = store.Get(context.Background(), query, result)
	data := map[string]any{
		"query":  query,
		"err":    err,
		"result": result.List(),
	}
	serveTemplate(w, b.page("stores"), storeHTML, data)
}

const storeHTML = `
{{define "body"}}
    Query: {{json .query}}<br>
    <hr>
    {{if .err}}
        Error: {{.err}}<br>
    {{else}}
        Results ({{len .result}})<br>
            {{range .result}}<hr><pre>{{yaml .}}</pre>{{end}}
        </pre>
    {{end}}
{{end}}
    `
