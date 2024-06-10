// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/build"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/korrel8r/korrel8r/pkg/rest/docs"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web [flags]",
	Short: "Start REST server. Listening address must be  provided via --http or --https.",
	Args:  cobra.NoArgs,
	Run: func(_ *cobra.Command, args []string) {
		if *htmlFlag || !*restFlag {
			log.Info("DEPRECATED --html and --rest are deprecated. HTML server no longer supported.")
		}
		if *specFlag != "" {
			spec := docs.SwaggerInfo.ReadDoc()
			if *specFlag == "-" {
				fmt.Println(spec)
			} else {
				must.Must(os.WriteFile(*specFlag, []byte(spec), 0666))
			}
			return
		}

		if *httpFlag == "" && *httpsFlag == "" {
			*httpFlag = ":8080" // Default if no port specified.
		}
		var s http.Server
		switch {
		case *httpFlag != "" && *httpsFlag != "":
			panic(fmt.Errorf("only one of --http or --https may be present"))
		case *httpFlag != "":
			s.Addr = *httpFlag
			docs.SwaggerInfo.Schemes = []string{"http"}
			if *certFlag != "" || *keyFlag != "" {
				panic(fmt.Errorf("--cert and --key not allowed with --http"))
			}

		case *httpsFlag != "":
			s.Addr = *httpsFlag
			docs.SwaggerInfo.Schemes = []string{"https"}
			if *certFlag == "" || *keyFlag == "" {
				panic(fmt.Errorf("--cert and --key are required for https"))
			}
		}

		engine, configs := newEngine()
		gin.DefaultWriter = logging.LogWriter()
		gin.SetMode(gin.ReleaseMode)
		gin.DisableConsoleColor()
		router := gin.New()
		router.Use(gin.Recovery())
		r := must.Must1(rest.New(engine, configs, router))
		defer r.Close()
		s.Handler = router
		pprof.Register(router) // Enable profiling

		if *httpFlag != "" {
			log.Info("listening for http", "addr", s.Addr, "version", build.Version)
			must.Must(s.ListenAndServe())
		} else {
			log.Info("listening for https", "addr", s.Addr, "version", build.Version)
			must.Must(s.ListenAndServeTLS(*certFlag, *keyFlag))
		}
	},
}

var (
	httpFlag, httpsFlag *string
	certFlag, keyFlag   *string
	specFlag            *string
	htmlFlag, restFlag  *bool
)

func init() {
	rootCmd.AddCommand(webCmd)
	httpFlag = webCmd.Flags().String("http", "", "host:port address for insecure http listener")
	httpsFlag = webCmd.Flags().String("https", "", "host:port address for secure https listener")
	certFlag = webCmd.Flags().String("cert", "", "TLS certificate file (PEM format) for https")
	keyFlag = webCmd.Flags().String("key", "", "Private key (PEM format) for https")
	specFlag = webCmd.Flags().String("spec", "", "Dump swagger spec to a file, '-' for stdout.")

	// DEPRECATED: remove in future version.
	htmlFlag = webCmd.Flags().Bool("html", false, "DEPRECATED - use korrel8rcli instead.")
	restFlag = webCmd.Flags().Bool("rest", true, "DEPRECATED - always enabled.")
}
