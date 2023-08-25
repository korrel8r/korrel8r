// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/browser"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web [flags]",
	Short: "Start a web server to interact with korrel8r from a browser.",
	Run: func(_ *cobra.Command, args []string) {
		gin.DefaultWriter = logging.LogWriter()
		if os.Getenv(gin.EnvGinMode) == "" { // Don't override an explicit env setting.
			gin.SetMode(gin.ReleaseMode)
			if *verbose >= 3 { // Enable gin debug mode and logging
				gin.SetMode(gin.DebugMode)
			}
		}
		router := gin.New()
		router.Use(gin.Recovery())
		if *verbose >= 2 {
			router.Use(gin.Logger())
		}
		engine := newEngine()
		if *serveHTML {
			defer must.Must1(browser.New(engine, router)).Close()
		}
		if *serveREST {
			defer must.Must1(api.New(engine, router)).Close()
		}
		log.Info("listening for http", "addr", *httpAddr)
		must.Must(http.ListenAndServe(*httpAddr, router))
	},
}

var (
	httpAddr             *string
	serveHTML, serveREST *bool
)

func init() {
	rootCmd.AddCommand(webCmd)
	httpAddr = webCmd.Flags().String("http", ":8080", "host:port address for web UI server")
	serveHTML = webCmd.Flags().Bool("html", true, "serve human-readabe HTML web pages")
	serveREST = webCmd.Flags().Bool("rest", true, "serve machine readable REST API")
}
