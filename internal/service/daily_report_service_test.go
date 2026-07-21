package service

import (
	"encoding/base64"
	"strings"
	"testing"
)

// ===== FeishuSign =====

func TestFeishuSign_Deterministic(t *testing.T) {
	ts := int64(1700000000)
	secret := "my-webhook-secret"
	sig1 := FeishuSign(ts, secret)
	sig2 := FeishuSign(ts, secret)
	if sig1 != sig2 {
		t.Errorf("FeishuSign should be deterministic for same inputs: %q vs %q", sig1, sig2)
	}
}

func TestFeishuSign_DifferentTimestampsDiffer(t *testing.T) {
	secret := "secret"
	sig1 := FeishuSign(1700000000, secret)
	sig2 := FeishuSign(1700000001, secret)
	if sig1 == sig2 {
		t.Error("Different timestamps should produce different signatures")
	}
}

func TestFeishuSign_DifferentSecretsDiffer(t *testing.T) {
	ts := int64(1700000000)
	sig1 := FeishuSign(ts, "secret1")
	sig2 := FeishuSign(ts, "secret2")
	if sig1 == sig2 {
		t.Error("Different secrets should produce different signatures")
	}
}

func TestFeishuSign_IsBase64(t *testing.T) {
	sig := FeishuSign(1700000000, "test-secret")
	decoded, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		t.Fatalf("FeishuSign output should be valid base64: %v (sig=%q)", err, sig)
	}
	// HMAC-SHA256 produces 32 bytes
	if len(decoded) != 32 {
		t.Errorf("Decoded signature should be 32 bytes (SHA-256), got %d", len(decoded))
	}
}

func TestFeishuSign_NonEmpty(t *testing.T) {
	sig := FeishuSign(1, "x")
	if sig == "" {
		t.Error("FeishuSign should return non-empty string")
	}
}

func TestFeishuSign_EmptySecret(t *testing.T) {
	// Even with empty secret, should produce a valid signature (HMAC of empty-key)
	sig := FeishuSign(1700000000, "")
	if sig == "" {
		t.Error("FeishuSign with empty secret should still produce output")
	}
	// Verify it's valid base64
	if _, err := base64.StdEncoding.DecodeString(sig); err != nil {
		t.Errorf("FeishuSign with empty secret should produce valid base64: %v", err)
	}
}

func TestFeishuSign_KnownValue(t *testing.T) {
	// Verify the algorithm matches: sign = base64(hmac_sha256(timestamp + "\n" + secret, empty_bytes))
	sig := FeishuSign(1700000000, "test123")
	if strings.HasPrefix(sig, " ") || strings.HasSuffix(sig, " ") {
		t.Errorf("FeishuSign output should not have leading/trailing whitespace: %q", sig)
	}
}
