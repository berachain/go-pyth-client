package hermes

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/berachain/go-pyth-client/types"
	"github.com/ethereum/go-ethereum/common"
	sse "github.com/r3labs/sse/v2"
	backoff "gopkg.in/cenkalti/backoff.v1"
)

// Resubscribe backoff parameters. These bound how fast we re-subscribe after the rare
// unrecoverable error; routine reconnects are handled inside the SSE client itself.
const (
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
)

// authHeaders builds the headers used to authenticate the SSE client, which does
// not go through the retryable HTTP client. It returns nil when no API key is set.
func authHeaders(apiKey string) map[string]string {
	if apiKey == "" {
		return nil
	}

	return map[string]string{"Authorization": "Bearer " + apiKey}
}

// Subscribe price feed from the streaming `v2/updates/price/stream` endpoint. Ensures this only
// happens once in the scope of runtime. Any further calls to this are unnecessary and no-ops.
func (c *Client) SubscribePriceStreaming(ctx context.Context, priceFeedIDs []string) {
	c.subscribeOnce.Do(func() {
		client := sse.NewClient(c.buildBatchURLStream(priceFeedIDs))
		client.Headers = authHeaders(c.cfg.APIKey)

		// hermes.pyth.network sits behind Cloudflare, which periodically resets the HTTP/2
		// stream (~5-12 min, INTERNAL_ERROR) by design while leaving the connection intact.
		// Reconnect through these resets indefinitely: the default strategy's 15-min
		// MaxElapsedTime is measured from subscription start and never reset across
		// reconnects, so it otherwise expires and surfaces a routine reset as a fatal error.
		// MaxElapsedTime = 0 means "never give up".
		reconnect := backoff.NewExponentialBackOff()
		reconnect.MaxElapsedTime = 0
		// Make the strategy context-aware. r3labs runs the reconnect loop via
		// backoff.RetryNotify, which only exits on a clean shutdown when the backoff carries
		// the caller's context (otherwise ensureContext wraps it in context.Background(), whose
		// Done() never fires). With MaxElapsedTime = 0 the loop never stops on its own, so
		// without this the subscribe goroutine would leak when the caller cancels ctx.
		client.ReconnectStrategy = backoff.WithContext(reconnect, ctx)

		// These reconnects are the recoverable, expected case (Cloudflare stream resets),
		// so log them at info. The price-staleness metric is what alerts if reconnects
		// stop delivering data; this is purely for visibility.
		client.ReconnectNotify = func(err error, d time.Duration) {
			c.logger.Info("SSE stream reconnecting", "error", err, "backoff", d.String())
		}

		subscribe := func() error {
			return client.SubscribeRawWithContext(ctx, func(msg *sse.Event) {
				c.handleSseEvent(msg)
			})
		}

		// Subscribe to the SSE using the context and the channel
		// Use goroutine since SSE Subscribe will block the current thread
		// see https://github.com/r3labs/sse/blob/master/README.md
		go c.subscribeWithRetries(ctx, subscribe)
	})
}

// Queries cached price feed update data, obtained from sse streaming endpoints.
// Returns the Pyth PriceFeed struct and the price feed update data for each pair.
func (c *Client) GetCachedLatestPriceUpdates(
	ctx context.Context, priceFeedIDs []string,
) (map[string]*types.LatestPriceData, error) {
	// Validate parameters
	if len(priceFeedIDs) == 0 {
		return nil, errors.New("zero length of price feed ids is an invalid input")
	}

	// Wait for the ready signal
	if err := c.waitForReady(ctx); err != nil {
		return nil, err
	}

	c.ssePriceCached.mu.RLock()
	defer c.ssePriceCached.mu.RUnlock()

	cachedUpdates := make(map[string]*types.LatestPriceData)

	for _, priceFeedID := range priceFeedIDs {
		priceFeedIDRaw := hex.EncodeToString(common.FromHex(priceFeedID))

		if _, ok := c.ssePriceCached.latestPrice[priceFeedIDRaw]; !ok {
			return nil, fmt.Errorf("this price feed has not been subscribed to: %s", priceFeedID)
		}

		cachedUpdates[priceFeedID] = c.ssePriceCached.latestPrice[priceFeedIDRaw]
	}

	return cachedUpdates, nil
}

