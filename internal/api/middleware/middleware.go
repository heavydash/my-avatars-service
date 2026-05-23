package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Setup настраивает базовый набор middleware
func Setup(e *echo.Echo) {
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())

	// Логирование HTTP запросов
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `[${time_rfc3339}] ${method} ${uri} ${status} ${latency_human} ${remote_ip} ${error}` + "\n",
	}))
}
