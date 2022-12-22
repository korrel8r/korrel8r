/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get DOMAIN URI_REF [NAME=VALUE...]",
	Short: "Execute URI_REF in the default store for DOMAIN and print the results",
	Long: `
URI_REF is a URI reference to select objects in this domain.
Optional NAME=VALUE arguments are added to URL query.
`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		domainName, _, _ := strings.Cut(args[0], "/") // Allow a class name, extract the domain.
		store, err := e.Store(domainName)
		check(err)
		u := must(referenceArgs(args[1:]))
		log.V(1).Info("getting", "query", u)
		result := newPrinter(os.Stdout)
		check(store.Get(context.Background(), u, result))
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
