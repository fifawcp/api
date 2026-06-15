package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCipher(t *testing.T, secret string) *Cipher {
	t.Helper()
	c, err := NewCipher(DeriveKey(secret, "test"))
	require.NoError(t, err)
	return c
}

func TestCipher_EncryptDecryptRoundTrip(t *testing.T) {
	c := newTestCipher(t, "super-secret")
	plaintext := []byte(`{"refresh_token":"rt_abc123"}`)

	ciphertext, err := c.Encrypt(plaintext)
	require.NoError(t, err)
	assert.False(t, strings.Contains(ciphertext, "rt_abc123"), "ciphertext must not leak plaintext")

	decrypted, err := c.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestCipher_DecryptWithWrongKeyFails(t *testing.T) {
	a := newTestCipher(t, "secret-a")
	b := newTestCipher(t, "secret-b")

	ciphertext, err := a.Encrypt([]byte("payload"))
	require.NoError(t, err)

	_, err = b.Decrypt(ciphertext)
	assert.Error(t, err, "a different key must not decrypt")
}

func TestCipher_DecryptTamperedCiphertextFails(t *testing.T) {
	c := newTestCipher(t, "secret")

	ciphertext, err := c.Encrypt([]byte("payload"))
	require.NoError(t, err)

	_, err = c.Decrypt(corruptFirstChar(ciphertext))
	assert.Error(t, err, "tampered ciphertext must fail authentication")
}

func TestCipher_EncryptIsNonDeterministic(t *testing.T) {
	c := newTestCipher(t, "secret")

	first, err := c.Encrypt([]byte("payload"))
	require.NoError(t, err)
	second, err := c.Encrypt([]byte("payload"))
	require.NoError(t, err)

	assert.NotEqual(t, first, second, "each encryption must use a fresh nonce")
}

func corruptFirstChar(s string) string {
	b := []byte(s)
	if b[0] == 'A' {
		b[0] = 'B'
	} else {
		b[0] = 'A'
	}
	return string(b)
}