// LastStreamUpdate reports the wall-clock time the SSE stream last delivered a valid price
// update from any feed. A zero time means no update has been received yet. This is the
// stream-liveness signal: callers can export `time.Since(LastStreamUpdate())` as a metric to
// alert when the whole stream goes dark, independent of how many times it reconnects.
func (c *Client) LastStreamUpdate() time.Time {
	c.ssePriceCached.mu.RLock()
	defer c.ssePriceCached.mu.RUnlock()

	return c.ssePriceCached.lastEventAt
}

// LastFeedPublishTime reports Pyth's publish_time for the latest cached price of the given
// feed. A zero time means the feed has not been seen yet. This is the per-feed staleness
// signal: callers can export `time.Since(LastFeedPublishTime(id))` as a per-feed metric
// (e.g. NoPriceUpdateSince{feed="..."}) to detect a single feed going stale at the source
// even while the stream itself stays alive.
func (c *Client) LastFeedPublishTime(priceFeedID string) time.Time {
	priceFeedIDRaw := hex.EncodeToString(common.FromHex(priceFeedID))

	c.ssePriceCached.mu.RLock()
	defer c.ssePriceCached.mu.RUnlock()

	lpd, ok := c.ssePriceCached.latestPrice[priceFeedIDRaw]
	if !ok || lpd.PriceFeed == nil || lpd.PriceFeed.Price.PublishTime == nil {
		return time.Time{}
	}

	return time.Unix(lpd.PriceFeed.Price.PublishTime.Int64(), 0)
}

// Handler of the sse streaming event.
func (c *Client) handleSseEvent(event *sse.Event) {
	// Decode the price from sse response to LatestPriceData.
	var priceResp latestPriceResponse
	if err := json.Unmarshal(event.Data, &priceResp); err != nil {
		c.logger.Error(
			"skipping msg, encountered an error when unmarshalling streaming data", "error", err,
		)

		return
	}

	// Unpack the price data and store in the local cache.
	c.ssePriceCached.mu.Lock()
	if err := c.resolveSsePrice(&priceResp); err != nil {
		c.logger.Error(
			"encountered an error when decoding price response from sse stream", "error", err,
		)
	} else {
		// Record stream liveness: the wall-clock time we last received a valid update from
		// any feed. Used as the transport-health backstop behind the per-feed publish_time
		// staleness signal (see LastStreamUpdate / LastFeedPublishTime).
		c.ssePriceCached.lastEventAt = time.Now()
	}
	c.ssePriceCached.mu.Unlock()

	// Close the channel to effectively broadcast the first update to consumers
	c.ssePriceCached.broadcastOnce.Do(func() {
		close(c.ssePriceCached.ready) // Signal that the first update has occurred
	})
}

// waitForReady waits for the ready signal. Once the channel is closed,
// reads from the channel are non-blocking and very fast, essentially becoming a no-op.
func (c *Client) waitForReady(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.ssePriceCached.ready:
			return nil
		default:
			// If the channel is not closed, wait for a short duration before checking again
			// to avoid busy waiting
			// choose 100ms since 1st price data usually comes <1s after subscription
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// subscribeWithRetries supervises the SSE subscription. The SSE client reconnects through
// transient disconnects on its own (see ReconnectStrategy / ReconnectNotify), so subscribe()
// only returns on a clean shutdown (context cancelled) or an error the client deemed
// unrecoverable. The unrecoverable case is logged at error level and we re-subscribe with a
// capped backoff rather than panicking: a crash is never better than a retry, and the
// price-staleness metric is the authoritative "things are broken" signal regardless.
func (c *Client) subscribeWithRetries(ctx context.Context, subscribe func() error) {
	wait := initialBackoff

	for {
		err := subscribe()

		// Distinguish a clean shutdown from a genuine failure.
		if ctx.Err() != nil {
			return
		}

		if err == nil {
			return
		}

		c.logger.Error(
			"SSE subscription returned an unrecoverable error, resubscribing...",
			"error", err, "backoff", wait.String(),
		)

		select {
		case <-ctx.Done():
			return
		// #nosec:G404 // jitter only, not security-sensitive.
		case <-time.After(wait + time.Duration(rand.Intn(1000))*time.Millisecond):
		}

		if wait < maxBackoff {
			if wait *= 2; wait > maxBackoff {
				wait = maxBackoff
			}
		}
	}
}
