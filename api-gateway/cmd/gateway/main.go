package main

import (
	"fmt"

	"github.com/AbhiramiRajeev/pulse-chat-platform/api-gateway/internal/config"
)


func main() {
	print("Starting gateway service")

	cfg :=config.Load()
	fmt.Println("Listening on Http port:",cfg.HTTPPort)
}