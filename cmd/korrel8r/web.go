// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/browser"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web [ADDR] [flags]",
	Short: "Start web server listening on host:port address ADDR (default :8080 for http, :8443 for https)",
	Args:  cobra.NoArgs,
	Run: func(_ *cobra.Command, args []string) {
		if *httpFlag == "" && *httpsFlag == "" {
			*httpFlag = ":8080" // Default if no port specified.
		}
		var s http.Server
		switch {
		case *httpFlag != "" && *httpsFlag != "":
			panic(fmt.Errorf("only one of --http or --https may be present"))
		case *httpFlag != "":
			s.Addr = *httpFlag
			if *certFlag != "" || *keyFlag != "" {
				panic(fmt.Errorf("--cert and --key not allowed with --http"))
			}
		case *httpsFlag != "":
			s.Addr = *httpsFlag
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
		if *verbose >= 2 {
			router.Use(gin.Logger())
		}
		if *htmlFlag {
			b := must.Must1(browser.New(engine, router, rootCmd.Version))
			defer b.Close()
		}
		if *restFlag {
			r := must.Must1(rest.New(engine, configs, router))
			defer r.Close()
		}
		s.Handler = router
		pprof.Register(router) // Enable profiling

		if *httpFlag != "" {
			log.Info("listening for http", "addr", s.Addr, "version", rootCmd.Version)
			must.Must(s.ListenAndServe())
		} else {
			log.Info("listening for https", "addr", s.Addr, "version", rootCmd.Version)
			must.Must(s.ListenAndServeTLS(*certFlag, *keyFlag))
		}
	},
}

var (
	htmlFlag, restFlag                     *bool
	httpFlag, httpsFlag, certFlag, keyFlag *string
)

func init() {
	rootCmd.AddCommand(webCmd)
	htmlFlag = webCmd.Flags().Bool("html", true, "serve human-readabe HTML web pages")
	restFlag = webCmd.Flags().Bool("rest", true, "serve machine readable REST API")
	httpFlag = webCmd.Flags().String("http", "", "host:port address for insecure http listener")
	httpsFlag = webCmd.Flags().String("https", "", "host:port address for secure https listener")
	certFlag = webCmd.Flags().String("cert", "", "TLS certificate file (PEM format) for https")
	keyFlag = webCmd.Flags().String("key", "", "Private key (PEM format) for https")
}
