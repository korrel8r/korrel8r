// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/client/pkg/browser"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web REMOTE-URL [LISTEN-ADDR]",
	Short: "Connect to REMOTE-URL and run an HTTP server listening on LISTEN-ADDR (default :8080)",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(_ *cobra.Command, args []string) error {
		remoteURL := args[0]
		gin.DefaultWriter = log.Writer()
		gin.SetMode(gin.ReleaseMode)
		gin.DisableConsoleColor()
		router := gin.New()
		router.Use(gin.Recovery())
		router.Use(gin.Logger())
		b, err := browser.New(newClient(remoteURL), router)
		if err != nil {
			return err
		}
		defer b.Close()
		s := http.Server{
			Addr:    *addr,
			Handler: router,
		}
		log.Printf("Listening on %v, connected to %v\n", *addr, remoteURL)
		return s.ListenAndServe()
	},
}

var (
	addr *string
)

func init() {
	rootCmd.AddCommand(webCmd)
	addr = webCmd.Flags().StringP("addr", "a", ":8080", "Listeing address for web server")
}
