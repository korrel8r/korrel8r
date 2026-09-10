// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"os"

	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/spf13/cobra"
)

var domainsCmd = &cobra.Command{
	Use:   "domains",
	Short: "List all domains with their descriptions and store configurations.",
	Long: `List all domains with their descriptions and store configurations.

This is the same information returned by the /domains REST endpoint.
Store configurations include an "error" entry for stores that could not be loaded.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		newPrinter(os.Stdout).Print(rest.ListDomains(e))
	},
}

func init() {
	rootCmd.AddCommand(domainsCmd)
}
