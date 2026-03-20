package tokens

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type tokenPair struct {
	Access  string
	Refresh string

	Expires time.Time
}

type TokenDBO struct {
	ID int `db:"id"`

	UserID    string    `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	Revoked   bool      `db:"revoked"`
	CreatedAt time.Time `db:"created_at"`
	ExpiresAt time.Time `db:"expires_at"`
}

type TokenHash string

func NewTokenHash(token string) TokenHash {
	tokenHash := sha256.Sum256([]byte(token))

	return TokenHash(hex.EncodeToString(tokenHash[:]))
}

func (th TokenHash) String() string {
	return string(th)
}

type Token struct {
	UserID string

	TokenHash TokenHash

	Revoked   bool
	CreatedAt time.Time
	ExpiresAt time.Time
}

func NewToken(userID, token string, expires time.Time) Token {
	return Token{
		UserID:    userID,
		TokenHash: NewTokenHash(token),
		Revoked:   false,
		CreatedAt: time.Now(),
		ExpiresAt: expires,
	}
}
