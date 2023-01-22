/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/korrel8r/korrel8r/internal/pkg/decoder"
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"

	"github.com/spf13/cobra"
)

// correlateCmd represents the correlate command
var correlateCmd = &cobra.Command{
	Use:   "correlate START_CLASS GOAL_CLASS [START_QUERY]",
	Short: "Correlate from START_CLASS to GOAL_CLASS. Use START_QUERY if present, read start object from stdin if not",
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		start, goal := must.Must1(e.Class(args[0])), must.Must1(e.Class(args[1]))
		paths := must.Must1(e.Graph().ShortestPaths(start, goal))
		log.V(1).Info("found paths", "paths", paths, "count", len(paths))

		starters := korrel8r.NewResult(start)
		if len(args) > 2 { // Get starters using query
			store := must.Must1(e.StoreErr(
				start.Domain().String()))
			query := start.Domain().Query(start)
			must.Must(json.Unmarshal([]byte(args[2]), query))
			must.Must(store.Get(ctx, query, starters))
		} else { // Read starters from stdin
			dec := decoder.New(os.Stdin)
			for {
				o := start.New()
				if err := dec.Decode(&o); err != nil {
					if err == io.EOF {
						break
					}
					must.Must(fmt.Errorf("error reading from stdin: %w", err))
				}
				starters.Append(o)
			}
		}
		var results engine.Results
		must.Must(e.FollowAll(ctx, starters.List(), nil, paths, &results))

		result := results.Last()
		if result == nil {
			must.Must(fmt.Errorf("no results"))
		}
		printer := newPrinter(os.Stdout)
		if *getFlag { // Get the objects and print those
			store := must.Must1(e.StoreErr(goal.Domain().String()))
			for _, q := range result.Queries.List {
				must.Must(store.Get(ctx, q, printer))
			}
		} else {
			for _, q := range results.Last().Queries.List {
				printer.Print(q)
			}
		}
	},
}

var (
	getFlag *bool
	endTime TimeFlag
)

func init() {
	rootCmd.AddCommand(correlateCmd)
	correlateCmd.Flags().Var(&endTime, "time", "find results up to this time")
	getFlag = correlateCmd.Flags().Bool("get", false, "Get objects from query")
}
