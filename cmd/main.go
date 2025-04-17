package main

import (
	"log/slog"
	"time"

	"fast/middleware/compress"

	"fast"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	app := fast.New(
		fast.Config{
			IdleTimeout: 30 * time.Second,
		},
	)

	app.Use(compress.New())

	app.Get("/test", func(ctx *fast.Ctx) error {
		return ctx.SendString("Hello World")
	})

	err := app.Listen(":8100")
	if err != nil {
		slog.Error("failed to start server", "error", err)
	}
}
