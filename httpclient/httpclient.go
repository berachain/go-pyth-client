// Package httpclient holds the offchain HTTP configuration and client
// construction shared by the Pyth Hermes and Benchmarks clients.
package httpclient

import (
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/berachain/go-pyth-client/types"
)

// BaseConfig holds the offchain HTTP parameters common to all Pyth API clients.
type BaseConfig struct {
	APIEndpoint string        // Base URL of the API.
	APIKey      string        // API key sent as `Authorization: Bearer <APIKey>`.
	HTTPTimeout time.Duration // Timeout applied to each HTTP request.
	MaxRetries  int           // Maximum number of retries per request.
}

// Validate checks that the shared HTTP configuration is well formed. An API key
// is required: every Pyth API client authenticates its requests.
func (c BaseConfig) Validate() error {
	if _, err := url.Parse(c.APIEndpoint); err != nil {
		return err
	}

	if c.HTTPTimeout <= 0 {
		return types.ErrInvalidHTTPTimeout
	}

	if c.MaxRetries < 0 {
		return types.ErrInvalidMaxRetries
	}

	return nil
}

// New builds an *http.Client from cfg. It is backed by a retryablehttp
// client, so the returned standard client transparently retries per cfg.MaxRetries.
// When cfg.APIKey is set, every request is decorated with an
// `Authorization: Bearer <APIKey>` header.
func New(cfg BaseConfig, logger retryablehttp.LeveledLogger) *http.Client {
	httpClient := retryablehttp.NewClient()
	httpClient.HTTPClient.Timeout = cfg.HTTPTimeout
	httpClient.Logger = logger
	httpClient.RetryMax = cfg.MaxRetries

	if cfg.APIKey != "" {
		base := httpClient.HTTPClient.Transport
		if base == nil {
			base = http.DefaultTransport
		}

		httpClient.HTTPClient.Transport = &authTransport{apiKey: cfg.APIKey, base: base}
	}

	// Expose the retryable client as a standard *http.Client
	return httpClient.StandardClient()
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
