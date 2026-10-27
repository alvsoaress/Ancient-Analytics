package opendota

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

var ErrPlayerNotFound = errors.New("perfil não disponível na OpenDota")

const playerCacheLimit = 128

type PlayerIdentity struct {
	AccountID    uint32 `json:"account_id"`
	SteamID      string `json:"steamid"`
	PersonaName  string `json:"personaname"`
	AvatarFull   string `json:"avatarfull"`
	AvatarMedium string `json:"avatarmedium"`
	Avatar       string `json:"avatar"`
	ProfileURL   string `json:"profileurl"`
}

type PlayerData struct {
	Profile         *PlayerIdentity `json:"profile"`
	RankTier        *int            `json:"rank_tier"`
	LeaderboardRank *int            `json:"leaderboard_rank"`
}

type playerCacheEntry struct {
	data      PlayerData
	expiresAt time.Time
}

func (c *Client) Player(ctx context.Context, accountID uint32) (PlayerData, error) {
	if accountID == 0 {
		return PlayerData{}, fmt.Errorf("accountId deve ser positivo")
	}
	c.playersMu.Lock()
	defer c.playersMu.Unlock()
	if err := ctx.Err(); err != nil {
		return PlayerData{}, err
	}
	if entry, ok := c.playersCache[accountID]; ok && time.Now().Before(entry.expiresAt) {
		return clonePlayer(entry.data), nil
	}
	data, err := c.fetchPlayer(ctx, accountID)
	if err != nil {
		return PlayerData{}, err
	}
	if c.playersCache == nil {
		c.playersCache = make(map[uint32]playerCacheEntry)
	}
	now := time.Now()
	for id, entry := range c.playersCache {
		if !now.Before(entry.expiresAt) {
			delete(c.playersCache, id)
		}
	}
	if len(c.playersCache) >= playerCacheLimit {
		var oldestID uint32
		var oldestExpiry time.Time
		for id, entry := range c.playersCache {
			if oldestExpiry.IsZero() || entry.expiresAt.Before(oldestExpiry) {
				oldestID, oldestExpiry = id, entry.expiresAt
			}
		}
		delete(c.playersCache, oldestID)
	}
	c.playersCache[accountID] = playerCacheEntry{data: data, expiresAt: now.Add(c.cacheTTL)}
	return clonePlayer(data), nil
}

func (c *Client) fetchPlayer(ctx context.Context, accountID uint32) (PlayerData, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/players/"+strconv.FormatUint(uint64(accountID), 10), nil)
	if err != nil {
		return PlayerData{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "AncientAnalytics/1.0")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return PlayerData{}, fmt.Errorf("consultar perfil OpenDota: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return PlayerData{}, ErrPlayerNotFound
	}
	if response.StatusCode != http.StatusOK {
		return PlayerData{}, fmt.Errorf("OpenDota retornou HTTP %d ao consultar perfil", response.StatusCode)
	}
	const maxProfileBytes = 1 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxProfileBytes+1))
	if err != nil {
		return PlayerData{}, fmt.Errorf("ler perfil OpenDota: %w", err)
	}
	if len(body) > maxProfileBytes {
		return PlayerData{}, fmt.Errorf("resposta de perfil OpenDota excede o limite permitido")
	}
	var data *PlayerData
	if err := json.Unmarshal(body, &data); err != nil {
		return PlayerData{}, fmt.Errorf("interpretar perfil OpenDota: %w", err)
	}
	if data == nil {
		return PlayerData{}, fmt.Errorf("resposta de perfil OpenDota inválida")
	}
	if data.Profile == nil {
		return PlayerData{}, ErrPlayerNotFound
	}
	if data.Profile.AccountID != accountID {
		return PlayerData{}, fmt.Errorf("identificador do perfil retornado não corresponde à conta solicitada")
	}
	if data.RankTier != nil && *data.RankTier < 0 || data.LeaderboardRank != nil && *data.LeaderboardRank < 0 {
		return PlayerData{}, fmt.Errorf("rank retornado pela OpenDota inválido")
	}
	return *data, nil
}

func clonePlayer(data PlayerData) PlayerData {
	if data.Profile != nil {
		profile := *data.Profile
		data.Profile = &profile
	}
	if data.RankTier != nil {
		rank := *data.RankTier
		data.RankTier = &rank
	}
	if data.LeaderboardRank != nil {
		rank := *data.LeaderboardRank
		data.LeaderboardRank = &rank
	}
	return data
}
