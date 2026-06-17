package benchmarks

import (
	"github.com/berachain/go-pyth-client/httpclient"
)

// Config holds the configuration for the Benchmarks client. Benchmarks has no
// parameters beyond the shared offchain HTTP config, so it is an alias.
type Config = httpclient.BaseConfig
