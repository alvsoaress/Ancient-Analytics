package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/alvsoaress/ancient-analytics-backend/internal/players"
)

func playerMatches(service PlayersService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		id, err := strconv.ParseUint(r.PathValue("accountId"), 10, 32)
		if err != nil || id == 0 {
			writePlayerError(w, http.StatusBadRequest, "INVALID_ACCOUNT_ID", "Informe um accountId válido do Dota 2.")
			return
		}
		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			writePlayerError(w, http.StatusBadRequest, "INVALID_PAGINATION", "Informe parâmetros de paginação válidos.")
			return
		}
		limit, err := paginationInteger(query, "limit", players.DefaultMatchLimit, 1, players.MaxMatchLimit)
		if err != nil {
			writePlayerError(w, http.StatusBadRequest, "INVALID_LIMIT", "limit deve ser informado uma única vez como inteiro entre 1 e 100.")
			return
		}
		offset, err := paginationInteger(query, "offset", 0, 0, players.MaxMatchOffset)
		if err != nil {
			writePlayerError(w, http.StatusBadRequest, "INVALID_OFFSET", "offset deve ser informado uma única vez como inteiro entre 0 e 2147483546.")
			return
		}
		result, err := service.Matches(r.Context(), uint32(id), limit, offset)
		if err != nil {
			status, code, message := http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", "Não foi possível consultar o histórico de partidas."
			if errors.Is(err, players.ErrInvalidPagination) {
				status, code, message = http.StatusBadRequest, "INVALID_PAGINATION", "Informe parâmetros de paginação válidos."
			} else if errors.Is(err, players.ErrInvalidAccountID) {
				status, code, message = http.StatusBadRequest, "INVALID_ACCOUNT_ID", "Informe um accountId válido do Dota 2."
			}
			writePlayerError(w, status, code, message)
			return
		}
		_ = json.NewEncoder(w).Encode(result)
	}
}

func paginationInteger(query url.Values, name string, fallback, minimum, maximum int) (int, error) {
	values, provided := query[name]
	if !provided {
		return fallback, nil
	}
	if len(values) != 1 {
		return 0, fmt.Errorf("parâmetro repetido: %s", name)
	}
	value, err := strconv.ParseUint(values[0], 10, 31)
	if err != nil || value < uint64(minimum) || value > uint64(maximum) {
		return 0, fmt.Errorf("parâmetro fora do intervalo permitido: %s", name)
	}
	return int(value), nil
}
