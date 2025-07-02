// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/mcp"
	"github.com/spf13/cobra"
)

// mcpCmd represents the mcp command
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP server for Korrel8r",
	Long: `Run korrel8r as an MCP stdio server that provides korrel8r resources and tools.
For an HTTP streaming or SSE server use the 'web' command with the '--mcp' flag.
`,
	Run: func(cmd *cobra.Command, args []string) {
		engine, _ := newEngine()
		log.Info("MCP server starting on stdio.")
		must.Must(mcp.ServeStdio(engine))
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	// FIXME
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// mcpCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// mcpCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
