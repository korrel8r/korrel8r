// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"fmt"
	"os"

	"text/template"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template [--file FILE|--template STRING]",
	Short: "Apply a Go template to the korrel8r engine.",
	Run: func(cmd *cobra.Command, args []string) {
		if (*templateFile == "" && *templateString == "") || (*templateFile != "" && *templateString != "") {
			must.Must(fmt.Errorf("exactly one of --file or --template must be provided"))
		}
		if *templateFile != "" {
			*templateString = string(must.Must1(os.ReadFile(*templateFile)))
		}
		e := newEngine()
		t := template.Must(template.New(*templateString).Funcs(e.TemplateFuncs()).Parse(*templateString))
		must.Must(t.Execute(os.Stdout, e))
	},
}

var templateFile, templateString *string

func init() {
	templateFile = templateCmd.Flags().StringP("file", "f", "", "read template from file")
	templateString = templateCmd.Flags().StringP("template", "t", "", "use template string")
	rootCmd.AddCommand(templateCmd)
}
