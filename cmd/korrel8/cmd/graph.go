/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Graph of correlation rules.",
	Run: func(_ *cobra.Command, args []string) {
		e := newEngine()
		gv := must(dot.MarshalMulti(e.Graph(), "", "", "  "))
		if *dir == "" && *httpFlag == "" { // Dump to stdout

			_ = must(os.Stdout.Write(gv))
			return
		}
		// Set up directory
		if *dir == "" {
			must(os.MkdirTemp("", *dir))
			log.Info("output directory", "dir", *dir)
		}
		check(os.Chdir(*dir))

		// Write DOT graph to .gv
		base := "rulegraph"
		gvFile := base + ".gv"
		check(os.WriteFile(gvFile, gv, 0664))

		// Write image
		imageFile := base + ".png"
		cmd := exec.Command("dot", "-x", "-Tpng", "-o", imageFile, gvFile)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		check(cmd.Run())

		// Write HTML file
		htmlFile := must(os.Create("index.html"))
		fmt.Fprintf(htmlFile, `<!DOCTYPE html PUBLIC " - //W3C//DTD xhtml 1.0 Strict//EN"
	"http://www.w3.org/1999/xhtml">
<head>
  <title>Korrel8ion Rule Graph (%v)</title>
	<meta http-equiv="Content-Type" content+"text/html; charset=utf-8/>
</head>
<body>
<h1>Korrel8ion Rule Graph (%v)</h1>
<img src="%v" usemap="#%v" />
`, e.Name(), e.Name(), imageFile, e.Name())

		cmd = exec.Command("dot", "-x", "-Tcmap", gvFile)
		cmd.Stderr = os.Stderr
		cmd.Stdout = htmlFile
		check(cmd.Run())

		fmt.Fprintf(htmlFile, "\n</body></html>\n")

		if *httpFlag != "" {
			log.Info("web server listening", "addr", "*httpFlag")
			check(http.ListenAndServe(*httpFlag, http.FileServer(http.Dir("."))))
		}
	},
}

var (
	httpFlag *string
	dir      *string
)

func init() {
	rootCmd.AddCommand(graphCmd)
	httpFlag = graphCmd.Flags().String("http", "", "show graph via HTTP server at host:port")
	dir = graphCmd.Flags().StringP("dir", "d", "", "dirctory to write graph files")
}
