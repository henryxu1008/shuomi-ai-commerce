package googlemerchant

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	domain "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	merchant "google.golang.org/api/merchantapi/products/v1"
	sigjson "sigs.k8s.io/json"
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
	if ctx.Err() != nil {
		return domain.TokenSet{}, domain.ErrRefreshNotSent
	}
	form := url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {previous.RefreshToken},
		"client_id": {c.config.ClientID}, "client_secret": {c.config.ClientSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.Endpoint.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return domain.TokenSet{}, domain.ErrRefreshNotSent
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Use the existing bounded, no-redirect, no-POST-replay transport. The
	// refresh response must preserve the distinction between omitted and
	// explicitly invalid refresh_token fields.
	started := time.Now()
	response, err := c.client.Do(req)
	if err != nil {
		return domain.TokenSet{}, domain.ErrGrantUnavailable
	}
	defer response.Body.Close()
	var body struct {
		AccessToken  string          `json:"access_token"`
		RefreshToken json.RawMessage `json:"refresh_token"`
		TokenType    string          `json:"token_type"`
		Scope        json.RawMessage `json:"scope"`
		ExpiresIn    json.Number     `json:"expires_in"`
		Error        string          `json:"error"`
	}
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return domain.TokenSet{}, domain.ErrGrantUnavailable
	}
	defer clear(raw)
	if duplicate, err := sigjson.UnmarshalStrict(raw, &body, sigjson.DisallowDuplicateFields); err != nil || len(duplicate) != 0 {
		return domain.TokenSet{}, domain.ErrGrantUnavailable
	}
	if response.StatusCode != http.StatusOK && (body.AccessToken != "" || body.RefreshToken != nil || body.ExpiresIn != "") {
		return domain.TokenSet{}, domain.ErrGrantUnavailable
	}
	switch response.StatusCode {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
		switch body.Error {
		case "invalid_grant", "access_denied":
			return domain.TokenSet{}, domain.ErrGrantRejected
		case "invalid_client":
			return domain.TokenSet{}, domain.RefreshRetry{After: refreshRetryAfter(response.Header.Get("Retry-After"))}
		}
	case http.StatusTooManyRequests:
		if body.Error == "rate_limit_exceeded" || body.Error == "slow_down" {
			return domain.TokenSet{}, domain.RefreshRetry{After: refreshRetryAfter(response.Header.Get("Retry-After"))}
		}
	}
	if response.StatusCode != http.StatusOK || body.Error != "" {
		return domain.TokenSet{}, domain.ErrGrantUnavailable
	}
	refresh := previous.RefreshToken
	if body.RefreshToken != nil && (json.Unmarshal(body.RefreshToken, &refresh) != nil ||
		string(body.RefreshToken) == "null" || !validToken(refresh)) {
		return domain.TokenSet{}, domain.ErrGrantUnavailable
	}
	if body.Scope != nil {
		var scope string
		if json.Unmarshal(body.Scope, &scope) != nil || string(body.Scope) == "null" {
			return domain.TokenSet{}, domain.ErrGrantUnavailable
		}
		if !slices.Contains(strings.Fields(scope), merchant.ContentScope) {
			return domain.TokenSet{}, domain.ErrGrantRejected
		}
	}
	lifetime, err := body.ExpiresIn.Int64()
	if err != nil || lifetime <= 0 || lifetime > int64(math.MaxInt64/time.Second) {
		return domain.TokenSet{}, domain.ErrGrantUnavailable
	}
	return grantResult(&oauth2.Token{AccessToken: body.AccessToken, RefreshToken: refresh, TokenType: body.TokenType,
		Expiry: started.Add(time.Duration(lifetime) * time.Second)}, nil)
}

func refreshRetryAfter(value string) time.Duration {
	if seconds, err := strconv.ParseUint(value, 10, 32); err == nil {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil {
		return time.Until(at)
	}
	return 0
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
