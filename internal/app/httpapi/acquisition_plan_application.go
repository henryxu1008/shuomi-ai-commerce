package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"io"
	"mime"
	"net/http"
	"net/url"
	sigjson "sigs.k8s.io/json"
	"strings"
	"task-processor/internal/app/productsourcing"
	supplyapp "task-processor/internal/app/supplychain"
	supplyhttp "task-processor/internal/app/supplychain/httpapi"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	planstore "task-processor/internal/integration/persistence/product/acquisitionplan"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
	"time"
	"unicode/utf8"
)

const acquisitionPlanBase = "/api/v1/workbench/sourcing/1688/plans"
const acquisitionRuleBase = "/api/v1/workbench/sourcing/1688/rules"

func WithAcquisitionPlans() CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.acquisitionPlans++ }
}

// Extend the current producer module, reusing the selected Preparation owner.
func composeAcquisitionPlans(ctx context.Context, module productAcquisitionModule, db *gorm.DB, deps routeAuthDependencies, permissions *authz.ListingKitAuthorizer, collections *collection.Service, full *supplyapp.Application) (productAcquisitionModule, error) {
	if module.service == nil || collections == nil {
		return module, sourcing.ErrAcquisitionUnavailable
	}
	discover, ok := module.provider.(sourcing.PublicProductDiscoverer)
	if !ok {
		return module, sourcing.ErrAcquisitionUnavailable
	}
	repo, err := planstore.NewRepository(ctx, db)
	if err != nil {
		return module, err
	}
	live := &productReviewLiveOrganizationAccess{resolver: deps.organizationResolver, now: time.Now}
	sourceAuth, err := sourcing.NewContextAuthorizer(live, permissions)
	if err != nil {
		return module, err
	}
	collectionAuth, err := collection.NewContextAuthorizer(live, permissions)
	if err != nil {
		return module, err
	}
	app := full
	if app == nil {
		core, e := composePreparationCore(ctx, db, deps, permissions, collections)
		if e != nil {
			return module, e
		}
		app = core.app
	}
	auth := productsourcing.PlanAuthority{Sourcing: sourceAuth, Collections: collectionAuth, Preparations: app.Authorization}
	service, err := productsourcing.NewAcquisitionPlanService(repo, repo, module.service, collections, app.Preparations, auth, discover)
	if err != nil {
		return module, err
	}
	bind := productReviewCapabilityBinder{now: time.Now}
	module.routes = append(module.routes, acquisitionPlanRoutes(service, bind.Bind)...)
	if full == nil {
		module.routes = append(module.routes, supplyhttp.PreparationRoutes(app, bind.Bind)...)
	}
	return module, nil
}

