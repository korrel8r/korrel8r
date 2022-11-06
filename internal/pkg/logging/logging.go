// package logging initializes the root logger and provides some helpers.
package logging

import (
	"log"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
)

// The root logger. Not functional till Init() is called.
var Log logr.Logger = stdr.New(log.New(os.Stderr, "korrel8 ", log.Ltime))

// Init sets up the Root logger.
func Init(verbosity int) {
	if n, err := strconv.Atoi(os.Getenv("KORREL8_VERBOSE")); err == nil && verbosity == 0 {
		verbosity = n
	}
	stdr.SetVerbosity((verbosity))
}
