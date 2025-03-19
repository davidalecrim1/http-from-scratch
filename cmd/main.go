package main

import (
	"log/slog"

	"http_from_scratch"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	s := http_from_scratch.NewServer(":8097")
	err := s.Start()
	if err != nil {
		slog.Error("failed to start server", "error", err)
	}
}
