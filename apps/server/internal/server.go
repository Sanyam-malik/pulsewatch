package internal

import (
	_ "github.com/sanyam-malik/pulsewatch/docs"
	"github.com/sanyam-malik/pulsewatch/internal/config"
	"github.com/sanyam-malik/pulsewatch/internal/modules/api_key"
	"github.com/sanyam-malik/pulsewatch/internal/modules/auth"
	"github.com/sanyam-malik/pulsewatch/internal/modules/badge"
	"github.com/sanyam-malik/pulsewatch/internal/modules/healthcheck"
	"github.com/sanyam-malik/pulsewatch/internal/modules/heartbeat"
	"github.com/sanyam-malik/pulsewatch/internal/modules/incident"
	"github.com/sanyam-malik/pulsewatch/internal/modules/maintenance"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor"
	"github.com/sanyam-malik/pulsewatch/internal/modules/monitor_group"
	"github.com/sanyam-malik/pulsewatch/internal/modules/notification_channel"
	"github.com/sanyam-malik/pulsewatch/internal/modules/proxy"
	"github.com/sanyam-malik/pulsewatch/internal/modules/queue"
	"github.com/sanyam-malik/pulsewatch/internal/modules/setting"
	"github.com/sanyam-malik/pulsewatch/internal/modules/status_page"
	"github.com/sanyam-malik/pulsewatch/internal/modules/tag"
	"github.com/sanyam-malik/pulsewatch/internal/modules/websocket"
	"github.com/sanyam-malik/pulsewatch/internal/version"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// @Summary      Get server version
// @Description  Returns the current server version
// @Tags         System
// @Produce      json
// @Success      200  {object}  map[string]string  "{"version": "1.2.3"}"
// @Router       /version [get]
func versionHandler(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"version": version.Version})
}

// @Summary      Get server health
// @Description  Returns the current server health
// @Tags         System
// @Produce      json
// @Success      200  {object}  map[string]string  "{"status": "success"}"
// @Router       /health [get]
func healthHandler(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

type Server struct {
	Router *gin.Engine
	Cfg    *config.Config
}

func ProvideServer(
	logger *zap.SugaredLogger,
	cfg *config.Config,
	monitorRoute *monitor.MonitorRoute,
	monitorController *monitor.MonitorController,
	monitorGroupRoute *monitor_group.Route,
	incidentRoute *incident.Route,
	authRoute *auth.Route,
	authController *auth.Controller,
	wsServer *websocket.Server,
	notificationChannelRoute *notification_channel.Route,
	notificationChannelController *notification_channel.Controller,
	proxyRoute *proxy.Route,
	proxyController *proxy.Controller,
	settingRoute *setting.Route,
	settingController *setting.Controller,
	heartbeatService heartbeat.Service,
	monitorService monitor.Service,
	queueService queue.Service,
	maintenanceRoute *maintenance.Route,
	maintenanceController *maintenance.Controller,
	statusPageRoute *status_page.Route,
	statusPageController *status_page.Controller,
	tagRoute *tag.Route,
	tagController *tag.Controller,
	badgeRoute *badge.Route,
	badgeController *badge.Controller,
	apiKeyRoute *api_key.Route,
	apiKeyController *api_key.Controller,
) *Server {
	// Initialize server based on mode
	var server *gin.Engine
	if cfg.Mode == "dev" {
		// Development: use default with logger and recovery middleware
		server = gin.Default()
	} else {
		// Production/Test: use clean instance with only recovery middleware
		gin.SetMode(gin.ReleaseMode)
		server = gin.New()
		server.Use(gin.Recovery()) // Always use recovery middleware to prevent crashes
	}

	server.RedirectTrailingSlash = false

	// CORS configuration
	server.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "X-Requested-With", "Content-Type", "Accept", "Authorization", "X-Group-ID"},
		ExposeHeaders:    []string{"Authorization"},
		AllowCredentials: true,
	}))

	// server.Use(LogMiddleware(logger))

	server.GET("/health", healthHandler)
	router := server.Group("/api/v1")
	router.GET("/health", healthHandler)
	router.GET("/version", versionHandler)

	// Connect routes
	monitorRoute.ConnectRoute(router, monitorController)
	authRoute.ConnectRoute(router, authController)
	monitorGroupRoute.ConnectRoute(router)
	incidentRoute.ConnectRoute(router)
	notificationChannelRoute.ConnectRoute(router, notificationChannelController)
	proxyRoute.ConnectRoute(router, proxyController)
	settingRoute.ConnectRoute(router, settingController)
	maintenanceRoute.ConnectRoute(router, maintenanceController)
	statusPageRoute.ConnectRoute(router, statusPageController)
	tagRoute.ConnectRoute(router, tagController)
	badgeRoute.ConnectRoute(router, badgeController)
	apiKeyRoute.ConnectRoute(router, apiKeyController)

	// Register push endpoint
	healthcheck.RegisterPushEndpoint(router, monitorService, heartbeatService, queueService, logger)

	// Swagger routes
	url := ginSwagger.URL("/swagger/doc.json")
	server.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))

	// WebSocket route
	server.GET("/socket.io/*f", func(c *gin.Context) {
		wsServer.ServeHTTP(c.Writer, c.Request)
	})
	server.POST("/socket.io/*f", func(c *gin.Context) {
		wsServer.ServeHTTP(c.Writer, c.Request)
	})

	return &Server{Router: server, Cfg: cfg}
}
