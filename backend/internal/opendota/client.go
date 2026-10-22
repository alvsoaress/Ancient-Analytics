package opendota

import (
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		baseURL: "https://api.opendota.com/api",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}
