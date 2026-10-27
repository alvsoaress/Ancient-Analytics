package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/alvsoaress/ancient-analytics-backend/internal/meta"
)

type MetaService interface {
	Heroes(
		ctx context.Context,
		minPicks int,
	) ([]meta.HeroPerformance, error)
}

type metaHeroesResponse struct {
	Scope  string                 `json:"scope"`
	Heroes []meta.HeroPerformance `json:"heroes"`
}

func metaHeroes(service MetaService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		heroes, err := service.Heroes(r.Context(), 0)

		w.Header().Set("Content-Type", "application/json")

		if err != nil {
			w.WriteHeader(http.StatusBadGateway)

			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "UPSTREAM_UNAVAILABLE",
					"message": "Não foi possível consultar as estatísticas de meta.",
				},
			})
			return
		}

		response := metaHeroesResponse{
			Scope:  "professional",
			Heroes: heroes,
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}
