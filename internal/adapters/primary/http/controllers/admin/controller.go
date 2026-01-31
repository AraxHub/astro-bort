package admin

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	astroUsecase "github.com/admin/tg-bots/astro-bot/internal/usecases/astro"
	"github.com/gin-gonic/gin"
)

type Controller struct {
	AstroService *astroUsecase.Service
	Log          *slog.Logger
}

func New(
	astroService *astroUsecase.Service,
	log *slog.Logger,
) *Controller {
	return &Controller{
		AstroService: astroService,
		Log:          log,
	}
}

func (c *Controller) RegisterRoutes(router *gin.Engine) {
	admin := router.Group("/admin")
	{
		admin.POST("/images/sync", c.syncImages)
	}
}

// SyncImagesRequest запрос на синхронизацию картинок
type SyncImagesRequest struct {
	BotID           string   `json:"bot_id" binding:"required"`      // Bot ID для отправки картинок
	SyncChatID      int64    `json:"sync_chat_id" binding:"required"` // Chat ID для отправки картинок
	MessageThreadID *int64   `json:"message_thread_id,omitempty"`    // ID топика форума (опционально)
	Themes          []string `json:"themes"`                         // Темы для синхронизации (опционально, по умолчанию все)
}

// SyncImagesResponse ответ на запрос синхронизации
type SyncImagesResponse struct {
	Success      bool     `json:"success"`
	Processed    int      `json:"processed"`
	Created      int      `json:"created"`
	Errors       []string `json:"errors,omitempty"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// syncImages синхронизирует картинки из S3 в Telegram и БД
func (c *Controller) syncImages(ctx *gin.Context) {
	var req SyncImagesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		c.Log.Warn("failed to bind sync images request", "error", err)
		ctx.JSON(http.StatusBadRequest, SyncImagesResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Вызываем бизнес-логику (если themes пустой - синхронизируются все темы)
	var messageThreadID *int64
	if req.MessageThreadID != nil && *req.MessageThreadID != 0 {
		messageThreadID = req.MessageThreadID
	}

	result, err := c.AstroService.SyncImages(
		ctx.Request.Context(),
		domain.BotId(req.BotID),
		req.SyncChatID,
		messageThreadID,
		req.Themes, // если пустой - синхронизируются все темы
	)

	if err != nil {
		c.Log.Error("failed to sync images", "error", err)
		ctx.JSON(http.StatusInternalServerError, SyncImagesResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		})
		return
	}

	response := SyncImagesResponse{
		Success:   len(result.Errors) == 0,
		Processed: result.Processed,
		Created:   result.Created,
		Errors:    result.Errors,
	}

	ctx.JSON(http.StatusOK, response)
}
