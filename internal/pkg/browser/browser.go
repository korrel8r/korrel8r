// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package browser implements an HTML UI for web browsers.
package browser

import (
	"context"
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/build"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/openshift"
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
	version    string
	engine     *engine.Engine
	console    *openshift.Console
	router     *gin.Engine
	images     http.FileSystem
	dir, files string
}

func New(e *engine.Engine, router *gin.Engine) (*Browser, error) {
	b := &Browser{
		engine:  e,
		router:  router,
		version: build.Version,
		images:  http.FS(must.Must1(fs.Sub(images, "images"))),
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
	cfg, err := k8s.GetConfig()
	if err != nil {
		return nil, err
	}
	kc, err := k8s.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	consoleURL, err := openshift.ConsoleURL(context.Background(), kc)
	if err != nil {
		return nil, err
	}
	b.console = openshift.NewConsole(consoleURL, kc)
	c := &correlate{browser: b}

	tmpl := template.Must(template.New("").Funcs(b.engine.TemplateFuncs()).ParseFS(templates, "templates/*.tmpl"))
	router.SetHTMLTemplate(tmpl)
	router.GET("/", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/correlate") })
	router.GET("/correlate", c.HTML)
	router.Static("/files", b.files)
	router.StaticFS("/images", b.images)
	// Display errors converted into URLs.
	router.GET("/error", func(c *gin.Context) {
		httpError(c, errors.New(c.Request.URL.Query().Get("err")), http.StatusNotFound)
	})

	return b, nil
}

// Close should be called on shutdown to clean up external resources.
func (b *Browser) Close() {
	if err := os.RemoveAll(b.dir); err != nil {
		log.Error(err, "Closing")
	}
}

func httpError(c *gin.Context, err error, code int) bool {
	if err != nil {
		_ = c.Error(err)
		c.HTML(code, "error.html.tmpl", c)
		log.Error(err, "Page error")
	}
	return err != nil
}
