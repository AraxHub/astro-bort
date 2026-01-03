package telegram

import (
	"log/slog"
	"net/http"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	telegramService "github.com/admin/tg-bots/astro-bot/internal/services/telegram"
	"github.com/gin-gonic/gin"
)

type Controller struct {
	TgService *telegramService.Service
	Log       *slog.Logger
}

func New(TgService *telegramService.Service, log *slog.Logger) *Controller {
	return &Controller{
		TgService: TgService,
		Log:       log,
	}
}

func (c *Controller) RegisterRoutes(router *gin.Engine) {
	router.POST("/webhook/", c.handleWebhook)
}

func (c *Controller) handleWebhook(ctx *gin.Context) {
	secretToken := ctx.GetHeader("X-Telegram-Bot-Api-Secret-Token")
	if secretToken == "" {
		c.Log.Error("secret token is required")
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "secret token required"})
		return
	}

	botID := domain.BotId(secretToken)

	// Валидируем, что bot_id существует в конфигурации
	if _, err := c.TgService.GetBotType(botID); err != nil {
		c.Log.Warn("unknown bot_id in webhook",
			"bot_id", botID,
			"error", err,
		)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "unknown bot_id"})
		return
	}

	var update domain.Update

	if err := ctx.ShouldBindJSON(&update); err != nil {
		c.Log.Error("failed to bind webhook request",
			"error", err,
			"bot_id", botID,
		)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.Log.Debug("received webhook update",
		"update_id", update.UpdateID,
		"bot_id", botID,
	)

	if err := c.TgService.HandleUpdate(ctx.Request.Context(), botID, &update); err != nil {
		c.Log.Error("failed to handle update",
			"error", err,
			"update_id", update.UpdateID,
			"bot_id", botID,
		)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process update"})
		return
	}

	// Telegram ожидает 200 OK в ответ
	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}
