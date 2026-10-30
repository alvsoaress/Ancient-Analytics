package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/alvsoaress/ancient-analytics-backend/internal/builds"
	"github.com/alvsoaress/ancient-analytics-backend/internal/meta"
)

type BuildsService interface {
	Hero(ctx context.Context, heroID int, position meta.Position) (builds.Result, error)
}

func heroBuilds(service BuildsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id, err := strconv.ParseUint(r.PathValue("heroId"), 10, 31)
		if err != nil || id == 0 {
			writePlayerError(w, http.StatusBadRequest, "INVALID_HERO_ID", "Informe um heroId válido do Dota 2.")
			return
		}
		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			writePlayerError(w, http.StatusBadRequest, "INVALID_POSITION", "Informe uma posição válida.")
			return
		}
		var position meta.Position
		if values, present := query["position"]; present {
			if len(values) != 1 {
				writePlayerError(w, http.StatusBadRequest, "INVALID_POSITION", "Informe uma única posição válida.")
				return
			}
			position, err = meta.ParsePosition(values[0])
			if err != nil {
				writePlayerError(w, http.StatusBadRequest, "INVALID_POSITION", "Informe uma posição válida.")
				return
			}
		}
		result, err := service.Hero(r.Context(), int(id), position)
		if err != nil {
			status, code, message := http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", "Não foi possível consultar as builds do herói."
			switch {
			case errors.Is(err, builds.ErrNotFound):
				status, code, message = http.StatusNotFound, "HERO_NOT_FOUND", "Herói não disponível no catálogo."
			case errors.Is(err, builds.ErrInvalidHeroID):
				status, code, message = http.StatusBadRequest, "INVALID_HERO_ID", "Informe um heroId válido do Dota 2."
			case errors.Is(err, builds.ErrInvalidPosition):
				status, code, message = http.StatusBadRequest, "INVALID_POSITION", "Informe uma posição válida."
			}
			writePlayerError(w, status, code, message)
			return
		}
		_ = json.NewEncoder(w).Encode(result)
	}
}
