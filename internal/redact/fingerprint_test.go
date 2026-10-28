package redact_test

import (
	"testing"

	"github.com/Alarm2024/stale/internal/redact"
)

func TestFingerprintLength(t *testing.T) {
	fp := redact.Fingerprint("https://rpc.example.com?key=secret")
	if len(fp) != 8 {
		t.Fatalf("len(Fingerprint) = %d, want 8", len(fp))
	}
	if fp == redact.Fingerprint("https://rpc.example.com?key=other") {
		t.Fatal("different URLs should yield different fingerprints")
	}
}
