package auth

import (
	"github.com/sanyam-malik/pulsewatch/internal/modules/bruteforce"

	"github.com/gin-gonic/gin"
)

type Route struct {
	controller      *Controller
	middleware      *MiddlewareProvider
	bruteforceGuard *bruteforce.Guard
}

func NewRoute(
	controller *Controller,
	middleware *MiddlewareProvider,
	bruteforceGuard *bruteforce.Guard,
) *Route {
	return &Route{
		controller,
		middleware,
		bruteforceGuard,
	}
}

func (r *Route) ConnectRoute(router *gin.RouterGroup, controller *Controller) {
	auth := router.Group("/auth")
	auth.POST("/register", controller.Register)

	auth.POST("/login", r.bruteforceGuard.Middleware(), controller.Login)

	auth.POST("/refresh", controller.RefreshToken)

	auth.Use(r.middleware.Auth())
	auth.POST("/2fa/setup", controller.SetupTwoFA)
	auth.POST("/2fa/verify", controller.VerifyTwoFA)
	auth.POST("/2fa/disable", controller.DisableTwoFA)
	auth.PUT("/password", controller.UpdatePassword)

	groups := router.Group("/groups")
	groups.Use(r.middleware.Auth())
	groups.GET("", controller.ListGroups)
	groups.POST("", controller.CreateGroup)
	groups.GET("/:groupId/members", controller.ListMembers)
	groups.POST("/:groupId/members", controller.AddMember)
	groups.PATCH("/:groupId/members/:userId", controller.UpdateMemberRole)
	groups.DELETE("/:groupId/members/:userId", controller.RemoveMember)
}
