package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Cipher шифрует значения секретов в БД (AES-256-GCM, имя и хост секрета — AAD).
type Cipher struct{ aead cipher.AEAD }

// NewCipher принимает ключ из SECRETS_KEY: base64 от ровно 32 байт.
func NewCipher(b64key string) (*Cipher, error) {
	key, err := base64.StdEncoding.DecodeString(b64key)
	if err != nil {
		return nil, fmt.Errorf("SECRETS_KEY is not valid base64: %w", err)
	}

	if len(key) != keyLen {
		return nil, fmt.Errorf(
			"SECRETS_KEY must decode to %d bytes, got %d",
			keyLen,
			len(key),
		)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	return &Cipher{aead: aead}, nil
}

// Encrypt возвращает base64(nonce || ciphertext).
func (c *Cipher) Encrypt(name, host, plain string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("encrypt secret %q: %w", name, err)
	}

	out := c.aead.Seal(nonce, nonce, []byte(plain), aad(name, host))

	return base64.StdEncoding.EncodeToString(out), nil
}

// aad связывает шифртекст с именем и хостом: подмена любого из них ломает расшифровку.
func aad(name, host string) []byte { return []byte(name + "\x00" + host) }

// Decrypt обратна Encrypt; имя и хост должны совпасть с использованными при шифровании.
func (c *Cipher) Decrypt(name, host, b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("decrypt secret %q: %w", name, err)
	}

	n := c.aead.NonceSize()
	if len(raw) < n {
		return "", fmt.Errorf("decrypt secret %q: ciphertext too short", name)
	}

	plain, err := c.aead.Open(nil, raw[:n], raw[n:], aad(name, host))
	if err != nil {
		return "", fmt.Errorf("decrypt secret %q: %w", name, err)
	}

	return string(plain), nil
}
