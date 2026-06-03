// Tiny HTTP service standing in for a payments API endpoint.
// Intentionally pure-stdlib (no third-party deps) so the resulting
// distroless static image has a minimal SBOM with effectively zero CVEs.
// That gives the ZTA admission pipeline a clean "Verified" sample to
// contrast against the deliberately vulnerable analytics-worker.
package main

import (
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

type chargeRequest struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
	Account  string `json:"account"`
}

type chargeResponse struct {
	TxID      string `json:"txId"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "payments-api"})
}

type quoteRequest struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type quoteResponse struct {
	Amount   int    `json:"amount"`
	Fee      int    `json:"fee"`
	Net      int    `json:"net"`
	Currency string `json:"currency"`
}

var allowedCurrencies = map[string]bool{"USD": true, "EUR": true, "RON": true}

// computeQuote is a pure helper (no I/O) shared by the HTTP handler and the
// unit tests: it validates the request and computes the processing fee
// (2.9% + 30 minor units) and the resulting net amount.
func computeQuote(amount int, currency string) (quoteResponse, error) {
	if amount <= 0 {
		return quoteResponse{}, errors.New("amount must be positive")
	}
	if !allowedCurrencies[currency] {
		return quoteResponse{}, errors.New("unsupported currency")
	}
	fee := int(math.Round(float64(amount)*0.029)) + 30
	return quoteResponse{Amount: amount, Fee: fee, Net: amount - fee, Currency: currency}, nil
}

func quote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req quoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp, err := computeQuote(req.Amount, req.Currency)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func charge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req chargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	resp := chargeResponse{
		TxID:      "tx-" + time.Now().UTC().Format("20060102150405.000"),
		Status:    "queued",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func main() {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/healthz", health)
	mux.HandleFunc("/charge", charge)
	mux.HandleFunc("/quote", quote)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("payments-api listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("listen failed: %v", err)
	}
}
