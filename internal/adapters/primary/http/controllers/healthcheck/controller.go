package healthcheckController

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type HealthCheckController struct {
	db  *sqlx.DB
	log *slog.Logger
}

func New(db *sqlx.DB, log *slog.Logger) *HealthCheckController {
	return &HealthCheckController{
		db:  db,
		log: log,
	}
}

func (c *HealthCheckController) RegisterRoutes(r *gin.Engine) {
	r.GET("/health", c.health)
	r.GET("/ready", c.ready)
}

// health базовая проверка (всегда возвращает 200)
func (c *HealthCheckController) health(ctx *gin.Context) {
	ctx.JSON(200, gin.H{
		"status":   "ok",
		"usecases": "astro-bot",
	})
}

// ready проверка готовности (проверяет подключение к БД)
func (c *HealthCheckController) ready(ctx *gin.Context) {
	// Пингуем БД
	if err := c.db.Ping(); err != nil {
		c.log.Error("Database not ready", "error", err)
		ctx.JSON(503, gin.H{
			"status": "not ready",
			"error":  "database unavailable",
		})
		return
	}

	ctx.JSON(200, gin.H{
		"status": "ready",
	})
}
