package storecenterapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"testing"
	"time"

	amazon "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/amazon"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/credentialprotection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
)

const amazonFixtureApp = "amzn1.sellerapps.app.2eca283f-9f5a-4d13-b16c-474000000057"
const amazonFixtureStore = "f64667a1-c09d-45ad-b8c5-fa4a1024fa7a"

func TestAmazonAuthorizationBindsSessionAndClaimsExchangeOnce(t *testing.T) {
	ctx := context.Background()
	app, repo, provider := newAmazonAuthorizationFixture(t)
	scope := storecenter.AmazonBrowserScope{OrganizationID: "org-a", SessionDigest: sha256.Sum256([]byte("browser-session-a"))}
	begin, err := app.Begin(ctx, scope, amazonFixtureStore, 7)
	require.NoError(t, err)
	require.NotEmpty(t, begin.FlowCookie)
	require.NotEqual(t, begin.FlowCookie, repo.begin.FlowDigest)
	flowHash := sha256.Sum256([]byte(begin.FlowCookie))
	require.Equal(t, hex.EncodeToString(flowHash[:]), repo.begin.FlowDigest)

	callback := storecenter.AmazonLoginCallback{
		FlowCookie: begin.FlowCookie, AmazonState: "amazon-state", SellerID: "SELLERA",
		AmazonCallbackURI: "https://amazon.com/apps/authorize/confirm/" + amazonFixtureApp,
	}
	otherSession := storecenter.AmazonBrowserScope{OrganizationID: "org-a", SessionDigest: sha256.Sum256([]byte("browser-session-b"))}
	_, err = app.Continue(ctx, otherSession, callback)
	require.ErrorIs(t, err, storecenter.ErrNotFound)

	callback.AmazonCallbackURI = "https://example.org/apps/authorize/confirm/" + amazonFixtureApp
	_, err = app.Continue(ctx, scope, callback)
	require.ErrorIs(t, err, storecenter.ErrOfficialAuthorizationRejected)
	require.Zero(t, repo.advanceCalls)

	callback.AmazonCallbackURI = "https://amazon.com/apps/authorize/confirm/" + amazonFixtureApp
	next, err := app.Continue(ctx, scope, callback)
	require.NoError(t, err)
	confirmation, err := url.Parse(next)
	require.NoError(t, err)
	final := storecenter.AmazonRedirectCallback{
		FlowCookie: begin.FlowCookie, SellerID: "SELLERB", State: confirmation.Query().Get("state"), Code: "single-use-code",
	}
	_, err = app.Complete(ctx, scope, final)
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	require.Zero(t, provider.exchanges)

	final.SellerID = "SELLERA"
	connection, err := app.Complete(ctx, scope, final)
	require.NoError(t, err)
	require.Equal(t, amazonFixtureStore, connection.StoreID)
	require.Equal(t, 1, provider.exchanges)
	require.Equal(t, 1, provider.verifications)
	require.NotContains(t, repo.sealed, "fixture-refresh-token")

	plain, err := app.protection.Open("fixture-key", repo.sealed, repo.aad)
	require.NoError(t, err)
	require.Equal(t, "fixture-refresh-token", string(plain))
	_, err = app.protection.Open("fixture-key", repo.sealed, []byte("other context"))
	require.Error(t, err)

	replayed, err := app.Complete(ctx, scope, final)
	require.NoError(t, err)
	require.Equal(t, connection, replayed)
	final.Code = "different-code"
	_, err = app.Complete(ctx, scope, final)
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	require.Equal(t, 1, provider.exchanges)
}

func TestAmazonAuthorizationUnknownExchangeRequiresFreshConsent(t *testing.T) {
	ctx := context.Background()
	app, repo, provider := newAmazonAuthorizationFixture(t)
	scope := storecenter.AmazonBrowserScope{OrganizationID: "org-a", SessionDigest: sha256.Sum256([]byte("session"))}
	start, err := app.Begin(ctx, scope, amazonFixtureStore, 7)
	require.NoError(t, err)
	confirm, err := app.Continue(ctx, scope, storecenter.AmazonLoginCallback{
		FlowCookie: start.FlowCookie, AmazonState: "amazon-state", SellerID: "SELLERA",
		AmazonCallbackURI: "https://amazon.com/apps/authorize/confirm/" + amazonFixtureApp,
	})
	require.NoError(t, err)
	redirect, err := url.Parse(confirm)
	require.NoError(t, err)
	callback := storecenter.AmazonRedirectCallback{FlowCookie: start.FlowCookie, SellerID: "SELLERA", State: redirect.Query().Get("state"), Code: "used-code"}
	provider.exchangeError = errors.New("private provider response")
	_, err = app.Complete(ctx, scope, callback)
	require.ErrorIs(t, err, storecenter.ErrOfficialExchangeUnknown)
	require.Empty(t, repo.sealed)
	_, err = app.Complete(ctx, scope, callback)
	require.ErrorIs(t, err, storecenter.ErrOfficialExchangeUnknown)
	require.Equal(t, 1, provider.exchanges)
	require.Zero(t, provider.verifications)
}

