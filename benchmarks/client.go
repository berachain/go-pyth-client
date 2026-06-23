package benchmarks

import (
	"context"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/berachain/go-pyth-client/httpclient"
	"github.com/berachain/go-pyth-client/types"
)

// Client is a client for the Pyth Benchmarks API (https://benchmarks.pyth.network/docs)
type Client struct {
	// Config for Pyth and HTTP calls.
	cfg *Config

	// HTTP client that handles retries with a default retry policy.
	client *http.Client

	// The logger to handle logs
	logger retryablehttp.LeveledLogger
}

// NewClient creates a client for the Pyth Benchmarks API.
func NewClient(cfg *Config, logger retryablehttp.LeveledLogger) (*Client, error) {
	// Ensure the given configuration is valid.
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	// Ensure an API key is provided.
	if cfg.APIKey == "" {
		return nil, types.ErrMissingAPIKey
	}

	// Build the retryable HTTP client
	httpClient := httpclient.New(*cfg, logger)

	return &Client{
		cfg:    cfg,
		client: httpClient,
		logger: logger,
	}, nil
}

// Shutdown gracefully shuts down the Pyth Benchmarks client.
func (c *Client) Shutdown() {
	c.client.CloseIdleConnections()
}

// get issues a context-aware GET request, as recommended by the net/http docs
// (build the request with NewRequestWithContext, then call Client.Do).
func (c *Client) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return c.client.Do(req)
}
