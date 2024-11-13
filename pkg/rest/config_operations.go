// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
)

// Config settings that can be modified at runtime via the API.
type Config struct {
	Verbose *int `json:"verbose,omitempty" form:"verbose"` // Verbose setting for logging.
}

// PutConfig handler
//
//	@router		/config [put]
//	@summary	Set verbose level for logging on a running server.
//	@param		verbose	query int false	"verbose setting for logging"
//	@success	200
//	@failure	default	{object}	any
func (a *API) PutConfig(c *gin.Context) {
	var config Config
	if !check(c, http.StatusBadRequest, c.BindQuery(&config)) {
		return
	}
	if config.Verbose != nil {
		log.V(1).Info("Verbose log level set via API", "level", *config.Verbose)
		logging.SetVerbose(*config.Verbose)
	}
	c.JSON(http.StatusOK, config)
}
