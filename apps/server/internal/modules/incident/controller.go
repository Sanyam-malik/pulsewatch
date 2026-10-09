package incident

import (
	"net/http"

	"github.com/sanyam-malik/pulsewatch/internal/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Controller struct {
	service Service
	logger  *zap.SugaredLogger
}

func NewController(service Service, logger *zap.SugaredLogger) *Controller {
	return &Controller{service: service, logger: logger}
}

func (c *Controller) FindAll(ctx *gin.Context) {
	incidents, err := c.service.FindAll(ctx)
	if err != nil {
		c.logger.Errorw("Failed to fetch incidents", "error", err)
		ctx.JSON(http.StatusInternalServerError, utils.NewFailResponse("Failed to fetch incidents"))
		return
	}
	ctx.JSON(http.StatusOK, utils.NewSuccessResponse("Incidents loaded", incidents))
}

func (c *Controller) Create(ctx *gin.Context) {
	var dto CreateDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, utils.NewFailResponse("Invalid request body"))
		return
	}
	if err := utils.Validate.Struct(dto); err != nil {
		ctx.JSON(http.StatusBadRequest, utils.NewFailResponse(err.Error()))
		return
	}
	incident, err := c.service.Create(ctx, &dto)
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, utils.NewSuccessResponse("Incident created", incident))
}

func (c *Controller) AddUpdate(ctx *gin.Context) {
	var dto AddUpdateDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, utils.NewFailResponse("Invalid request body"))
		return
	}
	if err := utils.Validate.Struct(dto); err != nil {
		ctx.JSON(http.StatusBadRequest, utils.NewFailResponse(err.Error()))
		return
	}
	incident, err := c.service.AddUpdate(ctx, ctx.Param("id"), &dto)
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, utils.NewSuccessResponse("Incident updated", incident))
}

func (c *Controller) FindPublicByStatusPageSlug(ctx *gin.Context) {
	incidents, err := c.service.FindPublicByStatusPageSlug(ctx, ctx.Param("slug"))
	if err != nil {
		c.logger.Errorw("Failed to fetch public status-page incidents", "error", err)
		ctx.JSON(http.StatusInternalServerError, utils.NewFailResponse("Failed to fetch incidents"))
		return
	}
	ctx.JSON(http.StatusOK, utils.NewSuccessResponse("Incidents loaded", incidents))
}

func (c *Controller) writeError(ctx *gin.Context, err error) {
	switch err.Error() {
	case "status page not found in the active group", "incident not found":
		ctx.JSON(http.StatusNotFound, utils.NewFailResponse(err.Error()))
	case "invalid incident status", "invalid status page ID", "incident title must be between 3 and 255 characters",
		"incident message must be between 1 and 4000 characters", "incident update must be between 1 and 4000 characters":
		ctx.JSON(http.StatusBadRequest, utils.NewFailResponse(err.Error()))
	default:
		c.logger.Errorw("Incident operation failed", "error", err)
		ctx.JSON(http.StatusInternalServerError, utils.NewFailResponse("Incident operation failed"))
	}
}
