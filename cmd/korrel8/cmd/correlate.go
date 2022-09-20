/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alanconway/korrel8/pkg/alert"
	"github.com/alanconway/korrel8/pkg/k8s"
	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/alanconway/korrel8/pkg/loki"
	"github.com/alanconway/korrel8/pkg/rules"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// correlateCmd represents the correlate command
var correlateCmd = &cobra.Command{
	// FIXME need string form for start/goal domain/class and start data.
	Use: "correlate --start FILE (FIXME: hard coded start + goal classes for demo)",
	//	Use:   "correlate",
	//	Short: "Correlate the objects in file START from DOMAIN to GOAL  a file of starting objects, return a query for correlated objects. '-' means use stdin",
	//	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		f := open(startFile)
		defer f.Close()
		// FIXME total ruleset
		ruleSet := korrel8.NewRuleSet(rules.K8sRules()...)
		ruleSet.Add(rules.K8sToLoki()...)
		cfg, err := config.GetConfig()
		exitErr(err)
		c, err := client.New(cfg, client.Options{})
		store, err := k8s.NewStore(c)
		exitErr(err)
		// FIXME Follower construct
		follower := korrel8.Follower{Stores: map[korrel8.Domain]korrel8.Store{k8s.Domain: store}}
		// FIXME hard coded for demo
		paths := ruleSet.FindPaths(alert.Class{}, loki.Class{})
		a := models.GettableAlert{}
		exitErr(json.NewDecoder(f).Decode(&a))
		var queries korrel8.Queries
		for _, p := range paths {
			q, err := follower.Follow(context.Background(), alert.Object{GettableAlert: &a}, nil, p)
			exitErr(err)
			queries = append(queries, q...)
		}
		fmt.Printf("\nresulting query set: %#v\n\n", queries)
	},
}

var (
	startFile, storeType string
)

func init() {
	rootCmd.AddCommand(correlateCmd)
	correlateCmd.Flags().StringVar(&startFile, "start", "", "file containing start objects")
	for _, f := range []string{"start"} {
		correlateCmd.MarkFlagRequired(f)
	}
}
