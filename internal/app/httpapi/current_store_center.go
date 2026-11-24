package httpapi

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	storeapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storecenter"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	zitadelruntime "github.com/henryxu1008/shuomi-ai-commerce/internal/authruntime/zitadel"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	kernelmodule "github.com/henryxu1008/shuomi-ai-commerce/internal/kernel/module"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/ledger/orgresource"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	storehttp "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/httpapi"
)

// WithStoreCenter borrows the separately owned native record pool. The runtime
// retains responsibility for closing them; this builder never opens cfg.Database.
func WithStoreCenter(records *gorm.DB) CurrentApplicationOption {
	return func(o *currentApplicationOptions) {
		o.storeCenters++
		o.storeCenterDB = records
	}
}

func WithStoreOfficialApplications(applications *storeapp.OfficialApplicationRegistry) CurrentApplicationOption {
	return func(o *currentApplicationOptions) {
		o.officialStoreConfigs++
		o.officialStoreApplications = applications
	}
}

// WithIsolatedAmazonStoreAuthorization supplies a Store authorization service
// for isolated application assembly. It has no runtime manifest or deployed
// service configuration until Amazon rollout admission is implemented.
func WithIsolatedAmazonStoreAuthorization(service storehttp.AmazonAuthorizationService) CurrentApplicationOption {
	return func(o *currentApplicationOptions) {
		o.amazonStoreAuthorizationCount++
		o.amazonStoreAuthorization = service
	}
}

func currentAmazonSessionProof(ctx context.Context) [32]byte {
	token := zitadelruntime.BearerTokenFromContext(ctx)
	if token == "" {
		return [32]byte{}
	}
	// Only the already verified Workbench bearer is available here. The
	// digest is scoped to this flow, and credential rotation requires new
	// consent rather than allowing the old callback to cross sessions.
	return sha256.Sum256([]byte("storecenter/amazon/workbench-session/v1\x00" + token))
}

func buildCurrentStoreCenterModule(ctx context.Context, records *gorm.DB, authorizer *authz.ListingKitAuthorizer, charges orgresource.ConsumerChargePort, applications *storeapp.OfficialApplicationRegistry, capabilities storecenter.RuntimeCapabilities) (kernelmodule.Module, error) {
	if records == nil {
		return nil, errors.New("store center requires its native record pool")
	}
	if err := storecenter.VerifyCurrentSchemaForCapabilities(ctx, records, capabilities); err != nil {
		return nil, fmt.Errorf("verify store center schema: %w", err)
	}
	if err := storecenter.VerifyRuntimePermissionsForCapabilities(ctx, records, capabilities); err != nil {
		return nil, err
	}
	if authorizer == nil {
		return nil, errors.New("store center authorizer unavailable")
	}
	repo, err := storecenter.NewMemberScopedStoreRepository(records, currentStoreMemberAuthorizer{authorizer: authorizer})
	if err != nil {
		return nil, err
	}
	repo = repo.WithShopifyLifecycle(capabilities.ShopifyLifecycle)
	audit, err := storecenter.NewGormAuditRepository(records)
	if err != nil {
		return nil, err
	}
	var connections *storeapp.OfficialConnections
	if applications == nil {
		connections, err = storeapp.NewUnconfiguredOfficialConnections(repo)
	} else {
		connections, err = storeapp.NewOfficialConnections(repo, applications)
	}
	if err != nil {
		return nil, err
	}
	service, err := storecenter.NewService(repo, audit, connections, time.Now)
	if err != nil {
		return nil, err
	}
	executor, err := storeapp.NewServiceLifecycleExecutor(repo, charges, connections)
	if err != nil {
		return nil, err
	}
	lifecycle, err := storecenter.NewServiceLifecycleApplication(repo, executor, connections, authorizer, currentStorePeriodPolicy{}, time.Now)
	if err != nil {
		return nil, err
	}
	handler, err := storehttp.NewHandlerWithServiceLifecycle(service, lifecycle)
	if err != nil {
		return nil, err
	}
	if err := handler.SetOfficialConnections(connections); err != nil {
		return nil, err
	}
	shopify, err := applications.ShopifyConnections(repo)
	if err != nil {
		return nil, err
	}
	if shopify != nil {
		if err := handler.SetShopifyConnections(shopify); err != nil {
			return nil, err
		}
	}
	google, err := applications.GoogleConnections(repo)
	if err != nil {
		return nil, err
	}
	if google != nil {
		if err := handler.SetGoogleConnections(google); err != nil {
			return nil, err
		}
	}
	module := storehttp.NewModule(handler)
	catalog, metadata, err := buildStoreCatalog(records, authorizer, applications, nil, nil, nil, capabilities)
	if err != nil {
		return nil, err
	}
	module.(interface {
		ConfigureCatalog(*storecenter.Catalog, *storecenter.MetadataRepository)
	}).ConfigureCatalog(catalog, metadata)
	return module, nil
}

