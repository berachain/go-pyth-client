package hermes

import (
	"net/url"
	"time"

	"github.com/berachain/go-pyth-client/types"
)

type Config struct {
	// Offchain parameters
	APIEndpoint string
	APIKey      string // API key sent as `Authorization: Bearer <APIKey>`.
	HTTPTimeout time.Duration
	MaxRetries  int

	// Onchain parameters
	UseMock bool // Uses the mock Pyth contract rather than the real one.
}

func (c *Config) Validate() error {
	_, err := url.Parse(c.APIEndpoint)
	if err != nil {
		return err
	}

	if c.APIKey == "" {
		return types.ErrMissingAPIKey
	}

	if c.HTTPTimeout <= 0 {
		return types.ErrInvalidHTTPTimeout
	}
	if c.MaxRetries < 0 {
		return types.ErrInvalidMaxRetries
	}

	return nil
}