type amazonAuthorizationFixtureRepo struct {
	begin        storecenter.AmazonAuthorizationBegin
	advance      storecenter.AmazonAuthorizationStep
	advanceCalls int
	claimed      bool
	codeDigest   string
	completed    bool
	sealed       string
	aad          []byte
}

func (r *amazonAuthorizationFixtureRepo) StartAmazonAuthorization(_ context.Context, begin storecenter.AmazonAuthorizationBegin) error {
	r.begin = begin
	return nil
}
func (r *amazonAuthorizationFixtureRepo) AdvanceAmazonAuthorization(_ context.Context, step storecenter.AmazonAuthorizationStep) error {
	if step.Scope != r.begin.Scope || step.FlowDigest != r.begin.FlowDigest || step.Application != r.begin.Application {
		return storecenter.ErrNotFound
	}
	r.advanceCalls++
	r.advance = step
	return nil
}
func (r *amazonAuthorizationFixtureRepo) ClaimAmazonAuthorization(_ context.Context, step storecenter.AmazonAuthorizationClaimCommand) (storecenter.AmazonAuthorizationClaim, error) {
	if step.Scope != r.advance.Scope || step.FlowDigest != r.advance.FlowDigest || step.StateDigest != r.advance.StateDigest ||
		step.SellerID != r.advance.SellerID || step.Application != r.advance.Application {
		return storecenter.AmazonAuthorizationClaim{}, storecenter.ErrNotFound
	}
	claim := storecenter.AmazonAuthorizationClaim{
		AttemptID: r.begin.AttemptID, StoreID: r.begin.StoreID, ActorID: "actor", MemberID: "member",
		SellerID: step.SellerID, Scope: step.Scope, Application: step.Application, Generation: 1,
	}
	if r.claimed && step.CodeDigest != r.codeDigest {
		return storecenter.AmazonAuthorizationClaim{}, storecenter.ErrNotFound
	}
	if r.completed {
		claim.Replay = true
		return claim, nil
	}
	if r.claimed {
		return storecenter.AmazonAuthorizationClaim{}, storecenter.ErrOfficialExchangeUnknown
	}
	r.claimed = true
	r.codeDigest = step.CodeDigest
	return claim, nil
}
func (r *amazonAuthorizationFixtureRepo) BindAmazonAuthorization(_ context.Context, claim storecenter.AmazonAuthorizationClaim, keyID, sealed string, _ time.Time) error {
	r.sealed, r.aad = sealed, claim.CredentialAAD()
	r.completed = true
	return nil
}

type amazonAuthorizationFixtureProvider struct {
	exchanges, verifications   int
	exchangeError, verifyError error
}

func (p *amazonAuthorizationFixtureProvider) ExchangeCode(context.Context, string) (amazon.Tokens, error) {
	p.exchanges++
	return amazon.Tokens{AccessToken: "fixture-access", RefreshToken: "fixture-refresh-token"}, p.exchangeError
}
func (p *amazonAuthorizationFixtureProvider) VerifyUSSeller(_ context.Context, accessToken, sellerID string) error {
	p.verifications++
	if p.verifyError != nil {
		return p.verifyError
	}
	if accessToken != "fixture-access" || sellerID != "SELLERA" {
		return amazon.ErrUnauthorized
	}
	return nil
}

func newAmazonAuthorizationFixture(t *testing.T) (*AmazonAuthorizations, *amazonAuthorizationFixtureRepo, *amazonAuthorizationFixtureProvider) {
	t.Helper()
	repo := &amazonAuthorizationFixtureRepo{}
	provider := &amazonAuthorizationFixtureProvider{}
	protection, err := credentialprotection.NewAESGCM("fixture-key", make([]byte, 32))
	require.NoError(t, err)
	app, err := NewAmazonAuthorizations(repo, provider, protection, storecenter.AmazonAuthorizationApplication{
		ID: amazonFixtureApp, Revision: "r1", RedirectURI: "https://workbench.example.org/amazon/redirect",
	}, time.Now)
	require.NoError(t, err)
	return app, repo, provider
}

