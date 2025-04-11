package main

import (
	"log/slog"

	"fast/middleware/compress"

	"fast"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	app := fast.New(
		fast.Config{},
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
