package main

import (
	"log"

	"block-explorer-backend/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("server start failed: %v", err)
	}
}
