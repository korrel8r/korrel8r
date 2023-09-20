// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package browser implements an HTML UI for web browsers.
package browser

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"context"

	_ "net/http/pprof"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/openshift"
	"github.com/korrel8r/korrel8r/pkg/openshift/console"
	"github.com/korrel8r/korrel8r/pkg/rules"
	"go.uber.org/multierr"
)

var (
	log = logging.Log()
	//go:embed templates
	templates embed.FS
	//go:embed images
	images embed.FS
)

// Browser implements HTTP handlers for web browsers.
type Browser struct {
	engine     *engine.Engine
	console    *console.Console
	router     *gin.Engine
	images     http.FileSystem
	dir, files string
}

func New(e *engine.Engine, router *gin.Engine) (*Browser, error) {
	b := &Browser{
		engine: e,
		router: router,
		images: http.FS(must.Must1(fs.Sub(images, "images"))),
	}
	var err error
	if b.dir, err = os.MkdirTemp("", "korrel8r"); err == nil {
		log.V(1).Info("working directory", "dir", b.dir)
		b.files = filepath.Join(b.dir, "files")
		err = os.Mkdir(b.files, 0700)
	}
	if err != nil {
		return nil, err
	}
	kc, _, err := k8s.NewClient()
	if err != nil {
		return nil, err
	}
	consoleURL, err := openshift.ConsoleURL(context.Background(), kc)
	if err != nil {
		return nil, err
	}
	b.console = console.New(consoleURL, e)
	c := &correlate{browser: b}

	tmpl := template.Must(template.New("").
		Funcs(rules.Funcs).
		Funcs(b.engine.TemplateFuncs()).
		Funcs(map[string]any{
			"asHTML":         func(s string) template.HTML { return template.HTML(s) },
			"queryToConsole": func(q korrel8r.Query) *url.URL { return c.checkURL(c.browser.console.QueryToConsoleURL(q)) },
		}).
		ParseFS(templates, "templates/*.tmpl"))
	router.SetHTMLTemplate(tmpl)
	router.GET("/", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/correlate") })
	router.GET("/correlate", c.HTML)
	router.Static("/files", b.files)
	router.StaticFS("/images", b.images)
	router.GET("/stores/:domain", b.stores)
	return b, nil
}

// Close should be called on shutdown to clean up external resources.
func (b *Browser) Close() {
	if err := os.RemoveAll(b.dir); err != nil {
		log.Error(err, "closing")
	}
}

func (b *Browser) stores(c *gin.Context) {
	domain, err := b.engine.DomainErr(c.Param("domain"))
	if httpError(c, err, http.StatusNotFound) {
		return
	}
	stores := b.engine.StoresFor(domain)
	if len(stores) == 0 {
		_ = httpError(c, korrel8r.StoreNotFoundErr{Domain: domain}, http.StatusNotFound)
		return
	}
	query, err := domain.Query(c.Request.URL.Query().Get("query"))
	if httpError(c, err, http.StatusNotFound) {
		return
	}
	result := korrel8r.NewResult(query.Class())
	for _, store := range stores {
		err = multierr.Append(err, store.Get(context.Background(), query, result))
	}
	c.HTML(http.StatusOK, "stores.html.tmpl", map[string]any{"query": query, "err": err, "result": result.List()})
}

func httpError(c *gin.Context, err error, code int) bool {
	if err != nil {
		_ = c.Error(err)
		c.HTML(code, "error.html.tmpl", c)
		log.Error(err, "page error")
	}
	return err != nil
}
