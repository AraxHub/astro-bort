package alerter

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
	"github.com/gin-gonic/gin"
)

type Controller struct {
	AlerterService service.IAlerterService
	Log            *slog.Logger
}

func New(alerterService service.IAlerterService, log *slog.Logger) *Controller {
	return &Controller{
		AlerterService: alerterService,
		Log:            log,
	}
}

func (c *Controller) RegisterRoutes(router *gin.Engine) {
	router.POST("/webhooks/railway", c.handleRailwayWebhook)
	router.POST("/webhooks/alert", c.handleGenericAlert)
}

func (c *Controller) handleRailwayWebhook(ctx *gin.Context) {
	var payload RailwayWebhookPayload

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		c.Log.Warn("failed to bind railway webhook request",
			"error", err,
		)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.Log.Debug("received railway webhook",
		"type", payload.Type,
		"service", payload.Resource.Service.Name,
		"project", payload.Resource.Project.Name,
		"severity", payload.Severity,
	)

	// Форматируем сообщение для Telegram
	message := c.formatMessage(payload)

	// Если алертер не настроен, просто логируем и возвращаем 200
	if c.AlerterService == nil {
		c.Log.Info("alerter service not configured, skipping alert",
			"type", payload.Type,
		)
		ctx.JSON(http.StatusOK, gin.H{"ok": true, "message": "alerter not configured"})
		return
	}

	// Отправляем алерт
	if err := c.AlerterService.SendAlert(ctx.Request.Context(), message); err != nil {
		c.Log.Warn("failed to send alert",
			"error", err,
			"type", payload.Type,
		)
		// Возвращаем 200, чтобы Railway не повторял запрос
		ctx.JSON(http.StatusOK, gin.H{"ok": false, "error": "failed to send alert"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}

// formatMessage форматирует Railway webhook payload в читаемое сообщение для Telegram
func (c *Controller) formatMessage(payload RailwayWebhookPayload) string {
	var builder strings.Builder

	// Заголовок с типом события и severity
	builder.WriteString("🚨 ")
	builder.WriteString(formatEventType(payload.Type))
	if payload.Severity != "" {
		builder.WriteString(" [")
		builder.WriteString(payload.Severity)
		builder.WriteString("]")
	}
	builder.WriteString("\n\n")

	// Информация о проекте и сервисе
	builder.WriteString("📦 ")
	builder.WriteString(payload.Resource.Project.Name)
	if payload.Resource.Service.Name != "" {
		builder.WriteString(" / ")
		builder.WriteString(payload.Resource.Service.Name)
	}
	builder.WriteString("\n")

	// Окружение
	if payload.Resource.Environment.Name != "" {
		builder.WriteString("🌍 Окружение: ")
		builder.WriteString(payload.Resource.Environment.Name)
		if payload.Resource.Environment.IsEphemeral {
			builder.WriteString(" (Ephemeral)")
		}
		builder.WriteString("\n")
	}

	// Статус деплоя
	if payload.Details.Status != "" {
		builder.WriteString("📊 Статус: ")
		builder.WriteString(formatStatus(payload.Details.Status))
		builder.WriteString("\n")
	}

	// Ветка и коммит
	if payload.Details.Branch != "" {
		builder.WriteString("🌿 Ветка: ")
		builder.WriteString(payload.Details.Branch)
		builder.WriteString("\n")
	}

	if payload.Details.CommitHash != "" {
		commitShort := payload.Details.CommitHash
		if len(commitShort) > 7 {
			commitShort = commitShort[:7]
		}
		builder.WriteString("🔹 Коммит: ")
		builder.WriteString(commitShort)
		if payload.Details.CommitAuthor != "" {
			builder.WriteString(" (")
			builder.WriteString(payload.Details.CommitAuthor)
			builder.WriteString(")")
		}
		builder.WriteString("\n")
	}

	if payload.Details.CommitMessage != "" {
		builder.WriteString("💬 Сообщение: ")
		// Ограничиваем длину сообщения коммита
		commitMsg := payload.Details.CommitMessage
		if len(commitMsg) > 100 {
			commitMsg = commitMsg[:100] + "..."
		}
		builder.WriteString(commitMsg)
		builder.WriteString("\n")
	}

	// Источник
	if payload.Details.Source != "" {
		builder.WriteString("🔗 Источник: ")
		builder.WriteString(payload.Details.Source)
		builder.WriteString("\n")
	}

	// Время
	if payload.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, payload.Timestamp); err == nil {
			builder.WriteString("⏰ Время: ")
			builder.WriteString(t.Format("02.01.2006 15:04:05"))
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// formatEventType форматирует тип события в читаемый формат
func formatEventType(eventType string) string {
	// Убираем точку и делаем заглавными первые буквы
	parts := strings.Split(eventType, ".")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
}

// formatStatus форматирует статус с эмодзи
func formatStatus(status string) string {
	statusUpper := strings.ToUpper(status)
	switch statusUpper {
	case "SUCCESS":
		return "✅ SUCCESS"
	case "FAILED":
		return "❌ FAILED"
	case "BUILDING":
		return "🔨 BUILDING"
	case "DEPLOYING":
		return "🚀 DEPLOYING"
	case "INACTIVE":
		return "💤 INACTIVE"
	default:
		return status
	}
}

// handleGenericAlert обрабатывает универсальный алерт в свободной форме
func (c *Controller) handleGenericAlert(ctx *gin.Context) {
	var payload GenericAlertPayload

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		c.Log.Warn("failed to bind generic alert request",
			"error", err,
		)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	// Валидация: message обязателен
	if payload.Message == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}

	c.Log.Debug("received generic alert",
		"message_length", len(payload.Message),
		"source", payload.Source,
	)

	if c.AlerterService == nil {
		c.Log.Info("alerter service not configured, skipping alert",
			"source", payload.Source,
		)
		ctx.JSON(http.StatusOK, gin.H{"ok": true, "message": "alerter not configured"})
		return
	}

	message := payload.Message
	if payload.Source != "" {
		message = fmt.Sprintf("🔔 Источник алерта: %s\n\n Сообщение:%s", payload.Source, payload.Message)
	}

	if err := c.AlerterService.SendAlert(ctx.Request.Context(), message); err != nil {
		c.Log.Warn("failed to send alert",
			"error", err,
			"source", payload.Source,
		)
		ctx.JSON(http.StatusOK, gin.H{"ok": false, "error": "failed to send alert"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}
