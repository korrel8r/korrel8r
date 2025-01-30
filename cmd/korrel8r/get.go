// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
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

// Common flags for neighbours and goals
var (
	class   string
	queries []string
	objects []string

	limit                 int
	since, until, timeout time.Duration
)

func startFlags(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&queries, "query", "q", nil, "Query string for start objects, can be multiple.")
	cmd.Flags().StringVar(&class, "class", "", "Class for serialized start objects")
	cmd.Flags().StringArrayVar(&objects, "object", nil, "Serialized start object, can be multiple.")
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
			must.Must(e.Get(context.Background(), q, constraint(), p))
		},
	}
)

func init() {
	rootCmd.AddCommand(objectsCmd)
	constraintFlags(objectsCmd)
}

func check(err error) {
	if traverse.IsPartialError(err) {
		fmt.Fprintln(os.Stderr, err)
	} else {
		must.Must(err)
	}
}

var (
	neighboursCmd = &cobra.Command{
		Use:   "neighbours",
		Short: "Get graph of nearest neighbours",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			e, _ := newEngine()
			ctx, cancel := korrel8r.WithConstraint(context.Background(), constraint())
			defer cancel()
			g, err := traverse.New(e, e.Graph()).Neighbours(ctx, start(e), depth)
			check(err)
			newPrinter(os.Stdout).Print(rest.NewGraph(g))
		},
	}
	depth int
)

func init() {
	rootCmd.AddCommand(neighboursCmd)
	startFlags(neighboursCmd)
	constraintFlags(neighboursCmd)
	neighboursCmd.Flags().IntVarP(&depth, "depth", "d", 2, "Depth of neighbourhood search.")
}

var (
	goalsCmd = &cobra.Command{
		Use:   "goals GOAL [GOAL...]",
		Short: "Execute QUERY, find all paths to GOAL classes.",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			e, _ := newEngine()
			var goals []korrel8r.Class
			for _, g := range args[1:] {
				goals = append(goals, must.Must1(e.Class(g)))
			}
			ctx, cancel := korrel8r.WithConstraint(context.Background(), constraint())
			defer cancel()
			g, err := traverse.New(e, e.Graph()).Goals(ctx, start(e), goals)
			check(err)
			newPrinter(os.Stdout).Print(rest.NewGraph(g))
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
	if timeout > 0 {
		c.Timeout = ptr.To(timeout)
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
		must.Must(fmt.Errorf("Must provide a class or at least one query."))
	}
	start := traverse.Start{Class: c}
	for _, q := range queries {
		start.Queries = append(start.Queries, must.Must1(e.Query(q)))
	}
	for _, o := range objects {
		start.Objects = append(start.Objects, must.Must1(c.Unmarshal([]byte(o))))
	}
	return start
}
