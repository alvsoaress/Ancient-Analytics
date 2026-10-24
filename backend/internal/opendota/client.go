package opendota

import (
	"net/http"
	"sync"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client

	heroesMu        sync.Mutex
	heroesCache     []Hero
	heroesExpiresAt time.Time
	cacheTTL        time.Duration
}

func NewClient() *Client {
	return &Client{
		baseURL: "https://api.opendota.com/api",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cacheTTL: 5 * time.Minute,
	}
}
