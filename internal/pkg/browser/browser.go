// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package browser implements an HTML UI for web browsers.
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
	images embed.FS
	//go:embed basepage.html.tmpl
	basePageHTML string
)

// Browser implements HTTP handlers for web browsers.
type Browser struct {
	engine  *engine.Engine
	console *console.Console
	dir     string
}

func New(e *engine.Engine, cfg *rest.Config, c client.Client) (*Browser, error) {
	b := &Browser{engine: e}
	var err error
	if b.dir, err = os.MkdirTemp("", "korrel8r"); err != nil {
		return nil, err
	}
	if err := os.Mkdir(filepath.Join(b.dir, "files"), 0700); err != nil {
		return nil, err
	}
	log.Info("working directory", "dir", b.dir)
	consoleURL, err := openshift.ConsoleURL(context.Background(), c)
	if err != nil {
		return nil, err
	}
	b.console = console.New(consoleURL, e)
	return b, nil
}

// Register handlers with a http.ServeMux, including a default "/" redirect handler.
func (b *Browser) Register(mux *http.ServeMux) {
	mux.Handle("/", http.RedirectHandler("/correlate", http.StatusMovedPermanently))
	mux.Handle("/correlate", &correlate{app: b})
	mux.Handle("/files/", http.FileServer(http.Dir(b.dir)))
	mux.Handle("/images/", http.FileServer(http.FS(images)))
	mux.HandleFunc("/stores/", b.stores)
	mux.HandleFunc("/error/", func(w http.ResponseWriter, req *http.Request) {
		// Handler that returns an error message from the URL, used when a link can't be generated due to error.
		httpError(w, errors.New(req.URL.Query().Get("err")), http.StatusInternalServerError)
	})
}

// Close should be called on shutdown to clean up external resources.
func (b *Browser) Close() {
	if err := os.RemoveAll(b.dir); err != nil {
		log.Error(err, "closing")
	}
}

// page tempate for all pages.
func (app *Browser) page(name string) *template.Template {
	return template.Must(
		template.New(name).
			Funcs(templaterule.Funcs).
			Funcs(app.engine.TemplateFuncs()).
			Funcs(map[string]any{
				"asHTML": func(s string) template.HTML { return template.HTML(s) },
			}).
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
	if httpError(w, err, code) || httpError(w, t.Execute(&b, data), code) {
		return
	}
	_, _ = w.Write(b.Bytes())
}

func (b *Browser) stores(w http.ResponseWriter, req *http.Request) {
	path := path.Base(req.URL.Path)
	log.V(2).Info("store handler", "path", path, "query", req.URL.RawQuery)

	domain, err := b.engine.DomainErr(path)
	if httpError(w, err, http.StatusNotFound) {
		return
	}
	store, err := b.engine.StoreErr(domain.String())
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
