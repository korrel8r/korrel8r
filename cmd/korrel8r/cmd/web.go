package cmd

import (
	"net/http"

	"github.com/korrel8r/korrel8r/cmd/korrel8r/webapp/browser"
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web [flags]",
	Short: "Start a web server to interact with korrel8r from a browser.",
	Run: func(_ *cobra.Command, args []string) {
		e := newEngine()
		cfg := restConfig()
		browser := must.Must1(browser.New(e, cfg, k8sClient(cfg)))
		browser.Register(http.DefaultServeMux)
		defer browser.Close()
		// FIXME
		// rest := must.Must1(rest.New(e, cfg, k8sClient(cfg)))
		log.Info("web ui listening", "addr", *httpAddr)
		must.Must(http.ListenAndServe(*httpAddr, nil))
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
