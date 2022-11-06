/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get DOMAIN QUERY",
	Short: "Execute QUERY in the default store for DOMAIN and print the results",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		domainName, queryString := args[0], args[1]
		e := newEngine()
		domainName, _, _ = strings.Cut(domainName, "/") // Allow a class name, extract the domain.
		store := e.Store(e.Domain(domainName))
		if store == nil {
			check(fmt.Errorf("no store for domain %v", domainName))
		}
		query, err := url.Parse(queryString)
		check(err, "invalid query URL: %v", queryString)
		result := newPrinter(os.Stdout)
		check(store.Get(context.Background(), query, result))
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
