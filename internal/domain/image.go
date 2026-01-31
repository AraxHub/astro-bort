package domain

import (
	"time"

	"github.com/google/uuid"
)

// ImageTheme тема картинки
type ImageTheme string

const (
	ImageThemeLove       ImageTheme = "Love"       // Любовь
	ImageThemeFuture     ImageTheme = "Future"     // Будущее
	ImageThemeCrisis     ImageTheme = "Crisis"     // Кризис
	ImageThemeBusiness   ImageTheme = "Business"   // Бизнес
	ImageThemePersonality ImageTheme = "Personality" // Личность
)

// AllImageThemes возвращает все доступные темы картинок
func AllImageThemes() []ImageTheme {
	return []ImageTheme{
		ImageThemeLove,
		ImageThemeFuture,
		ImageThemeCrisis,
		ImageThemeBusiness,
		ImageThemePersonality,
	}
}

// String возвращает строковое представление темы
func (t ImageTheme) String() string {
	return string(t)
}

// S3Path возвращает путь к папке темы в S3 (themes/Love/)
func (t ImageTheme) S3Path() string {
	return "themes/" + string(t) + "/"
}

// IsValid проверяет, является ли тема валидной
func (t ImageTheme) IsValid() bool {
	switch t {
	case ImageThemeLove, ImageThemeFuture, ImageThemeCrisis, ImageThemeBusiness, ImageThemePersonality:
		return true
	default:
		return false
	}
}

// Image представляет метаданные картинки из S3
type Image struct {
	ID        uuid.UUID   `json:"id" db:"id"`
	Filename  string      `json:"filename" db:"filename"`
	TgFileID  string      `json:"tg_file_id" db:"tg_file_id"`
	Theme     *ImageTheme `json:"theme,omitempty" db:"theme"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
}
