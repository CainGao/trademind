// Package crypto 密码与加解密工具（规范 V1.0 §6.1/§6.2）。
//
// 密码：bcrypt cost=12
// AI Key 等敏感字段：AES-256-GCM（密钥运行时随机生成并持久化到 settings 表）
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// ===== 密码 =====

// HashPassword 密码 bcrypt 加密（规范 §6.1: cost=12）。
func HashPassword(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword 校验密码。匹配返回 nil，不匹配返回错误。
func CheckPassword(hash, pwd string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd))
}

// ===== AES-256-GCM 加密（用于 AI Key 等敏感字段）=====

// Encryptor AES-256-GCM 加密器。
type Encryptor struct {
	key []byte // 必须 32 字节
}

// NewEncryptor 创建加密器。key 必须为 32 字节。
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, errors.New("AES key 必须 32 字节")
	}
	return &Encryptor{key: key}, nil
}

// Encrypt 加密为 base64 字符串（nonce 前置）。
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	// Seal: 返回 nonce + ciphertext + tag
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// Decrypt 解密 base64 字符串。
func (e *Encryptor) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("密文过短")
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
