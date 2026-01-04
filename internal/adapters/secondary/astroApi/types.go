package astroApi

// BirthData представляет данные о рождении для API запроса
type BirthData struct {
	Year        int    `json:"year"`
	Month       int    `json:"month"`
	Day         int    `json:"day"`
	Hour        int    `json:"hour"`
	Minute      int    `json:"minute"`
	Second      int    `json:"second,omitempty"`
	City        string `json:"city"`
	CountryCode string `json:"country_code"`
}

// Person представляет субъекта натальной карты
type Person struct {
	Name      string    `json:"name"`
	BirthData BirthData `json:"birth_data"`
}

// ChartOptions представляет опции для расчета карты
type ChartOptions struct {
	HouseSystem  string   `json:"house_system"`  // "P" для Плацидуса
	ZodiacType   string   `json:"zodiac_type"`   // "Tropic" для тропического
	ActivePoints []string `json:"active_points"` // ["Sun", "Moon", ...]
	Precision    int      `json:"precision"`
}

// PositionsOptions представляет опции для запроса позиций (новый API)
type PositionsOptions struct {
	HouseSystem  string   `json:"house_system"`           // "P" для Плацидуса
	Language     string   `json:"language,omitempty"`     // "en"
	Tradition    string   `json:"tradition,omitempty"`    // "universal"
	DetailLevel  string   `json:"detail_level,omitempty"` // "standard"
	ZodiacType   string   `json:"zodiac_type"`            // "Tropic" для тропического
	ActivePoints []string `json:"active_points"`          // ["Sun", "Moon", ...]
	Precision    int      `json:"precision"`
}

// PositionsRequest представляет запрос на получение позиций (новый API)
type PositionsRequest struct {
	Subject Person           `json:"subject"`
	Options PositionsOptions `json:"options"`
}

// PositionsResponse представляет ответ API позиций
type PositionsResponse struct {
	RawJSON string `json:"-"` // Оригинальный JSON ответ для вывода
}

// NatalChartRequest представляет запрос на расчет натальной карты
type NatalChartRequest struct {
	Subject Person       `json:"subject"`
	Options ChartOptions `json:"options"`
}

// NatalChartResponse представляет ответ API
type NatalChartResponse struct {
	Status    string          `json:"status"`
	Code      int             `json:"code,omitempty"`
	Message   string          `json:"message,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Data      *NatalChartData `json:"data,omitempty"`
	RawJSON   string          `json:"-"` // Оригинальный JSON ответ для вывода
}

// NatalChartData представляет данные натальной карты (используется только для парсинга ответа)
type NatalChartData struct {
	Planets []PlanetPosition `json:"planets,omitempty"`
	Houses  []HousePosition  `json:"houses,omitempty"`
	Aspects []Aspect         `json:"aspects,omitempty"`
}

// PlanetPosition представляет позицию планеты (используется только для парсинга ответа)
type PlanetPosition struct {
	Name   string  `json:"name"`
	Sign   string  `json:"sign"`
	Degree float64 `json:"degree"`
	House  int     `json:"house,omitempty"`
}

// HousePosition представляет позицию дома (используется только для парсинга ответа)
type HousePosition struct {
	House  int     `json:"house"`
	Sign   string  `json:"sign"`
	Degree float64 `json:"degree"`
}

// Aspect представляет аспект между планетами (используется только для парсинга ответа)
type Aspect struct {
	Planet1 string  `json:"planet1"`
	Planet2 string  `json:"planet2"`
	Aspect  string  `json:"aspect"`
	Orb     float64 `json:"orb"`
}
