// Async worker that simulates draining a payment queue. No HTTP server,
// no third-party deps — keeps the image SBOM minimal so the strict
// SCA accepts it as Verified.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func processOne(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Second):
		log.Printf("payment batch at %s", time.Now().UTC().Format(time.RFC3339))
	}
}

// parseInterval is a pure helper (unit-testable) that turns the TICK_INTERVAL
// env value into a positive duration, falling back to def for empty, invalid
// or non-positive values.
func parseInterval(raw string, def time.Duration) time.Duration {
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return def
	}
	return d
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	interval := parseInterval(os.Getenv("TICK_INTERVAL"), 2*time.Second)

	log.Printf("payments-worker starting (tick=%s)", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Printf("payments-worker shutdown")
			return
		case <-ticker.C:
			processOne(ctx)
		}
	}
}
