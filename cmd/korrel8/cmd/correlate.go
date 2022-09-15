/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
)

// correlateCmd represents the correlate command
var correlateCmd = &cobra.Command{
	// FIXME need string form for start/goal domain/class and start data.
	Use:   "correlate DOMAIN START",
	Short: "Correlate the objects in file START from DOMAIN to GOAL  a file of starting objects, return a query for correlated objects. '-' means use stdin",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		f := open(startFile)
		defer f.Close()
		panic("FIXME")
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
