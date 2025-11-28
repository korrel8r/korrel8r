// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/ptr"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/spf13/cobra"
)

// Common flags for neighbors and goals
var (
	class                 string
	queries               []string
	objects               []string
	withRules             bool
	limit                 int
	since, until, timeout time.Duration

	// Gather graph options into a struct.
	graphOptions = rest.GraphOptions{
		Rules: &withRules,
	}
)

func startFlags(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&queries, "query", "q", nil, "Query string for start objects, can be multiple.")
	cmd.Flags().StringVar(&class, "class", "", "Class for serialized start objects")
	cmd.Flags().StringArrayVar(&objects, "object", nil, "Serialized start object, can be multiple.")
	cmd.Flags().BoolVar(&withRules, "rules", false, "Include rule names in returned graph")
}

func constraintFlags(cmd *cobra.Command) {
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit total number of results.")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Timeout for store requests.")
	cmd.Flags().DurationVar(&since, "since", 0, "Only get results since this long ago.")
	cmd.Flags().DurationVar(&until, "until", 0, "Only get results until this long ago.")
}

var (
	objectsCmd = &cobra.Command{
		Use:     "objects QUERY",
		Short:   "Execute QUERY and print the results",
		Aliases: []string{"get"},
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			e, _ := newEngine()
			q := must.Must1(e.Query(args[0]))
			p := newPrinter(os.Stdout)
			defer p.Close()
			ctx, cancel := e.WithTimeout(context.Background(), timeout)
			defer cancel()
			must.Must(e.Get(ctx, q, constraint(), p))
		},
	}
)

func init() {
	rootCmd.AddCommand(objectsCmd)
	constraintFlags(objectsCmd)
}

var (
	neighborsCmd = &cobra.Command{
		Use:   "neighbors",
		Short: "Get graph of nearest neighbors",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			e, _ := newEngine()
			ctx, cancel := e.WithTimeout(context.Background(), timeout)
			defer cancel()
			g, err := traverse.Neighbors(ctx, e, start(e), depth)
			must.Must(err)
			newPrinter(os.Stdout).Print(rest.NewGraph(g, &graphOptions))
		},
	}
	depth int
)

func init() {
	rootCmd.AddCommand(neighborsCmd)
	startFlags(neighborsCmd)
	constraintFlags(neighborsCmd)
	neighborsCmd.Flags().IntVarP(&depth, "depth", "d", 3, "Depth of neighborhood search.")
}

var (
	goalsCmd = &cobra.Command{
		Use:   "goals GOAL [GOAL...]",
		Short: "Execute QUERY, find all paths to GOAL classes.",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			e, _ := newEngine()
			var goals []korrel8r.Class
			for _, g := range args {
				goals = append(goals, must.Must1(e.Class(g)))
			}
			ctx, cancel := e.WithTimeout(context.Background(), timeout)
			defer cancel()
			g, err := traverse.Goals(ctx, e, start(e), goals)
			must.Must(err)
			newPrinter(os.Stdout).Print(rest.NewGraph(g, &graphOptions))
		},
	}
)

func init() {
	rootCmd.AddCommand(goalsCmd)
	startFlags(goalsCmd)
	constraintFlags(goalsCmd)
}

func constraint() *korrel8r.Constraint {
	c := &korrel8r.Constraint{}
	if limit > 0 {
		c.Limit = ptr.To(limit)
	}
	now := time.Now()
	if since > 0 {
		c.Start = ptr.To(now.Add(-since))
	}
	if until > 0 {
		c.End = ptr.To(now.Add(-until))
	}
	return c
}

func start(e *engine.Engine) traverse.Start {
	var c korrel8r.Class
	switch {
	case class != "":
		c = must.Must1(e.Class(class))
	case len(queries) > 0:
		c = must.Must1(e.Query(queries[0])).Class()
	default:
		must.Must(fmt.Errorf("must provide a class or at least one query"))
	}
	start := traverse.Start{Class: c, Constraint: constraint()}
	for _, q := range queries {
		start.Queries = append(start.Queries, must.Must1(e.Query(q)))
	}
	for _, o := range objects {
		start.Objects = append(start.Objects, must.Must1(c.Unmarshal([]byte(o))))
	}
	return start
}
