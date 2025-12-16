package middlewares

import (
	"time"

	"github.com/gin-gonic/gin"
	"log/slog"
)

func RequestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Получаем информацию о запросе
		req := c.Request

		// Логируем входящий запрос
		log.Info("incoming request",
			"method", req.Method,
			"path", req.URL.Path,
			"query", req.URL.RawQuery,
			"user_agent", req.UserAgent(),
			"remote_addr", req.RemoteAddr,
			"content_length", req.ContentLength,
		)

		// Продолжаем обработку запроса
		c.Next()

		// Засекаем время окончания
		latency := time.Since(start)

		// Получаем статус ответа
		status := c.Writer.Status()

		// Определяем уровень логирования в зависимости от статуса
		var logLevel slog.Level
		switch {
		case status >= 500:
			logLevel = slog.LevelError
		case status >= 400:
			logLevel = slog.LevelWarn
		default:
			logLevel = slog.LevelInfo
		}

		// Логируем ответ
		log.LogAttrs(nil, logLevel, "request completed",
			slog.String("method", req.Method),
			slog.String("path", req.URL.Path),
			slog.String("query", req.URL.RawQuery),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.Int("response_size", c.Writer.Size()),
			slog.String("user_agent", req.UserAgent()),
			slog.String("remote_addr", req.RemoteAddr),
		)
	}
}
