package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// Cipher does authenticated symmetric encryption (AES-256-GCM) for secrets that must be
// recoverable in plaintext, unlike the one-way hashing used for tokens at rest.
type Cipher struct {
	aead cipher.AEAD
}

// DeriveKey turns a shared secret into a 32-byte AES-256 key bound to a context label, so
// the same secret yields independent keys for different uses.
func DeriveKey(secret, context string) []byte {
	sum := sha256.Sum256([]byte(context + ":" + secret))
	return sum[:]
}

func NewCipher(key []byte) (*Cipher, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt returns base64(nonce || ciphertext||tag).
func (c *Cipher) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := c.aead.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (c *Cipher) Decrypt(ciphertext string) ([]byte, error) {
	sealed, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}
	nonceSize := c.aead.NonceSize()
	if len(sealed) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, body := sealed[:nonceSize], sealed[nonceSize:]
	return c.aead.Open(nil, nonce, body, nil)
}
