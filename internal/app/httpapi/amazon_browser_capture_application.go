package httpapi

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"mime"
	"net/http"
	"strings"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
	"task-processor/internal/workbenchcontext"
	"time"
)

const amazonCaptureBase = "/api/v1/workbench/sourcing/amazon/browser-captures"

func WithAmazonBrowserCapture() CurrentApplicationOption {
	return func(options *currentApplicationOptions) { options.amazonBrowserCaptures++ }
}

type amazonCaptureService interface {
	Capture(context.Context, string, []byte) (productsourcing.AmazonCaptureResult, error)
	Verify(context.Context, string, []byte) (productsourcing.AmazonCaptureResult, error)
	ByKey(context.Context, string) (productsourcing.AmazonCaptureResult, error)
	Read(context.Context, string) (productsourcing.AmazonCaptureResult, error)
	Published(context.Context, string) (productsourcing.AmazonCaptureResult, error)
}
type amazonCaptureModule struct{ routes []httproute.Descriptor }

func (amazonCaptureModule) Name() string { return "amazon-browser-capture" }
func (amazonCaptureModule) Enabled(cfg *config.Config) bool {
	return cfg != nil && cfg.Workbench.Enabled
}
func (m amazonCaptureModule) Register(reg *kernelmodule.Registry) error {
	reg.AddRoutes(m.routes...)
	return nil
}

func buildAmazonCaptureModule(ctx context.Context, db *gorm.DB, deps routeAuthDependencies, permissions *authz.ListingKitAuthorizer, capabilities acquisitionstore.RuntimeCapabilities) (kernelmodule.Module, error) {
	if db == nil || deps.organizationResolver == nil || permissions == nil || collectionstore.VerifySchema(ctx, db) != nil {
		return nil, sourcing.ErrSourcePublicationUnavailable
	}
	for _, table := range []string{"product_source_publications", "product_source_publication_receipts", "product_snapshot_versions", "product_snapshot_heads"} {
		if !db.Migrator().HasTable(table) {
			return nil, sourcing.ErrSourcePublicationUnavailable
		}
	}
	if err := acquisitionstore.VerifyRuntimePermissions(ctx, db, capabilities); err != nil {
		return nil, err
	}
	write := &productReviewLiveOrganizationAccess{resolver: deps.organizationResolver, now: time.Now}
	auth, err := productsourcing.NewAmazonCaptureContextAuthorizer(write, amazonReadAccess{deps.organizationResolver}, permissions)
	if err != nil {
		return nil, err
	}
	service, err := productsourcing.NewAmazonBrowserCapture(db, auth)
	if err != nil {
		return nil, err
	}
	return amazonCaptureModule{amazonCaptureRoutes(service, (productReviewCapabilityBinder{now: time.Now}).Bind)}, nil
}

type amazonReadAccess struct{ resolver organizationIdentityResolver }

func (a amazonReadAccess) ResolveLiveRoles(ctx context.Context, org, actor string) ([]string, error) {
	capability, ok := ctx.Value(productReviewCapabilityContextKey{}).(productReviewRequestCapability)
	identity, authenticated := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || !authenticated || a.resolver == nil || capability.actorID != actor || capability.effectiveOrganizationID != org || identity.EffectiveOrganizationID != org || identity.UserID != actor || !time.Now().Before(identity.TokenExpiresAt) {
		return nil, sourcing.ErrPublicationForbidden
	}
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	resolved, err := a.resolver.Resolve(bounded, httproute.OrganizationAccessPolicyCachedRead, workbenchcontext.ResolveInput{Identity: authidentity.AuthenticatedIdentity{UserID: actor, HomeOrganizationID: capability.homeOrganizationID, TokenExpiresAt: capability.tokenExpiresAt}, BearerToken: capability.bearerToken, RequestedOrganizationID: org})
	if err != nil {
		return nil, sourcing.ErrPublicationForbidden
	}
	if resolved.UserID != actor || resolved.TenantID != org || resolved.EffectiveOrganizationID != org || resolved.EffectiveMemberID != identity.EffectiveMemberID {
		return nil, sourcing.ErrPublicationForbidden
	}
	return append([]string(nil), resolved.Roles...), nil
}

