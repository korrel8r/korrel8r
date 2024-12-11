// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"sync"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

var profileOnce sync.Once

// WebProfile enables profiling REST endpoints.
func WebProfile(router *gin.Engine) {
	profileOnce.Do(func() { pprof.Register(router) })
}
