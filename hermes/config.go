package hermes

import (
	"github.com/berachain/go-pyth-client/httpclient"
)

type Config struct {
	// Offchain parameters.
	httpclient.BaseConfig

	// Onchain parameters.
	UseMock bool // Uses the mock Pyth contract rather than the real one.
}
