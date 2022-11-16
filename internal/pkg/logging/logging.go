// package logging initializes the root logger and provides some helpers.
package logging

import (
	"encoding/json"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

// The root logger. Not functional till Init() is called.
var Log logr.Logger

func init() {
	Log = stdr.New(log.New(os.Stderr, "korrel8 ", log.Ltime))
	if n, err := strconv.Atoi(os.Getenv("KORREL8_VERBOSE")); err == nil {
		stdr.SetVerbosity(n)
	}
}

// Init sets verbosity for the Root logger.
func Init(verbosity int) {
	if verbosity > 0 {
		stdr.SetVerbosity((verbosity))
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
		return err.Error()
	}
	return string(b)
}

type logJSON struct{ v any }

func (l logJSON) MarshalLog() any { return JSONString(l.v) }

func JSON(v any) logr.Marshaler { return logJSON{v: v} }
