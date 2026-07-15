package benchmarks_test

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/berachain/go-pyth-client/benchmarks"
	"github.com/berachain/go-pyth-client/httpclient"
)

// This file contains test utils that are shared for tests of Benchmarks.

var (
	testPairs = []string{
		"0xc9d8b075a5c69303365ae23633d4e085199bf5c520a3b90fed1322a0342ffc33", // WBTC/USD
		"0x9d4294bbcd1174d6f2003ec365831e64cc31d9f6f15a2b85399db8d5000960f6", // WETH/USD
		"0x962088abcfdbdb6e30db2e340c8cf887d9efb311b1f2f17b155a63dbb6d40265", // BERA/USD
	}

	testConfig = benchmarks.Config{
		APIEndpoint: "https://benchmarks.pyth.network",
		APIKey:      httpclient.NewSecretWrapper(os.Getenv("PYTH_API_KEY")),
		HTTPTimeout: 1 * time.Second,
		MaxRetries:  2,
	}

	testTime = time.Now().Add(-2 * time.Hour) // 2 hours ago
)

func setUp(tb testing.TB) *benchmarks.Client {
	tb.Helper()

	// These tests hit the real Benchmarks API, which requires a valid key. Skip them when no key
	// is configured (e.g. local runs without PYTH_API_KEY set) rather than panicking.
	if testConfig.APIKey.Reveal() == "" {
		tb.Skip("no Pyth API key configured; set PYTH_API_KEY to run this test")
	}

	pythClient, err := benchmarks.NewClient(&testConfig, slog.Default())
	if err != nil {
		tb.Fatalf("failed to create benchmarks client: %v", err)
	}

	return pythClient
}
