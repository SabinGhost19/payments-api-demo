package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComputeQuote(t *testing.T) {
	// fee = round(10000 * 0.029) + 30 = 290 + 30 = 320; net = 9680
	q, err := computeQuote(10000, "USD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Fee != 320 || q.Net != 9680 || q.Amount != 10000 || q.Currency != "USD" {
		t.Fatalf("unexpected quote: %+v", q)
	}
}

func TestComputeQuoteRejectsBadInput(t *testing.T) {
	if _, err := computeQuote(0, "USD"); err == nil {
		t.Fatal("expected error for non-positive amount")
	}
	if _, err := computeQuote(100, "JPY"); err == nil {
		t.Fatal("expected error for unsupported currency")
	}
}

func TestQuoteHandlerOK(t *testing.T) {
	body := strings.NewReader(`{"amount":10000,"currency":"EUR"}`)
	req := httptest.NewRequest(http.MethodPost, "/quote", body)
	rec := httptest.NewRecorder()
	quote(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp quoteResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Fee != 320 || resp.Net != 9680 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestQuoteHandlerRejectsBadCurrency(t *testing.T) {
	body := strings.NewReader(`{"amount":100,"currency":"JPY"}`)
	req := httptest.NewRequest(http.MethodPost, "/quote", body)
	rec := httptest.NewRecorder()
	quote(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	health(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "payments-api") {
		t.Fatalf("unexpected health body: %s", rec.Body.String())
	}
}
