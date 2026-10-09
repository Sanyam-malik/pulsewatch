package incident

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
	publicPages := router.Group("status-pages")
	publicPages.GET("/slug/:slug/incidents", r.controller.FindPublicByStatusPageSlug)

	incidents := router.Group("incidents")
	incidents.Use(r.auth.AllAuth())
	incidents.GET("", r.controller.FindAll)
	incidents.POST("", r.controller.Create)
	incidents.POST("/:id/updates", r.controller.AddUpdate)
}
