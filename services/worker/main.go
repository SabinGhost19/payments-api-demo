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
		log.Printf("processed payment batch at %s", time.Now().UTC().Format(time.RFC3339))
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	interval := 2 * time.Second
	if v := os.Getenv("TICK_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}

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
