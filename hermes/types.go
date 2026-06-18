package hermes

import (
	"sync"
	"time"

	"github.com/berachain/go-pyth-client/types"
)

// Cached data returned from the `/v2/updates/price/stream` stream with mutex feature.
// Streaming data may return hundreds of ms after subscription, so the ready channel is used
// to block read operation until cached data gets populated.
type ssePriceData struct {
	mu            sync.RWMutex
	ready         chan struct{}
	broadcastOnce sync.Once
	latestPrice   map[string]*types.LatestPriceData

	// lastEventAt is the wall-clock time the most recent valid update was received from any
	// feed. It is the stream-liveness (transport-health) signal; per-feed price freshness is
	// derived from each feed's Pyth publish_time instead. Guarded by mu.
	lastEventAt time.Time
}

// JSON formatted price returned from the `v2/updates/price/latest` endpoint.
type price struct {
	Price       string `json:"price"`
	Conf        string `json:"conf"`
	Expo        int32  `json:"expo"`
	PublishTime int64  `json:"publish_time"`
}

// JSON response returned from the `v2/updates/price/latest` endpoint.
//
//nolint:revive // needed for JSON unmarshalling.
type latestPriceResponse struct {
	Binary struct {
		Data []string `json:"data"`
	} `json:"binary"`
	Parsed []struct {
		ID       string `json:"id"`
		Price    price  `json:"price"`
		EmaPrice price  `json:"ema_price"`
	} `json:"parsed"`
}
