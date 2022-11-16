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
	Use:   "get DOMAIN QUERY_URL [NAME=VALUE...]",
	Short: "Execute QUERY_URL in the default store for DOMAIN and print the results",
	Long: `
QUERY_URL is a valid URL query for this domain.
Optional NAME=VALUE arguments are added to URL query.
`,
	Args: cobra.MinimumNArgs(2),
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
		q := query.Query()
		for _, nv := range args[2:] {
			n, v, ok := strings.Cut(nv, "=")
			if !ok {
				check(fmt.Errorf("not a name=value argument: %v", nv))
			}
			q.Set(n, v)
		}
		query.RawQuery = q.Encode()
		result := newPrinter(os.Stdout)
		check(store.Get(context.Background(), query, result))
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
