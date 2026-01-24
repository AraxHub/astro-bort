package astro

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/usecases/astro/texts"
	"github.com/google/uuid"
)

// SendWeeklyForecastPush отправляет пуш "прогноз на неделю" всем пользователям
// Отправляется в Пн 10:00
// Отправляет сообщение с кнопкой "Прочитать" пользователям, у которых last_seen_at > 3 часа
func (s *Service) SendWeeklyForecastPush(ctx context.Context) error {
	s.Log.Info("starting weekly forecast push job")

	// Получаем пользователей, у которых last_seen_at > 3 часа или NULL
	users, err := s.UserRepo.GetUsersWithLastSeenOlderThan(ctx, 3)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	if len(users) == 0 {
		s.Log.Info("no users found for weekly forecast push")
		return nil
	}

	s.Log.Info("found users for weekly forecast push", "count", len(users))

	// Создаём генератор случайных чисел с текущим временем как seed
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Отправляем сообщения с задержкой между ними
	for i, user := range users {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond): // задержка между отправками
			}
		}

		// Получаем botID из последнего запроса пользователя
		botID, err := s.RequestRepo.GetBotIDForUser(ctx, user.ID)
		if err != nil {
			// Если запросов нет, пытаемся получить из последнего платежа
			if s.PaymentRepo != nil {
				paymentBotID, paymentErr := s.PaymentRepo.GetBotIDForUser(ctx, user.ID)
				if paymentErr == nil {
					botID = domain.BotId(paymentBotID)
				} else {
					// Если и платежей нет, пропускаем пользователя
					s.Log.Warn("failed to get bot_id for user (no requests or payments), skipping",
						"error", err,
						"payment_error", paymentErr,
						"user_id", user.ID)
					continue
				}
			} else {
				// Если PaymentRepo не настроен, пропускаем
				s.Log.Warn("failed to get bot_id for user, skipping (no payment repo)",
					"error", err,
					"user_id", user.ID)
				continue
			}
		}

		// Выбираем случайное сообщение
		message := texts.WeeklyForecastMessages[rng.Intn(len(texts.WeeklyForecastMessages))]

		// Создаём inline-клавиатуру с кнопкой "Прочитать"
		keyboard := map[string]interface{}{
			"inline_keyboard": [][]map[string]interface{}{
				{
					{
						"text":          "Прочитать",
						"callback_data": fmt.Sprintf("weekly_forecast:%s", user.ID.String()),
					},
				},
			},
		}

		// Отправляем сообщение с кнопкой
		if err := s.TelegramService.SendMessageWithKeyboard(ctx, botID, user.TelegramChatID, message, keyboard); err != nil {
			s.Log.Warn("failed to send weekly forecast push",
				"error", err,
				"user_id", user.ID,
				"bot_id", botID)
			// Продолжаем отправку остальным пользователям
			continue
		}

		s.Log.Debug("weekly forecast push sent",
			"user_id", user.ID,
			"bot_id", botID)
	}

	s.Log.Info("weekly forecast push job completed", "sent", len(users))
	return nil
}

