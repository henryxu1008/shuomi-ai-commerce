package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	storeapp "task-processor/internal/app/storecenter"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/storecenter"
	storehttp "task-processor/internal/storecenter/httpapi"
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
	module := storehttp.NewModule(handler)
	catalog, metadata, err := buildStoreCatalog(records, authorizer, applications, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	module.(interface {
		ConfigureCatalog(*storecenter.Catalog, *storecenter.MetadataRepository)
	}).ConfigureCatalog(catalog, metadata)
	return module, nil
}

type currentStorePeriodPolicy struct{}

func (currentStorePeriodPolicy) MaxQuantity(context.Context, string, storecenter.ServiceCommand) (int64, error) {
	return 12, nil
}

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
	for _, expected := range []struct {
		method, path, permission string
		policy                   httproute.OrganizationAccessPolicy
	}{
		{http.MethodPost, "/api/v1/workbench/stores/:store_id/browser-operations", authz.PermissionWorkbenchStoreBrowserUse, httproute.OrganizationAccessPolicyLiveWrite},
		{http.MethodGet, "/api/v1/workbench/stores/:store_id/browser-operations/:operation_id", "", httproute.OrganizationAccessPolicyNone},
		{http.MethodPost, "/api/v1/workbench/stores/:store_id/browser-operations/:operation_id/reports", "", httproute.OrganizationAccessPolicyNone},
		{http.MethodGet, "/api/v1/workbench/stores/:store_id/audit", authz.PermissionWorkbenchStoreRead, httproute.OrganizationAccessPolicyLiveWrite},
	} {
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