func acquisitionPlanRoutes(service *productsourcing.AcquisitionPlanService, bind func(context.Context, string) (context.Context, error)) []httproute.Descriptor {
	specs := []struct{ method, path, action string }{
		{"GET", acquisitionPlanBase, "list"}, {"POST", acquisitionPlanBase, "create"}, {"GET", acquisitionPlanBase + "/by-key/:key", "by-key"}, {"GET", acquisitionPlanBase + "/:plan_id", "read"}, {"POST", acquisitionPlanBase + "/:plan_id/advance", "advance"},
		{"GET", acquisitionRuleBase, "rules"}, {"POST", acquisitionRuleBase, "mutate-rule"},
	}
	routes := []httproute.Descriptor{}
	for _, spec := range specs {
		routes = append(routes, httproute.Descriptor{Method: spec.method, Path: spec.path, Module: "product-acquisition", Permission: "product_sourcing.write", AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 50 * time.Second, Handler: httproute.WithRequestBodyReadTimeout(3*time.Second, func(c *gin.Context) {
			c.Header("Cache-Control", "no-store")
			if service == nil || bind == nil {
				writeAcquisitionError(c, sourcing.ErrAcquisitionUnavailable)
				return
			}
			ctx, err := bind(c.Request.Context(), c.GetHeader("Authorization"))
			if err != nil {
				writeAcquisitionError(c, sourcing.ErrPublicationForbidden)
				return
			}
			query, queryErr := url.ParseQuery(c.Request.URL.RawQuery)
			after := query.Get("after")
			if queryErr != nil || len(query) > 0 && (spec.action != "list" || len(query) != 1 || len(query["after"]) != 1 || !acquisitionHTTPUUID(after)) {
				writeAcquisitionError(c, sourcing.ErrInvalidAcquisition)
				return
			}
			if spec.method == "GET" && (c.Request.ContentLength != 0 || len(c.Request.TransferEncoding) > 0) {
				httproute.RejectUnreadRequestBody(c)
				writeAcquisitionError(c, sourcing.ErrInvalidAcquisition)
				return
			}
			var output any
			switch spec.action {
			case "list":
				var values []sourcing.AcquisitionPlan
				values, err = service.List(ctx, after)
				var transfer bool
				if err == nil {
					transfer, err = service.CanTransfer(ctx)
				}
				output = gin.H{"schemaVersion": 1, "plans": values, "canAutoTransfer": transfer}
				if len(values) > 0 {
					output.(gin.H)["nextCursor"] = values[len(values)-1].ID
				}
			case "rules":
				var values []sourcing.AcquisitionRuleTemplate
				values, err = service.Rules(ctx)
				output = gin.H{"schemaVersion": 1, "rules": values}
			case "read":
				output, err = service.Read(ctx, c.Param("plan_id"))
			case "by-key":
				output, err = service.ByKey(ctx, c.Param("key"))
			case "create":
				var input productsourcing.CreatePlanInput
				key, e := readPlanRequest(c.Request, &input, true)
				if e != nil {
					err = e
					break
				}
				output, err = service.Create(ctx, key, input)
			case "advance":
				var input struct {
					ExpectedRevision int64 `json:"expectedRevision"`
				}
				_, e := readPlanRequest(c.Request, &input, false)
				if e != nil {
					err = e
					break
				}
				output, err = service.Advance(ctx, c.Param("plan_id"), input.ExpectedRevision)
			case "mutate-rule":
				var input sourcing.RuleTemplateMutation
				key, e := readPlanRequest(c.Request, &input, true)
				if e != nil {
					err = e
					break
				}
				output, err = service.MutateRule(ctx, key, input)
			}
			if err != nil {
				if errors.Is(err, preparation.ErrForbidden) || errors.Is(err, collection.ErrForbidden) {
					err = sourcing.ErrPublicationForbidden
				}
				if errors.Is(err, preparation.ErrConflict) || errors.Is(err, collection.ErrConflict) {
					err = sourcing.ErrAcquisitionConflict
				}
				writeAcquisitionError(c, err)
				return
			}
			raw, e := json.Marshal(output)
			if e != nil || len(raw) > 2<<20 {
				writeAcquisitionError(c, sourcing.ErrSourcePublicationTooLarge)
				return
			}
			c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
		})})
	}
	return routes
}

func readPlanRequest(request *http.Request, input any, requireKey bool) (string, error) {
	key := ""
	if requireKey {
		keys := request.Header.Values("Idempotency-Key")
		if len(keys) != 1 || !acquisitionHTTPUUID(keys[0]) {
			return key, sourcing.ErrInvalidAcquisition
		}
		key = keys[0]
	}
	media, parameters, e := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if e != nil || media != "application/json" || parameters["charset"] != "" && !strings.EqualFold(parameters["charset"], "utf-8") || request.Header.Get("Content-Encoding") != "" && request.Header.Get("Content-Encoding") != "identity" || request.Body == nil {
		return key, sourcing.ErrInvalidAcquisition
	}
	raw, e := io.ReadAll(io.LimitReader(request.Body, sourcing.MaxPlanBodyBytes+1))
	if len(raw) > sourcing.MaxPlanBodyBytes {
		return key, sourcing.ErrSourcePublicationTooLarge
	}
	if e != nil || !utf8.Valid(raw) {
		return key, sourcing.ErrInvalidAcquisition
	}
	strict, e := sigjson.UnmarshalStrict(raw, input, sigjson.DisallowDuplicateFields, sigjson.DisallowUnknownFields)
	if e != nil || len(strict) > 0 {
		return key, sourcing.ErrInvalidAcquisition
	}
	return key, nil
}

func validateAcquisitionPlanDescriptor(route httproute.Descriptor) error {
	for _, expected := range acquisitionPlanRoutes(nil, nil) {
		if route.Method == expected.Method && route.Path == expected.Path {
			if route.Module != expected.Module || route.Permission != expected.Permission || route.AuthPolicy != expected.AuthPolicy || route.OrganizationAccessPolicy != expected.OrganizationAccessPolicy || route.OrganizationTargetResolver != nil || route.RequestTimeout != expected.RequestTimeout || route.Handler == nil {
				return errors.New("acquisition plan loses its live scoped contract")
			}
			return nil
		}
	}
	return errors.New("acquisition plan route not admitted")
}
