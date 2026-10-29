package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/storecenter"
)

const ModuleName = "store-center"

const requestBodyReadTimeout = 30 * time.Second

type routeModule struct{ handler *Handler }

func NewModule(handler *Handler) kernelmodule.Module { return routeModule{handler: handler} }

// Composition supplies current directory and saved observation ports once,
// before registry registration. It never opens pools or installs schema.
func (m routeModule) ConfigureCatalog(catalog *storecenter.Catalog, metadata *storecenter.MetadataRepository) {
	m.handler.catalog, m.handler.metadata = catalog, metadata
}

func (m routeModule) ConfigureBrowser(service *storecenter.BrowserOperations) error {
	return m.handler.SetBrowserOperations(service)
}

func (routeModule) Name() string { return ModuleName }

func (m routeModule) Enabled(cfg *config.Config) bool {
	return m.handler != nil && cfg != nil && cfg.Workbench.Enabled
}

func (m routeModule) Register(registry *kernelmodule.Registry) error {
	if m.handler.browser != nil {
		registry.AddRoutes(m.handler.browserRoutes()...)
	}
	if lifecycle := m.handler.tikTokLifecycleRoute(); lifecycle != nil {
		registry.AddRoutes(*lifecycle)
	}
	if m.handler.metadata != nil {
		registry.AddRoutes(
			route(http.MethodGet, "/api/v1/workbench/stores/:store_id/summary", authz.PermissionWorkbenchStoreRead, httproute.OrganizationAccessPolicyLiveWrite, m.handler.StoreSummary),
			route(http.MethodGet, "/api/v1/workbench/store-catalog", authz.PermissionWorkbenchStoreRead, httproute.OrganizationAccessPolicyLiveWrite, m.handler.catalogList),
			route(http.MethodGet, "/api/v1/workbench/store-groups", authz.PermissionWorkbenchStoreRead, httproute.OrganizationAccessPolicyLiveWrite, m.handler.Groups),
			route(http.MethodGet, "/api/v1/workbench/stores/:store_id/profile", authz.PermissionWorkbenchStoreRead, httproute.OrganizationAccessPolicyLiveWrite, m.handler.Profile),
			route(http.MethodPost, "/api/v1/workbench/store-metadata", authz.PermissionWorkbenchStoreUpdate, httproute.OrganizationAccessPolicyLiveWrite, m.handler.MetadataCommand),
			route(http.MethodGet, "/api/v1/workbench/store-metadata/:operation_id", authz.PermissionWorkbenchStoreUpdate, httproute.OrganizationAccessPolicyLiveWrite, m.handler.MetadataReceipt),
		)
	}
	registry.AddRoutes(
		route(http.MethodGet, "/api/v1/workbench/stores", authz.PermissionWorkbenchStoreRead, httproute.OrganizationAccessPolicyLiveWrite, m.handler.List),
		route(http.MethodPost, "/api/v1/workbench/stores", authz.PermissionWorkbenchStoreCreate, httproute.OrganizationAccessPolicyLiveWrite, m.handler.Create),
		route(http.MethodGet, "/api/v1/workbench/stores/:store_id", authz.PermissionWorkbenchStoreRead, httproute.OrganizationAccessPolicyLiveWrite, m.handler.Get),
		route(http.MethodPut, "/api/v1/workbench/stores/:store_id", authz.PermissionWorkbenchStoreUpdate, httproute.OrganizationAccessPolicyLiveWrite, m.handler.Update),
		route(http.MethodPost, "/api/v1/workbench/stores/:store_id/disable", authz.PermissionWorkbenchStoreLifecycle, httproute.OrganizationAccessPolicyLiveWrite, m.handler.Disable),
		route(http.MethodPost, "/api/v1/workbench/stores/:store_id/enable", authz.PermissionWorkbenchStoreLifecycle, httproute.OrganizationAccessPolicyLiveWrite, m.handler.Enable),
		route(http.MethodDelete, "/api/v1/workbench/stores/:store_id", authz.PermissionWorkbenchStoreDelete, httproute.OrganizationAccessPolicyLiveWrite, m.handler.Delete),
	)
	if m.handler.serviceLifecycle != nil {
		registry.AddRoutes(
			route(http.MethodPost, "/api/v1/workbench/stores/:store_id/activate", authz.PermissionWorkbenchStoreLifecycle, httproute.OrganizationAccessPolicyLiveWrite, m.handler.Activate),
			route(http.MethodPost, "/api/v1/workbench/stores/:store_id/renew", authz.PermissionWorkbenchStoreLifecycle, httproute.OrganizationAccessPolicyLiveWrite, m.handler.Renew),
			route(http.MethodPost, "/api/v1/workbench/stores/:store_id/reactivate", authz.PermissionWorkbenchStoreLifecycle, httproute.OrganizationAccessPolicyLiveWrite, m.handler.Reactivate),
		)
	}
	if m.handler.officialConnections != nil {
		registry.AddRoutes(
			route(http.MethodGet, "/api/v1/workbench/stores/:store_id/connection", authz.PermissionWorkbenchStoreRead, httproute.OrganizationAccessPolicyLiveWrite, m.handler.ReadOfficialConnection),
			route(http.MethodGet, "/api/v1/workbench/stores/:store_id/connection/applications", authz.PermissionWorkbenchStoreRead, httproute.OrganizationAccessPolicyLiveWrite, m.handler.OfficialApplications),
			route(http.MethodPost, "/api/v1/workbench/stores/:store_id/connection/begin", authz.PermissionWorkbenchStoreUpdate, httproute.OrganizationAccessPolicyLiveWrite, m.handler.BeginOfficialConnection),
			route(http.MethodPost, "/api/v1/workbench/stores/:store_id/connection/complete", authz.PermissionWorkbenchStoreUpdate, httproute.OrganizationAccessPolicyLiveWrite, m.handler.CompleteOfficialConnection),
			route(http.MethodPost, "/api/v1/workbench/stores/:store_id/connection/token", authz.PermissionWorkbenchStoreUpdate, httproute.OrganizationAccessPolicyLiveWrite, m.handler.InstallOfficialToken),
			route(http.MethodPost, "/api/v1/workbench/stores/:store_id/connection/query", authz.PermissionWorkbenchStoreUpdate, httproute.OrganizationAccessPolicyLiveWrite, m.handler.ResumeOfficialConnectionQuery),
			route(http.MethodPost, "/api/v1/workbench/stores/:store_id/connection/refresh", authz.PermissionWorkbenchStoreUpdate, httproute.OrganizationAccessPolicyLiveWrite, m.handler.RefreshOfficialConnection),
			route(http.MethodPost, "/api/v1/workbench/stores/:store_id/connection/disconnect", authz.PermissionWorkbenchStoreUpdate, httproute.OrganizationAccessPolicyLiveWrite, m.handler.DisconnectOfficialConnection),
		)
	}
	return nil
}

func route(method, path, permission string, access httproute.OrganizationAccessPolicy, handler func(*gin.Context)) httproute.Descriptor {
	return httproute.Descriptor{
		Method: method, Path: path, Module: ModuleName, Permission: permission,
		AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: access, Handler: httproute.WithRequestBodyReadTimeout(requestBodyReadTimeout, handler),
	}
}
