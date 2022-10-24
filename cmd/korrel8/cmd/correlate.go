/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/korrel8/korrel8/internal/pkg/decoder"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/spf13/cobra"
)

// correlateCmd represents the correlate command
var correlateCmd = &cobra.Command{
	Use:   "correlate START_CLASS GOAL_CLASS [QUERY]",
	Short: "Correlate from START_CLASS to GOAL_CLASS, using QUERY or object data from stdin",
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		start, goal := must(e.ParseClass(args[0])), must(e.ParseClass(args[1]))
		path := e.Graph.ShortestPath(start, goal)
		starters := korrel8.NewSetResult(start)

		if len(args) == 3 { // Get starters using query
			query := must(korrel8.ParseQuery(start.Domain(), args[2]))
			store := must(e.Store(start.Domain()))
			store.Get(context.Background(), query, starters)
		} else { // Get starters from stdin
			dec := decoder.New(os.Stdin)
			for {
				o := start.New()
				if err := dec.Decode(&o); err != nil {
					if err == io.EOF {
						break
					}
					check(fmt.Errorf("error reading from stdin: %w", err))
				}
				starters.Append(o)
			}
			// FIXME allow constraint
			startObjs := starters.List()
			debug.Info("following objects", "objects", startObjs)
			queries := must(e.Follow(context.Background(), startObjs, nil, path))
			debug.Info("resulting queries", "queries", queries)
			if *resultType.Value == "data" {
				store := must(e.Store(goal.Domain()))
				printer := newPrinter(os.Stdout)
				for _, q := range queries {
					err := store.Get(context.Background(), q, printer)
					check(err, "%v query %v: %w", goal.Domain(), q, err)
				}
			} else {
				for _, q := range queries {
					empty := url.URL{}
					switch *resultType.Value {
					case "rest":
						fmt.Println(q.REST(&empty))
					case "string":
						fmt.Println(q.String())
					case "console":
						fmt.Println(q.Console(&empty))
					default:
						check(fmt.Errorf("invalid value for --result"))
					}
				}
			}
		}
	},
}

var (
	baseURL    URLFlag
	interval   *time.Duration
	endTime    TimeFlag
	resultType = NewEnumFlag("string", "rest", "console", "data")
)

func init() {
	rootCmd.AddCommand(correlateCmd)
	correlateCmd.Flags().VarP(resultType, "result", "r", "form of result: query string, REST URL, console URL or object data")

	// FIXME implement time and interval
	interval = correlateCmd.Flags().Duration("interval", time.Minute*10, "limit results to this interval")
	correlateCmd.Flags().Var(&endTime, "time", "find results up to this time")
}
