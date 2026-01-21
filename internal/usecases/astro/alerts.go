package astro

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

const (
	infoString = "❌ Ошибка обработки запроса\n%v\n\n"
	members    = "@nhoj41_3 @matarseks @romanovnl"
)

// sendAlertOrLog отправляет алерт в Telegram канал, не падает если алертер не настроен
func (s *Service) sendAlertOrLog(ctx context.Context, status *domain.Status) {
	if s.AlerterService == nil {
		return
	}

	message := s.formatAlertMessage(status)
	if message == "" {
		return
	}

	if err := s.AlerterService.SendAlert(ctx, message); err != nil {
		s.Log.Warn("failed to send alert (non-critical)",
			"error", err,
			"object_id", status.ObjectID,
			"status", status.Status,
		)
	}
}

// formatAlertMessage форматирует сообщение для алерта на основе статуса
func (s *Service) formatAlertMessage(status *domain.Status) string {
	var builder strings.Builder

	requestID := status.ObjectID.String()

	switch domain.RequestStatus(status.Status) {
	case domain.RequestError:
		builder.WriteString(fmt.Sprintf(infoString, members))
		builder.WriteString(fmt.Sprintf("🆔 Request ID: `%s`\n", requestID))

		// Определяем поток
		if len(status.Metadata) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(status.Metadata, &metadata); err == nil {
				if phase, ok := metadata["phase"].(string); ok {
					if phase == string(domain.PhaseSend) {
						builder.WriteString("📤 Прямой поток (-> request)\n")
					} else if phase == string(domain.PhaseReceive) {
						builder.WriteString("📥 Обратный поток (← response)\n")
					}
				}
			}
		}

		// Сообщение об ошибке
		if status.ErrorMessage != nil {
			errMsg := *status.ErrorMessage
			builder.WriteString(fmt.Sprintf("💬 Ошибка:%s\n", errMsg))
		}

	default:
		return ""
	}

	return builder.String()
}
