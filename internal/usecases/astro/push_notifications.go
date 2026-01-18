package astro

import (
	"context"
	"fmt"
	"time"
)

// SendWeeklyForecastPush отправляет пуш "прогноз на неделю" всем пользователям
// Отправляется в Пн 10:00
// Для платников чередуется неделя через неделю
func (s *Service) SendWeeklyForecastPush(ctx context.Context) error {
	s.Log.Info("starting weekly forecast push job")

	// TODO: реализация будет добавлена позже
	// 1. Получить всех активных пользователей
	// 2. Разделить на бесплатников и платников
	// 3. Для платников: проверить, нужно ли отправлять на этой неделе (чередование)
	// 4. Сгенерировать промпт для RAG с текущими позициями планет
	// 5. Для каждого пользователя: создать Request с RequestTypePushWeeklyForecast
	// 6. Отправить в RAG через Kafka

	return fmt.Errorf("not implemented yet")
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
