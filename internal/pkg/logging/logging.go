// package logging initializes the root logger and provides some helpers.
package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

const verboseEnv = "KORREL8R_VERBOSE"

var root logr.Logger

// The root logger.
func Log() logr.Logger { return root }

func init() { // Set env verbosity on init, Init() can over-ride.
	root = stdr.New(log.New(os.Stderr, "korrel8r ", log.Ltime))
	if n, err := strconv.Atoi(os.Getenv(verboseEnv)); err == nil {
		stdr.SetVerbosity(n)
	}
}

// Init sets verbosity for the Root logger.
func Init(verbosity int) {
	if verbosity != 0 { // If not set, let env verbosity stand
		stdr.SetVerbosity(verbosity)
	}
}

// Log url slices properly, url.URL only has a pointer-receiver String() method.
type URLs []url.URL

func (u URLs) MarshalLog() any {
	p := make([]*url.URL, len(u))
	for i, v := range u {
		p[i] = &v
	}
	return p
}

// JSONString returns the JSON marshaled string from v, or the error message if marshal fails
func JSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%q", err.Error())
	}
	return string(b)
}

type logJSON struct{ v any }

func (l logJSON) MarshalLog() any { return JSONString(l.v) }

// JSON wraps a value so it will be printed as JSON if logged.
func JSON(v any) logr.Marshaler { return logJSON{v: v} }
