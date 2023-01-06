package cmd

import (
	"net/http"

	"github.com/korrel8/korrel8/internal/pkg/must"
	"github.com/korrel8/korrel8/internal/pkg/webui"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start a web UI server.",
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
