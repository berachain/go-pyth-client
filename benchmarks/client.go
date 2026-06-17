package benchmarks

import (
	"net/http"

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

	// Setup and configure the retryable HTTP client.
	httpClient := retryablehttp.NewClient()
	httpClient.HTTPClient.Timeout = cfg.HTTPTimeout
	httpClient.Logger = logger
	httpClient.RetryMax = cfg.MaxRetries

	// Inject the API key as an `Authorization: Bearer` header on every request.
	if cfg.APIKey != "" {
		base := httpClient.HTTPClient.Transport
		if base == nil {
			base = http.DefaultTransport
		}
		httpClient.HTTPClient.Transport = &authTransport{apiKey: cfg.APIKey, base: base}
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

// authTransport injects an `Authorization: Bearer` header into every request.
type authTransport struct {
	apiKey string
	base   http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request before mutating it, per the http.RoundTripper contract.
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	return t.base.RoundTrip(req)
}
