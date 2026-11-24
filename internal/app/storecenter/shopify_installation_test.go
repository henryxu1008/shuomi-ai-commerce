package storecenterapp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
)

type startedShopifyTransport func(*http.Request) (*http.Response, error)

func (fn startedShopifyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestShopifyStartedInstallExchangesOwnerCodeBeforeWorkbenchSignIn(t *testing.T) {
	const appID = "gid://shopify/App/812"
	const domain = "fixture.myshopify.com"
	const secret = "fixture-shopify-secret"
	const primaryID = "gid://shopify/AppInstallation/27"
	exchanges, reads := 0, 0
	installationID, development, accountOwner := primaryID, true, true
	transport := &http.Client{Transport: startedShopifyTransport(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "https", r.URL.Scheme)
		require.Equal(t, domain, r.URL.Host)
		response := ""
		switch r.URL.Path {
		case "/admin/oauth/access_token":
			exchanges++
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			params, err := url.ParseQuery(string(body))
			require.NoError(t, err)
			require.Equal(t, "fixture-code", params.Get("code"))
			response = "{\"access_token\":\"owner-token\",\"expires_in\":3600,\"scope\":\"write_products\",\"associated_user_scope\":\"write_products\",\"associated_user\":{\"account_owner\":" + strconv.FormatBool(accountOwner) + "}}"
		case "/admin/api/2026-10/graphql.json":
			reads++
			require.Equal(t, "owner-token", r.Header.Get("X-Shopify-Access-Token"))
			response = "{\"data\":{\"shop\":{\"id\":\"gid://shopify/Shop/19\",\"myshopifyDomain\":\"" + domain +
				"\",\"currencyCode\":\"USD\",\"plan\":{\"partnerDevelopment\":" + strconv.FormatBool(development) +
				"}},\"currentAppInstallation\":{\"id\":\"" + installationID + "\",\"app\":{\"id\":\"" + appID +
				"\"},\"accessScopes\":[{\"handle\":\"write_products\"}]}}}"
		default:
			t.Fatalf("unexpected provider path %s", r.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(response))}, nil
	})}
	oauth, err := wire.NewOAuthClient(transport, "fixture-client", secret, "https://workbench.example/api/shopify/callback")
	require.NoError(t, err)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	issuer, err := wire.NewAppTokenIssuer(transport, "fixture-client", "fixture-key", key)
	require.NoError(t, err)
	protection, err := wire.NewPairingProtection("fixture-key", []byte(strings.Repeat("k", 32)))
	require.NoError(t, err)
	repo, err := storecenter.NewMemberScopedStoreRepository(newShopifyChargeStoreDatabase(t), serviceContextAccess{})
	require.NoError(t, err)
	app, err := NewShopifyConnections(repo, ShopifyApplication{AppID: appID, InstallURL: "https://apps.shopify.com/fixture",
		OAuth: oauth, Admin: wire.NewDevelopmentAdminClient(transport), Issuer: issuer, Protection: protection, StartedInstallation: true})
	require.NoError(t, err)

	public := context.Background()
	member := serviceContext("operator", "member-a", authz.EnterpriseRoleKey("org-a", 1))
	store, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: uuid.NewString(), OrganizationID: "org-a", ActorSubject: "operator",
		Name: "Fixture Store", Platform: "shopify", Region: "US", CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC()})
	require.NoError(t, err)
	store, _, err = repo.CreateOrReplay(member, "org-a", store)
	require.NoError(t, err)
	browser, state := strings.Repeat("b", 43), strings.Repeat("s", 43)
	signed := func(callback bool) string {
		params := url.Values{"shop": {domain}, "timestamp": {strconv.FormatInt(time.Now().Unix(), 10)}}
		if callback {
			params.Set("state", state)
			params.Set("code", "fixture-code")
		}
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(params.Encode()))
		params.Set("hmac", hex.EncodeToString(mac.Sum(nil)))
		return params.Encode()
	}

	attempt := uuid.NewString()
	_, err = app.BeginStartedInstallation(public, attempt, browser, state, "shop="+domain)
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	target, err := app.BeginStartedInstallation(public, attempt, browser, state, signed(false))
	require.NoError(t, err)
	require.Contains(t, target, "grant_options%5B%5D=per-user")
	_, err = app.StageStartedInstallation(public, attempt, strings.Repeat("x", 43), state, signed(true))
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	require.Zero(t, exchanges)
	staged, err := app.StageStartedInstallation(public, attempt, browser, state, signed(true))
	require.NoError(t, err)
	require.Equal(t, domain, staged)
	_, err = app.StageStartedInstallation(public, attempt, browser, state, signed(true))
	require.NoError(t, err)
	require.Equal(t, 1, exchanges)
	require.Equal(t, 1, reads)
	command := storecenter.OfficialConnectionCommand{OrganizationID: "org-a", StoreID: store.ID(), AttemptID: attempt, ExpectedStoreVersion: store.Version()}
	_, err = app.ConfirmStartedInstallation(public, command, browser)
	require.Error(t, err)
	require.Equal(t, 1, reads)
	installationID = "gid://shopify/AppInstallation/99"
	_, err = app.ConfirmStartedInstallation(member, command, browser)
	require.ErrorIs(t, err, storecenter.ErrVersionConflict)
	_, err = repo.ReadShopifyBinding(member, "org-a", store.ID())
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	installationID = primaryID
	bound, err := app.ConfirmStartedInstallation(member, command, browser)
	require.NoError(t, err)
	require.Equal(t, primaryID, bound.InstallationID)
	replayed, err := app.ConfirmStartedInstallation(member, command, browser)
	require.NoError(t, err)
	require.Equal(t, bound, replayed)
	require.Equal(t, 3, reads)

	// A successful Shopify token exchange alone must not enroll employees
	// or admit production stores into the development-only candidate.
	attempt = uuid.NewString()
	accountOwner = false
	_, err = app.BeginStartedInstallation(public, attempt, browser, state, signed(false))
	require.NoError(t, err)
	_, err = app.StageStartedInstallation(public, attempt, browser, state, signed(true))
	require.ErrorIs(t, err, storecenter.ErrShopifyOwnerRequired)
	require.Equal(t, 3, reads)
	attempt = uuid.NewString()
	development, accountOwner = false, true
	_, err = app.BeginStartedInstallation(public, attempt, browser, state, signed(false))
	require.NoError(t, err)
	_, err = app.StageStartedInstallation(public, attempt, browser, state, signed(true))
	require.ErrorIs(t, err, storecenter.ErrOfficialConnectionUnavailable)
	require.Equal(t, 4, reads)
}
