package store

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/pkg/secret"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSecretStore(
	t *testing.T,
	withKey bool,
) (*SecretStore, pgxmock.PgxPoolIface) {
	t.Helper()

	pool, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	s := &SecretStore{q: sqlcgen.New(pool)}

	if withKey {
		k := make([]byte, 32)
		_, err := rand.Read(k)
		require.NoError(t, err)

		s.cipher, err = secret.NewCipher(base64.StdEncoding.EncodeToString(k))
		require.NoError(t, err)
	}

	return s, pool
}

func TestSecretStore_SetGetRoundtrip(t *testing.T) {
	s, pool := newSecretStore(t, true)

	pool.ExpectExec("insert into secrets").
		WithArgs("tok", pgxmock.AnyArg(), "billing.local:8080").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	require.NoError(t, s.Set(
		context.Background(), "tok", " Billing.Local:8080 ", "plain-value-1"))

	// Достаём шифртекст через повторный Encrypt нельзя (nonce случайный), поэтому шифруем сами.
	stored, err := s.cipher.Encrypt(
		"tok",
		"billing.local:8080",
		"plain-value-1",
	)
	require.NoError(t, err)
	assert.NotContains(t, stored, "plain-value-1")

	pool.ExpectQuery("select value, host from secrets").
		WithArgs("tok").
		WillReturnRows(pgxmock.NewRows([]string{"value", "host"}).
			AddRow(stored, "billing.local:8080"))

	got, host, err := s.Get(context.Background(), "tok")
	require.NoError(t, err)
	assert.Equal(t, "plain-value-1", got)
	assert.Equal(t, "billing.local:8080", host)
	assert.Equal(t, "***", secret.Redact("plain-value-1"))
	require.NoError(t, pool.ExpectationsWereMet())
}

func TestSecretStore_NoKey(t *testing.T) {
	s, _ := newSecretStore(t, false)

	require.ErrorContains(
		t,
		s.Set(context.Background(), "a", "h", "b"),
		"SECRETS_KEY is not set",
	)

	_, _, err := s.Get(context.Background(), "a")
	require.ErrorContains(t, err, "SECRETS_KEY is not set")
}

func TestSecretStore_GetNotFound(t *testing.T) {
	s, pool := newSecretStore(t, true)

	pool.ExpectQuery("select value, host from secrets").
		WithArgs("nope").
		WillReturnError(pgx.ErrNoRows)

	_, _, err := s.Get(context.Background(), "nope")
	require.ErrorContains(t, err, `"nope" not found`)
}

func TestSecretStore_SetEmptyHost(t *testing.T) {
	s, _ := newSecretStore(t, true)

	require.ErrorContains(
		t,
		s.Set(context.Background(), "a", " ", "b"),
		"host is empty",
	)
}

func TestSecretStore_GetHostTampered(t *testing.T) {
	s, pool := newSecretStore(t, true)

	stored, err := s.cipher.Encrypt("tok", "billing.local", "v")
	require.NoError(t, err)

	pool.ExpectQuery("select value, host from secrets").
		WithArgs("tok").
		WillReturnRows(pgxmock.NewRows([]string{"value", "host"}).
			AddRow(stored, "evil.example"))

	_, _, err = s.Get(context.Background(), "tok")
	require.Error(t, err)
}

func TestSecretStore_List(t *testing.T) {
	s, pool := newSecretStore(t, false)

	pool.ExpectQuery("select name, host from secrets").
		WillReturnRows(pgxmock.NewRows([]string{"name", "host"}).
			AddRow("a", "h1").AddRow("b", "h2"))

	got, err := s.List(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []SecretInfo{{"a", "h1"}, {"b", "h2"}}, got)
}

func TestSecretStore_Delete(t *testing.T) {
	s, pool := newSecretStore(t, false)

	pool.ExpectExec("delete from secrets").
		WithArgs("ok").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	require.NoError(t, s.Delete(context.Background(), "ok"))

	pool.ExpectExec("delete from secrets").
		WithArgs("gone").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))
	require.ErrorContains(
		t,
		s.Delete(context.Background(), "gone"),
		`"gone" not found`,
	)

	pool.ExpectExec("delete from secrets").
		WithArgs("used").
		WillReturnError(&pgconn.PgError{Code: "23503"})
	require.ErrorContains(
		t,
		s.Delete(context.Background(), "used"),
		`secret "used" is used by queries`,
	)
}
