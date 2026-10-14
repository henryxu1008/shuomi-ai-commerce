package storecenterapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/storecenter"
	authorization "task-processor/internal/storecenter/tiktokshop"
)

type TikTokRulesProvider interface {
	QueryListingRules(context.Context, string, string) (marketplace.ListingRules, error)
	QueryCategories(context.Context, string, string, marketplace.ListingMethod) ([]marketplace.Category, error)
	QueryCategoryRules(context.Context, string, string, marketplace.RuleSelection) (marketplace.CategoryRules, error)
	QueryAttributes(context.Context, string, string, marketplace.RuleSelection) ([]marketplace.AttributeRule, error)
	QueryWarehouses(context.Context, string, string) ([]marketplace.Warehouse, error)
	QueryGlobalWarehouses(context.Context, string) ([]marketplace.GlobalWarehouse, error)
}

type TikTokProductStatusProvider interface {
	QueryProductStatus(context.Context, string, string, marketplace.ProductStatusExpectation) (marketplace.ProductStatus, error)
}

func (h *TikTokRulesHandle) ProductStatus(ctx context.Context, expected marketplace.ProductStatusExpectation) (marketplace.ProductStatus, error) {
	tokens, err := h.credential(ctx, "seller.product.basic")
	if err != nil {
		return marketplace.ProductStatus{}, err
	}
	provider, ok := h.owner.applications.tiktokProvider.(TikTokProductStatusProvider)
	if !ok {
		return marketplace.ProductStatus{}, ErrProductAccessChanged
	}
	value, err := provider.QueryProductStatus(ctx, tokens.AccessToken, h.material.Binding.Shop.Cipher, expected)
	if err != nil {
		return marketplace.ProductStatus{}, err
	}
	if _, err = h.credential(ctx, "seller.product.basic"); err != nil {
		return marketplace.ProductStatus{}, err
	}
	return value, nil
}

type TikTokRulesHandle struct {
	owner         *OfficialProductAccess
	subject       storecenter.ProductExecutionSubject
	application   authorization.Application
	material      storecenter.TikTokProductExecutionMaterial
	binding       MerchantBinding
	connectionRef string
	expiresAt     time.Time
}

