// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"context"
	"os"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/mcp"
	mcplib "github.com/modelcontextprotocol/go-sdk/mcp"
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
		engine, _ := newEngine()
		server := mcp.NewServer(engine)
		if mcpDumpFlag {
			mcpDump(server)
			return
		}
		log.Info("MCP server starting on stdio.")
		must.Must(server.ServeStdio(context.Background()))
	},
}

var mcpDumpFlag bool

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.Flags().BoolVar(&mcpDumpFlag, "dump", false, "Dump MCP metadata to stdout")
}

func mcpDump(s *mcp.Server) {
	ctx := context.Background()
	ct, st := mcplib.NewInMemoryTransports()
	ss := must.Must1(s.Connect(ctx, st))
	c := mcplib.NewClient(&mcplib.Implementation{Name: "client"}, nil)
	cs := must.Must1(c.Connect(ctx, ct))
	defer func() { _ = cs.Close(); _ = ss.Wait() }()
	p := newPrinter(os.Stdout)
	tools := must.Must1(cs.ListTools(ctx, &mcplib.ListToolsParams{}))
	p.Print(tools)
	resources := must.Must1(cs.ListResources(ctx, &mcplib.ListResourcesParams{}))
	p.Print(resources)
	prompts := must.Must1(cs.ListPrompts(ctx, &mcplib.ListPromptsParams{}))
	p.Print(prompts)
}