func TestAmazonAuthorizationDoesNotBindWithoutSellerProof(t *testing.T) {
	ctx := context.Background()
	app, repo, provider := newAmazonAuthorizationFixture(t)
	scope := storecenter.AmazonBrowserScope{OrganizationID: "org-a", SessionDigest: sha256.Sum256([]byte("session-a"))}
	begin, err := app.Begin(ctx, scope, amazonFixtureStore, 7)
	require.NoError(t, err)
	wire, err := json.Marshal(begin)
	require.NoError(t, err)
	require.NotContains(t, string(wire), begin.FlowCookie)
	redirect, err := app.Continue(ctx, scope, storecenter.AmazonLoginCallback{
		FlowCookie: begin.FlowCookie, AmazonState: "amazon-state", SellerID: "SELLERB",
		AmazonCallbackURI: "https://amazon.com/apps/authorize/confirm/" + amazonFixtureApp,
	})
	require.NoError(t, err)
	target, err := url.Parse(redirect)
	require.NoError(t, err)
	callback := storecenter.AmazonRedirectCallback{FlowCookie: begin.FlowCookie, SellerID: "SELLERB", State: target.Query().Get("state"), Code: "one-time-code"}
	otherOrganization := storecenter.AmazonBrowserScope{OrganizationID: "org-b", SessionDigest: scope.SessionDigest}
	_, err = app.Complete(ctx, otherOrganization, callback)
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	require.Zero(t, provider.exchanges)
	_, err = app.Complete(ctx, scope, callback)
	require.ErrorIs(t, err, storecenter.ErrOfficialAuthorizationRejected)
	require.Equal(t, 1, provider.verifications)
	require.Empty(t, repo.sealed)
	require.False(t, repo.completed)
}

func TestAmazonAuthorizationRejectsBadCodeBeforeClaim(t *testing.T) {
	ctx := context.Background()
	app, repo, provider := newAmazonAuthorizationFixture(t)
	scope := storecenter.AmazonBrowserScope{OrganizationID: "org-a", SessionDigest: sha256.Sum256([]byte("session-a"))}
	begin, err := app.Begin(ctx, scope, amazonFixtureStore, 7)
	require.NoError(t, err)
	redirect, err := app.Continue(ctx, scope, storecenter.AmazonLoginCallback{
		FlowCookie: begin.FlowCookie, AmazonState: "amazon-state", SellerID: "SELLERA",
		AmazonCallbackURI: "https://amazon.com/apps/authorize/confirm/" + amazonFixtureApp,
	})
	require.NoError(t, err)
	target, err := url.Parse(redirect)
	require.NoError(t, err)
	_, err = app.Complete(ctx, scope, storecenter.AmazonRedirectCallback{
		FlowCookie: begin.FlowCookie, SellerID: "SELLERA", State: target.Query().Get("state"), Code: "invalid code",
	})
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	require.False(t, repo.claimed)
	require.Zero(t, provider.exchanges)
}

func TestAmazonAuthorizationTreatsSellerProbeOutageAsUnknown(t *testing.T) {
	ctx := context.Background()
	app, repo, provider := newAmazonAuthorizationFixture(t)
	scope := storecenter.AmazonBrowserScope{OrganizationID: "org-a", SessionDigest: sha256.Sum256([]byte("session-a"))}
	begin, err := app.Begin(ctx, scope, amazonFixtureStore, 7)
	require.NoError(t, err)
	redirect, err := app.Continue(ctx, scope, storecenter.AmazonLoginCallback{
		FlowCookie: begin.FlowCookie, AmazonState: "amazon-state", SellerID: "SELLERA",
		AmazonCallbackURI: "https://amazon.com/apps/authorize/confirm/" + amazonFixtureApp,
	})
	require.NoError(t, err)
	target, err := url.Parse(redirect)
	require.NoError(t, err)
	provider.verifyError = amazon.ErrUnavailable
	callback := storecenter.AmazonRedirectCallback{FlowCookie: begin.FlowCookie, SellerID: "SELLERA", State: target.Query().Get("state"), Code: "one-time-code"}
	_, err = app.Complete(ctx, scope, callback)
	require.ErrorIs(t, err, storecenter.ErrOfficialExchangeUnknown)
	_, err = app.Complete(ctx, scope, callback)
	require.ErrorIs(t, err, storecenter.ErrOfficialExchangeUnknown)
	require.Equal(t, 1, provider.exchanges)
	require.Equal(t, 1, provider.verifications)
	require.Empty(t, repo.sealed)
}
