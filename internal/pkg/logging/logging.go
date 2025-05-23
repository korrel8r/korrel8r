// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package logging initializes the root logger and provides some helpers.
package logging

import (
	"io"
	"log"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

const verboseEnv = "KORREL8R_VERBOSE"

var root logr.Logger

// Log returns the root logger.
func Log() logr.Logger { return root }

// LogWriter returns the destination for the root logger,  so other logs can be directed to it.
func LogWriter() io.Writer { return os.Stderr }

func init() { // Set env verbosity on init, Init() can over-ride.
	root = stdr.New(log.New(os.Stderr, "korrel8r ", log.Ltime))
	if n, err := strconv.Atoi(os.Getenv(verboseEnv)); err == nil {
		stdr.SetVerbosity(n)
	}
}

// Init sets verbosity based on flag or environment.
// Defaults to 1 if neither is set.
// Sets the flag to actual setting.
func Init(verbosity *int) {
	if *verbosity == 0 {
		*verbosity, _ = strconv.Atoi(os.Getenv(verboseEnv))
	}
	klogInit(*verbosity)
	SetVerbose(*verbosity)
}

// SetVerbose sets the logging verbosity for the entire process.
func SetVerbose(level int) {
	if level < 0 {
		level = 0
	}
	stdr.SetVerbosity(level)
	klogVerbose(level)
}
