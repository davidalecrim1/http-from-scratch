package recovery

import (
	"log/slog"

	"fast"
)

func New() fast.Handler {
	return func(c *fast.Ctx) error {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("recovering from panic", "error", err)
				c.Status(fast.StatusServiceUnavailable).SendString("received an error while executing")
			}
		}()

		return c.Next()
	}
}
