/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get DOMAIN QUERY",
	Short: "Execute QUERY in the default store for DOMAIN and print the results",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		domain, query := korrel8.Domain(args[0]), args[1]
		store := stores[domain]
		result, err := store.Query(context.Background(), query)
		exitErr(err)
		switch *output {
		case "json":
			e := json.NewEncoder(os.Stdout)
			if *pretty {
				e.SetIndent("", "  ")
			}
			e.Encode(result)
		case "yaml":
			for _, o := range result {
				b, err := yaml.Marshal(o)
				exitErr(err)
				fmt.Printf("---\n%s", b)
			}
		default:
			exitErr(fmt.Errorf("invalid output type: %v", *output))
		}
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
