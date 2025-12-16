package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Encoding string `envconfig:"ENCODING"`
	Level    string `envconfig:"LEVEL"`
}

func New(app string, cfg *Config) *slog.Logger {
	if cfg == nil {
		cfg = &Config{
			Encoding: "console",
			Level:    "info",
		}
	}

	if cfg.Level == "" {
		cfg.Level = "info"
	}

	if cfg.Encoding == "" {
		cfg.Encoding = "console"
	}

	// Парсим уровень логирования
	level := parseLevel(cfg.Level)

	// Создаем опции для логгера
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true, // fasle source - убирает длинные пути к файлам
	}

	var handler slog.Handler

	switch cfg.Encoding {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "console":
		handler = NewConsoleHandler(os.Stderr, opts)
	default:
		panic(fmt.Errorf("invalid logger config: encoding %s is not supported", cfg.Encoding))
	}

	// Создаем логгер с контекстом
	logger := slog.New(handler).With(
		"app", app,
	)

	return logger
}

// parseLevel парсит строковый уровень в slog.Level
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		panic(fmt.Errorf("invalid logger config: level %s is not supported", level))
	}
}

// ConsoleHandler реализует красивый консольный вывод для slog
type ConsoleHandler struct {
	handler slog.Handler
}

func NewConsoleHandler(w io.Writer, opts *slog.HandlerOptions) *ConsoleHandler {
	return &ConsoleHandler{
		handler: slog.NewTextHandler(w, opts),
	}
}

func (h *ConsoleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *ConsoleHandler) Handle(ctx context.Context, record slog.Record) error {
	// Простая реализация - используем TextHandler
	// В реальном проекте можно сделать более красивый вывод
	return h.handler.Handle(ctx, record)
}

func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ConsoleHandler{
		handler: h.handler.WithAttrs(attrs),
	}
}

func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	return &ConsoleHandler{
		handler: h.handler.WithGroup(name),
	}
}

// SetDefault устанавливает логгер по умолчанию
func SetDefault(logger *slog.Logger) {
	slog.SetDefault(logger)
}

// Helper функции для удобства
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	slog.DebugContext(ctx, msg, args...)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	slog.InfoContext(ctx, msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	slog.WarnContext(ctx, msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	slog.ErrorContext(ctx, msg, args...)
}
