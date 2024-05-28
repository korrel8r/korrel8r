// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"os"

	"github.com/korrel8r/korrel8r/client/pkg/swagger/client/operations"
	"github.com/korrel8r/korrel8r/client/pkg/swagger/models"
	"github.com/spf13/cobra"
)

// Common flags
var (
	queries []string
	class   string
	objects []string
	goals   []string
	depth   int64
	rules   bool
)

func makeStart() *models.Start {
	return &models.Start{
		Class:      class,
		Constraint: nil, // TODO support for constraints.
		Objects:    objects,
		Queries:    queries,
	}
}

var domainsCmd = &cobra.Command{
	Use:   "domains URL",
	Short: "Get a list of domains and store configuration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient(args[0])
		ok, err := c.Operations.GetDomains(&operations.GetDomainsParams{})
		check(err)
		NewPrinter(output.String(), os.Stdout)(ok.Payload)
	},
}

func init() {
	rootCmd.AddCommand(domainsCmd)
}

var neighboursCmd = &cobra.Command{
	Use:   "neighbours URL [FLAGS]",
	Short: "Get graph of nearest neighbours",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient(args[0])
		ok, err := c.Operations.PostGraphsNeighbours(&operations.PostGraphsNeighboursParams{
			Request: &models.Neighbours{
				Depth: depth,
				Start: makeStart(),
			},
			Rules: &rules,
		})
		check(err)
		NewPrinter(output.String(), os.Stdout)(ok.Payload)
	},
}

var goalsCmd = &cobra.Command{
	Use:   "goals URL [FLAGS]",
	Short: "Get graph of nearest goals",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		c := newClient(args[0])
		ok, err := c.Operations.PostGraphsGoals(&operations.PostGraphsGoalsParams{
			Request: &models.Goals{
				Goals: goals,
				Start: makeStart(),
			},
			Rules: &rules,
		})
		check(err)
		NewPrinter(output.String(), os.Stdout)(ok.Payload)
	},
}

func commonFlags(cmd *cobra.Command) {
	rootCmd.AddCommand(cmd)
	cmd.Flags().StringArrayVar(&queries, "query", nil, "Query string for start objects, can be multiple.")
	cmd.Flags().StringVar(&class, "class", "", "Class for serialized objects")
	cmd.Flags().StringArrayVar(&objects, "object", nil, "Serialized start object, can be multiple.")
	cmd.Flags().BoolVar(&rules, "rules", false, "Include per-rule information in returned graph.")
}

func init() {
	commonFlags(neighboursCmd)
	neighboursCmd.Flags().Int64Var(&depth, "depth", 2, "Depth of neighbourhood search.")
	commonFlags(goalsCmd)
	goalsCmd.Flags().StringArrayVar(&goals, "goal", nil, "Serialized start goal, can be multiple.")
}
