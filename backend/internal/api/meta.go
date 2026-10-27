package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/alvsoaress/ancient-analytics-backend/internal/meta"
	"github.com/alvsoaress/ancient-analytics-backend/internal/opendota"
)

type MetaService interface {
	Heroes(
		ctx context.Context,
		minPicks int,
		sortBy string,
	) ([]meta.HeroPerformance, error)
	HeroesByPosition(ctx context.Context, minPicks int, sortBy string, position meta.Position) (meta.PositionResult, error)
}

type metaHeroesResponse struct {
	Scope          string                   `json:"scope"`
	Heroes         []meta.HeroPerformance   `json:"heroes"`
	Position       string                   `json:"position,omitempty"`
	PositionSource string                   `json:"positionSource,omitempty"`
	Sample         *opendota.PositionSample `json:"sample,omitempty"`
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

		var position meta.Position
		positionValues, positionProvided := query["position"]
		if positionProvided && len(positionValues) == 1 {
			position, err = meta.ParsePosition(positionValues[0])
		}
		if positionProvided && (len(positionValues) != 1 || err != nil) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "INVALID_POSITION",
					"message": "position deve ser informado uma única vez como carry, mid, offlane, support4 ou support5.",
				},
			})
			return
		}

		response := metaHeroesResponse{Scope: "professional"}
		if positionProvided {
			var result meta.PositionResult
			result, err = service.HeroesByPosition(r.Context(), minPicks, sortBy, position)
			response.Heroes = result.Heroes
			response.Position = position.String()
			response.PositionSource = "opendota.position_est"
			response.Sample = &result.Sample
		} else {
			response.Heroes, err = service.Heroes(r.Context(), minPicks, sortBy)
		}

		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			code := "UPSTREAM_UNAVAILABLE"
			message := "Não foi possível consultar as estatísticas de meta."
			if positionProvided {
				code = "POSITION_DATA_UNAVAILABLE"
				message = "Não foi possível obter uma amostra profissional com posições estimadas."
			}

			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    code,
					"message": message,
				},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}