type currentStorePeriodPolicy = storecenter.Phase1ServiceQuantityPolicy

// Current Store descriptors resolve live Organization access before invoking
// this capability. No member ID or administrator flag comes from the request.
type currentStoreMemberAuthorizer struct{ authorizer *authz.ListingKitAuthorizer }

func (a currentStoreMemberAuthorizer) AuthorizeStoreMember(ctx context.Context, organizationID string) (storecenter.StoreMemberAccess, error) {
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.EffectiveOrganizationID != organizationID || identity.TenantID != organizationID || !authidentity.IsBoundedIdentifier(identity.EffectiveMemberID) || !authz.AllowedOrganization(ctx, a.authorizer, identity.UserID, organizationID, identity.Roles, authz.PermissionWorkbenchStoreRead) {
		return storecenter.StoreMemberAccess{}, storecenter.ErrNotFound
	}
	granted := false
	for _, grant := range identity.OrganizationGrants {
		if grant.OrganizationID == organizationID && grant.AuthorizationID == identity.EffectiveMemberID {
			granted = true
			break
		}
	}
	if !granted {
		return storecenter.StoreMemberAccess{}, storecenter.ErrNotFound
	}
	return storecenter.StoreMemberAccess{OrganizationID: organizationID, ActorID: identity.UserID, MemberID: identity.EffectiveMemberID, Administrator: a.authorizer.IsTenantAdmin(identity.UserID, identity.Roles), CanWrite: authz.AllowedOrganization(ctx, a.authorizer, identity.UserID, organizationID, identity.Roles, authz.PermissionWorkbenchStoreUpdate)}, nil
}

var currentStoreCenterRoutes = []struct{ method, path, permission string }{
	{http.MethodGet, "/api/v1/workbench/stores/:store_id/summary", authz.PermissionWorkbenchStoreRead},
	{http.MethodGet, "/api/v1/workbench/store-catalog", authz.PermissionWorkbenchStoreRead},
	{http.MethodGet, "/api/v1/workbench/store-groups", authz.PermissionWorkbenchStoreRead},
	{http.MethodGet, "/api/v1/workbench/stores/:store_id/profile", authz.PermissionWorkbenchStoreRead},
	{http.MethodPost, "/api/v1/workbench/store-metadata", authz.PermissionWorkbenchStoreUpdate},
	{http.MethodGet, "/api/v1/workbench/store-metadata/:operation_id", authz.PermissionWorkbenchStoreUpdate},
	{http.MethodGet, "/api/v1/workbench/stores", authz.PermissionWorkbenchStoreRead},
	{http.MethodPost, "/api/v1/workbench/stores", authz.PermissionWorkbenchStoreCreate},
	{http.MethodGet, "/api/v1/workbench/stores/:store_id", authz.PermissionWorkbenchStoreRead},
	{http.MethodPut, "/api/v1/workbench/stores/:store_id", authz.PermissionWorkbenchStoreUpdate},
	{http.MethodPost, "/api/v1/workbench/stores/:store_id/disable", authz.PermissionWorkbenchStoreLifecycle},
	{http.MethodPost, "/api/v1/workbench/stores/:store_id/enable", authz.PermissionWorkbenchStoreLifecycle},
	{http.MethodDelete, "/api/v1/workbench/stores/:store_id", authz.PermissionWorkbenchStoreDelete},
	{http.MethodPost, "/api/v1/workbench/stores/:store_id/activate", authz.PermissionWorkbenchStoreLifecycle},
	{http.MethodPost, "/api/v1/workbench/stores/:store_id/renew", authz.PermissionWorkbenchStoreLifecycle},
	{http.MethodPost, "/api/v1/workbench/stores/:store_id/reactivate", authz.PermissionWorkbenchStoreLifecycle},
	{http.MethodGet, "/api/v1/workbench/stores/:store_id/connection", authz.PermissionWorkbenchStoreRead},
	{http.MethodGet, "/api/v1/workbench/stores/:store_id/connection/applications", authz.PermissionWorkbenchStoreRead},
	{http.MethodPost, "/api/v1/workbench/stores/:store_id/connection/begin", authz.PermissionWorkbenchStoreUpdate},
	{http.MethodPost, "/api/v1/workbench/stores/:store_id/connection/complete", authz.PermissionWorkbenchStoreUpdate},
	{http.MethodPost, "/api/v1/workbench/stores/:store_id/connection/token", authz.PermissionWorkbenchStoreUpdate},
	{http.MethodPost, "/api/v1/workbench/stores/:store_id/connection/query", authz.PermissionWorkbenchStoreUpdate},
	{http.MethodPost, "/api/v1/workbench/stores/:store_id/connection/refresh", authz.PermissionWorkbenchStoreUpdate},
	{http.MethodPost, "/api/v1/workbench/stores/:store_id/connection/disconnect", authz.PermissionWorkbenchStoreUpdate},
}

