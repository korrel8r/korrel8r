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
	"sigs.k8s.io/yaml"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get DOMAIN QUERY",
	Short: "Execute QUERY in the default store for DOMAIN and print the results",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		domain, query := args[0], args[1]
		e := engine()
		store := e.Stores[domain]
		if store == nil {
			exitErr(fmt.Errorf("unknown domain name %q", domain))
		}
		result := must(store.Query(context.Background(), query))
		switch *output {
		case "json":
			e := json.NewEncoder(os.Stdout)
			if *pretty {
				e.SetIndent("", "  ")
			}
			exitErr(e.Encode(result))
		case "yaml":
			for _, o := range result {
				fmt.Printf("---\n%s", must(yaml.Marshal(o)))
			}
		default:
			exitErr(fmt.Errorf("invalid output type: %v", *output))
		}
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
