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
	"github.com/korrel8r/korrel8r/internal/pkg/tlsprofile"
	"github.com/korrel8r/korrel8r/pkg/mcp"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/korrel8r/korrel8r/pkg/rest/auth"
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
			s.TLSConfig = must.Must1(tlsprofile.NewTLSConfig(*tlsMinVersionFlag, *tlsCipherSuitesFlag, *tlsCurvesFlag))
		}
		if *httpFlag != "" && (len(*tlsCipherSuitesFlag) > 0 || *tlsMinVersionFlag != "" || len(*tlsCurvesFlag) > 0) {
			panic(fmt.Errorf("--tls-min-version, --tls-cipher-suites, and --tls-curves are not allowed with --http"))
		}

		engine, configs := newEngine()
		gin.SetMode(gin.ReleaseMode)
		router := gin.New()
		router.Use(gin.Recovery())
		// Middleware to add authentication and timeout to the request context.
		router.Use(func(c *gin.Context) {
			ctx, cancel := engine.WithTimeout(auth.Context(c.Request), 0)
			defer cancel()
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		})

		var restAPI *rest.API
		if *restFlag {
			restAPI = must.Must1(rest.New(engine, configs, router))
			log.V(0).Info("REST endpoint", "path", restAPI.BasePath)
		}
		if *mcpFlag {
			router.Any(mcp.StreamablePath, gin.WrapH(mcp.NewServer(engine, restAPI).HTTPHandler()))
			log.V(0).Info("MCP Streamable endpoint", "path", mcp.StreamablePath)
		}
		if *mcpSSEFlag {
			router.Any(mcp.SSEPath, gin.WrapH(mcp.NewServer(engine, restAPI).SSEHandler()))
			log.V(0).Info("MCP SSE endpoint", "path", mcp.SSEPath)
		}
		s.Handler = router
		if profileTypeFlag.String() == "http" {
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
	mcpSSEFlag          *bool
	restFlag            *bool
	tlsMinVersionFlag   *string
	tlsCipherSuitesFlag *[]string
	tlsCurvesFlag       *[]string
	WebProfile          func()
)

func init() {
	rootCmd.AddCommand(webCmd)
	certFlag = webCmd.Flags().String("cert", "", "TLS certificate file (PEM format) for https")
	httpFlag = webCmd.Flags().String("http", "", "host:port address for insecure http listener")
	httpsFlag = webCmd.Flags().String("https", "", "host:port address for secure https listener")
	keyFlag = webCmd.Flags().String("key", "", "Private key (PEM format) for https")
	mcpFlag = webCmd.Flags().Bool("mcp", true, "Enable MCP streamable HTTP protocol on "+mcp.StreamablePath)
	mcpSSEFlag = webCmd.Flags().Bool("mcpSSE", true, "Enable MCP Server-Sent Events protocol server on "+mcp.SSEPath)
	restFlag = webCmd.Flags().Bool("rest", true, "Enable HTTP REST server on "+rest.BasePath)
	specFlag = webCmd.Flags().String("spec", "", "Write OpenAPI specification to a file, '-' for stdout.")
	tlsCipherSuitesFlag = webCmd.Flags().StringSlice("tls-cipher-suites", nil, "Comma-separated list of TLS cipher suites for https (IANA names)")
	tlsCurvesFlag = webCmd.Flags().StringSlice("tls-curves", nil, "Comma-separated list of TLS curves for https (e.g. CurveP256, CurveP384, CurveP521, X25519)")
	tlsMinVersionFlag = webCmd.Flags().String("tls-min-version", "", "Minimum TLS version for https (e.g. VersionTLS12, VersionTLS13)")
}
