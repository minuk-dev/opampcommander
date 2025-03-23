package client

import (
	"github.com/go-resty/resty/v2"
)

type Client struct {
	Endpoint string
	Client   *resty.Client
}

func NewClient(endpoint string) *Client {
	return &Client{
		Endpoint: endpoint,
		Client:   resty.New().SetBaseURL(endpoint),
	}
}
