package crypto

import (
	"crypto/rand"
	"strings"
	"testing"
)

// ===== AES-256-GCM 加密器 =====

func TestNewEncryptor_ValidKey(t *testing.T) {
	key := make([]byte, 32)
	e, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("NewEncryptor with 32-byte key should succeed, got: %v", err)
	}
	if e == nil {
		t.Fatal("Encryptor should not be nil")
	}
}

func TestNewEncryptor_InvalidKeyLength(t *testing.T) {
	for _, size := range []int{0, 1, 16, 31, 33, 64} {
		key := make([]byte, size)
		_, err := NewEncryptor(key)
		if err == nil {
			t.Errorf("NewEncryptor with %d-byte key should fail", size)
		}
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatal(err)
	}

	cases := []string{
		"sk-deepseek-abc123",
		"这是一个中文 API Key 🔑",
		"",
		"x",
		strings.Repeat("a", 1000),
	}
	for _, pt := range cases {
		ct, err := enc.Encrypt(pt)
		if err != nil {
			t.Fatalf("Encrypt(%q) error: %v", pt, err)
		}
		decrypted, err := enc.Decrypt(ct)
		if err != nil {
			t.Fatalf("Decrypt error: %v", err)
		}
		if decrypted != pt {
			t.Errorf("RoundTrip mismatch: got %q, want %q", decrypted, pt)
		}
	}
}

func TestEncrypt_ProducesDifferentCiphertext(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := NewEncryptor(key)

	plaintext := "sk-test-key-12345"
	ct1, _ := enc.Encrypt(plaintext)
	ct2, _ := enc.Encrypt(plaintext)
	if ct1 == ct2 {
		t.Error("Same plaintext should produce different ciphertexts (nonce randomization)")
	}
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 1

	enc1, _ := NewEncryptor(key1)
	enc2, _ := NewEncryptor(key2)

	ct, _ := enc1.Encrypt("secret data")
	_, err := enc2.Decrypt(ct)
	if err == nil {
		t.Error("Decrypt with wrong key should fail")
	}
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := NewEncryptor(key)
	_, err := enc.Decrypt("!!!not-valid-base64!!!")
	if err == nil {
		t.Error("Decrypt of invalid base64 should fail")
	}
}

func TestDecrypt_CiphertextTooShort(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := NewEncryptor(key)
	// base64 of 5 bytes — shorter than nonce size (12 bytes for GCM)
	short := "AAAAAAA=" // decodes to ~5 bytes
	_, err := enc.Decrypt(short)
	if err == nil {
		t.Error("Decrypt of too-short ciphertext should fail")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	enc, _ := NewEncryptor(key)

	ct, _ := enc.Encrypt("sensitive data")
	// Tamper with the ciphertext by flipping last chars
	tampered := ct[:len(ct)-2] + "AB"
	_, err := enc.Decrypt(tampered)
	if err == nil {
		t.Error("Decrypt of tampered ciphertext should fail (GCM integrity)")
	}
}

// ===== 密码 bcrypt =====

func TestHashPassword_AndCheckPassword(t *testing.T) {
	pwd := "admin123"
	hash, err := HashPassword(pwd)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == "" {
		t.Fatal("Hash should not be empty")
	}
	if hash == pwd {
		t.Fatal("Hash should differ from plaintext")
	}
	if err := CheckPassword(hash, pwd); err != nil {
		t.Errorf("CheckPassword with correct password should succeed: %v", err)
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, _ := HashPassword("correct-pwd")
	if err := CheckPassword(hash, "wrong-pwd"); err == nil {
		t.Error("CheckPassword with wrong password should fail")
	}
}

func TestHashPassword_DifferentHashesEachCall(t *testing.T) {
	h1, _ := HashPassword("same-password")
	h2, _ := HashPassword("same-password")
	if h1 == h2 {
		t.Error("bcrypt should produce different hashes for same password (salt)")
	}
}
