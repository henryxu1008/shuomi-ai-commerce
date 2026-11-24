package googlemerchant

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidGrantInput = errors.New("invalid Google Merchant authorization request")
	ErrGrantRejected     = errors.New("Google Merchant authorization was rejected")
	ErrGrantUnavailable  = errors.New("Google Merchant authorization could not be confirmed")
)

type Application struct {
	ID, Revision, CallbackURL string
}

// TokenSet stays in the credential path. It must not be returned to a client.
type TokenSet struct {
	AccessToken  string    `json:"-"`
	RefreshToken string    `json:"-"`
	ExpiresAt    time.Time `json:"-"`
}

type Authorization interface {
	AuthorizationURL(state, verifier string) (string, error)
	Exchange(context.Context, string, string) (TokenSet, error)
	Refresh(context.Context, TokenSet) (TokenSet, error)
}
