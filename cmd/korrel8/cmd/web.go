package cmd

import (
	"net/http"

	"github.com/korrel8/korrel8/internal/pkg/webui"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start a web UI server.",
	Run: func(_ *cobra.Command, args []string) {
		e := newEngine()
		cfg := restConfig()
		ui := must(webui.New(e, cfg, k8sClient(cfg)))
		defer ui.Close()
		mux := http.NewServeMux()
		ui.HandlerFuncs(mux)
		log.Info("web ui listening", "addr", *httpAddr)
		check(http.ListenAndServe(*httpAddr, mux))
	},
}

var httpAddr *string

func init() {
	rootCmd.AddCommand(webCmd)
	httpAddr = webCmd.Flags().String("http", ":8080", "host:port address for web UI server")
}
