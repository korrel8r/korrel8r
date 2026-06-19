// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Command korrel8r-docgen generates markdown documentation for the MCP API.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	mcpserver "github.com/korrel8r/korrel8r/pkg/mcp"
	"github.com/korrel8r/korrel8r/pkg/session"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Build a minimal engine — we only need the tool metadata, not real stores.
	d := mock.NewDomain("mock", "a")
	e, err := engine.Build().Domains(d).Stores(mock.NewStore(d)).Engine()
	if err != nil {
		return fmt.Errorf("building engine: %w", err)
	}

	// Create an MCP server and list its tools via an in-memory client.
	s := mcpserver.NewServer(session.NewSingleManager(e))
	ct, st := mcp.NewInMemoryTransports()
	ctx := context.Background()
	ss, err := s.Connect(ctx, st, nil)
	if err != nil {
		return fmt.Errorf("server connect: %w", err)
	}
	defer func() { _ = ss.Wait() }()

	c := mcp.NewClient(&mcp.Implementation{Name: "docgen"}, nil)
	cs, err := c.Connect(ctx, ct, nil)
	if err != nil {
		return fmt.Errorf("client connect: %w", err)
	}
	defer func() { _ = cs.Close() }()

	result, err := cs.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return fmt.Errorf("listing tools: %w", err)
	}

	// Sort tools by name for stable output.
	tools := result.Tools
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })

	w := os.Stdout
	fmt.Fprintln(w, "Korrel8r provides an [MCP](https://modelcontextprotocol.io/) server with the following tools.")
	fmt.Fprintln(w)

	// Table of contents
	for _, t := range tools {
		fmt.Fprintf(w, "- [%s](#%s)\n", t.Name, t.Name)
	}
	fmt.Fprintln(w)

	for _, t := range tools {
		fmt.Fprintf(w, "## %s\n\n", t.Name)
		fmt.Fprintln(w, strings.TrimSpace(t.Description))
		fmt.Fprintln(w)

		if t.InputSchema != nil {
			writeSchemaTable(w, "Input", t.InputSchema)
		}
		if t.OutputSchema != nil {
			writeSchemaTable(w, "Output", t.OutputSchema)
		}
	}
	return nil
}

// writeSchemaTable renders a JSON Schema as a markdown parameter table.
func writeSchemaTable(w *os.File, label string, schema any) {
	data, err := json.Marshal(schema)
	if err != nil {
		return
	}
	var s jsonSchema
	if err := json.Unmarshal(data, &s); err != nil {
		return
	}
	if len(s.Properties) == 0 {
		return
	}

	requiredSet := map[string]bool{}
	for _, r := range s.Required {
		requiredSet[r] = true
	}

	fmt.Fprintf(w, "### %s parameters\n\n", label)
	fmt.Fprintln(w, "| Parameter | Type | Required | Description |")
	fmt.Fprintln(w, "|-----------|------|----------|-------------|")

	// Sort properties for stable output.
	names := make([]string, 0, len(s.Properties))
	for name := range s.Properties {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		prop := s.Properties[name]
		typ := schemaType(prop)
		req := ""
		if requiredSet[name] {
			req = "yes"
		}
		desc := prop.Description
		fmt.Fprintf(w, "| `%s` | %s | %s | %s |\n", name, typ, req, desc)
	}
	fmt.Fprintln(w)
}

type jsonSchema struct {
	Properties map[string]jsonSchemaProperty `json:"properties"`
	Required   []string                      `json:"required"`
}

type jsonSchemaProperty struct {
	Type        any    `json:"type"`
	Description string `json:"description"`
	Ref         string `json:"$ref"`
	Items       *struct {
		Type string `json:"type"`
		Ref  string `json:"$ref"`
	} `json:"items"`
}

func schemaType(p jsonSchemaProperty) string {
	if p.Ref != "" {
		return refName(p.Ref)
	}
	switch t := p.Type.(type) {
	case string:
		if t == "array" && p.Items != nil {
			if p.Items.Ref != "" {
				return refName(p.Items.Ref) + "[]"
			}
			return p.Items.Type + "[]"
		}
		return t
	case []any:
		// Nullable types come as ["null", "string"] — filter out null and show the real type.
		var real []string
		for _, v := range t {
			s := fmt.Sprint(v)
			if s != "null" {
				real = append(real, s)
			}
		}
		if len(real) == 1 {
			// Recurse to handle array items
			p.Type = real[0]
			return schemaType(p)
		}
		return strings.Join(real, " | ")
	default:
		return "object"
	}
}

func refName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}
