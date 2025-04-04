package http

import (
	"net/http"
	"sync"
	"time"
)

type Client struct {
	client *http.Client
	once   sync.Once
}

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
