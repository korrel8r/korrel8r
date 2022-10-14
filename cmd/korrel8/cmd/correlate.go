/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/korrel8/korrel8/internal/pkg/decoder"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

// correlateCmd represents the correlate command
var correlateCmd = &cobra.Command{
	Use:   "correlate START_CLASS GOAL_CLASS [START_FILE]",
	Short: "Correlate from START_CLASS to GOAL_CLASS starting from in START_FILE. '-' means use stdin",
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		startClass, goalClass := must(e.ParseClass(args[0])), must(e.ParseClass(args[1]))
		startReader := os.Stdin
		if len(args) > 2 && args[2] != "-" {
			startReader = open(args[2])
			defer startReader.Close()
		}
		start := startClass.New()
		check(decoder.New(startReader).Decode(&start))

		paths := e.Rules.FindPaths(startClass, goalClass)

		var (
			queries []korrel8.Query
			merr    error
		)

		for _, p := range paths {
			q, err := e.Follow(context.Background(), start, nil, p)
			merr = multierr.Append(merr, err)
			queries = append(queries, q...)
		}
		for _, err := range multierr.Errors(merr) {
			log.V(1).Error(err, "ignored")
		}
		fmt.Printf("\nresulting queries: %v\n\n", queries)
	},
}

func init() {
	rootCmd.AddCommand(correlateCmd)
}
