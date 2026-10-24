package secret

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testKey(t *testing.T, n int) string {
	t.Helper()

	b := make([]byte, n)
	_, err := rand.Read(b)
	require.NoError(t, err)

	return base64.StdEncoding.EncodeToString(b)
}

func TestCipher_Roundtrip(t *testing.T) {
	c, err := NewCipher(testKey(t, 32))
	require.NoError(t, err)

	enc, err := c.Encrypt("tok", "h.local", "hunter2-secret")
	require.NoError(t, err)

	raw, err := base64.StdEncoding.DecodeString(enc)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "hunter2-secret")
	assert.NotContains(t, enc, "hunter2-secret")

	got, err := c.Decrypt("tok", "h.local", enc)
	require.NoError(t, err)
	assert.Equal(t, "hunter2-secret", got)
}

func TestCipher_Failures(t *testing.T) {
	c, err := NewCipher(testKey(t, 32))
	require.NoError(t, err)

	enc, err := c.Encrypt("tok", "h.local", "value")
	require.NoError(t, err)

	other, err := NewCipher(testKey(t, 32))
	require.NoError(t, err)

	_, err = other.Decrypt("tok", "h.local", enc)
	require.Error(t, err)

	_, err = c.Decrypt("other-name", "h.local", enc)
	require.ErrorContains(t, err, "other-name")

	_, err = c.Decrypt("tok", "evil.example", enc)
	require.Error(t, err)

	_, err = c.Decrypt("tok", "h.local", "!!!")
	require.Error(t, err)

	_, err = c.Decrypt("tok", "h.local", "AAAA")
	require.Error(t, err)
}

func TestNewCipher_BadKey(t *testing.T) {
	_, err := NewCipher(testKey(t, 16))
	require.ErrorContains(t, err, "SECRETS_KEY")
	require.ErrorContains(t, err, "32 bytes")

	_, err = NewCipher("not base64 !")
	require.ErrorContains(t, err, "SECRETS_KEY")
}
