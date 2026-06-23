package hermes_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/berachain/go-pyth-client/hermes"
	"github.com/stretchr/testify/assert"
)

func TestSubscribePriceStreaming(t *testing.T) {
	ctx, pythClient := setUp()

	pythClient.SubscribePriceStreaming(ctx, testPairs)

	prices, err := pythClient.GetCachedLatestPriceUpdates(ctx, testPairs)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(prices))
	for _, pair := range testPairs {
		assert.Contains(t, prices, pair)
	}
}

func TestSubscribePriceStreaming_EmptyRequests(t *testing.T) {
	ctx, pythClient := setUp()

	pythClient.SubscribePriceStreaming(ctx, testPairs)

	var empty_pair = []string{}

	prices, err := pythClient.GetCachedLatestPriceUpdates(ctx, empty_pair)
	assert.Error(t, err)
	assert.Nil(t, prices)
}

func TestSubscribePriceStreaming_PriceFeedNotSubscribed(t *testing.T) {
	ctx, pythClient := setUp()

	pythClient.SubscribePriceStreaming(ctx, testPairs)

	var feed = []string{
		"0xf67b033925d73d43ba4401e00308d9b0f26ab4fbd1250e8b5407b9eaade7e1f4", // HONEY/USD
	}

	prices, err := pythClient.GetCachedLatestPriceUpdates(ctx, feed)
	assert.Error(t, err)
	assert.Nil(t, prices)
}

// TestSubscribePriceStreaming_StopsOnContextCancel is a regression test for a goroutine leak on
// shutdown. The SSE reconnect loop runs inside r3labs via backoff.RetryNotify, which only exits on
// context cancellation when the ReconnectStrategy carries the caller's context. With
// MaxElapsedTime = 0 (never give up) and a non-context-aware backoff, ensureContext wrapped it in
// context.Background() — whose Done() never fires — so cancelling ctx never stopped the loop and
// the subscribe goroutine leaked forever. This test drives the real r3labs + backoff path against
// a local SSE server, cancels the context, and asserts the goroutine actually returns.
func TestSubscribePriceStreaming_StopsOnContextCancel(t *testing.T) {
	connected := make(chan struct{}, 1)

	// Minimal SSE server: open the stream, emit a keepalive comment, then hold the connection
	// open until the client disconnects (i.e. until the caller's context is cancelled).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			// A comment line keeps the SSE parser happy without needing a valid price payload.
			_, _ = w.Write([]byte(": connected\n\n"))
			f.Flush()
		}
		select {
		case connected <- struct{}{}:
		default:
		}
		<-r.Context().Done()
	}))
	defer srv.Close()

	cfg := testConfig
	cfg.APIEndpoint = srv.URL
	pythClient, err := hermes.NewClient(&cfg, slog.Default())
	assert.NoError(t, err)

	// Count subscribe goroutines by their stack root. Other tests in this package subscribe with
	// a background context that is never cancelled, so they leave their own subscribeWithRetries
	// goroutines parked in this shared process — measure a delta against that baseline rather than
	// an absolute count.
	const marker = "hermes.(*Client).subscribeWithRetries"
	subscribeGoroutines := func() int { return strings.Count(goroutineDump(), marker) }

	base := subscribeGoroutines()

	ctx, cancel := context.WithCancel(context.Background())
	pythClient.SubscribePriceStreaming(ctx, testPairs)

	// Wait until the stream is actually established before cancelling, so we exercise the
	// reconnect loop's cancellation path rather than a connect-time failure.
	select {
	case <-connected:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("SSE stream never connected to the test server")
	}

	cancel()

	// With the fix our subscribe goroutine returns and the count drops back to baseline; with the
	// bug it spins on the cancelled context forever and the count stays at baseline+1 until this
	// times out.
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if subscribeGoroutines() <= base {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf(
		"subscribe goroutine still parked in the retry loop after context cancel (leaked): "+
			"want <= %d subscribe goroutines, still have %d:\n%s",
		base, subscribeGoroutines(), goroutineDump(),
	)
}

// goroutineDump returns the stack traces of all running goroutines.
func goroutineDump() string {
	buf := make([]byte, 1<<16)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return string(buf[:n])
		}
		buf = make([]byte, 2*len(buf))
	}
}

// To run this benchmark only without other tests: `go test -run=^$ -bench=BenchmarkGetCachedLatestPriceUpdates`
func BenchmarkGetCachedLatestPriceUpdates(b *testing.B) {
	ctx, pythClient := setUp()

	pythClient.SubscribePriceStreaming(ctx, testPairs)

	for i := 0; i < b.N; i++ {
		prices, err := pythClient.GetCachedLatestPriceUpdates(ctx, testPairs)
		assert.NoError(b, err)
		assert.Equal(b, 5, len(prices))
	}
}
