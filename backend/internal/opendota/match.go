package opendota

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

var ErrMatchNotFound = errors.New("partida não disponível na OpenDota")

type MatchData struct {
	MatchID      int64          `json:"match_id"`
	Duration     *int           `json:"duration"`
	StartTime    *int64         `json:"start_time"`
	GameMode     *int           `json:"game_mode"`
	LobbyType    *int           `json:"lobby_type"`
	RadiantWin   *bool          `json:"radiant_win"`
	RadiantScore *int           `json:"radiant_score"`
	DireScore    *int           `json:"dire_score"`
	Players      []*MatchPlayer `json:"players"`
}

type MatchPlayer struct {
	AccountID     *uint32 `json:"account_id"`
	PersonaName   *string `json:"personaname"`
	PlayerSlot    *int    `json:"player_slot"`
	HeroID        *int    `json:"hero_id"`
	Kills         *int    `json:"kills"`
	Deaths        *int    `json:"deaths"`
	Assists       *int    `json:"assists"`
	GoldPerMinute *int    `json:"gold_per_min"`
	XPPerMinute   *int    `json:"xp_per_min"`
	HeroDamage    *int    `json:"hero_damage"`
	TowerDamage   *int    `json:"tower_damage"`
	HeroHealing   *int    `json:"hero_healing"`
	LastHits      *int    `json:"last_hits"`
	Denies        *int    `json:"denies"`
	Level         *int    `json:"level"`
}

func (c *Client) Match(ctx context.Context, matchID int64) (MatchData, error) {
	if matchID <= 0 || matchID > 9007199254740991 {
		return MatchData{}, fmt.Errorf("matchId inválido")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/matches/"+strconv.FormatInt(matchID, 10), nil)
	if err != nil {
		return MatchData{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "AncientAnalytics/1.0")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return MatchData{}, fmt.Errorf("consultar partida OpenDota: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return MatchData{}, ErrMatchNotFound
	}
	if response.StatusCode != http.StatusOK {
		return MatchData{}, fmt.Errorf("OpenDota retornou HTTP %d ao consultar partida", response.StatusCode)
	}
	const maxMatchBytes = 16 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxMatchBytes+1))
	if err != nil {
		return MatchData{}, fmt.Errorf("ler partida OpenDota: %w", err)
	}
	if len(body) > maxMatchBytes {
		return MatchData{}, fmt.Errorf("resposta de partida excede o limite permitido")
	}
	var data *MatchData
	if err := json.Unmarshal(body, &data); err != nil {
		return MatchData{}, fmt.Errorf("interpretar partida OpenDota: %w", err)
	}
	if data == nil || data.MatchID != matchID {
		return MatchData{}, fmt.Errorf("partida retornada não corresponde ao matchId solicitado")
	}
	for _, value := range []*int{data.Duration, data.GameMode, data.LobbyType, data.RadiantScore, data.DireScore} {
		if value != nil && *value < 0 {
			return MatchData{}, fmt.Errorf("partida contém valor negativo inválido")
		}
	}
	if data.StartTime != nil && *data.StartTime < 0 {
		return MatchData{}, fmt.Errorf("instante de início da partida inválido")
	}
	if len(data.Players) > 256 {
		return MatchData{}, fmt.Errorf("quantidade de jogadores da partida inválida")
	}
	slots := make(map[int]bool, len(data.Players))
	for _, p := range data.Players {
		if p == nil {
			return MatchData{}, fmt.Errorf("jogador da partida inválido")
		}
		if p.PlayerSlot != nil {
			slot := *p.PlayerSlot
			if slot < 0 || slot > 255 || slots[slot] {
				return MatchData{}, fmt.Errorf("posição do jogador no time inválida ou repetida")
			}
			slots[slot] = true
		}
		for _, value := range []*int{p.HeroID, p.Kills, p.Deaths, p.Assists, p.GoldPerMinute, p.XPPerMinute, p.HeroDamage, p.TowerDamage, p.HeroHealing, p.LastHits, p.Denies, p.Level} {
			if value != nil && *value < 0 {
				return MatchData{}, fmt.Errorf("estatística do jogador inválida")
			}
		}
	}
	return *data, nil
}
