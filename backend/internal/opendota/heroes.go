package opendota

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"
)

type Hero struct {
	ID            int      `json:"id"`
	LocalizedName string   `json:"localized_name"`
	PrimaryAttr   string   `json:"primary_attr"`
	AttackType    string   `json:"attack_type"`
	Roles         []string `json:"roles"`
}

func (c *Client) Heroes(ctx context.Context) ([]Hero, error) {
	c.heroesMu.Lock()
	defer c.heroesMu.Unlock()

	if c.heroesCache != nil &&
		time.Now().Before(c.heroesExpiresAt) {
		return slices.Clone(c.heroesCache), nil
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/heroes",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("criar requisição de heróis: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("consultar heróis: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"OpenDota retornou HTTP %d ao consultar heróis",
			response.StatusCode,
		)
	}

	var heroes []Hero

	if err := json.NewDecoder(response.Body).Decode(&heroes); err != nil {
		return nil, fmt.Errorf("interpretar resposta de heróis: %w", err)
	}

	if c.heroesCache != nil &&
		time.Now().Before(c.heroesExpiresAt) {
		return cloneHeroes(c.heroesCache), nil
	}

	c.heroesCache = heroes
	c.heroesExpiresAt = time.Now().Add(c.cacheTTL)

	return cloneHeroes(c.heroesCache), nil
}

func cloneHeroes(heroes []Hero) []Hero {
	cloned := slices.Clone(heroes)

	for i := range cloned {
		cloned[i].Roles = slices.Clone(cloned[i].Roles)
	}

	return cloned
}
