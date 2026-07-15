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

// SecretWrapper holds a sensitive value (such as an API key) so it is not
// accidentally exported or logged. Its String method redacts the value, so the
// only way to obtain the underlying secret is the explicit Reveal method.
type SecretWrapper struct {
	value string
}

// NewSecretWrapper wraps value in a SecretWrapper.
func NewSecretWrapper(value string) SecretWrapper {
	return SecretWrapper{value: value}
}

// String implements fmt.Stringer, returning a redacted placeholder so the
// secret is never emitted by fmt-based formatting or logging.
func (SecretWrapper) String() string {
	return "[REDACTED]"
}

// Reveal returns the underlying secret value. This is the only way to read it,
// making every access to the raw secret explicit at the call site.
func (sw SecretWrapper) Reveal() string {
	return sw.value
}

// BaseConfig holds the offchain HTTP parameters common to all Pyth API clients.
type BaseConfig struct {
	APIEndpoint string        // Base URL of the API.
	APIKey      SecretWrapper // API key sent as `Authorization: Bearer <APIKey>`.
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

	if cfg.APIKey.Reveal() != "" {
		base := httpClient.HTTPClient.Transport
		if base == nil {
			base = http.DefaultTransport
		}

		httpClient.HTTPClient.Transport = &authTransport{apiKey: cfg.APIKey.Reveal(), base: base}
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