var currentStoreBrowserRoutes = []struct {
	method, path, permission string
	policy                   httproute.OrganizationAccessPolicy
}{
	{http.MethodPost, "/api/v1/workbench/stores/:store_id/browser-operations", authz.PermissionWorkbenchStoreBrowserUse, httproute.OrganizationAccessPolicyLiveWrite},
	{http.MethodGet, "/api/v1/workbench/stores/:store_id/browser-operations/:operation_id", "", httproute.OrganizationAccessPolicyNone},
	{http.MethodPost, "/api/v1/workbench/stores/:store_id/browser-operations/:operation_id/reports", "", httproute.OrganizationAccessPolicyNone},
	{http.MethodGet, "/api/v1/workbench/stores/:store_id/audit", authz.PermissionWorkbenchStoreRead, httproute.OrganizationAccessPolicyLiveWrite},
}

func (a currentStoreMemberAuthorizer) AuthorizeStoreBrowser(ctx context.Context, org string) error {
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.EffectiveOrganizationID != org || !authz.AllowedOrganization(ctx, a.authorizer, identity.UserID, org, identity.Roles, authz.PermissionWorkbenchStoreBrowserUse) {
		return storecenter.ErrBrowserDenied
	}
	_, err := a.AuthorizeStoreMember(ctx, org)
	return err
}

func (a currentStoreMemberAuthorizer) ReceiptSubject(ctx context.Context) (string, error) {
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || !authidentity.IsBoundedIdentifier(identity.UserID) || identity.TokenExpiresAt.IsZero() || !time.Now().Before(identity.TokenExpiresAt) {
		return "", storecenter.ErrNotFound
	}
	return identity.UserID, nil
}

func validateCurrentStoreDescriptor(d httproute.Descriptor) error {
	if d.Path == storehttp.ShopifyPairingPath {
		if d.Module == storehttp.ModuleName && d.Method == http.MethodPost &&
			d.AuthPolicy == httproute.AuthPolicyCurrentIdentity && d.OrganizationAccessPolicy == httproute.OrganizationAccessPolicyLiveWrite &&
			d.Permission == authz.PermissionWorkbenchStoreUpdate && d.OrganizationTargetResolver == nil &&
			d.RequestTimeout == storehttp.ShopifyPairingTimeout && !d.RejectUnreadRequestBody && d.Handler != nil {
			return nil
		}
		return errors.New("Shopify Store pairing route loses live write authority or request bounds")
	}
	if storehttp.IsGoogleStoreDescriptor(d.Method, d.Path, d.Permission) && d.Module == storehttp.ModuleName &&
		d.AuthPolicy == httproute.AuthPolicyCurrentIdentity && d.OrganizationAccessPolicy == httproute.OrganizationAccessPolicyLiveWrite &&
		d.OrganizationTargetResolver == nil && d.Handler != nil {
		return nil
	}
	if storehttp.IsAmazonAuthorizationPath(d.Path) {
		method, permission, timeout := http.MethodPost, authz.PermissionWorkbenchStoreUpdate, 35*time.Second
		if d.Path == storehttp.AmazonAuthorizationStatusPath {
			method, permission, timeout = http.MethodGet, authz.PermissionWorkbenchStoreRead, 5*time.Second
		}
		if d.Module == storehttp.ModuleName && d.Method == method &&
			d.AuthPolicy == httproute.AuthPolicyCurrentIdentity && d.OrganizationAccessPolicy == httproute.OrganizationAccessPolicyLiveWrite &&
			d.Permission == permission && d.OrganizationTargetResolver == nil &&
			d.RequestTimeout == timeout && d.RejectUnreadRequestBody && d.Handler != nil {
			return nil
		}
		return errors.New("Amazon Store route loses callback session or live write authority")
	}
	for _, expected := range currentStoreBrowserRoutes {
		if d.Method == expected.method && d.Path == expected.path && d.Permission == expected.permission && d.AuthPolicy == httproute.AuthPolicyCurrentIdentity && d.OrganizationAccessPolicy == expected.policy && d.OrganizationTargetResolver == nil && d.Module == storehttp.ModuleName && d.Handler != nil && d.RequestTimeout == 15*time.Second && d.RejectUnreadRequestBody == (d.Method == http.MethodGet) {
			return nil
		}
	}
	for _, expected := range currentStoreCenterRoutes {
		if expected.method == d.Method && expected.path == d.Path && d.Module == storehttp.ModuleName && d.AuthPolicy == httproute.AuthPolicyCurrentIdentity && d.OrganizationAccessPolicy == httproute.OrganizationAccessPolicyLiveWrite && d.OrganizationTargetResolver == nil && d.Permission == expected.permission && d.Handler != nil {
			return nil
		}
	}
	return errors.New("store center route loses current identity, live permission or CRUD scope")
}
