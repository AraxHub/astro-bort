package testController

import (
	"github.com/admin/tg-bots/astro-bot/internal/usecases/test"
	"log/slog"
	"net/http"

	"github.com/admin/tg-bots/astro-bot/internal/domain"

	"github.com/gin-gonic/gin"
)

type Controller struct {
	TestService *testService.Service
	Log         *slog.Logger
}

func New(testService *testService.Service, log *slog.Logger) *Controller {
	return &Controller{
		TestService: testService,
		Log:         log,
	}
}

func (c *Controller) RegisterRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")
	{
		v1.POST("/test", c.handleTest)
	}
}

func (c *Controller) handleTest(ctx *gin.Context) {
	var req TestReq

	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		c.Log.Error(err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	test := &domain.Test{
		Filed1: req.Field1,
		Filed2: req.Field2,
	}

	err = c.TestService.SaveTest(ctx.Request.Context(), test)
	if err != nil {
		c.Log.Error(err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"testID": test.ID})
}
