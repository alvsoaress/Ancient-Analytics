package opendota

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Hero struct {
	ID            int    `json:"id"`
	LocalizedName string `json:"localized_name"`
}

func (c *Client) Heroes(ctx context.Context) ([]Hero, error) {
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

	return heroes, nil
}
