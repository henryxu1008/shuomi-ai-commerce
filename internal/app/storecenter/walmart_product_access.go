package storecenterapp

import (
	"context"
	"encoding/json"
	"sort"
	"sync/atomic"
	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/storecenter"
	auth "task-processor/internal/storecenter/walmart"
	"time"
)

type WalmartRulesProvider interface {
	Rules(context.Context, auth.TokenSet, marketplace.RuleSelection) (marketplace.RuleSnapshot, error)
	Taxonomy(context.Context, auth.TokenSet, string) (json.RawMessage, error)
	Search(context.Context, auth.TokenSet, string) (json.RawMessage, error)
	ProductBySKU(context.Context, auth.TokenSet, string) (*marketplace.Product, error)
	FeedPage(context.Context, auth.TokenSet, string, int) (marketplace.FeedObservation, error)
}
type WalmartMutationProvider interface {
	SubmitFeed(context.Context, auth.TokenSet, marketplace.Draft, string) (marketplace.AcceptedFeed, error)
}
type WalmartRulesHandle struct {
	owner         *OfficialProductAccess
	subject       storecenter.ProductExecutionSubject
	grant         storecenter.WalmartGrant
	binding       MerchantBinding
	connectionRef string
	expiresAt     time.Time
}

func (h *WalmartRulesHandle) Catalog(ctx context.Context, feedType string) (marketplace.RuleCatalog, error) {
	var out marketplace.RuleCatalog
	if feedType != marketplace.FullFeed && feedType != marketplace.MatchFeed {
		return out, marketplace.ErrInvalid
	}
	provider, ok := h.owner.applications.walmartProvider.(interface{ SpecVersion(string) string })
	if !ok {
		return out, marketplace.ErrUnavailable
	}
	version := provider.SpecVersion(feedType)
	if version == "" {
		return out, marketplace.ErrUnavailable
	}
	if feedType == marketplace.MatchFeed {
		return marketplace.RuleCatalog{FeedType: feedType, Version: version, ProductTypes: []string{}}, nil
	}
	raw, e := h.Taxonomy(ctx, feedType)
	if e != nil {
		return out, e
	}
	var body struct {
		Taxonomy []struct {
			Groups []struct {
				Types []struct {
					Name string `json:"productTypeName"`
				} `json:"productType"`
			} `json:"productTypeGroup"`
		} `json:"itemTaxonomy"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return out, marketplace.ErrUnavailable
	}
	seen := map[string]bool{}
	for _, c := range body.Taxonomy {
		for _, g := range c.Groups {
			for _, pt := range g.Types {
				if pt.Name != "" && len(pt.Name) <= 200 {
					seen[pt.Name] = true
				}
			}
		}
	}
	if len(seen) == 0 || len(seen) > 10000 {
		return out, marketplace.ErrUnavailable
	}
	out = marketplace.RuleCatalog{FeedType: feedType, Version: version, ProductTypes: []string{}}
	for name := range seen {
		out.ProductTypes = append(out.ProductTypes, name)
	}
	sort.Strings(out.ProductTypes)
	return out, nil
}

func (a *OfficialProductAccess) authorizeWalmart(ctx context.Context, s storecenter.ProductExecutionSubject, id string, expected *MerchantBinding) (*WalmartRulesHandle, error) {
	if a == nil || ctx == nil || ctx.Err() != nil || !a.applications.WalmartConfigured() || s.Purpose != storecenter.ProductPurposeRules && s.Purpose != storecenter.ProductPurposePublish {
		return nil, ErrProductAccessChanged
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	m, e := a.reader.ReadProductExecution(ctx, s, id, a.authorization, a.now())
	if e != nil || m.Walmart == nil || m.Platform != storecenter.PlatformWalmart {
		return nil, ErrProductAccessChanged
	}
	g := *m.Walmart
	app := a.applications.walmartProvider.Application()
	if g.AppID != app.ClientID || g.AppRevision != app.Revision || g.Environment != app.Environment || g.OrganizationID != s.OrganizationID || g.StoreID != id || !g.CanRead(a.now()) || !a.now().Before(m.ServiceExpiresAt) {
		return nil, ErrProductAccessChanged
	}
	if _, ok := a.applications.walmartProvider.(WalmartRulesProvider); !ok {
		return nil, ErrProductAccessChanged
	}
	tokens, e := a.applications.walmartProtection.Open(g.CredentialBinding(), g.KeyID, g.Ciphertext)
	if e != nil || tokens.SellerID != g.SellerID {
		return nil, ErrProductAccessChanged
	}
	detail, e := a.applications.walmartProvider.TokenDetail(ctx, tokens)
	if e != nil || !detail.Allows(s.Purpose, a.now()) || marketplace.Hash(detail.Scopes) != g.ScopesHash {
		return nil, ErrProductAccessChanged
	}
	b := MerchantBinding{Platform: storecenter.PlatformWalmart, Site: "walmart-us", OrganizationID: s.OrganizationID, StoreID: id, StoreVersion: m.StoreVersion, ConnectionRevision: m.Connection.Version, ApplicationRevision: g.AppRevision, ApplicationID: g.AppID, SupplierIdentityHash: g.MerchantIdentityHash, ServiceExpiresAt: m.ServiceExpiresAt, CredentialExpiresAt: g.AccessExpiresAt, ScopesHash: g.ScopesHash, Walmart: storecenter.WalmartMerchantBinding{Market: "US", Environment: g.Environment, GrantID: g.ID, Generation: g.Generation, GrantVersion: g.Version}}
	if !b.ValidWalmartApplication() || expected != nil && b != *expected {
		return nil, ErrProductAccessChanged
	}
	h := &WalmartRulesHandle{owner: a, subject: s, grant: g, binding: b, connectionRef: m.Connection.AttemptID, expiresAt: a.now().Add(10 * time.Second)}
	if _, e = h.credential(ctx, s.Purpose); e != nil {
		return nil, e
	}
	return h, nil
}
func (a *OfficialProductAccess) AuthorizeWalmartRules(ctx context.Context, s storecenter.ProductExecutionSubject, id string, expected *MerchantBinding) (*WalmartRulesHandle, error) {
	if s.Purpose != storecenter.ProductPurposeRules {
		return nil, ErrProductAccessChanged
	}
	return a.authorizeWalmart(ctx, s, id, expected)
}
func (h *WalmartRulesHandle) Binding() MerchantBinding {
	if h == nil {
		return MerchantBinding{}
	}
	return h.binding
}
func (h *WalmartRulesHandle) credential(ctx context.Context, purpose string) (auth.TokenSet, error) {
	var zero auth.TokenSet
	if h == nil || ctx == nil || ctx.Err() != nil || !h.owner.now().Before(h.expiresAt) || purpose == storecenter.ProductPurposePublish && h.subject.Purpose != purpose {
		return zero, ErrProductAccessChanged
	}
	app := h.owner.applications.walmartProvider.Application()
	if app.ClientID != h.grant.AppID || app.Revision != h.grant.AppRevision || app.Environment != h.grant.Environment {
		return zero, ErrProductAccessChanged
	}
	m, e := h.owner.reader.ReadProductExecution(ctx, h.subject, h.binding.StoreID, h.owner.authorization, h.owner.now())
	if e != nil || m.Walmart == nil || *m.Walmart != h.grant || m.Connection.AttemptID != h.connectionRef || m.Connection.Version != h.binding.ConnectionRevision || m.StoreVersion != h.binding.StoreVersion || !m.ServiceExpiresAt.Equal(h.binding.ServiceExpiresAt) {
		return zero, ErrProductAccessChanged
	}
	tokens, e := h.owner.applications.walmartProtection.Open(h.grant.CredentialBinding(), h.grant.KeyID, h.grant.Ciphertext)
	if e != nil {
		return zero, ErrProductAccessChanged
	}
	detail, e := h.owner.applications.walmartProvider.TokenDetail(ctx, tokens)
	if e != nil || !detail.Allows(purpose, h.owner.now()) || marketplace.Hash(detail.Scopes) != h.grant.ScopesHash {
		return zero, ErrProductAccessChanged
	}
	return tokens, nil
}
func (h *WalmartRulesHandle) Rules(ctx context.Context, s marketplace.RuleSelection) (marketplace.RuleSnapshot, error) {
	t, e := h.credential(ctx, storecenter.ProductPurposeRules)
	if e != nil {
		return marketplace.RuleSnapshot{}, e
	}
	out, e := h.owner.applications.walmartProvider.(WalmartRulesProvider).Rules(ctx, t, s)
	if e != nil {
		return out, e
	}
	if _, e = h.credential(ctx, storecenter.ProductPurposeRules); e != nil {
		return marketplace.RuleSnapshot{}, e
	}
	return out, nil
}
func (h *WalmartRulesHandle) Taxonomy(ctx context.Context, feed string) (json.RawMessage, error) {
	t, e := h.credential(ctx, storecenter.ProductPurposeRules)
	if e != nil {
		return nil, e
	}
	out, e := h.owner.applications.walmartProvider.(WalmartRulesProvider).Taxonomy(ctx, t, feed)
	if e != nil {
		return nil, e
	}
	_, e = h.credential(ctx, storecenter.ProductPurposeRules)
	return out, e
}
func (h *WalmartRulesHandle) Search(ctx context.Context, identifier string) (json.RawMessage, error) {
	t, e := h.credential(ctx, storecenter.ProductPurposeRules)
	if e != nil {
		return nil, e
	}
	out, e := h.owner.applications.walmartProvider.(WalmartRulesProvider).Search(ctx, t, identifier)
	if e != nil {
		return nil, e
	}
	_, e = h.credential(ctx, storecenter.ProductPurposeRules)
	return out, e
}
func (h *WalmartRulesHandle) ProductBySKU(ctx context.Context, sku string) (*marketplace.Product, error) {
	t, e := h.credential(ctx, storecenter.ProductPurposeRules)
	if e != nil {
		return nil, e
	}
	out, e := h.owner.applications.walmartProvider.(WalmartRulesProvider).ProductBySKU(ctx, t, sku)
	if e != nil {
		return nil, e
	}
	_, e = h.credential(ctx, storecenter.ProductPurposeRules)
	return out, e
}
func (h *WalmartRulesHandle) FeedPage(ctx context.Context, feed string, offset int) (marketplace.FeedObservation, error) {
	t, e := h.credential(ctx, "feed_result")
	if e != nil {
		return marketplace.FeedObservation{}, e
	}
	out, e := h.owner.applications.walmartProvider.(WalmartRulesProvider).FeedPage(ctx, t, feed, offset)
	if e != nil {
		return out, e
	}
	_, e = h.credential(ctx, "feed_result")
	return out, e
}

type WalmartMutationHandle struct {
	authority *WalmartRulesHandle
	sent      atomic.Bool
}

func (a *OfficialProductAccess) AuthorizeWalmartMutation(ctx context.Context, s storecenter.ProductExecutionSubject, id string, expected *MerchantBinding) (*WalmartMutationHandle, error) {
	if s.Purpose != storecenter.ProductPurposePublish {
		return nil, ErrProductAccessChanged
	}
	h, e := a.authorizeWalmart(ctx, s, id, expected)
	if e != nil {
		return nil, e
	}
	if _, ok := a.applications.walmartProvider.(WalmartMutationProvider); !ok {
		return nil, ErrProductAccessChanged
	}
	return &WalmartMutationHandle{authority: h}, nil
}
func (h *WalmartMutationHandle) Binding() MerchantBinding {
	if h == nil {
		return MerchantBinding{}
	}
	return h.authority.Binding()
}
func (h *WalmartMutationHandle) SubmitFeed(ctx context.Context, d marketplace.Draft, correlation string) (marketplace.AcceptedFeed, error) {
	var zero marketplace.AcceptedFeed
	if h == nil || d.ValidateWire() != nil {
		return zero, ErrProductAccessChanged
	}
	tokens, e := h.authority.credential(ctx, storecenter.ProductPurposePublish)
	if e != nil || !h.sent.CompareAndSwap(false, true) {
		return zero, ErrProductAccessChanged
	}
	return h.authority.owner.applications.walmartProvider.(WalmartMutationProvider).SubmitFeed(ctx, tokens, d, correlation)
}