func (a *OfficialProductAccess) AuthorizeTikTokRules(ctx context.Context, subject storecenter.ProductExecutionSubject, storeID string, expected *MerchantBinding) (*TikTokRulesHandle, error) {
	if subject.Purpose != storecenter.ProductPurposeRules {
		return nil, ErrProductAccessChanged
	}
	return a.authorizeTikTok(ctx, subject, storeID, expected)
}
func (a *OfficialProductAccess) authorizeTikTok(ctx context.Context, subject storecenter.ProductExecutionSubject, storeID string, expected *MerchantBinding) (*TikTokRulesHandle, error) {
	if a == nil || ctx == nil || ctx.Err() != nil || subject.Purpose != storecenter.ProductPurposeRules && subject.Purpose != storecenter.ProductPurposePublish && subject.Purpose != storecenter.ProductPurposeImage || !a.applications.TikTokConfigured() {
		return nil, ErrProductAccessChanged
	}
	started := a.now()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	material, err := a.reader.ReadProductExecution(ctx, subject, storeID, a.authorization, a.now())
	if err != nil || material.Platform != storecenter.PlatformTikTokShop || material.TikTok == nil {
		return nil, ErrProductAccessChanged
	}
	app := a.applications.tiktokProvider.Application()
	private := *material.TikTok
	grant, err := authorization.RestoreGrant(private.Grant)
	if err != nil || !grant.CanSend(a.now()) || private.Grant.OrganizationID != subject.OrganizationID || private.Grant.AppKey != app.AppKey || private.Grant.AppRevision != app.Revision || private.Binding.OrganizationID != subject.OrganizationID || private.Binding.StoreID != storeID || private.Binding.GrantID != private.Grant.ID || private.Binding.AppKey != app.AppKey || private.Binding.Status != "active" || !a.now().Before(material.ServiceExpiresAt) {
		return nil, ErrProductAccessChanged
	}
	tokens, err := a.applications.tiktokProtection.Open(grantCredentialBinding(private.Grant), private.Grant.KeyID, private.Grant.Ciphertext)
	if err != nil || sellerIdentityHash(tokens.OpenID) != private.Grant.SellerIdentityHash || !hasTikTokScopes(tokens, "seller.authorization.info", "seller.product.basic") {
		return nil, ErrProductAccessChanged
	}
	if _, ok := a.applications.tiktokProvider.(TikTokRulesProvider); !ok {
		return nil, ErrProductAccessChanged
	}
	shops, err := a.applications.tiktokProvider.AuthorizedShops(ctx, tokens.AccessToken)
	if err != nil {
		return nil, ErrProductAccessChanged
	}
	shop, err := authorization.SelectUSCrossBorderShop(shops, private.Binding.Shop.Code)
	if err != nil || shop.ID != private.Binding.Shop.ID {
		return nil, ErrProductAccessChanged
	}
	scopes := append([]string(nil), tokens.GrantedScopes...)
	sort.Strings(scopes)
	raw, _ := json.Marshal(scopes)
	hash := sha256.Sum256(raw)
	binding := MerchantBinding{Platform: storecenter.PlatformTikTokShop, OrganizationID: subject.OrganizationID, StoreID: storeID, Site: "tiktokshop-us", StoreVersion: material.StoreVersion, ConnectionRevision: material.Connection.Version, ApplicationRevision: app.Revision, ApplicationID: app.AppKey, SupplierIdentityHash: private.Grant.SellerIdentityHash, ServiceExpiresAt: material.ServiceExpiresAt, CredentialExpiresAt: tokens.AccessExpiresAt, ScopesHash: hex.EncodeToString(hash[:]), TikTok: storecenter.TikTokMerchantBinding{GrantID: private.Grant.ID, ShopID: shop.ID, GrantGeneration: private.Grant.Generation, GrantRevision: private.Grant.Revision, BindingRevision: private.Binding.Version}}
	if !binding.ValidTikTokApplication() || expected != nil && *expected != binding {
		return nil, ErrProductAccessChanged
	}
	fresh, err := a.reader.ReadProductExecution(ctx, subject, storeID, a.authorization, a.now())
	if err != nil || fresh.TikTok == nil || *fresh.TikTok != private || fresh.StoreVersion != material.StoreVersion || fresh.Connection.Version != material.Connection.Version || fresh.Connection.AttemptID != material.Connection.AttemptID {
		return nil, ErrProductAccessChanged
	}
	// Use the freshly verified Shop cipher. It never enters the public binding.
	private.Binding.Shop = shop
	expires := started.Add(10 * time.Second)
	return &TikTokRulesHandle{owner: a, subject: subject, application: app, material: private, binding: binding, connectionRef: material.Connection.AttemptID, expiresAt: expires}, nil
}

