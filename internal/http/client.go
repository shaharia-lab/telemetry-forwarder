package http

import (
	"net/http"
	"sync"
	"time"
)

// Client is a singleton HTTP client with a custom transport configuration.
type Client struct {
	client *http.Client
	once   sync.Once
}

// Client creates a new instance of the HTTP client.
func (h *Client) Client() *http.Client {
	h.once.Do(func() {
		h.client = &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        500,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
				MaxConnsPerHost:     250,
			},
		}
	})
	return h.client
}
