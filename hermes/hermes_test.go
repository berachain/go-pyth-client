package hermes_test

import (
	"context"
	"time"

	"log/slog"

	"github.com/berachain/go-pyth-client/hermes"
)

// This file contains test utils that are shared for tests of Hermes.

var testPairs = []string{
	"0xc9d8b075a5c69303365ae23633d4e085199bf5c520a3b90fed1322a0342ffc33", // WBTC/USD
	"0x9d4294bbcd1174d6f2003ec365831e64cc31d9f6f15a2b85399db8d5000960f6", // WETH/USD
	"0x962088abcfdbdb6e30db2e340c8cf887d9efb311b1f2f17b155a63dbb6d40265", // BERA/USD
}

var testConfig = hermes.Config{
	// Offchain parameters
	APIEndpoint: "https://hermes.pyth.network",
	HTTPTimeout: 1 * time.Second,
	MaxRetries:  2,

	// Onchain parameters
	UseMock: true, // Uses the mock Pyth contract rather than the real one.
}

func setUp() (context.Context, *hermes.Client) {
	// set up Pyth client and subscribe
	pythClient, _ := hermes.NewClient(&testConfig, slog.Default())

	return context.Background(), pythClient
}