// HandleWeeklyForecastCallback обрабатывает нажатие кнопки "Прочитать" для недельного прогноза
// Создаёт Request и отправляет в RAG
func (s *Service) HandleWeeklyForecastCallback(ctx context.Context, botID domain.BotId, user *domain.User) error {
	s.Log.Info("handling weekly forecast callback",
		"user_id", user.ID,
		"bot_id", botID)

	// Промпт для RAG (из texts для возможности редактирования тех-меном)
	ragPrompt := texts.WeeklyForecastRAGPrompt

	// Проверяем наличие натальной карты
	if user.NatalChartFetchedAt == nil {
		if err := s.fetchAndSaveNatalChart(ctx, user); err != nil {
			s.Log.Error("failed to fetch natal chart for weekly forecast",
				"error", err,
				"user_id", user.ID,
			)
			if sendErr := s.sendMessage(ctx, botID, user.TelegramChatID, "Для получения прогноза необходимо указать дату рождения. Используйте команду /reset_birth_data"); sendErr != nil {
				s.Log.Warn("failed to notify user about error", "error", sendErr)
			}
			return fmt.Errorf("failed to fetch natal chart: %w", err)
		}
	}

	// Создаём Request с типом RequestTypePushWeeklyForecast
	request := &domain.Request{
		ID:          uuid.New(),
		UserID:      user.ID,
		BotID:       botID,
		TGUpdateID:  nil, // для push нет update_id
		RequestType: domain.RequestTypePushWeeklyForecast,
		RequestText: ragPrompt,
		CreatedAt:   time.Now(),
	}

	if err := s.RequestRepo.Create(ctx, request); err != nil {
		s.Log.Error("failed to create weekly forecast request",
			"error", err,
			"user_id", user.ID,
		)
		if sendErr := s.sendMessage(ctx, botID, user.TelegramChatID, "Ошибка при создании запроса. Попробуйте позже."); sendErr != nil {
			s.Log.Warn("failed to notify user about error", "error", sendErr)
		}
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Получаем натальную карту для отправки в RAG
	natalReport, err := s.UserRepo.GetNatalChart(ctx, user.ID)
	if err != nil {
		s.Log.Error("failed to get natal chart for RAG",
			"error", err,
			"user_id", user.ID,
			"request_id", request.ID,
		)
		if sendErr := s.sendMessage(ctx, botID, user.TelegramChatID, "Ошибка при получении данных. Попробуйте позже."); sendErr != nil {
			s.Log.Warn("failed to notify user about error", "error", sendErr)
		}
		return fmt.Errorf("failed to get natal chart: %w", err)
	}

	// Отправляем в RAG через Kafka
	if s.KafkaProducer != nil {
		_, _, err := s.KafkaProducer.SendRAGRequest(ctx, request.ID, request.BotID, user.TelegramChatID, request.RequestText, natalReport)
		if err != nil {
			s.Log.Error("failed to send weekly forecast request to kafka",
				"error", err,
				"request_id", request.ID,
				"user_id", user.ID,
			)
			if sendErr := s.sendMessage(ctx, botID, user.TelegramChatID, "Ошибка при отправке запроса. Попробуйте позже."); sendErr != nil {
				s.Log.Warn("failed to notify user about error", "error", sendErr)
			}
			return fmt.Errorf("failed to send request to kafka: %w", err)
		}

		s.Log.Info("weekly forecast request sent to kafka",
			"request_id", request.ID,
			"user_id", user.ID,
		)
	} else {
		s.Log.Warn("kafka producer not configured, cannot send weekly forecast request",
			"request_id", request.ID,
		)
		return fmt.Errorf("kafka producer not configured")
	}

	return nil
}

// SendSituationalWarningPush отправляет пуш "ситуативное предупреждение" всем пользователям
// Отправляется в Ср 13:00 и Вс 9:00
// Для платников чередуется неделя через неделю
func (s *Service) SendSituationalWarningPush(ctx context.Context) error {
	s.Log.Info("starting situational warning push job")

	// TODO: реализация будет добавлена позже
	// 1. Получить всех активных пользователей
	// 2. Разделить на бесплатников и платников
	// 3. Для платников: проверить, нужно ли отправлять на этой неделе (чередование)
	// 4. Сгенерировать промпт для RAG с текущими позициями планет
	// 5. Для каждого пользователя: создать Request с RequestTypePushSituational
	// 6. Отправить в RAG через Kafka

	return fmt.Errorf("not implemented yet")
}

// SendPremiumLimitPush отправляет пуш "платный лимит" пользователям
// Отправляется в Пт 13:00
// Разный текст для бесплатников и платников (хардкодный, без RAG)
func (s *Service) SendPremiumLimitPush(ctx context.Context) error {
	s.Log.Info("starting premium limit push job")

	// TODO: реализация будет добавлена позже
	// 1. Получить всех активных пользователей
	// 2. Разделить на бесплатников и платников
	// 3. Для бесплатников: отправлять текст о лимите (зависит от FreeMsgCount)
	// 4. Для платников: отправлять текст о глубоком разборе
	// 5. Создавать Request с RequestTypePushPremiumLimit для истории
	// 6. Отправлять сообщения напрямую (без RAG)

	return fmt.Errorf("not implemented yet")
}

// shouldSendToPaidUsers проверяет, нужно ли отправлять пуш платным пользователям на этой неделе
// Чередование: одна неделя отправляем, следующая - нет
func (s *Service) shouldSendToPaidUsers() bool {
	_, week := time.Now().ISOWeek()
	// Если неделя чётная - отправляем, если нечётная - нет (или наоборот, зависит от стартовой недели)
	return week%2 == 0
}
