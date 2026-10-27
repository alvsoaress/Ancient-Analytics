package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/alvsoaress/ancient-analytics-backend/internal/matches"
)

type MatchesService interface {
	Details(ctx context.Context, matchID int64) (matches.Details, error)
}

func matchDetails(service MatchesService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id, err := strconv.ParseUint(r.PathValue("matchId"), 10, 53)
		if err != nil || id == 0 {
			writePlayerError(w, http.StatusBadRequest, "INVALID_MATCH_ID", "Informe um matchId válido do Dota 2.")
			return
		}
		result, err := service.Details(r.Context(), int64(id))
		if err != nil {
			status, code, message := http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", "Não foi possível consultar os detalhes da partida."
			switch {
			case errors.Is(err, matches.ErrNotFound):
				status, code, message = http.StatusNotFound, "MATCH_NOT_FOUND", "Partida não disponível na OpenDota."
			case errors.Is(err, matches.ErrInvalidMatchID):
				status, code, message = http.StatusBadRequest, "INVALID_MATCH_ID", "Informe um matchId válido do Dota 2."
			}
			writePlayerError(w, status, code, message)
			return
		}
		_ = json.NewEncoder(w).Encode(result)
	}
}
