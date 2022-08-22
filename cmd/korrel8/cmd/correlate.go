/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
)

// correlateCmd represents the correlate command
var correlateCmd = &cobra.Command{
	Use:   "correlate context-file store-type",
	Short: "Given a file of context objects, return a query for correlated objects",
	Long:  `If context-file is not given, use stdin`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		f, err := open(contextFile)
		exitErr(err)
		defer f.Close()
		// FIXME decode objects loop, add translators later...
	},
}

// FIXME take objects from other sources, e.g. oc output.

var (
	contextFile, storeType string
)

func init() {
	rootCmd.AddCommand(correlateCmd)
	correlateCmd.Flags().StringVar(&contextFile, "context", "", "file containing context objects")
	correlateCmd.Flags().StringVar(&storeType, "store", "", "store type for returned query")
	for _, f := range []string{"context", "store"} {
		correlateCmd.MarkFlagRequired(f)
	}
}
