package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code atteso %d, ricevuto %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if body != "OK" {
		t.Errorf("corpo della risposta atteso %q, ricevuto %q", "OK", body)
	}
}