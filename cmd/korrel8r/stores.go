// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"encoding/json"
	"os"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/spf13/cobra"
)

var storesCmd = &cobra.Command{
	Use:   "stores [DOMAIN...]",
	Short: "List the stores configured for the listed domains, or for all domains if none are listed.",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		e, _ := newEngine()
		stores := map[string][]config.Store{}
		var domains []korrel8r.Domain
		for _, d := range args {
			domains = append(domains, must.Must1(e.Domain(d)))
		}
		if len(domains) == 0 {
			domains = e.Domains()
		}
		for _, d := range domains {
			stores[d.Name()] = e.StoreConfigsFor(d)
		}
		p := &jsonPrinter{Encoder: json.NewEncoder(os.Stdout)}
		p.SetIndent("", "  ")
		_ = p.Encode(stores)
	},
}

func init() {
	rootCmd.AddCommand(storesCmd)
}
