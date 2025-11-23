package main

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func main() {
	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")
	passphrase := os.Getenv("WEBHOOK_PASSPHRASE")
	enableLiveOrders := os.Getenv("ENABLE_LIVE_ORDERS") == "true"

	binanceClient := NewBinanceClient(apiKey, secretKey)

	app := fiber.New(
		fiber.Config{
			Prefork:        false,
			ReadBufferSize: 8192,
			BodyLimit:      10 * 1024, // 10KB
			JSONDecoder:    Json.Unmarshal,
			JSONEncoder:    Json.Marshal,
			ReadTimeout:    5 * time.Second,
			WriteTimeout:   10 * time.Second,
			IdleTimeout:    10 * time.Second,
			ErrorHandler: func(c *fiber.Ctx, err error) error {
				code := fiber.StatusInternalServerError

				var e *fiber.Error
				if errors.As(err, &e) {
					code = e.Code
				}

				return c.Status(code).JSON(fiber.Map{
					"error": err.Error(),
				})
			},
		},
	)

	app.Use(limiter.New(limiter.Config{
		Max:        20,
		Expiration: 1 * time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).SendString("Rate limit exceeded")
		},
	}))

	app.Post("/webhook", Handler(binanceClient, passphrase, enableLiveOrders))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s...", port)
	log.Fatal(app.Listen(":" + port))
}
