package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/alvsoaress/ancient-analytics-backend/internal/meta"
)

type MetaService interface {
	Heroes(
		ctx context.Context,
		minPicks int,
		sortBy string,
	) ([]meta.HeroPerformance, error)
}

type metaHeroesResponse struct {
	Scope  string                 `json:"scope"`
	Heroes []meta.HeroPerformance `json:"heroes"`
}

func metaHeroes(service MetaService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		query, err := url.ParseQuery(r.URL.RawQuery)
		minPicks := 0
		values, provided := query["minPicks"]
		if err == nil && provided && len(values) == 1 {
			minPicks, err = strconv.Atoi(values[0])
		}
		if err != nil || (provided && len(values) != 1) || minPicks < 0 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "INVALID_MIN_PICKS",
					"message": "minPicks deve ser um único número inteiro maior ou igual a zero.",
				},
			})
			return
		}

		sortBy := ""
		sortValues, sortProvided := query["sortBy"]
		if sortProvided && len(sortValues) == 1 {
			sortBy = sortValues[0]
		}
		if (sortProvided && len(sortValues) != 1) ||
			(sortBy != "" && sortBy != "winRate" && sortBy != "picks") {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "INVALID_SORT_BY",
					"message": "sortBy deve ser informado uma única vez como winRate ou picks.",
				},
			})
			return
		}

		heroes, err := service.Heroes(r.Context(), minPicks, sortBy)

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
