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

	// Отправляем статистику в алерт
	if s.AlerterService != nil {
		alertMsg := fmt.Sprintf("📊 Weekly Forecast Push завершён\n\n"+
			"Отправлено сообщений: %d",
			len(users))
		if err := s.AlerterService.SendAlert(ctx, alertMsg); err != nil {
			s.Log.Warn("failed to send weekly forecast push alert", "error", err)
		}
	}

	return nil
}

// HandleWeeklyForecastCallback обрабатывает нажатие кнопки "Прочитать" для недельного прогноза
// Создаёт Request и отправляет в RAG
func (s *Service) HandleWeeklyForecastCallback(ctx context.Context, botID domain.BotId, user *domain.User, messageID int64, chatID int64) error {
	s.Log.Info("handling weekly forecast callback",
		"user_id", user.ID,
		"bot_id", botID,
		"message_id", messageID,
		"chat_id", chatID)

	// Убираем кнопку из сообщения (передаём пустой reply_markup)
	if err := s.TelegramService.EditMessageReplyMarkup(ctx, botID, chatID, messageID, nil); err != nil {
		s.Log.Warn("failed to remove button from message, continuing anyway",
			"error", err,
			"user_id", user.ID,
			"message_id", messageID,
		)
		// Продолжаем работу даже если не удалось убрать кнопку
	}

	// Отправляем случайное сообщение "секундочку..."
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	generatingMessage := texts.WeeklyForecastGeneratingMessages[rng.Intn(len(texts.WeeklyForecastGeneratingMessages))]
	if err := s.sendMessage(ctx, botID, chatID, generatingMessage); err != nil {
		s.Log.Warn("failed to send generating message, continuing anyway",
			"error", err,
			"user_id", user.ID,
		)
		// Продолжаем работу даже если не удалось отправить сообщение
	}

	ragPrompt := texts.WeeklyForecastRAGPrompt

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

	if s.KafkaProducer != nil {
		_, _, err := s.KafkaProducer.SendRAGRequest(ctx, request.ID, request.BotID, user.TelegramChatID, request.RequestText, natalReport, request.RequestType)
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

// HandlePremiumLimitPaymentCallback обрабатывает нажатие на кнопку "Заплатить" в Premium Limit Push
func (s *Service) HandlePremiumLimitPaymentCallback(ctx context.Context, botID domain.BotId, user *domain.User) error {
	s.Log.Info("handling premium limit payment callback",
		"user_id", user.ID,
		"bot_id", botID)

	// Проверяем, что пользователь действительно бесплатный и лимит израсходован
	isPaidUser := user.IsPaid || user.ManualGranted
	if isPaidUser {
		s.Log.Warn("paid user clicked premium limit pay button",
			"user_id", user.ID,
			"bot_id", botID)
		if err := s.sendMessage(ctx, botID, user.TelegramChatID, "У вас уже есть активная подписка."); err != nil {
			s.Log.Warn("failed to send message to paid user", "error", err)
		}
		return nil
	}

	remaining := s.FreeMessagesLimit - user.FreeMsgCount
	if remaining > 0 {
		s.Log.Warn("free user with remaining limit clicked pay button",
			"user_id", user.ID,
			"bot_id", botID,
			"remaining", remaining)
		if err := s.sendMessage(ctx, botID, user.TelegramChatID, fmt.Sprintf("У вас ещё осталось %d бесплатных вопросов.", remaining)); err != nil {
			s.Log.Warn("failed to send message to free user", "error", err)
		}
		return nil
	}

	// Создаём платеж (invoice отправится автоматически)
	if s.PaymentService != nil {
		productID := "monthly_feed"
		productTitle := texts.BuyMonthlyFeedTitle
		description := texts.BuyMonthlyFeedDescription
		amount := s.StarsPrice

		_, paymentErr := s.PaymentService.CreatePayment(
			ctx,
			botID,
			user.ID,
			user.TelegramChatID,
			productID,
			productTitle,
			description,
			amount,
		)
		if paymentErr != nil {
			s.Log.Error("failed to create payment for premium limit push",
				"error", paymentErr,
				"user_id", user.ID,
				"bot_id", botID,
			)
			if sendErr := s.sendMessage(ctx, botID, user.TelegramChatID, "Ошибка при создании платежа. Попробуйте позже."); sendErr != nil {
				s.Log.Warn("failed to notify user about payment error", "error", sendErr)
			}
			return fmt.Errorf("failed to create payment: %w", paymentErr)
		}

		s.Log.Info("payment created for premium limit push",
			"user_id", user.ID,
			"bot_id", botID)
	} else {
		s.Log.Error("payment service not configured",
			"user_id", user.ID,
			"bot_id", botID)
		if err := s.sendMessage(ctx, botID, user.TelegramChatID, "Платежи временно недоступны. Попробуйте позже."); err != nil {
			s.Log.Warn("failed to notify user about payment unavailability", "error", err)
		}
		return fmt.Errorf("payment service not configured")
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

	// Получаем пользователей, у которых last_seen_at > 1 час или NULL
	users, err := s.UserRepo.GetUsersWithLastSeenOlderThan(ctx, 1)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	if len(users) == 0 {
		s.Log.Info("no users found for premium limit push")
		return nil
	}

	// Фильтруем только пользователей с натальной картой
	var usersWithChart []*domain.User
	for _, user := range users {
		if user.NatalChartFetchedAt != nil {
			usersWithChart = append(usersWithChart, user)
		}
	}

	if len(usersWithChart) == 0 {
		s.Log.Info("no users with natal chart found for premium limit push")
		return nil
	}

	s.Log.Info("found users for premium limit push", "count", len(usersWithChart))

	// Разделяем на платников и бесплатников
	var paidUsers []*domain.User
	var freeUsers []*domain.User

	for _, user := range usersWithChart {
		if user.IsPaid {
			paidUsers = append(paidUsers, user)
		} else {
			freeUsers = append(freeUsers, user)
		}
	}

	s.Log.Info("users split",
		"paid_count", len(paidUsers),
		"free_count", len(freeUsers))

	// Создаём генератор случайных чисел
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Обрабатываем платников (неделя через неделю)
	if len(paidUsers) > 0 && s.shouldSendToPaidUsers() {
		s.Log.Info("sending premium limit push to paid users", "count", len(paidUsers))
		for i, user := range paidUsers {
			if i > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(100 * time.Millisecond):
				}
			}

			// Получаем botID из последнего запроса или платежа
			botID, err := s.RequestRepo.GetBotIDForUser(ctx, user.ID)
			if err != nil {
				if s.PaymentRepo != nil {
					paymentBotID, paymentErr := s.PaymentRepo.GetBotIDForUser(ctx, user.ID)
					if paymentErr == nil {
						botID = domain.BotId(paymentBotID)
					} else {
						s.Log.Warn("failed to get bot_id for paid user, skipping",
							"error", err,
							"payment_error", paymentErr,
							"user_id", user.ID)
						continue
					}
				} else {
					s.Log.Warn("failed to get bot_id for paid user, skipping (no payment repo)",
						"error", err,
						"user_id", user.ID)
					continue
				}
			}

			// Выбираем случайное сообщение для платников
			message := texts.PremiumLimitPaidMessages[rng.Intn(len(texts.PremiumLimitPaidMessages))]

			// Создаём Request для истории
			request := &domain.Request{
				ID:          uuid.New(),
				UserID:      user.ID,
				BotID:       botID,
				TGUpdateID:  nil,
				RequestType: domain.RequestTypePushPremiumLimit,
				RequestText: message,
				CreatedAt:   time.Now(),
			}

			if err := s.RequestRepo.Create(ctx, request); err != nil {
				s.Log.Warn("failed to create premium limit push request, continuing anyway",
					"error", err,
					"user_id", user.ID)
			}

			// Отправляем сообщение
			if err := s.sendMessage(ctx, botID, user.TelegramChatID, message); err != nil {
				s.Log.Warn("failed to send premium limit push to paid user, continuing anyway",
					"error", err,
					"user_id", user.ID,
					"bot_id", botID)
				continue
			}

			// Обновляем last_seen_at после успешной отправки
			if err := s.UserRepo.UpdateLastSeen(ctx, user.ID); err != nil {
				s.Log.Warn("failed to update last_seen_at for paid user, continuing anyway",
					"error", err,
					"user_id", user.ID)
			}

			s.Log.Debug("premium limit push sent to paid user",
				"user_id", user.ID,
				"bot_id", botID)
		}
	} else if len(paidUsers) > 0 {
		s.Log.Info("skipping paid users this week (alternation)")
	}

	// Обрабатываем бесплатников (каждую неделю)
	if len(freeUsers) > 0 {
		s.Log.Info("sending premium limit push to free users", "count", len(freeUsers))
		for i, user := range freeUsers {
			if i > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(100 * time.Millisecond):
				}
			}

			// Получаем botID из последнего запроса или платежа
			botID, err := s.RequestRepo.GetBotIDForUser(ctx, user.ID)
			if err != nil {
				if s.PaymentRepo != nil {
					paymentBotID, paymentErr := s.PaymentRepo.GetBotIDForUser(ctx, user.ID)
					if paymentErr == nil {
						botID = domain.BotId(paymentBotID)
					} else {
						s.Log.Warn("failed to get bot_id for free user, skipping",
							"error", err,
							"payment_error", paymentErr,
							"user_id", user.ID)
						continue
					}
				} else {
					s.Log.Warn("failed to get bot_id for free user, skipping (no payment repo)",
						"error", err,
						"user_id", user.ID)
					continue
				}
			}

			// Определяем текст в зависимости от лимита
			var message string
			remaining := s.FreeMessagesLimit - user.FreeMsgCount
			if remaining > 0 {
				message = texts.FormatPremiumLimitFreeWithLimit(remaining)
			} else {
				message = texts.PremiumLimitFreeNoLimit
			}

			// Создаём Request для истории
			request := &domain.Request{
				ID:          uuid.New(),
				UserID:      user.ID,
				BotID:       botID,
				TGUpdateID:  nil,
				RequestType: domain.RequestTypePushPremiumLimit,
				RequestText: message,
				CreatedAt:   time.Now(),
			}

			if err := s.RequestRepo.Create(ctx, request); err != nil {
				s.Log.Warn("failed to create premium limit push request, continuing anyway",
					"error", err,
					"user_id", user.ID)
			}

			// Отправляем сообщение
			// Если лимит израсходован, добавляем кнопку "Заплатить"
			if remaining <= 0 {
				keyboard := map[string]interface{}{
					"inline_keyboard": [][]map[string]interface{}{
						{
							{
								"text":          "Заплатить",
								"callback_data": fmt.Sprintf("premium_limit_pay:%s", user.ID.String()),
							},
						},
					},
				}

				if err := s.sendMessageWithKeyboard(ctx, botID, user.TelegramChatID, message, keyboard); err != nil {
					s.Log.Warn("failed to send premium limit push to free user with button, continuing anyway",
						"error", err,
						"user_id", user.ID,
						"bot_id", botID)
					continue
				}
			} else {
				if err := s.sendMessage(ctx, botID, user.TelegramChatID, message); err != nil {
					s.Log.Warn("failed to send premium limit push to free user, continuing anyway",
						"error", err,
						"user_id", user.ID,
						"bot_id", botID)
					continue
				}
			}

			// Обновляем last_seen_at после успешной отправки
			if err := s.UserRepo.UpdateLastSeen(ctx, user.ID); err != nil {
				s.Log.Warn("failed to update last_seen_at for free user, continuing anyway",
					"error", err,
					"user_id", user.ID)
			}

			s.Log.Debug("premium limit push sent to free user",
				"user_id", user.ID,
				"bot_id", botID,
				"remaining", remaining)
		}
	}

	var paidSent int
	if s.shouldSendToPaidUsers() {
		paidSent = len(paidUsers)
	} else {
		paidSent = 0
	}
	freeSent := len(freeUsers)
	totalSent := paidSent + freeSent

	s.Log.Info("premium limit push job completed",
		"paid_sent", paidSent,
		"free_sent", freeSent,
		"total_sent", totalSent)

	// Отправляем статистику в алерт
	if s.AlerterService != nil {
		alertMsg := fmt.Sprintf("📊 Premium Limit Push завершён\n\n"+
			"Платники: %d\n"+
			"Бесплатники: %d\n"+
			"Всего отправлено: %d",
			paidSent, freeSent, totalSent)
		if err := s.AlerterService.SendAlert(ctx, alertMsg); err != nil {
			s.Log.Warn("failed to send premium limit push alert", "error", err)
		}
	}

	return nil
}

// shouldSendToPaidUsers для платников одна неделя отправляем, следующая - нет
func (s *Service) shouldSendToPaidUsers() bool {
	_, week := time.Now().ISOWeek()
	return week%2 == 0
}
