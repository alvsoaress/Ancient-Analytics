package main

import (
	"log"
	"net/http"

	"github.com/alvsoaress/ancient-analytics-backend/internal/api"
)

func main() {
	handler := api.NewHandler()

	log.Println("API disponível em http://127.0.0.1:8080")

	err := http.ListenAndServe("127.0.0.1:8080", handler)
	if err != nil {
		log.Fatal(err)
	}
}
