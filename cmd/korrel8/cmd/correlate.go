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
	"github.com/korrel8/korrel8/internal/pkg/openshift"
	"github.com/korrel8/korrel8/pkg/engine"
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
		path := e.Graph().ShortestPath(start, goal)
		starters := korrel8.NewSetResult(start)

		// FIXME include constraint

		if len(args) == 3 { // Get starters using query
			query := must(url.Parse(args[2]))
			store := must(e.Store(start.Domain()))
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
		queries := must(e.Follow(ctx, starters.List(), nil, path))
		printResult(e, goal, queries)
	},
}

func printResult(e *engine.Engine, goal korrel8.Class, queries []korrel8.Query) {
	formatter := func(q *korrel8.Query) *korrel8.Query { return q }
	if *formatFlag != "" {
		formatter = goal.Domain().Formatter(*formatFlag)
		if formatter == nil {
			check(fmt.Errorf("unknown URL format: %q", *formatFlag))
		}
		if *formatFlag == "console" { // FIXME this is messy
			c := k8sClient(restConfig())
			base := url.URL{
				Scheme: "https",
				Path:   "/",
				Host:   must(openshift.RouteHost(context.Background(), c, openshift.ConsoleNSName))}
			export1 := formatter
			formatter = func(ref *korrel8.Query) *url.URL { return base.ResolveReference(export1(ref)) }
		}
	}
	for _, q := range queries {
		fmt.Println(formatter(&q))
	}
}

var (
	formatFlag   *string
	intervalFlag *time.Duration
	endTime      TimeFlag
)

func init() {
	rootCmd.AddCommand(correlateCmd)
	formatFlag = correlateCmd.Flags().String("format", "", "format for URLs, e.g. console")
	intervalFlag = correlateCmd.Flags().Duration("interval", time.Minute*10, "limit results to this interval")
	correlateCmd.Flags().Var(&endTime, "time", "find results up to this time")
}
