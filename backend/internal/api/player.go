package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/alvsoaress/ancient-analytics-backend/internal/players"
)

type PlayersService interface {
	Profile(ctx context.Context, accountID uint32) (players.Profile, error)
	Matches(ctx context.Context, accountID uint32, limit, offset int) (players.MatchHistory, error)
}

func playerProfile(service PlayersService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		accountID, err := strconv.ParseUint(r.PathValue("accountId"), 10, 32)
		if err != nil || accountID == 0 {
			writePlayerError(w, http.StatusBadRequest, "INVALID_ACCOUNT_ID", "Informe um accountId válido do Dota 2.")
			return
		}
		profile, err := service.Profile(r.Context(), uint32(accountID))
		if err != nil {
			status, code, message := http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", "Não foi possível consultar o perfil do jogador."
			switch {
			case errors.Is(err, players.ErrNotFound):
				status, code, message = http.StatusNotFound, "PLAYER_NOT_FOUND", "Perfil não disponível na OpenDota."
			case errors.Is(err, players.ErrInvalidAccountID):
				status, code, message = http.StatusBadRequest, "INVALID_ACCOUNT_ID", "Informe um accountId válido do Dota 2."
			}
			writePlayerError(w, status, code, message)
			return
		}
		_ = json.NewEncoder(w).Encode(profile)
	}
}

func writePlayerError(w http.ResponseWriter, status int, code, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}
