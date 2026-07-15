package hermes_test

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestGetLatestPriceUpdatesSync(t *testing.T) {
	ctx, pythClient := setUp(t)

	prices, err := pythClient.GetLatestPriceUpdatesSync(ctx, testPairs)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(prices))
	for _, pair := range testPairs {
		assert.Contains(t, prices, hex.EncodeToString(common.FromHex(pair)))
	}
}

// To run this benchmark only without other tests: `go test -run=^$ -bench=BenchmarkGetLatestPriceUpdatesSync`
func BenchmarkGetLatestPriceUpdatesSync(b *testing.B) {
	ctx, benchmarkClient := setUp(b)

	for i := 0; i < b.N; i++ {
		_, err := benchmarkClient.GetLatestPriceUpdatesSync(ctx, testPairs)
		assert.NoError(b, err)
	}
}
