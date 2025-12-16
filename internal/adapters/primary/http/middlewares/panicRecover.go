package middlewares

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func RecoveryLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("PANIC CAUGHT",
					"panic", r,
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"full_path", c.FullPath(),
					"client_ip", c.ClientIP(),
					"user_agent", c.Request.UserAgent(),
				)

				// Выводим стек трейс отдельно для читаемости
				log.Error("Stack trace:",
					"stack", string(debug.Stack()),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
					"data":  []interface{}{},
				})
			}
		}()
		c.Next()
	}
}
