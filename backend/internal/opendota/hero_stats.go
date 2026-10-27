package opendota

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"
)

type HeroStat struct {
	ID            int    `json:"id"`
	LocalizedName string `json:"localized_name"`
	ProPick       int    `json:"pro_pick"`
	ProWin        int    `json:"pro_win"`
}

func (c *Client) HeroStats(ctx context.Context) ([]HeroStat, error) {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	if c.statsCache != nil &&
		time.Now().Before(c.statsExpiresAt) {
		return slices.Clone(c.statsCache), nil
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/heroStats",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"criar requisição de estatísticas: %w",
			err,
		)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf(
			"consultar estatísticas: %w",
			err,
		)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"OpenDota retornou HTTP %d ao consultar estatísticas",
			response.StatusCode,
		)
	}

	var stats []HeroStat

	if err := json.NewDecoder(response.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf(
			"interpretar resposta de estatísticas: %w",
			err,
		)
	}

	c.statsCache = stats
	c.statsExpiresAt = time.Now().Add(c.cacheTTL)

	return slices.Clone(c.statsCache), nil
}
