//go:build swag

package swag

import (
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/middlewares/security"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/swag"
)

func Setup(r *gin.RouterGroup) {
	swagger := swag.GetSwagger("swagger")
	if swagger == nil {
		logger.Warnf("%s not registered, please check if the swagger docs directory exists in your project", "swagger")
		return
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerfiles.Handler,
		ginSwagger.PersistAuthorization(true)))
	security.PermitRoute("/swagger/*any")

	logger.Infof("swagger initialized successfully, please visit http://%s/swagger/index.html to view the API documentation.",
		config.GetString("server.addr")+config.GetString("server.base_path"))
}
