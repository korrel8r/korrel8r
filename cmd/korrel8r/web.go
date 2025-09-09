// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/build"
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/mcp"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web [flags]",
	Short: "Start REST server. Listening address must be  provided via --http or --https.",
	Args:  cobra.NoArgs,
	Run: func(_ *cobra.Command, args []string) {
		if *specFlag != "" {
			var out = os.Stdout
			if *specFlag != "-" {
				out = must.Must1(os.Create(*specFlag))
				defer func() { _ = out.Close() }()
			}
			j := json.NewEncoder(out)
			j.SetIndent("", "  ")
			must.Must(j.Encode(rest.Spec))
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
		gin.SetMode(gin.ReleaseMode)
		router := gin.New()
		router.Use(gin.Recovery())

		if *restFlag {
			rest := must.Must1(rest.New(engine, configs, router))
			log.V(0).Info("Enable REST endpoint", "path", rest.BasePath)
		}
		if *mcpFlag {
			router.POST(mcp.StreamablePath, gin.WrapH(mcp.NewServer(engine).HTTPHandler()))
			log.V(0).Info("Enable MCP Streamable endpoint", "path", mcp.StreamablePath)
		}
		s.Handler = router
		if *profileFlag == "http" {
			rest.WebProfile(router)
		}
		log := log.WithValues("addr", s.Addr, "version", build.Version, "configuration", *configFlag)
		if *httpFlag != "" {
			log.Info("Server: Listening for http")
			must.Must(s.ListenAndServe())
		} else {
			log.Info("Server: Listening for https")
			must.Must(s.ListenAndServeTLS(*certFlag, *keyFlag))
		}
	},
}

var (
	httpFlag, httpsFlag *string
	certFlag, keyFlag   *string
	specFlag            *string
	mcpFlag             *bool
	restFlag            *bool
	WebProfile          func()
)

func init() {
	rootCmd.AddCommand(webCmd)
	httpFlag = webCmd.Flags().String("http", "", "host:port address for insecure http listener")
	httpsFlag = webCmd.Flags().String("https", "", "host:port address for secure https listener")
	certFlag = webCmd.Flags().String("cert", "", "TLS certificate file (PEM format) for https")
	keyFlag = webCmd.Flags().String("key", "", "Private key (PEM format) for https")
	specFlag = webCmd.Flags().String("spec", "", "Dump OpenAPI specification to a file, '-' for stdout.")
	restFlag = webCmd.Flags().Bool("rest", true, "Enable HTTP REST server on "+rest.BasePath)
	mcpFlag = webCmd.Flags().Bool("mcp", true, "Enable MCP streamable HTTP protocol on "+mcp.StreamablePath)
}
