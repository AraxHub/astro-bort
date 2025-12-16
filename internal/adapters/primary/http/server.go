package server

import (
	"net"
	"net/http"
	"time"

	"log/slog"

	"github.com/gin-gonic/gin"
)

type Config struct {
	Host                    string        `envconfig:"HOST"`
	Port                    string        `envconfig:"PORT"`
	WriteTimeout            time.Duration `envconfig:"WRITE_TIMEOUT" default:"60s"`
	ReadTimeout             time.Duration `envconfig:"READ_TIMEOUT" default:"3s"`
	ReadHeaderTimeout       time.Duration `envconfig:"READ_HEADER_TIMEOUT" default:"3s"`
	IdleTimeout             time.Duration `envconfig:"IDLE_TIMEOUT" default:"15s"`
	EnableLoggingMiddleware bool          `envconfig:"ENABLE_LOGGING_MIDDLEWARE" default:"false"`
}

type Controller interface {
	RegisterRoutes(router *gin.Engine)
}

func NewHTTPServer(
	cfg *Config,
	logger *slog.Logger,
	controllers ...Controller,
) *http.Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Регистрируем маршруты всех контроллеров
	for _, controller := range controllers {
		controller.RegisterRoutes(router)
	}

	server := &http.Server{
		Handler:           router,
		Addr:              net.JoinHostPort(cfg.Host, cfg.Port),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	return server
}
