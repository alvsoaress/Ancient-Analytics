package opendota

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type PlayerMatch struct {
	MatchID    int64  `json:"match_id"`
	HeroID     *int   `json:"hero_id"`
	PlayerSlot *int   `json:"player_slot"`
	RadiantWin *bool  `json:"radiant_win"`
	Duration   *int   `json:"duration"`
	StartTime  *int64 `json:"start_time"`
	Kills      *int   `json:"kills"`
	Deaths     *int   `json:"deaths"`
	Assists    *int   `json:"assists"`
	GameMode   *int   `json:"game_mode"`
	LobbyType  *int   `json:"lobby_type"`
}

func (c *Client) PlayerMatches(ctx context.Context, accountID uint32, limit, offset int) ([]PlayerMatch, error) {
	// O item extra permite detectar a próxima página sem contar todo o histórico.
	if accountID == 0 || limit < 1 || limit > 101 || offset < 0 || offset > 2147483546 {
		return nil, fmt.Errorf("conta ou paginação de partidas inválida")
	}
	query := url.Values{
		"limit": {strconv.Itoa(limit)}, "offset": {strconv.Itoa(offset)},
		"sort": {"match_id"}, "significant": {"0"},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/players/"+strconv.FormatUint(uint64(accountID), 10)+"/matches?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "AncientAnalytics/1.0")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("consultar partidas OpenDota: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenDota retornou HTTP %d ao consultar partidas", response.StatusCode)
	}
	const maxMatchBytes = 2 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxMatchBytes+1))
	if err != nil {
		return nil, fmt.Errorf("ler partidas OpenDota: %w", err)
	}
	if len(body) > maxMatchBytes {
		return nil, fmt.Errorf("resposta de partidas excede o limite permitido")
	}
	var matches []PlayerMatch
	if err := json.Unmarshal(body, &matches); err != nil {
		return nil, fmt.Errorf("interpretar partidas OpenDota: %w", err)
	}
	if matches == nil || len(matches) > limit {
		return nil, fmt.Errorf("lista de partidas OpenDota inválida")
	}
	seen := make(map[int64]bool, len(matches))
	for _, match := range matches {
		if match.MatchID <= 0 || seen[match.MatchID] {
			return nil, fmt.Errorf("identificador de partida inválido ou repetido")
		}
		seen[match.MatchID] = true
		for _, value := range []*int{match.HeroID, match.Duration, match.Kills, match.Deaths, match.Assists, match.GameMode, match.LobbyType} {
			if value != nil && *value < 0 {
				return nil, fmt.Errorf("partida OpenDota contém valor negativo inválido")
			}
		}
		if match.StartTime != nil && *match.StartTime < 0 || match.PlayerSlot != nil && (*match.PlayerSlot < 0 || *match.PlayerSlot > 255) {
			return nil, fmt.Errorf("data ou time da partida OpenDota inválido")
		}
	}
	return matches, nil
}
