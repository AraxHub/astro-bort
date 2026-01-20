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
	router.POST("/webhook", c.handleWebhook)
}
//TODO fix на стороне бизнес-логики с флоу даты рождения, временное решение, чтобы не зациклить бота
func (c *Controller) handleWebhook(ctx *gin.Context) {
	secretToken := ctx.GetHeader("X-Telegram-Bot-Api-Secret-Token")
	if secretToken == "" {
		c.Log.Error("secret token is required")
		// Возвращаем 200, чтобы Telegram не повторял запрос
		ctx.JSON(http.StatusOK, gin.H{"ok": false, "error": "secret token required"})
		return
	}

	botID := domain.BotId(secretToken)

	if _, err := c.TgService.GetBotType(botID); err != nil {
		c.Log.Error("unknown bot_id in webhook",
			"bot_id", botID,
			"error", err,
		)
		// Возвращаем 200, чтобы Telegram не повторял запрос
		ctx.JSON(http.StatusOK, gin.H{"ok": false, "error": "unknown bot_id"})
		return
	}

	var update domain.Update

	if err := ctx.ShouldBindJSON(&update); err != nil {
		c.Log.Error("failed to bind webhook request",
			"error", err,
			"bot_id", botID,
		)
		// Возвращаем 200, чтобы Telegram не повторял запрос
		ctx.JSON(http.StatusOK, gin.H{"ok": false, "error": "invalid request"})
		return
	}

	c.Log.Debug("received webhook update",
		"update_id", update.UpdateID,
		"bot_id", botID,
	)

	if err := c.TgService.HandleUpdate(ctx.Request.Context(), botID, &update); err != nil {
		if !domain.IsBusinessError(err) {
			c.Log.Error("failed to handle webhook update",
				"error", err,
				"bot_id", botID)
		}
		// Возвращаем 200, чтобы Telegram не повторял запрос
		ctx.JSON(http.StatusOK, gin.H{"ok": false, "error": "failed to process update"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}
