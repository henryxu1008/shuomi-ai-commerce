package googlemerchant

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	domain "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	merchant "google.golang.org/api/merchantapi/products/v1"
)

var challengeInput = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)

type OAuth struct {
	config oauth2.Config
	client *http.Client
}

func NewOAuth(application domain.Application, secret string, httpClient *http.Client) (*OAuth, error) {
	callback, err := url.Parse(application.CallbackURL)
	if err != nil || callback.Scheme != "https" || callback.Hostname() == "" || callback.User != nil ||
		callback.Fragment != "" || callback.RawQuery != "" || callback.ForceQuery || callback.Path == "" ||
		!authidentity.IsBoundedIdentifier(application.ID) || !authidentity.IsBoundedIdentifier(application.Revision) ||
		!validToken(secret) {
		return nil, domain.ErrInvalidGrantInput
	}
	return &OAuth{
		config: oauth2.Config{
			ClientID: application.ID, ClientSecret: secret, RedirectURL: application.CallbackURL,
			Endpoint: google.Endpoint, Scopes: []string{merchant.ContentScope},
		},
		client: boundedHTTPClient(httpClient),
	}, nil
}

func (c *OAuth) AuthorizationURL(state, verifier string) (string, error) {
	if c == nil || !challengeInput.MatchString(state) || !challengeInput.MatchString(verifier) {
		return "", domain.ErrInvalidGrantInput
	}
	return c.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"), oauth2.S256ChallengeOption(verifier)), nil
}

func (c *OAuth) Exchange(ctx context.Context, code, verifier string) (domain.TokenSet, error) {
	if c == nil || ctx == nil || !validToken(code) || !challengeInput.MatchString(verifier) {
		return domain.TokenSet{}, domain.ErrInvalidGrantInput
	}
	result, err := c.config.Exchange(context.WithValue(ctx, oauth2.HTTPClient, c.client), code, oauth2.VerifierOption(verifier))
	return grantResult(result, err)
}

func (c *OAuth) Refresh(ctx context.Context, previous domain.TokenSet) (domain.TokenSet, error) {
	if c == nil || ctx == nil || !validToken(previous.RefreshToken) {
		return domain.TokenSet{}, domain.ErrInvalidGrantInput
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.client)
	result, err := c.config.TokenSource(ctx, &oauth2.Token{RefreshToken: previous.RefreshToken}).Token()
	return grantResult(result, err)
}

func grantResult(token *oauth2.Token, err error) (domain.TokenSet, error) {
	if err != nil {
		var rejected *oauth2.RetrieveError
		if errors.As(err, &rejected) && (rejected.ErrorCode == "invalid_grant" || rejected.ErrorCode == "access_denied") {
			return domain.TokenSet{}, domain.ErrGrantRejected
		}
		return domain.TokenSet{}, domain.ErrGrantUnavailable
	}
	if token == nil || !validToken(token.AccessToken) || !validToken(token.RefreshToken) ||
		!strings.EqualFold(token.TokenType, "Bearer") || !time.Now().Before(token.Expiry) {
		return domain.TokenSet{}, domain.ErrGrantUnavailable
	}
	if scope := token.Extra("scope"); scope != nil {
		granted, ok := scope.(string)
		if !ok || !slices.Contains(strings.Fields(granted), merchant.ContentScope) {
			return domain.TokenSet{}, domain.ErrGrantRejected
		}
	}
	return domain.TokenSet{
		AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, ExpiresAt: token.Expiry.UTC(),
	}, nil
}

func (c *OAuth) BeginAuthorization(state string) (string, error) {
	if c == nil {
		return "", domain.ErrInvalidGrantInput
	}
	return c.AuthorizationURL(state, c.verifier(state))
}

func (c *OAuth) ExchangeAuthorization(ctx context.Context, state, code string) (domain.TokenSet, error) {
	if c == nil || !challengeInput.MatchString(state) {
		return domain.TokenSet{}, domain.ErrInvalidGrantInput
	}
	return c.Exchange(ctx, code, c.verifier(state))
}

// The server derives a different PKCE verifier for every random state. The
// browser sees the challenge but cannot reconstruct the verifier.
func (c *OAuth) verifier(state string) string {
	key := hmac.New(sha256.New, []byte(c.config.ClientSecret))
	_, _ = key.Write([]byte("google-merchant/oauth/pkce/v1:" + state))
	return base64.RawURLEncoding.EncodeToString(key.Sum(nil))
}
