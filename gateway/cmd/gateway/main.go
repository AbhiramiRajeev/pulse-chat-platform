package main

import (
	"log"
	"net/http"

	"github.com/AbhiramiRajeev/pulse-chat-platform/gateway/internal/router"
)

func main() {
	r := router.NewRouter()

	log.Println("Gateway listening on :8080")

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}