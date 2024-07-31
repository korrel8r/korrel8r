// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/build"
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
		gin.SetMode(gin.ReleaseMode)
		router := gin.New()
		router.Use(gin.Recovery())
		r, err := rest.New(engine, configs, router)
		must.Must(err)
		defer r.Close()
		s.Handler = router
		if *profileFlag {
			pprof.Register(router)
		}
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
	profileFlag         *bool
)

const (
	profileEnv = "KORREL8R_PROFILE"
)

func init() {
	rootCmd.AddCommand(webCmd)
	httpFlag = webCmd.Flags().String("http", "", "host:port address for insecure http listener")
	httpsFlag = webCmd.Flags().String("https", "", "host:port address for secure https listener")
	certFlag = webCmd.Flags().String("cert", "", "TLS certificate file (PEM format) for https")
	keyFlag = webCmd.Flags().String("key", "", "Private key (PEM format) for https")
	specFlag = webCmd.Flags().String("spec", "", "Dump swagger spec to a file, '-' for stdout.")
	profileDefault, _ := strconv.ParseBool(os.Getenv(profileEnv))
	profileFlag = webCmd.Flags().Bool("profile", profileDefault, "Enable HTTP profiling, see https://pkg.go.dev/net/http/pprof")
}
