// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package logging initializes the root logger and provides some helpers.
//
// Log statements should be logged at the correct level to make logs readable for non-developers
// but also useful for debugging.
//
// - 0: Cannot be hidden, only for events that should *always* be seen by a service operator or command line user.
//   Examples: service startup notice, fatal errors, events requiring human intervention.
// - 1: Low volume info/warn messages useful to service operator. Don't assume the reader understands the code.
// - 2: Low-volume debugging - setup, shutdown, occasional state changes.
// - 3: Per-request debugging - events that occur on every incoming service request.
// - 4: Per-rule-evaluation debugging - many per request.
// - 5: Per-query-execution debugging - many per rule evaluation.

package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

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
	// Redirect standard log output to use the logr.Logger
	log.SetOutput(&logrWriter{logger: root})
}

// logrWriter is an io.Writer that forwards writes to a logr.Logger
type logrWriter struct {
	logger logr.Logger
}

// Write implements io.Writer, forwarding the message to the logr.Logger at V(3)
func (w *logrWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		w.logger.V(3).Info(msg)
	}
	return len(p), nil
}

// Init sets verbosity based on flag or environment.
// Defaults to 1 if neither is set.
// Sets the flag to actual setting.
func Init(verbosity *int) {
	if *verbosity == 0 {
		*verbosity, _ = strconv.Atoi(os.Getenv(verboseEnv))
	}
	SetVerbose(*verbosity)
}

// JSON wraps a value so it is logged as plain JSON rather than Go's default formatting.
// Implements logr.Marshaler: marshalling is deferred until the log line is actually emitted.
// The value is marshalled to JSON then unmarshalled to generic types (map/slice/string/number),
// which funcr (used by stdr) formats as unquoted JSON-structured output with correct field names.
func JSON(v any) logr.Marshaler { return jsonValue{v} }

type jsonValue struct{ v any }

func (j jsonValue) MarshalLog() any {
	b, err := json.Marshal(j.v)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err)
	}
	var generic any
	if err := json.Unmarshal(b, &generic); err != nil {
		return string(b)
	}
	return generic
}

// SetVerbose sets the logging verbosity for the entire process.
func SetVerbose(level int) {
	if level < 0 {
		level = 0
	}
	stdr.SetVerbosity(level)
}
