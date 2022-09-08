package cmd

import (
	"fmt"
	"os"

	"github.com/alanconway/korrel8/pkg/k8s"
	"github.com/alanconway/korrel8/pkg/korrel8"
)

func exitErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func must[T any](v T, err error) T { exitErr(err); return v }

func open(name string) (f *os.File) {
	if name == "-" {
		return os.Stdin
	} else {
		return must(os.Open(name))
	}
}

var stores = map[korrel8.Domain]korrel8.Store{
	k8s.Domain: must(k8s.NewStore(must(k8s.NewClient()))),
}
