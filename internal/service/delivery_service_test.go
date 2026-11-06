package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func TestSignPayload(t *testing.T) {
	secret := "test-secret"
	timestamp := "1730874600"
	payload := []byte(`{"event":"CLICK"}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "."))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	got := SignPayload(secret, timestamp, payload)
	if got != expected {
		t.Fatalf("expected signature %s, got %s", expected, got)
	}
}

func TestCalculateBackoff(t *testing.T) {
	cases := []struct {
		attempt  int
		expected time.Duration
	}{
		{attempt: 0, expected: 1 * time.Second},
		{attempt: 1, expected: 1 * time.Second},
		{attempt: 2, expected: 5 * time.Second},
		{attempt: 3, expected: 30 * time.Second},
		{attempt: 4, expected: 2 * time.Minute},
		{attempt: 5, expected: 10 * time.Minute},
		{attempt: 6, expected: 10 * time.Minute},
		{attempt: 99, expected: 10 * time.Minute},
	}

	for _, tc := range cases {
		got := CalculateBackoff(tc.attempt)
		if got != tc.expected {
			t.Fatalf("attempt %d expected %v, got %v", tc.attempt, tc.expected, got)
		}
	}
}
