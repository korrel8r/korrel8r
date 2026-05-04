// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"context"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/mcp"
	"github.com/korrel8r/korrel8r/pkg/session"
	"github.com/spf13/cobra"
)

// mcpCmd represents the mcp command
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP stdio server",
	Long: `Run korrel8r as an MCP server communicating via stdin/stdout.
Allows korrel8r to be run as a sub-process by an MCP tool.
For a HTTP streaming server use the 'web' command with the '--mcp' flag.
`,
	Run: func(cmd *cobra.Command, args []string) {
		configs := must.Must1(config.Load(*configFlag))
		e := must.Must1(newEngineWithConfigs(configs))
		server := mcp.NewServer(session.NewSingle(e))
		log.Info("MCP server starting on stdio.")
		must.Must(server.ServeStdio(context.Background()))
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}
