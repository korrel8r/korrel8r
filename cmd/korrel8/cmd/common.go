package cmd

import (
	"fmt"
	"os"
)

func exitErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func open(name string) (f *os.File, err error) {

	if name == "-" {
		return os.Stdin, nil
	} else {
		return os.Open(name)
	}
}
