/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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
		domain, ok := e.Domains[domainName]
		if !ok {
			check(fmt.Errorf("unknown domain name %q", domainName))
		}
		store, ok := e.Stores[domain.String()]
		if !ok {
			check(fmt.Errorf("no store for domaina %q", domainName))
		}
		query := domain.NewQuery()
		err := json.Unmarshal([]byte(queryString), query)
		check(err, "bad query %q: %v", queryString, err)
		result := newPrinter(os.Stdout)
		check(store.Get(context.Background(), query, result))
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
