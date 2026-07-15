package benchmarks_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
)

func TestGetPriceUpdatesSync(t *testing.T) {
	pythClient := setUp(t)

	prices, err := pythClient.GetHistoricalPriceUpdatesSync(
		context.Background(), testTime, testPairs,
	)
	assert.Nil(t, err)
	assert.Equal(t, len(testPairs), len(prices))

	for _, pair := range testPairs {
		expected, err := common.ParseHexOrString(pair)
		assert.Nil(t, err)

		priceID := hex.EncodeToString(expected)

		assert.Contains(t, prices, priceID)

		// Correct Price Feed ID returned
		assert.Equal(
			t,
			prices[priceID].Id[:], expected,
			fmt.Sprintf("Price feed ID for %s is incorrect", pair),
		)

		// Valid, non-zero prices returned
		assert.Greater(t, prices[priceID].Price.Price, int64(0))
		assert.Greater(t, prices[priceID].EmaPrice.Price, int64(0))

		// Within 5 seconds of the desired time
		assert.LessOrEqual(
			t,
			math.Abs(float64(prices[priceID].Price.PublishTime.Int64())-float64(testTime.Unix())),
			float64(5*time.Second),
		)
	}
}
