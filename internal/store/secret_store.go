package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"support_bot/internal/db/sqlcgen"
	"support_bot/internal/models"
	"support_bot/internal/pkg/secret"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SecretStore хранит зашифрованные секреты (таблица secrets).
// cipher может быть nil: тогда Set и Get возвращают ошибку.
type SecretStore struct {
	q      *sqlcgen.Queries
	cipher *secret.Cipher
}

func NewSecretStore(pool *pgxpool.Pool, c *secret.Cipher) *SecretStore {
	return &SecretStore{q: sqlcgen.New(pool), cipher: c}
}

var errNoSecretsKey = errors.New("SECRETS_KEY is not set")

// SecretInfo — имя секрета и хост, на который его разрешено отправлять.
type SecretInfo struct{ Name, Host string }

// Set сохраняет секрет, привязанный к хосту (host[:port], хранится в нижнем регистре).
func (s *SecretStore) Set(ctx context.Context, name, host, plain string) error {
	if s.cipher == nil {
		return errNoSecretsKey
	}

	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return errors.New("secret host is empty")
	}

	enc, err := s.cipher.Encrypt(name, host, plain)
	if err != nil {
		return err
	}

	if err := s.q.UpsertSecret(
		ctx,
		sqlcgen.UpsertSecretParams{Name: name, Value: enc, Host: host},
	); err != nil {
		return fmt.Errorf("save secret %q: %w", name, err)
	}

	return nil
}

// Get расшифровывает секрет и регистрирует значение для маскирования в ошибках.
func (s *SecretStore) Get(
	ctx context.Context,
	name string,
) (value, host string, err error) {
	if s.cipher == nil {
		return "", "", errNoSecretsKey
	}

	row, err := s.q.GetSecret(ctx, name)
	if err != nil {
		if errors.Is(translateNoRows(err), models.ErrNotFound) {
			return "", "", fmt.Errorf("secret %q not found", name)
		}

		return "", "", fmt.Errorf("get secret %q: %w", name, err)
	}

	v, err := s.cipher.Decrypt(name, row.Host, row.Value)
	if err != nil {
		return "", "", err
	}

	secret.Register(secret.String(v))

	return v, row.Host, nil
}

func (s *SecretStore) List(ctx context.Context) ([]SecretInfo, error) {
	rows, err := s.q.ListSecrets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}

	out := make([]SecretInfo, len(rows))
	for i, r := range rows {
		out[i] = SecretInfo{Name: r.Name, Host: r.Host}
	}

	return out, nil
}

func (s *SecretStore) Delete(ctx context.Context, name string) error {
	n, err := s.q.DeleteSecret(ctx, name)

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return fmt.Errorf("secret %q is used by queries", name)
	}

	if err != nil {
		return fmt.Errorf("delete secret %q: %w", name, err)
	}

	if n == 0 {
		return fmt.Errorf("secret %q not found", name)
	}

	return nil
}
