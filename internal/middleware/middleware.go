package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"
)

func New(logger *zap.Logger) *fiber.App {

	app := fiber.New(
		fiber.Config{
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	)
	app.Use(recover.New())
	app.Use(loggerMiddleware(logger))
	return app
}

func loggerMiddleware(logger *zap.Logger) fiber.Handler {
	return func (c *fiber.Ctx) error {
		start := time.Now()
		
		logger.Info("Incoming request",
		zap.String("Method", c.Method()),
		zap.String("URL", c.OriginalURL()),
		zap.ByteString("Request body", c.Body()),)
	
		err := c.Next()

		duration := time.Since(start)
		logger.Info("Outgoing response",
		zap.String("Method", c.Method()),
		zap.String("URL", c.OriginalURL()),
		zap.ByteString("Response body", c.Response().Body()),
		zap.Int("Status", c.Response().StatusCode()),
		zap.Duration("Duration", duration),
	)
		return err
	}
}