package cmd

import (
	"net/http"

	"github.com/korrel8r/korrel8r/cmd/korrel8r/webui"
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web [flags]",
	Short: "Start a web server to interact with korrel8r from a browser.",
	Run: func(_ *cobra.Command, args []string) {
		e := newEngine()
		cfg := restConfig()
		ui := must.Must1(webui.New(e, cfg, k8sClient(cfg)))
		defer ui.Close()
		log.Info("web ui listening", "addr", *httpAddr)
		must.Must(http.ListenAndServe(*httpAddr, ui.Mux))
	},
}

var httpAddr *string

func init() {
	rootCmd.AddCommand(webCmd)
	httpAddr = webCmd.Flags().String("http", ":8080", "host:port address for web UI server")
}