func amazonCaptureRoutes(service amazonCaptureService, bind func(context.Context, string) (context.Context, error)) []httproute.Descriptor {
	specs := []struct{ method, path, action string }{{"POST", amazonCaptureBase, "capture"}, {"POST", amazonCaptureBase + "/verify", "verify"}, {"GET", amazonCaptureBase + "/by-key/:key", "by-key"}, {"GET", amazonCaptureBase + "/:capture_id", "read"}, {"GET", amazonCaptureBase + "/:capture_id/published", "published"}}
	routes := make([]httproute.Descriptor, 0, len(specs))
	for _, spec := range specs {
		handler := func(c *gin.Context) {
			c.Header("Cache-Control", "no-store")
			if service == nil || bind == nil {
				writeAmazonCaptureError(c, sourcing.ErrSourcePublicationUnavailable)
				return
			}
			ctx, err := bind(c.Request.Context(), c.GetHeader("Authorization"))
			if err != nil {
				writeAmazonCaptureError(c, sourcing.ErrPublicationForbidden)
				return
			}
			identity, authenticated := authidentity.AuthenticatedIdentityFromContext(ctx)
			expectedMembers := c.Request.Header.Values("X-Expected-Member-ID")
			if !authenticated || len(expectedMembers) != 1 || !authidentity.IsBoundedIdentifier(expectedMembers[0]) || expectedMembers[0] != identity.EffectiveMemberID {
				writeAmazonCaptureError(c, sourcing.ErrPublicationForbidden)
				return
			}
			if c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
				writeAmazonCaptureError(c, sourcing.ErrInvalidBrowserCapture)
				return
			}
			var result productsourcing.AmazonCaptureResult
			if spec.action == "capture" || spec.action == "verify" {
				key, body, readErr := readAmazonCaptureRequest(c.Request)
				if readErr != nil {
					writeAmazonCaptureError(c, readErr)
					return
				}
				if spec.action == "capture" {
					result, err = service.Capture(ctx, key, body)
				} else {
					result, err = service.Verify(ctx, key, body)
				}
			} else {
				if c.Request.Body != nil {
					body, e := io.ReadAll(io.LimitReader(c.Request.Body, 1))
					if e != nil || len(body) != 0 {
						writeAmazonCaptureError(c, sourcing.ErrInvalidBrowserCapture)
						return
					}
				}
				id := c.Param("capture_id")
				if spec.action == "by-key" {
					id = c.Param("key")
				}
				if !acquisitionHTTPUUID(id) {
					writeAmazonCaptureError(c, sourcing.ErrInvalidBrowserCapture)
					return
				}
				switch spec.action {
				case "by-key":
					result, err = service.ByKey(ctx, id)
				case "published":
					result, err = service.Published(ctx, id)
				default:
					result, err = service.Read(ctx, id)
				}
			}
			if err != nil {
				writeAmazonCaptureError(c, err)
				return
			}
			c.JSON(http.StatusOK, result)
		}
		permission, policy := authz.PermissionWorkbenchSupplyRead, httproute.OrganizationAccessPolicyCachedRead
		if spec.action == "capture" {
			permission, policy = authz.PermissionProductSourcingWrite, httproute.OrganizationAccessPolicyLiveWrite
		}
		routes = append(routes, httproute.Descriptor{Method: spec.method, Path: spec.path, Module: "amazon-browser-capture", Permission: permission, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: policy, RequestTimeout: 20 * time.Second, Handler: httproute.WithRequestBodyReadTimeout(5*time.Second, handler)})
	}
	return routes
}
func readAmazonCaptureRequest(request *http.Request) (string, []byte, error) {
	keys := request.Header.Values("Idempotency-Key")
	if len(keys) != 1 || !acquisitionHTTPUUID(keys[0]) {
		return "", nil, sourcing.ErrInvalidBrowserCapture
	}
	media, parameters, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || parameters["charset"] != "" && !strings.EqualFold(parameters["charset"], "utf-8") || request.Header.Get("Content-Encoding") != "" || request.Body == nil {
		return "", nil, sourcing.ErrInvalidBrowserCapture
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, sourcing.BrowserCaptureMaxBytes+1))
	if err != nil {
		return "", nil, sourcing.ErrInvalidBrowserCapture
	}
	if len(body) > sourcing.BrowserCaptureMaxBytes {
		return "", nil, sourcing.ErrSourcePublicationTooLarge
	}
	if _, err = sourcing.ParseAmazonBrowserCapture(body); err != nil {
		return "", nil, err
	}
	return keys[0], body, nil
}
func writeAmazonCaptureError(c *gin.Context, err error) {
	status, code := 503, "CAPTURE_UNAVAILABLE"
	switch {
	case errors.Is(err, sourcing.ErrInvalidBrowserCapture), errors.Is(err, sourcing.ErrInvalidAcquisition), errors.Is(err, sourcing.ErrInvalidSourcePublication):
		status, code = 400, "INVALID_CAPTURE"
	case errors.Is(err, sourcing.ErrPublicationForbidden), errors.Is(err, collection.ErrForbidden):
		status, code = 403, "CAPTURE_FORBIDDEN"
	case errors.Is(err, sourcing.ErrSourcePublicationNotFound):
		status, code = 404, "CAPTURE_NOT_FOUND"
	case errors.Is(err, sourcing.ErrSourcePublicationConflict), errors.Is(err, collection.ErrConflict):
		status, code = 409, "CAPTURE_CONFLICT"
	case errors.Is(err, sourcing.ErrSourcePublicationTooLarge):
		status, code = 413, "SOURCE_TOO_LARGE"
	case errors.Is(err, sourcing.ErrSourcePublicationStateInvalid):
		status, code = 503, "CAPTURE_STATE_INVALID"
	case errors.Is(err, sourcing.ErrSourcePublicationOutcomeUnknown), errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		status, code = 503, "OUTCOME_UNKNOWN"
	}
	c.JSON(status, gin.H{"code": code, "message": code})
}
