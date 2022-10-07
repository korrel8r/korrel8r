/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
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
		domain, query := args[0], args[1]
		e := newEngine()
		store := e.Stores[domain]
		if store == nil {
			check(fmt.Errorf("unknown domain name %q", domain))
		}
		result := newPrinter(os.Stdout)
		check(store.Get(context.Background(), query, result))
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
