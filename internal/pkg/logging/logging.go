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
	log.SetLogger(zap.New(zap.Level(zapcore.Level(-verbose)), zap.UseDevMode(true)))
}

// BeginEnd logs Info "begin "+msg..., and returns a function to defer to log "end "+msg...
func BeginEnd(log logr.Logger, msg string, args ...any) func() {
	log.Info("begin "+msg, args...)
	return func() { log.Info("end   "+msg, args...) }
}
