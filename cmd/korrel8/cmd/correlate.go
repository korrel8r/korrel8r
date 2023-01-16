/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/korrel8/korrel8/internal/pkg/decoder"
	"github.com/korrel8/korrel8/internal/pkg/must"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/graph"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/unique"
	"github.com/korrel8/korrel8/pkg/uri"
	"github.com/spf13/cobra"
)

// correlateCmd represents the correlate command
var correlateCmd = &cobra.Command{
	Use:   "correlate START_CLASS GOAL_CLASS [URI_REF [NAME=VALUE...]]",
	Short: "Correlate from START_CLASS to GOAL_CLASS. Use URI_REF if present, read object from stdin if not",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		start, goal := must.Must1(e.ParseClass(args[0])), must.Must1(e.ParseClass(args[1]))
		var paths []graph.MultiPath
		switch {
		case *allFlag:
			paths = must.Must1(e.Graph().AllPaths(start, goal))
		case *kFlag > 0:
			paths = must.Must1(e.Graph().KShortestPaths(start, goal, *kFlag))
		default:
			paths = must.Must1(e.Graph().ShortestPaths(start, goal))
		}
		log.V(1).Info("found paths", "paths", paths, "count", len(paths))
		starters := korrel8.NewResult(start)

		if len(args) >= 3 { // Get starters using query
			query := must.Must1(referenceArgs(args[2:]))
			store, err := e.Store(start.Domain().String())
			must.Must(err)
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
		refs := unique.NewList[uri.Reference]()
		for _, path := range paths {
			refs.Append(must.Must1(e.Follow(ctx, starters.List(), nil, path))...)
		}
		if *getFlag {
			printObjects(e, goal, refs.List)
		} else {
			printRefs(refs.List)
		}
	},
}

func printRefs(refs []uri.Reference) {
	for _, ref := range refs {
		fmt.Println(ref)
	}
}

func printObjects(e *engine.Engine, goal korrel8.Class, refs []uri.Reference) {
	d := goal.Domain()
	store, err := e.Store(d.String())
	must.Must(err)
	result := newPrinter(os.Stdout)
	for _, ref := range refs {
		must.Must(store.Get(context.Background(), ref, result))
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
