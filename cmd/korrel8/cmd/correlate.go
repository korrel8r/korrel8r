/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
)

// correlateCmd represents the correlate command
var correlateCmd = &cobra.Command{
	Use:   "correlate [start file] [store type]",
	Short: "Given a file of starting objects, return a query for correlated objects. '-' means use stdin",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		f, err := open(startFile)
		exitErr(err)
		defer f.Close()
		// FIXME implement
	},
}

var (
	startFile, storeType string
)

func init() {
	rootCmd.AddCommand(correlateCmd)
	correlateCmd.Flags().StringVar(&startFile, "start", "", "file containing start objects")
	correlateCmd.Flags().StringVar(&storeType, "store", "", "store type for returned query")
	for _, f := range []string{"context", "store"} {
		correlateCmd.MarkFlagRequired(f)
	}
}
