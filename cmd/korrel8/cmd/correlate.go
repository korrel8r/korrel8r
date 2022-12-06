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

	"github.com/korrel8/korrel8/internal/pkg/decoder"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/graph"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/unique"
	"github.com/spf13/cobra"
)

// correlateCmd represents the correlate command
var correlateCmd = &cobra.Command{
	Use:   "correlate START_CLASS GOAL_CLASS [QUERY_URL [NAME=VALUE...]]",
	Short: "Correlate from START_CLASS to GOAL_CLASS. Use QUERY_URL if present, read object from stdin if not",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		start, goal := must(e.ParseClass(args[0])), must(e.ParseClass(args[1]))
		var paths []graph.MultiPath
		switch {
		case *allFlag:
			paths = must(e.Graph().AllPaths(start, goal))
		case *kFlag > 0:
			paths = must(e.Graph().KShortestPaths(start, goal, *kFlag))
		default:
			paths = must(e.Graph().ShortestPaths(start, goal))
		}
		log.V(1).Info("found paths", "paths", paths, "count", len(paths))
		starters := korrel8.NewSetResult(start)

		// FIXME include constraint

		if len(args) >= 3 { // Get starters using query
			query := must(queryFromArgs(args[2:]))
			store, err := e.Store(start.Domain().String())
			check(err)
			check(store.Get(ctx, query, starters))
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
		if *getFlag {
			printObjects(e, goal, queries.List)
		} else {
			printQueries(e, goal, queries.List)
		}
	},
}

func printQueries(e *engine.Engine, goal korrel8.Class, queries []korrel8.Query) {
	for _, q := range queries {
		fmt.Println(&q)
	}
}

func printObjects(e *engine.Engine, goal korrel8.Class, queries []korrel8.Query) {
	d := goal.Domain()
	store, err := e.Store(d.String())
	check(err)
	result := newPrinter(os.Stdout)
	for _, q := range queries {
		check(store.Get(context.Background(), &q, result))
	}
}

var (
	allFlag, getFlag *bool
	kFlag            *int
	endTime          TimeFlag
)

func init() {
	rootCmd.AddCommand(correlateCmd)
	correlateCmd.Flags().Var(&endTime, "time", "find results up to this time")
	kFlag = correlateCmd.Flags().IntP("kshortest", "k", 0, "Use K-shortest paths")
	allFlag = correlateCmd.Flags().BoolP("allpaths", "a", false, "Use all paths")
	getFlag = correlateCmd.Flags().Bool("get", false, "Get objects from query")
}
