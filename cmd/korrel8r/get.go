// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"os"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get DOMAIN:CLASS:QUERY",
	Short: "Execute QUERY and print the results",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		q := must.Must1(e.Query(args[0]))
		s := must.Must1(e.StoreErr(q.Class().Domain()))
		log.V(3).Info("get", "query", q, "class", korrel8r.ClassName(q.Class()))
		result := newPrinter(os.Stdout)
		must.Must(s.Get(context.Background(), q, result))
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