func hasTikTokScopes(tokens authorization.TokenSet, required ...string) bool {
	for _, scope := range required {
		found := false
		for _, granted := range tokens.GrantedScopes {
			if granted == scope {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (h *TikTokRulesHandle) Binding() MerchantBinding {
	if h == nil {
		return MerchantBinding{}
	}
	return h.binding
}
func (h *TikTokRulesHandle) credential(ctx context.Context, scopes ...string) (authorization.TokenSet, error) {
	if h == nil || h.owner == nil || ctx == nil || ctx.Err() != nil || !h.owner.now().Before(h.expiresAt) || h.owner.applications.tiktokProvider.Application() != h.application {
		return authorization.TokenSet{}, ErrProductAccessChanged
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	material, err := h.owner.reader.ReadProductExecution(ctx, h.subject, h.binding.StoreID, h.owner.authorization, h.owner.now())
	if err != nil || material.TikTok == nil || material.TikTok.Grant != h.material.Grant || material.TikTok.Binding.Version != h.material.Binding.Version || material.TikTok.Binding.Status != "active" || material.TikTok.Binding.Shop.ID != h.binding.TikTok.ShopID || material.Connection.AttemptID != h.connectionRef || material.Connection.Version != h.binding.ConnectionRevision || material.StoreVersion != h.binding.StoreVersion || !material.ServiceExpiresAt.Equal(h.binding.ServiceExpiresAt) {
		return authorization.TokenSet{}, ErrProductAccessChanged
	}
	tokens, err := h.owner.applications.tiktokProtection.Open(grantCredentialBinding(h.material.Grant), h.material.Grant.KeyID, h.material.Grant.Ciphertext)
	if err != nil || !hasTikTokScopes(tokens, scopes...) {
		return authorization.TokenSet{}, ErrProductAccessChanged
	}
	return tokens, nil
}

func (h *TikTokRulesHandle) ListingRules(ctx context.Context) (marketplace.ListingRules, error) {
	tokens, err := h.credential(ctx, "seller.product.basic")
	if err != nil {
		return marketplace.ListingRules{}, err
	}
	return h.owner.applications.tiktokProvider.(TikTokRulesProvider).QueryListingRules(ctx, tokens.AccessToken, h.material.Binding.Shop.Cipher)
}
func categoryScope(method marketplace.ListingMethod) string {
	if method == marketplace.GlobalPublishing {
		return "seller.global_product.category.info"
	}
	return "seller.product.basic"
}
func (h *TikTokRulesHandle) Categories(ctx context.Context, method marketplace.ListingMethod) ([]marketplace.Category, error) {
	if method != marketplace.GlobalPublishing && method != marketplace.LocalReplication {
		return nil, marketplace.ErrListingRules
	}
	tokens, err := h.credential(ctx, categoryScope(method))
	if err != nil {
		return nil, err
	}
	return h.owner.applications.tiktokProvider.(TikTokRulesProvider).QueryCategories(ctx, tokens.AccessToken, h.material.Binding.Shop.Cipher, method)
}
func (h *TikTokRulesHandle) CategoryRules(ctx context.Context, selection marketplace.RuleSelection) (marketplace.CategoryRules, error) {
	if !marketplace.ValidSelection(selection) {
		return marketplace.CategoryRules{}, marketplace.ErrListingRules
	}
	tokens, err := h.credential(ctx, categoryScope(selection.Method))
	if err != nil {
		return marketplace.CategoryRules{}, err
	}
	return h.owner.applications.tiktokProvider.(TikTokRulesProvider).QueryCategoryRules(ctx, tokens.AccessToken, h.material.Binding.Shop.Cipher, selection)
}
func (h *TikTokRulesHandle) Attributes(ctx context.Context, selection marketplace.RuleSelection) ([]marketplace.AttributeRule, error) {
	if !marketplace.ValidSelection(selection) {
		return nil, marketplace.ErrListingRules
	}
	tokens, err := h.credential(ctx, categoryScope(selection.Method))
	if err != nil {
		return nil, err
	}
	return h.owner.applications.tiktokProvider.(TikTokRulesProvider).QueryAttributes(ctx, tokens.AccessToken, h.material.Binding.Shop.Cipher, selection)
}
func (h *TikTokRulesHandle) Warehouses(ctx context.Context) ([]marketplace.Warehouse, error) {
	tokens, err := h.credential(ctx, "seller.logistics")
	if err != nil {
		return nil, err
	}
	return h.owner.applications.tiktokProvider.(TikTokRulesProvider).QueryWarehouses(ctx, tokens.AccessToken, h.material.Binding.Shop.Cipher)
}
func (h *TikTokRulesHandle) GlobalWarehouses(ctx context.Context) ([]marketplace.GlobalWarehouse, error) {
	tokens, err := h.credential(ctx, "seller.shop.info")
	if err != nil {
		return nil, err
	}
	return h.owner.applications.tiktokProvider.(TikTokRulesProvider).QueryGlobalWarehouses(ctx, tokens.AccessToken)
}
