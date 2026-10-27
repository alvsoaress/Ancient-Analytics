package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

type HeroesClient interface {
	Heroes(ctx context.Context) ([]opendota.Hero, error)
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func NewHandler(client HeroesClient, metaService MetaService) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", health)

	mux.HandleFunc("GET /api/heroes", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		heroes, err := client.Heroes(r.Context())

		w.Header().Set("Content-Type", "application/json")

		if err != nil {
			w.WriteHeader(http.StatusBadGateway)

			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "UPSTREAM_UNAVAILABLE",
					"message": "Não foi possível consultar os heróis.",
				},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(heroes)
	})

	mux.HandleFunc("GET /api/meta/heroes", metaHeroes(metaService))

	return mux
}
