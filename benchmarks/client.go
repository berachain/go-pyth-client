package benchmarks

import (
	"github.com/hashicorp/go-retryablehttp"
)

// Client is a client for the Pyth Benchmarks API (https://benchmarks.pyth.network/docs)
type Client struct {
	// Config for Pyth and HTTP calls.
	cfg *Config

	// HTTP client that handles retries with a default retry policy.
	client *retryablehttp.Client

	// The logger to handle logs
	logger retryablehttp.LeveledLogger
}

// NewClient creates a client for the Pyth Benchmarks API.
func NewClient(cfg *Config, logger retryablehttp.LeveledLogger) (*Client, error) {
	// Ensure the given configuration is valid.
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// The upgraded Benchmarks endpoints require an API key, so an authenticated
	// client is used.
	httpClient, err := cfg.NewAuthenticatedClient(logger)
	if err != nil {
		return nil, err
	}

	return &Client{
		cfg:    cfg,
		client: httpClient,
		logger: logger,
	}, nil
}

// Shutdown gracefully shuts down the Pyth Benchmarks client.
func (c *Client) Shutdown() {
	c.client.HTTPClient.CloseIdleConnections()
}
