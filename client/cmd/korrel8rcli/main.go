// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	_ "embed"
	"log"
	"net/url"
	"path/filepath"

	"os"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/korrel8r/korrel8r/client/pkg/swagger/client"
	"github.com/spf13/cobra"
)

const (
	remoteEnv = "KORREL8R_REMOTE"
)

var (
	rootCmd = &cobra.Command{
		Use:   "korrel8rcli COMMAND",
		Short: "REST client for a remote korrel8r server.",
	}

	// Global Flags
	output = EnumFlag("yaml", "json-pretty", "json")
)

func main() {
	rootCmd.PersistentFlags().VarP(output, "output", "o", "Output format")
	log.SetPrefix(filepath.Base(os.Args[0]) + ": ")
	log.SetFlags(0)
	check(rootCmd.Execute())
}

func check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func newClient(urlStr string) *client.RESTAPI {
	u, err := url.Parse(urlStr)
	check(err)
	if u.Path == "" {
		u.Path = client.DefaultBasePath
	}
	return client.New(httptransport.New(u.Host, u.Path, []string{u.Scheme}), nil)
}
