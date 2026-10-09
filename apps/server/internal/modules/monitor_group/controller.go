package monitor_group

import (
	"net/http"
	"strings"

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
	groups, err := c.service.FindAll(ctx)
	if err != nil {
		c.logger.Errorw("Failed to fetch monitor groups", "error", err)
		ctx.JSON(http.StatusInternalServerError, utils.NewFailResponse("Failed to fetch monitor groups"))
		return
	}
	ctx.JSON(http.StatusOK, utils.NewSuccessResponse("Monitor groups loaded", groups))
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
	group, err := c.service.Create(ctx, &dto)
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, utils.NewSuccessResponse("Monitor group created", group))
}

func (c *Controller) FindByID(ctx *gin.Context) {
	group, err := c.service.FindByID(ctx, ctx.Param("id"))
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	if group == nil {
		ctx.JSON(http.StatusNotFound, utils.NewFailResponse("Monitor group not found"))
		return
	}
	ctx.JSON(http.StatusOK, utils.NewSuccessResponse("Monitor group loaded", group))
}

func (c *Controller) Update(ctx *gin.Context) {
	var dto UpdateDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, utils.NewFailResponse("Invalid request body"))
		return
	}
	if err := utils.Validate.Struct(dto); err != nil {
		ctx.JSON(http.StatusBadRequest, utils.NewFailResponse(err.Error()))
		return
	}
	group, err := c.service.Update(ctx, ctx.Param("id"), &dto)
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	if group == nil {
		ctx.JSON(http.StatusNotFound, utils.NewFailResponse("Monitor group not found"))
		return
	}
	ctx.JSON(http.StatusOK, utils.NewSuccessResponse("Monitor group updated", group))
}

func (c *Controller) SetMonitors(ctx *gin.Context) {
	var dto SetMonitorsDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, utils.NewFailResponse("Invalid request body"))
		return
	}
	group, err := c.service.SetMonitors(ctx, ctx.Param("id"), dto.MonitorIDs)
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	if group == nil {
		ctx.JSON(http.StatusNotFound, utils.NewFailResponse("Monitor group not found"))
		return
	}
	ctx.JSON(http.StatusOK, utils.NewSuccessResponse("Monitor group members updated", group))
}

func (c *Controller) Delete(ctx *gin.Context) {
	if err := c.service.Delete(ctx, ctx.Param("id")); err != nil {
		c.writeError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, utils.NewSuccessResponse[any]("Monitor group deleted", nil))
}

func (c *Controller) writeError(ctx *gin.Context, err error) {
	switch {
	case err.Error() == "monitor group not found":
		ctx.JSON(http.StatusNotFound, utils.NewFailResponse(err.Error()))
	case err.Error() == "monitor group name already exists":
		ctx.JSON(http.StatusConflict, utils.NewFailResponse(err.Error()))
	case err.Error() == "all monitors must belong to the active group",
		strings.HasPrefix(err.Error(), "invalid monitor ID"),
		strings.HasPrefix(err.Error(), "monitor group name"):
		ctx.JSON(http.StatusBadRequest, utils.NewFailResponse(err.Error()))
	default:
		c.logger.Errorw("Monitor group operation failed", "error", err)
		ctx.JSON(http.StatusInternalServerError, utils.NewFailResponse("Monitor group operation failed"))
	}
}
