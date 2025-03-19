package main

import (
	"log"
	"log/slog"

	"http-from-scratch/app/server"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	s := server.NewServer(":8097")
	log.Fatal(s.Start())
}
