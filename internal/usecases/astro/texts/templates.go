package texts

import (
	"fmt"
	"strings"
	"time"
)

// FormatUnknownCommand форматирует сообщение о неизвестной команде
func FormatUnknownCommand(command string) string {
	return fmt.Sprintf(UnknownCommand, command)
}

// FormatBirthDateSuccess форматирует сообщение об успешном сохранении данных (но ошибка карты)
func FormatBirthDateSuccessButChartError(date, time, place string) string {
	return fmt.Sprintf(BirthDateSuccessButChartError, date, time, place)
}

// FormatBirthDateSuccess форматирует сообщение об успешном сохранении данных
func FormatBirthDateSuccess(date, time, place string) string {
	return fmt.Sprintf(BirthDateSuccess, date, time, place)
}

// FormatSubscriptionExpired форматирует сообщение об истекшей подписке
func FormatSubscriptionExpired(freeMessagesLimit int) string {
	return fmt.Sprintf(SubscriptionExpired, freeMessagesLimit)
}

// FormatMyInfo форматирует информацию о пользователе
func FormatMyInfo(birthDateTime *time.Time, birthPlace *string, natalChartExists bool, natalChartFetchedAt *time.Time, isPaidUser bool, manualGranted bool, freeMsgCount int, freeMessagesLimit int, expiryDate *time.Time) string {
	var message strings.Builder
	message.WriteString(MyInfoHeader)

	// Дата рождения
	if birthDateTime != nil {
		message.WriteString(fmt.Sprintf("📅 Дата рождения: %s\n", birthDateTime.Format("02.01.2006 15:04")))
		if birthPlace != nil {
			message.WriteString(fmt.Sprintf("📍 Место рождения: %s\n", *birthPlace))
		}
	} else {
		message.WriteString(MyInfoBirthDateNotSet)
	}

	// Натальная карта
	if natalChartExists {
		message.WriteString(MyInfoNatalChartExists)
		if natalChartFetchedAt != nil {
			message.WriteString(fmt.Sprintf("   Получена: %s\n", natalChartFetchedAt.Format("02.01.2006 15:04")))
		}
	} else {
		message.WriteString(MyInfoNatalChartNotSet)
		if birthDateTime != nil && birthPlace != nil {
			message.WriteString(MyInfoNatalChartUseStart)
		} else {
			message.WriteString(MyInfoNatalChartUseReset)
		}
	}

	message.WriteString("\n")

	// Тариф и сообщения
	if isPaidUser {
		message.WriteString(MyInfoTariffPaid)
		if !manualGranted {
			if expiryDate != nil {
				message.WriteString(MyInfoTariffUnlimited)
				message.WriteString(fmt.Sprintf("   Тариф активен до %s 🎉\n", expiryDate.Format("02.01.2006")))
			} else {
				message.WriteString(MyInfoTariffActive)
			}
		} else {
			message.WriteString(MyInfoTariffActiveManual)
		}
	} else {
		message.WriteString(MyInfoTariffNotPaid)
		remaining := freeMessagesLimit - freeMsgCount
		if remaining < 0 {
			remaining = 0
		}
		message.WriteString(fmt.Sprintf("🆓 Бесплатных сообщений осталось: %d из %d 🐱\n", remaining, freeMessagesLimit))
	}

	return message.String()
}

// pluralizeQuestion правильно склоняет слово "вопрос" в зависимости от числа
func pluralizeQuestion(count int) string {
	// Получаем последнюю цифру и две последние цифры
	lastDigit := count % 10
	lastTwoDigits := count % 100

	// Исключения для 11-14
	if lastTwoDigits >= 11 && lastTwoDigits <= 14 {
		return "вопросов"
	}

	// Склонение по последней цифре
	switch lastDigit {
	case 1:
		return "вопрос"
	case 2, 3, 4:
		return "вопроса"
	default:
		return "вопросов"
	}
}

// FormatPremiumLimitFreeWithLimit форматирует сообщение для бесплатников с остатком лимита
func FormatPremiumLimitFreeWithLimit(remaining int) string {
	questionWord := pluralizeQuestion(remaining)
	return fmt.Sprintf(PremiumLimitFreeWithLimit, remaining, questionWord)
}
