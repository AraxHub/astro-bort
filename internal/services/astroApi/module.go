package astroApi

import (
	"context"
	"fmt"
	"time"

	astroApiAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/astroApi"
	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
)

// Service реализует IAstroAPIService для работы с астро-API
type Service struct {
	client *astroApiAdapter.Client
}

// New создаёт новый сервис для работы с астро-API
func New(client *astroApiAdapter.Client) service.IAstroAPIService {
	return &Service{
		client: client,
	}
}

// CalculateNatalChart рассчитывает натальную карту по дате рождения и месту
func (s *Service) CalculateNatalChart(ctx context.Context, birthDateTime time.Time, birthPlace string) (domain.NatalChart, error) {
	// Парсим место рождения (ожидаем формат "City, CountryCode" или просто "City")
	city, countryCode := parseBirthPlace(birthPlace)

	// Формируем BirthData из time.Time
	birthData := astroApiAdapter.BirthData{
		Year:        birthDateTime.Year(),
		Month:       int(birthDateTime.Month()),
		Day:         birthDateTime.Day(),
		Hour:        birthDateTime.Hour(),
		Minute:      birthDateTime.Minute(),
		Second:      birthDateTime.Second(),
		City:        city,
		CountryCode: countryCode,
	}

	// Формируем запрос
	req := astroApiAdapter.NatalChartRequest{
		Subject: astroApiAdapter.Person{
			Name:      "User", // Имя не важно для API
			BirthData: birthData,
		},
		Options: astroApiAdapter.ChartOptions{
			HouseSystem:  "P", // Плацидус
			ZodiacType:   "Tropic",
			ActivePoints: []string{"Sun", "Moon", "Mercury", "Venus", "Mars", "Jupiter", "Saturn", "Uranus", "Neptune", "Pluto"},
			Precision:    2,
		},
	}

	resp, err := s.client.CalculateNatalChart(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate natal chart: %w", err)
	}

	//костыль
	if resp.RawJSON == "" {
		return nil, fmt.Errorf("astro API returned empty response")
	}

	if resp.Status != "" && resp.Status != "success" {
		return nil, fmt.Errorf("astro API returned error: status=%s, code=%d, message=%s, raw_response=%s",
			resp.Status, resp.Code, resp.Message, resp.RawJSON)
	}

	// Возвращаем RawJSON как domain.NatalChart
	return domain.NatalChart(resp.RawJSON), nil
}

// GetNatalReport получает натальный отчёт по дате рождения и месту
func (s *Service) GetNatalReport(ctx context.Context, birthDateTime time.Time, birthPlace string) (domain.NatalReport, error) {
	city, countryCode := parseBirthPlace(birthPlace)

	birthData := astroApiAdapter.BirthData{
		Year:        birthDateTime.Year(),
		Month:       int(birthDateTime.Month()),
		Day:         birthDateTime.Day(),
		Hour:        birthDateTime.Hour(),
		Minute:      birthDateTime.Minute(),
		Second:      birthDateTime.Second(),
		City:        city,
		CountryCode: countryCode,
	}

	// Формируем запрос
	req := astroApiAdapter.NatalChartRequest{
		Subject: astroApiAdapter.Person{
			Name:      "User", // Имя не важно для API
			BirthData: birthData,
		},
		Options: astroApiAdapter.ChartOptions{
			HouseSystem:  "P", // Плацидус
			ZodiacType:   "Tropic",
			ActivePoints: []string{"Sun", "Moon", "Mercury", "Venus", "Mars", "Jupiter", "Saturn", "Uranus", "Neptune", "Pluto"},
			Precision:    2,
		},
	}

	// Выполняем запрос
	rawJSON, err := s.client.GetNatalReport(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get natal report: %w", err)
	}

	// Возвращаем raw JSON как domain.NatalReport
	return domain.NatalReport(rawJSON), nil
}

// todo рефактор работы с городами
// parseBirthPlace парсит место рождения на город и код страны
// Ожидаемые форматы: "City, CountryCode" или "City" (тогда используем дефолтный код)
func parseBirthPlace(birthPlace string) (city, countryCode string) {
	// Простой парсинг: если есть запятая, разделяем
	// В реальности может потребоваться более сложная логика
	if birthPlace == "" {
		return "Unknown", "US" // Дефолтные значения
	}

	// Пытаемся найти запятую
	for i, char := range birthPlace {
		if char == ',' {
			city = birthPlace[:i]
			countryCode = birthPlace[i+1:]
			// Убираем пробелы
			if len(city) > 0 && city[0] == ' ' {
				city = city[1:]
			}
			if len(countryCode) > 0 && countryCode[0] == ' ' {
				countryCode = countryCode[1:]
			}
			if countryCode == "" {
				countryCode = "US" // Дефолт если код страны не указан
			}
			return city, countryCode
		}
	}

	// Если запятой нет, используем весь текст как город
	return birthPlace, "US" // Дефолтный код страны
}
