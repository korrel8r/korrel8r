// package logging initializes the root logger and provides some helpers.
package logging

import (
	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// The root logger. Not functional till Init() is called.
var Log logr.Logger = log.Log

// Init sets up the Root logger.
func Init(verbose int) {
	if verbose > 0 {
		verbose = -verbose
	}
	log.SetLogger(zap.New(zap.Level(zapcore.Level(verbose)), zap.UseDevMode(true), zap.ConsoleEncoder()))
}
