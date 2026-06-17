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

// Validate checks that the shared HTTP configuration is well formed. The API key
// is not checked here: it is required only when an authenticated client is
// created (see NewAuthenticatedClient).
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

// NewClient builds a retryablehttp.Client with the given timeout, retry count,
// and logger, without any authentication.
func NewClient(
	timeout time.Duration, maxRetries int, logger retryablehttp.LeveledLogger,
) *retryablehttp.Client {
	httpClient := retryablehttp.NewClient()
	httpClient.HTTPClient.Timeout = timeout
	httpClient.Logger = logger
	httpClient.RetryMax = maxRetries

	return httpClient
}

// NewAuthenticatedClient builds a retryablehttp.Client from the config, injecting
// the API key as an `Authorization: Bearer` header on every request. It requires
// an API key and returns types.ErrMissingAPIKey when one is not configured.
func (c BaseConfig) NewAuthenticatedClient(
	logger retryablehttp.LeveledLogger,
) (*retryablehttp.Client, error) {
	if c.APIKey == "" {
		return nil, types.ErrMissingAPIKey
	}

	httpClient := NewClient(c.HTTPTimeout, c.MaxRetries, logger)

	base := httpClient.HTTPClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	httpClient.HTTPClient.Transport = &authTransport{apiKey: c.APIKey, base: base}

	return httpClient, nil
}

// AuthHeaders returns the headers used to authenticate requests, for clients
// (such as the SSE client) that do not use the retryable HTTP client. It returns
// nil when no API key is configured.
func (c BaseConfig) AuthHeaders() map[string]string {
	if c.APIKey == "" {
		return nil
	}

	return map[string]string{"Authorization": "Bearer " + c.APIKey}
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
