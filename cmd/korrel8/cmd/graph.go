/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/spf13/cobra"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

// graphCmd represents the draw command
var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Graph of correlation rules.",
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		b, err := dot.MarshalMulti(e.Graph(), "", "", "  ")
		check(err)
		if *httpFlag == "" {
			fmt.Println(string(b))
		} else {
			cmd := exec.Command("dot", "-x", "-Tsvg")
			cmd.Stdin = bytes.NewBuffer(b)
			img, err := cmd.Output()
			check(err)
			fmt.Printf("web server listening on %v", *httpFlag)
			check(http.ListenAndServe(*httpFlag, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "image/svg+xml")
				w.Write(img)
			})))
		}
	},
}

var (
	httpFlag *string
)

func init() {
	rootCmd.AddCommand(graphCmd)
	httpFlag = graphCmd.Flags().String("http", "", "show graph via HTTP server at host:port")
}
