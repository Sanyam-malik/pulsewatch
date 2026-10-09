package monitor_group

import (
	"github.com/sanyam-malik/pulsewatch/internal/modules/middleware"

	"github.com/gin-gonic/gin"
)

type Route struct {
	controller *Controller
	auth       *middleware.AuthChain
}

func NewRoute(controller *Controller, auth *middleware.AuthChain) *Route {
	return &Route{controller: controller, auth: auth}
}

func (r *Route) ConnectRoute(router *gin.RouterGroup) {
	group := router.Group("monitor-groups")
	group.Use(r.auth.AllAuth())
	group.GET("", r.controller.FindAll)
	group.POST("", r.controller.Create)
	group.GET("/:id", r.controller.FindByID)
	group.PATCH("/:id", r.controller.Update)
	group.PUT("/:id/monitors", r.controller.SetMonitors)
	group.DELETE("/:id", r.controller.Delete)
}
