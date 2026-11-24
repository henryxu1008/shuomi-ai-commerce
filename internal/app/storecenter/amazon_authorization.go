package storecenterapp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/amazon"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/credentialprotection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

const amazonAuthorizationLifetime = 9 * time.Minute

type AmazonAuthorizationProvider interface {
	ExchangeCode(context.Context, string) (amazon.Tokens, error)
	VerifyUSSeller(context.Context, string, string) error
}

type AmazonAuthorizations struct {
	owner       storecenter.AmazonAuthorizationStore
	provider    AmazonAuthorizationProvider
	protection  *credentialprotection.AESGCM
	application storecenter.AmazonAuthorizationApplication
	now         func() time.Time
}

func NewAmazonAuthorizations(owner storecenter.AmazonAuthorizationStore, provider AmazonAuthorizationProvider,
	protection *credentialprotection.AESGCM, app storecenter.AmazonAuthorizationApplication, now func() time.Time,
) (*AmazonAuthorizations, error) {
	if owner == nil || provider == nil || protection == nil || now == nil ||
		!boundedAmazonParameter(app.Revision, 128) || app.RedirectURI == "" {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	if _, err := amazon.ConsentURL(app.ID, "configuration-check", app.Beta); err != nil {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	return &AmazonAuthorizations{owner: owner, provider: provider, protection: protection, application: app, now: now}, nil
}

type AmazonConnectionStart struct {
	AuthorizationURL string
	FlowCookie       string `json:"-"`
	ExpiresAt        time.Time
}

type AmazonLoginCallback struct {
	FlowCookie, AmazonCallbackURI, AmazonState, SellerID, Version string
}

type AmazonRedirectCallback struct {
	FlowCookie, State, SellerID, Code string
}

type AmazonConnectionResult struct {
	StoreID    string
	Generation int64
}

func (a *AmazonAuthorizations) Begin(ctx context.Context, scope storecenter.AmazonBrowserScope, storeID string, version int64) (AmazonConnectionStart, error) {
	if !validAmazonScope(scope) || uuid.Validate(storeID) != nil || version <= 0 {
		return AmazonConnectionStart{}, storecenter.ErrNotFound
	}
	flow := amazonRandomParameter()
	consent, err := amazon.ConsentURL(a.application.ID, amazonRandomParameter(), a.application.Beta)
	if err != nil {
		return AmazonConnectionStart{}, storecenter.ErrOfficialConnectionUnavailable
	}
	now := a.now().UTC()
	start := storecenter.AmazonAuthorizationBegin{
		Scope: scope, Application: a.application, StoreID: storeID, AttemptID: uuid.NewString(),
		FlowDigest: amazonDigest(flow), ExpectedStoreVersion: version, Now: now, ExpiresAt: now.Add(amazonAuthorizationLifetime),
	}
	if err := a.owner.StartAmazonAuthorization(ctx, start); err != nil {
		return AmazonConnectionStart{}, err
	}
	return AmazonConnectionStart{AuthorizationURL: consent, FlowCookie: flow, ExpiresAt: start.ExpiresAt}, nil
}

func (a *AmazonAuthorizations) Continue(ctx context.Context, scope storecenter.AmazonBrowserScope, callback AmazonLoginCallback) (string, error) {
	if !validAmazonScope(scope) || !validAmazonRandomParameter(callback.FlowCookie) || !validAmazonSellerID(callback.SellerID) {
		return "", storecenter.ErrNotFound
	}
	expectedVersion := ""
	if a.application.Beta {
		expectedVersion = "beta"
	}
	if callback.Version != expectedVersion {
		return "", storecenter.ErrOfficialAuthorizationRejected
	}
	state := amazonRandomParameter()
	target, err := amazon.ContinueAuthorization(a.application.ID, a.application.RedirectURI,
		callback.AmazonCallbackURI, callback.AmazonState, state, a.application.Beta)
	if err != nil {
		return "", storecenter.ErrOfficialAuthorizationRejected
	}
	step := storecenter.AmazonAuthorizationStep{
		Scope: scope, Application: a.application, FlowDigest: amazonDigest(callback.FlowCookie),
		StateDigest: amazonDigest(state), SellerID: callback.SellerID, Now: a.now().UTC(),
	}
	if err := a.owner.AdvanceAmazonAuthorization(ctx, step); err != nil {
		return "", err
	}
	return target, nil
}

func (a *AmazonAuthorizations) Complete(ctx context.Context, scope storecenter.AmazonBrowserScope, callback AmazonRedirectCallback) (AmazonConnectionResult, error) {
	out := AmazonConnectionResult{}
	if !validAmazonScope(scope) || !validAmazonRandomParameter(callback.FlowCookie) ||
		!validAmazonRandomParameter(callback.State) || !validAmazonSellerID(callback.SellerID) ||
		!visibleAmazonParameter(callback.Code, 8192) {
		return out, storecenter.ErrNotFound
	}
	claim, err := a.owner.ClaimAmazonAuthorization(ctx, storecenter.AmazonAuthorizationClaimCommand{
		AmazonAuthorizationStep: storecenter.AmazonAuthorizationStep{
			Scope: scope, Application: a.application, FlowDigest: amazonDigest(callback.FlowCookie),
			StateDigest: amazonDigest(callback.State), SellerID: callback.SellerID, Now: a.now().UTC(),
		},
		CodeDigest: amazonDigest(callback.Code),
	})
	if err != nil {
		return out, err
	}
	if claim.Scope != scope || claim.Application != a.application || claim.SellerID != callback.SellerID ||
		uuid.Validate(claim.StoreID) != nil || uuid.Validate(claim.AttemptID) != nil ||
		claim.Generation < 1 || claim.ActorID == "" || claim.MemberID == "" {
		return out, storecenter.ErrOfficialExchangeUnknown
	}
	out = AmazonConnectionResult{StoreID: claim.StoreID, Generation: claim.Generation}
	if claim.Replay {
		return out, nil
	}
	if err := a.exchangeAndBind(ctx, claim, callback.Code); err != nil {
		return AmazonConnectionResult{}, err
	}
	return out, nil
}

func (a *AmazonAuthorizations) exchangeAndBind(ctx context.Context, claim storecenter.AmazonAuthorizationClaim, code string) error {
	tokens, err := a.provider.ExchangeCode(ctx, code)
	if err != nil {
		// The claimed state is consumed. Even an ambiguous exchange cannot
		// acquire another dispatch permission for this code.
		return storecenter.ErrOfficialExchangeUnknown
	}
	if err := a.provider.VerifyUSSeller(ctx, tokens.AccessToken, claim.SellerID); err != nil {
		if errors.Is(err, amazon.ErrSellerIneligible) || errors.Is(err, amazon.ErrUnauthorized) {
			return storecenter.ErrOfficialAuthorizationRejected
		}
		return storecenter.ErrOfficialExchangeUnknown
	}
	plain := []byte(tokens.RefreshToken)
	defer clear(plain)
	sealed, err := a.protection.Seal(plain, claim.CredentialAAD())
	if err != nil {
		return storecenter.ErrOfficialExchangeUnknown
	}
	// If the caller disconnected after exchange, still give the original
	// claim a bounded chance to persist under the current Store grants.
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	if err := a.owner.BindAmazonAuthorization(saveCtx, claim, a.protection.KeyID(), sealed, a.now().UTC()); err != nil {
		return storecenter.ErrOfficialExchangeUnknown
	}
	return nil
}

func amazonRandomParameter() string {
	var value [32]byte
	// Go's crypto/rand Read fills the buffer or terminates the process.
	_, _ = rand.Read(value[:])
	return base64.RawURLEncoding.EncodeToString(value[:])
}

func amazonDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func validAmazonRandomParameter(value string) bool {
	if len(value) != base64.RawURLEncoding.EncodedLen(32) {
		return false
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	return err == nil && len(decoded) == 32
}

func validAmazonSellerID(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	return strings.IndexFunc(value, func(char rune) bool {
		return !(char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' ||
			char >= '0' && char <= '9' || char == '-' || char == '_')
	}) == -1
}

func visibleAmazonParameter(value string, limit int) bool {
	return len(value) > 0 && len(value) <= limit && strings.IndexFunc(value, func(char rune) bool {
		return char < '!' || char > '~'
	}) == -1
}

func validAmazonScope(scope storecenter.AmazonBrowserScope) bool {
	return boundedAmazonParameter(scope.OrganizationID, storecenter.MaxOrganizationIDBytes) && scope.SessionDigest != [32]byte{}
}

func boundedAmazonParameter(value string, limit int) bool {
	return len(value) > 0 && len(value) <= limit && strings.TrimSpace(value) == value &&
		!strings.ContainsAny(value, "\r\n\x00")
}
