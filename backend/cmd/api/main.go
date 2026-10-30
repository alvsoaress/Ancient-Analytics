package main

import (
	"log"
	"net/http"

	"github.com/alvsoaress/ancient-analytics-backend/internal/api"
	"github.com/alvsoaress/ancient-analytics-backend/internal/builds"
	"github.com/alvsoaress/ancient-analytics-backend/internal/matches"
	"github.com/alvsoaress/ancient-analytics-backend/internal/meta"
	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
	"github.com/alvsoaress/ancient-analytics-backend/internal/players"
)

func main() {
	client := opendota.NewClient()
	metaService := meta.NewService(client)
	playersService := players.NewService(client)
	matchesService := matches.NewService(client)
	buildsService := builds.NewService(client)

	handler := api.NewHandler(client, metaService, playersService, matchesService, buildsService)
	log.Println("API disponível em http://127.0.0.1:8080")

	err := http.ListenAndServe("127.0.0.1:8080", handler)
	if err != nil {
		log.Fatal(err)
	}
}
