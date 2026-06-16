package benchmarks

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/berachain/go-pyth-client/bindings/apyth"
)

// GetHistoricalPriceUpdatesSync queries the `v1/updates/price/{timestamp}` endpoint for all price
// feed IDs together. Takes the price feed keys (uses corresponding Pyth feed ID). Returns the Pyth
// PriceFeed struct and the price feed update data for each pair.
func (c *Client) GetHistoricalPriceUpdatesSync(
	_ context.Context, timestamp time.Time, priceFeedIDs []string,
) (map[string]*apyth.PythStructsPriceFeed, error) {
	// Validate parameters.
	if len(priceFeedIDs) == 0 {
		return nil, nil
	}

	// Build and fire the request.
	resp, err := c.client.Get(c.buildBatchURL(timestamp, priceFeedIDs))
	if err != nil {
		return nil, err
	}

	// Parse the response.
	var priceResp priceResponse
	if err = json.NewDecoder(resp.Body).Decode(&priceResp); err != nil {
		return nil, err
	}

	priceResults := make(map[string]*apyth.PythStructsPriceFeed, len(priceResp.Parsed))
	if err = resolveMany(&priceResp, priceResults); err != nil {
		return nil, err
	}
	return priceResults, nil
}

// Builds the API endpoint for querying multiple feeds on `v1/updates/price/{timestamp}`.
func (c *Client) buildBatchURL(timestamp time.Time, priceFeedIDs []string) string {
	// Batch the price feed IDs into a single query string.
	urlComponents := make([]string, len(priceFeedIDs)+3)
	urlComponents[0] = c.cfg.APIEndpoint
	urlComponents[1] = priceUpdateAPI
	urlComponents[2] = strconv.FormatInt(timestamp.Unix(), 10) + "?"
	for i, priceFeedID := range priceFeedIDs {
		urlComponents[i+3] = "ids=" + priceFeedID + "&"
	}
	return strings.Join(urlComponents, "")
}
