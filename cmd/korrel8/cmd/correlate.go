/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/korrel8/korrel8/internal/pkg/decoder"
	"github.com/korrel8/korrel8/internal/pkg/openshift"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/unique"
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
		paths := must(e.Graph().ShortestPaths(start, goal))
		starters := korrel8.NewSetResult(start)

		// FIXME include constraint

		if len(args) == 3 { // Get starters using query
			query := must(url.Parse(args[2]))
			store := e.Store(start.Domain())
			if store == nil {
				check(fmt.Errorf("domain has no store: %v", start.Domain()))
			}
			store.Get(ctx, query, starters)
		} else { // Read starters from stdin
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
		}
		queries := unique.NewList[url.URL]()
		for _, path := range paths {
			queries.Append(must(e.Follow(ctx, starters.List(), nil, path))...)
		}
		printResult(e, goal, queries.List)
	},
}

func printResult(e *engine.Engine, goal korrel8.Class, queries []korrel8.Query) {
	rewrite := func(q *korrel8.Query) *url.URL { return q }
	if *consoleFlag {
		d := goal.Domain()

		if transform := d.URLRewriter("console"); transform != nil {
			c := k8sClient(restConfig())
			base := must(openshift.ConsoleURL(ctx, c))
			rewrite = func(q *korrel8.Query) *url.URL { return base.ResolveReference(transform.FromQuery(q)) }
		}
	}
	for _, q := range queries {
		fmt.Println(rewrite(&q))
	}
}

var (
	consoleFlag  *bool
	intervalFlag *time.Duration
	endTime      TimeFlag
)

func init() {
	rootCmd.AddCommand(correlateCmd)
	consoleFlag = correlateCmd.Flags().Bool("console", false, "Print openshift console URLs instead of queries")
	intervalFlag = correlateCmd.Flags().Duration("interval", time.Minute*10, "limit results to this interval")
	correlateCmd.Flags().Var(&endTime, "time", "find results up to this time")
}
