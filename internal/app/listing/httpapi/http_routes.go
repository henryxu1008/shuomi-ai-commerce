package listinghttp

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"mime"
	"net/http"
	"net/url"
	sigjson "sigs.k8s.io/json"
	"strconv"
	"strings"
	listingapp "task-processor/internal/app/listing"
	"task-processor/internal/app/workbenchmetrics"
	"task-processor/internal/httproute"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"time"
	"unicode/utf8"
)

const ListingBasePath = "/api/v1/workbench/supply-preparations"
const listingMaxBytes = 2 << 20

// Preparation-only composition exposes neither store targeting nor execution.
func IsPreparationRoute(method, path string) bool {
	if method == http.MethodPost {
		return path == ListingBasePath+"/transfer"
	}
	if method != http.MethodGet {
		return false
	}
	switch path {
	case ListingBasePath, ListingBasePath + "/:preparation_id", ListingBasePath + "/:preparation_id/sources", ListingBasePath + "/sources/:source_id", ListingBasePath + "/transfers/by-key/:key":
		return true
	}
	return false
}
func PreparationRoutes(app *listingapp.Application, bind func(context.Context, string) (context.Context, error)) []httproute.Descriptor {
	result := []httproute.Descriptor{}
	for _, route := range ListingRoutes(app, bind) {
		if IsPreparationRoute(route.Method, route.Path) {
			result = append(result, route)
		}
	}
	return result
}

// Explicit read-only selection composition; execution descriptors are absent.
func IsPrivateDraftReadRoute(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	switch path {
	case ListingBasePath, ListingBasePath + "/:preparation_id", ListingBasePath + "/:preparation_id/sources", ListingBasePath + "/sources/:source_id", ListingBasePath + "/:preparation_id/stages", ListingBasePath + "/sources/:source_id/targets/:store_id", ListingBasePath + "/records/:record_id":
		return true
	}
	return false
}
func PrivateDraftReadRoutes(app *listingapp.Application, bind func(context.Context, string) (context.Context, error)) []httproute.Descriptor {
	var out []httproute.Descriptor
	for _, route := range ListingRoutes(app, bind) {
		if IsPrivateDraftReadRoute(route.Method, route.Path) {
			out = append(out, route)
		}
	}
	return out
}

func ListingRoutes(app *listingapp.Application, bind func(context.Context, string) (context.Context, error)) []httproute.Descriptor {
	specs := []struct{ method, path, action, permission string }{
		{"POST", "/upload-plans", "create-upload-plan", preparation.PermissionManage},
		{"GET", "/upload-plans", "upload-plans", preparation.PermissionRead},
		{"GET", "/upload-plans/by-key/:key", "upload-plan-key", preparation.PermissionRead},
		{"GET", "/upload-plans/:plan_id", "upload-plan", preparation.PermissionRead},
		{"POST", "/upload-plans/:plan_id/preview", "upload-plan-preview", preparation.PermissionManage},
		{"POST", "/upload-plans/:plan_id/cancel", "cancel-upload-plan", preparation.PermissionManage},
		{"POST", "/upload-plans/:plan_id/continue", "continue-upload-plan", preparation.PermissionManage},
		{"GET", "/upload-plans/:plan_id/commands/:key", "upload-plan-command", preparation.PermissionRead},
		{"GET", "/optimization-options", "optimization-options", preparation.PermissionManage},
		{"GET", "", "list", preparation.PermissionRead}, {"POST", "/transfer", "transfer", preparation.PermissionManage}, {"GET", "/transfers/by-key/:key", "transfer-read", preparation.PermissionRead},
		{"GET", "/:preparation_id/sources", "sources", preparation.PermissionRead}, {"GET", "/sources/:source_id", "source", preparation.PermissionRead},
		{"GET", "/:preparation_id/stages", "stages", preparation.PermissionRead},
		{"GET", "/cockpit-metrics", "cockpit-metrics", preparation.PermissionRead},
		{"GET", "/walmart-feed-metrics", "walmart-feed-metrics", preparation.PermissionRead},
		{"GET", "/tiktok-publish-metrics", "tiktok-publish-metrics", preparation.PermissionRead},
		{"GET", "/shein-publish-metrics", "shein-publish-metrics", preparation.PermissionRead},
		{"GET", "/temu-create-metrics", "temu-create-metrics", preparation.PermissionRead},
		{"GET", "/submission-overview-metrics", "submission-overview-metrics", preparation.PermissionRead},
		{"POST", "/shein-sales-mapping-preview", "shein-sales-mapping-preview", preparation.PermissionManage},
		{"POST", "/shein-completion-preview", "shein-completion-preview", preparation.PermissionManage},
		{"POST", "/shein-color-split-preview", "color-split-preview", preparation.PermissionManage},
		{"POST", "/shein-color-splits", "create-color-split", preparation.PermissionManage},
		{"GET", "/shein-color-split-commands/:key", "color-split-command", preparation.PermissionRead},
		{"GET", "/sources/:source_id/stores/:store_id/color-splits", "color-split-list", preparation.PermissionRead},
		{"GET", "/sources/:source_id/color-splits/:family_id", "color-split-family", preparation.PermissionRead},
		{"GET", "/:preparation_id", "preparation", preparation.PermissionRead}, {"GET", "/:preparation_id/operations", "operations", preparation.PermissionRead},
		{"GET", "/sources/:source_id/publications/:store_id", "publication", preparation.PermissionRead},
		{"GET", "/sources/:source_id/publications/:store_id/status/:receipt_id", "publication-status", preparation.PermissionRead},
		{"GET", "/sources/:source_id/publications/:store_id/status", "publication-status", preparation.PermissionRead},
		{"POST", "/shein-publication/status", "shein-status", preparation.PermissionRead},
		{"POST", "/shein-publication/prepare", "shein-shelf-prepare", preparation.PermissionSubmit},
		{"POST", "/shein-publication/confirm", "shein-shelf-confirm", preparation.PermissionSubmit},
		{"POST", "/shein-publication/result", "shein-shelf-result", preparation.PermissionRead},
		{"GET", "/sources/:source_id/targets/:store_id", "target", preparation.PermissionRead}, {"POST", "/targets", "save-target", preparation.PermissionManage}, {"POST", "/target-rules", "rules", preparation.PermissionRead}, {"GET", "/target-commands/:key", "target-command", preparation.PermissionRead}, {"GET", "/records/:record_id", "record", preparation.PermissionRead},
		{"POST", "/uploads/resolve", "resolve-upload", preparation.PermissionManage},
		{"GET", "/uploads/:record_id/:attempt_id", "upload-attempt", preparation.PermissionRead},
		{"POST", "/images/approve", "approve", preparation.PermissionManage}, {"POST", "/images/inventory", "inventory", preparation.PermissionManage},
		{"POST", "/operations", "create-operation", preparation.PermissionManage}, {"GET", "/operations/by-key/:key", "operation-key", preparation.PermissionRead}, {"GET", "/operations/:operation_id", "operation", preparation.PermissionRead}, {"GET", "/operations/:operation_id/items", "operation-items", preparation.PermissionRead}, {"POST", "/operations/:operation_id/ensure", "ensure", preparation.PermissionManage}, {"POST", "/operations/:operation_id/cancel", "cancel", preparation.PermissionManage},
	}
	result := []httproute.Descriptor{}
	for _, spec := range specs {
		budget := 20 * time.Second
		if strings.HasPrefix(spec.path, "/shein-publication/") || strings.Contains(spec.path, "/status") {
			budget = 30 * time.Second
		}
		result = append(result, httproute.Descriptor{Method: spec.method, Path: ListingBasePath + spec.path, Module: "supply-chain", Permission: spec.permission, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: budget, Handler: httproute.WithRequestBodyReadTimeout(3e9, func(c *gin.Context) {
			c.Header("Cache-Control", "no-store")
			if app == nil || bind == nil {
				listingError(c, preparation.ErrUnavailable)
				return
			}
			ctx, err := bind(c.Request.Context(), c.GetHeader("Authorization"))
			if err != nil {
				listingError(c, preparation.ErrForbidden)
				return
			}
			var output any
			if spec.method == http.MethodGet || spec.action == "ensure" || spec.action == "cancel" {
				if e := listingEmptyBody(c.Request); e != nil {
					listingError(c, e)
					return
				}
			}
			if spec.action != "submission-overview-metrics" && spec.action != "cockpit-metrics" && spec.action != "walmart-feed-metrics" && spec.action != "tiktok-publish-metrics" && spec.action != "shein-publish-metrics" && spec.action != "temu-create-metrics" && spec.action != "upload-plans" && spec.action != "optimization-options" && spec.action != "list" && spec.action != "sources" && spec.action != "stages" && spec.action != "operation-items" && spec.action != "operations" && c.Request.URL.RawQuery != "" {
				listingError(c, preparation.ErrInvalid)
				return
			}
			switch spec.action {
			case "submission-overview-metrics", "cockpit-metrics", "walmart-feed-metrics", "tiktok-publish-metrics", "shein-publish-metrics", "temu-create-metrics":
				now := time.Now()
				filter, e := workbenchmetrics.Parse(c.Request.URL.RawQuery, now)
				if e != nil {
					listingError(c, preparation.ErrInvalid)
					return
				}
				readCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
				defer cancel()
				switch spec.action {
				case "submission-overview-metrics":
					output, err = app.SubmissionOverview(readCtx, filter, now)
				case "walmart-feed-metrics":
					output, err = app.WalmartFeedMetrics(readCtx, filter, now)
				case "temu-create-metrics":
					output, err = app.TemuCreateMetrics(readCtx, filter, now)
				case "shein-publish-metrics":
					output, err = app.SheinPublishMetrics(readCtx, filter, now)
				case "tiktok-publish-metrics":
					output, err = app.TikTokPublishMetrics(readCtx, filter, now)
				default:
					output, err = app.CockpitMetrics(readCtx, filter, now)
				}
			case "cancel-upload-plan", "create-upload-plan", "upload-plans", "upload-plan", "upload-plan-key", "upload-plan-preview", "continue-upload-plan", "upload-plan-command":
				if app.UploadPlans == nil {
					listingError(c, preparation.ErrUnavailable)
					return
				}
				switch spec.action {
				case "cancel-upload-plan":
					var in struct {
						ExpectedRevision int64 `json:"expectedRevision"`
					}
					if _, e := listingBody(c.Request, &in, false); e != nil {
						listingError(c, e)
						return
					}
					output, err = app.UploadPlans.Cancel(ctx, c.Param("plan_id"), in.ExpectedRevision)
				case "upload-plans":
					q, e := listingQuery(c.Request)
					if e != nil || q.Keyword != "" {
						listingError(c, preparation.ErrInvalid)
						return
					}
					output, err = app.UploadPlans.List(ctx, q)
				case "upload-plan":
					output, err = app.UploadPlans.Read(ctx, c.Param("plan_id"))
				case "upload-plan-key":
					output, err = app.UploadPlans.ReadCreation(ctx, c.Param("key"))
				case "upload-plan-command":
					output, err = app.UploadPlans.ReadContinuation(ctx, c.Param("plan_id"), c.Param("key"))
				case "create-upload-plan":
					var in preparation.UploadPlanInput
					key, e := listingBody(c.Request, &in, true)
					if e != nil {
						listingError(c, e)
						return
					}
					output, err = app.UploadPlans.Create(ctx, key, in)
				case "upload-plan-preview", "continue-upload-plan":
					var in preparation.UploadPlanContinueInput
					key, e := listingBody(c.Request, &in, spec.action == "continue-upload-plan")
					if e != nil {
						listingError(c, e)
						return
					}
					if spec.action == "upload-plan-preview" {
						output, err = app.UploadPlans.Preview(ctx, c.Param("plan_id"), in)
					} else {
						var receipt preparation.UploadPlanAdvanceReceipt
						receipt, err = app.UploadPlans.Continue(ctx, c.Param("plan_id"), key, in)
						output = receipt
						if err == nil {
							for _, id := range receipt.OperationIDs {
								if ctx.Err() != nil {
									break
								}
								if _, e := app.Execution.EnsureExecution(ctx, id); e != nil {
									break
								}
							}
						}
					}
				}
			case "shein-status", "shein-shelf-prepare", "shein-shelf-confirm", "shein-shelf-result":
				var in listingapp.SheinShelfInput
				key, e := listingBody(c.Request, &in, spec.action == "shein-shelf-prepare" || spec.action == "shein-shelf-confirm")
				if e != nil {
					listingError(c, e)
					return
				}
				if app.SheinPublication == nil {
					listingError(c, preparation.ErrUnavailable)
					return
				}
				switch spec.action {
				case "shein-status":
					output, err = app.SheinPublication.Status(ctx, in.SourceID, in.StoreID, in.ReceiptID, in.StockOffset)
				case "shein-shelf-prepare":
					output, err = app.SheinPublication.Prepare(ctx, key, in)
				case "shein-shelf-confirm":
					output, err = app.SheinPublication.Confirm(ctx, key, in)
				case "shein-shelf-result":
					output, err = app.SheinPublication.Read(ctx, in.Key, in)
				}
			case "color-split-preview":
				var in record.ColorSplitRequest
				_, e := listingBody(c.Request, &in, false)
				if e != nil {
					listingError(c, e)
					return
				}
				if app.Targets == nil {
					listingError(c, record.ErrUnavailable)
					return
				}
				output, err = app.Targets.PreviewColorSplit(ctx, in)
			case "create-color-split":
				var in record.ColorSplitPlan
				key, e := listingBody(c.Request, &in, true)
				if e != nil {
					listingError(c, e)
					return
				}
				if app.Targets == nil {
					listingError(c, record.ErrUnavailable)
					return
				}
				output, err = app.Targets.CreateColorSplit(ctx, key, in)
			case "color-split-command":
				if app.Targets == nil {
					listingError(c, record.ErrUnavailable)
					return
				}
				output, err = app.Targets.ReadColorSplitCommand(ctx, c.Param("key"))
			case "color-split-list":
				if app.Targets == nil {
					listingError(c, record.ErrUnavailable)
					return
				}
				var items []record.ColorSplitReceipt
				items, err = app.Targets.ListColorSplitFamilies(ctx, c.Param("source_id"), c.Param("store_id"))
				output = gin.H{"items": items}
			case "color-split-family":
				if app.Targets == nil {
					listingError(c, record.ErrUnavailable)
					return
				}
				output, err = app.Targets.ReadColorSplitFamily(ctx, c.Param("source_id"), c.Param("family_id"))
			case "shein-sales-mapping-preview":
				var in record.SheinSalesRequest
				if _, e := listingBody(c.Request, &in, false); e != nil {
					listingError(c, e)
					return
				}
				if app.Targets == nil {
					listingError(c, record.ErrUnavailable)
					return
				}
				output, err = app.Targets.PreviewSheinSalesMapping(ctx, in)
			case "shein-completion-preview":
				var in record.SheinImportRequest
				_, e := listingBody(c.Request, &in, false)
				if e != nil {
					listingError(c, e)
					return
				}
				if !collection.ValidID(in.SourceID) || !collection.ValidID(in.BaseRecordID) || !collection.ValidID(in.CompletionDraft.RecordID) || in.CompletionDraft.Revision < 1 {
					listingError(c, preparation.ErrInvalid)
					return
				}
				if app.Targets == nil {
					listingError(c, preparation.ErrUnavailable)
					return
				}
				output, err = app.Targets.PreviewSheinImport(ctx, in)
			case "upload-attempt":
				if c.Request.URL.RawQuery != "" {
					listingError(c, preparation.ErrInvalid)
					return
				}
				output, err = app.ReadUploadAttempt(ctx, c.Param("record_id"), c.Param("attempt_id"))
			case "optimization-options":
				q, e := listingQuery(c.Request)
				if e != nil || q.Keyword != "" {
					listingError(c, preparation.ErrInvalid)
					return
				}
				if app.OptimizationOptions == nil {
					output = listingapp.OptimizationOptions{Titles: []listingapp.TitleOptimizationChoice{}, Reason: "智能体优化未配置"}
					break
				}
				output, err = app.OptimizationOptions(ctx, q)
			case "preparation":
				output, err = app.Preparations.Read(ctx, c.Param("preparation_id"))
			case "stages":
				values, e := url.ParseQuery(c.Request.URL.RawQuery)
				if e != nil || len(values["storeId"]) != 1 || !collection.ValidID(values.Get("storeId")) || len(values["stage"]) != 1 || !validStage(values.Get("stage")) || len(values["planId"]) > 1 || values.Has("planId") && !collection.ValidID(values.Get("planId")) || len(values["sourceKind"]) > 1 || values.Has("sourceKind") && !collection.ValidSourceKind(values.Get("sourceKind")) {
					listingError(c, preparation.ErrInvalid)
					return
				}
				storeID, stage, sourceKind := values.Get("storeId"), values.Get("stage"), values.Get("sourceKind")
				values.Del("storeId")
				values.Del("stage")
				planID := values.Get("planId")
				values.Del("planId")
				values.Del("sourceKind")
				request := c.Request.Clone(ctx)
				request.URL.RawQuery = values.Encode()
				q, e := listingQuery(request)
				if e != nil {
					listingError(c, e)
					return
				}
				output, err = app.Stages(ctx, c.Param("preparation_id"), storeID, stage, q, sourceKind, planID)
			case "operations":
				values, e := url.ParseQuery(c.Request.URL.RawQuery)
				if e != nil || len(values["storeId"]) != 1 || !collection.ValidID(values.Get("storeId")) {
					listingError(c, preparation.ErrInvalid)
					return
				}
				storeID := values.Get("storeId")
				values.Del("storeId")
				request := c.Request.Clone(ctx)
				request.URL.RawQuery = values.Encode()
				q, e := listingQuery(request)
				if e != nil {
					listingError(c, e)
					return
				}
				output, err = app.Operations.List(ctx, c.Param("preparation_id"), storeID, q)
			case "list", "sources", "operation-items":
				q, e := listingQuery(c.Request)
				if e != nil {
					listingError(c, e)
					return
				}
				switch spec.action {
				case "list":
					output, err = app.Preparations.List(ctx, q)
				case "sources":
					output, err = app.Preparations.ListSources(ctx, c.Param("preparation_id"), q)
				case "operation-items":
					output, err = app.OperationItems(ctx, c.Param("operation_id"), q)
				}
			case "transfer":
				var in preparation.TransferInput
				key, e := listingBody(c.Request, &in, true)
				if e != nil {
					listingError(c, e)
					return
				}
				output, err = app.Preparations.Transfer(ctx, key, in)
			case "transfer-read":
				output, err = app.Preparations.ReadByKey(ctx, c.Param("key"))
			case "source":
				output, err = app.Source(ctx, c.Param("source_id"))
			case "target":
				output, err = app.ReadTarget(ctx, c.Param("source_id"), c.Param("store_id"))
			case "publication":
				output, err = app.Publication(ctx, c.Param("source_id"), c.Param("store_id"))
			case "publication-status":
				output, err = app.PublicationStatus(ctx, c.Param("source_id"), c.Param("store_id"), c.Param("receipt_id"))
			case "record":
				output, err = app.ReadRecord(ctx, c.Param("record_id"))
			case "target-command":
				output, err = app.ReadTargetCommand(ctx, c.Param("key"))
			case "save-target", "rules":
				var body listingTargetBody
				key, e := listingBody(c.Request, &body, spec.action == "save-target")
				if e != nil {
					listingError(c, e)
					return
				}
				in, e := body.domain()
				if e != nil {
					listingError(c, e)
					return
				}
				if spec.action == "save-target" {
					output, err = app.Targets.Create(ctx, key, in)
				} else {
					output, err = app.QueryRules(ctx, in)
				}
			case "resolve-upload":
				var in listingapp.ResolveUploadInput
				_, e := listingBody(c.Request, &in, false)
				if e != nil {
					listingError(c, e)
					return
				}
				output, err = app.ResolveUpload(ctx, in)
			case "approve":
				var body listingImageApprovalBody
				key, e := listingBody(c.Request, &body, true)
				if e != nil {
					listingError(c, e)
					return
				}
				selection, e := body.Selection.domain()
				if e != nil {
					listingError(c, e)
					return
				}
				in := body.SourceApprovalCommand
				in.Selection = selection
				if in.ActionID != "" && in.ActionID != key {
					listingError(c, preparation.ErrInvalid)
					return
				}
				in.ActionID = key
				if app.SourceMaterialsOnly && len(in.Approved) != 0 {
					listingError(c, preparation.ErrInvalid)
					return
				}
				if app.Approvals == nil {
					listingError(c, preparation.ErrUnavailable)
					return
				}
				output, err = app.Approvals.Approve(ctx, in)
			case "inventory":
				var body listingImageSourceBody
				_, e := listingBody(c.Request, &body, false)
				if e != nil {
					listingError(c, e)
					return
				}
				in, e := body.domain()
				if e != nil {
					listingError(c, e)
					return
				}
				inventory, e := app.Inventory(ctx, in)
				err = e
				output = listingImageInventory(inventory)
			case "create-operation":
				var in preparation.OperationInput
				key, e := listingBody(c.Request, &in, true)
				if e != nil {
					listingError(c, e)
					return
				}
				if in.Action == preparation.OperationOptimize {
					if app.AuthorizeOptimization == nil {
						listingError(c, record.ErrNotReady)
						return
					}
					if e := app.AuthorizeOptimization(ctx, in); e != nil {
						listingError(c, e)
						return
					}
				}
				receipt, e := app.Operations.Create(ctx, key, in)
				if e != nil {
					listingError(c, e)
					return
				}
				operation, e := app.Execution.EnsureExecution(ctx, receipt.Operation.ID)
				if e == nil || errors.Is(e, preparation.ErrUnknown) {
					receipt.Operation = operation
					output = receipt
				} else {
					output = receipt
				}
			case "operation-key":
				output, err = app.OperationByKey(ctx, c.Param("key"))
			case "operation":
				output, err = app.Operations.Read(ctx, c.Param("operation_id"))
			case "ensure", "cancel":
				if spec.action == "ensure" {
					output, err = app.Execution.EnsureExecution(ctx, c.Param("operation_id"))
				} else {
					output, err = app.Operations.Cancel(ctx, c.Param("operation_id"))
				}
			}
			if err != nil {
				listingError(c, err)
				return
			}
			raw, e := json.Marshal(listingPublicResponse(output))
			if e != nil || len(raw) > listingMaxBytes {
				listingError(c, preparation.ErrUnavailable)
				return
			}
			c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
		})})
	}
	return result
}
func listingBody(r *http.Request, in any, requiresKey bool) (string, error) {
	return listingBodyLimit(r, in, requiresKey, listingMaxBytes)
}
func listingBodyLimit(r *http.Request, in any, requiresKey bool, maximum int64) (string, error) {
	keys := r.Header.Values("Idempotency-Key")
	key := ""
	if len(keys) == 1 {
		key = keys[0]
	}
	if requiresKey && (len(keys) != 1 || !collection.ValidID(key)) || !requiresKey && len(keys) > 0 || r.URL.RawQuery != "" || r.Body == nil {
		return "", preparation.ErrInvalid
	}
	media, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8") || r.Header.Get("Content-Encoding") != "" && r.Header.Get("Content-Encoding") != "identity" {
		return "", preparation.ErrInvalid
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, maximum+1))
	if err != nil || int64(len(raw)) > maximum || !utf8.Valid(raw) {
		return "", preparation.ErrInvalid
	}
	strict, err := sigjson.UnmarshalStrict(raw, in, sigjson.DisallowDuplicateFields, sigjson.DisallowUnknownFields)
	if err != nil || len(strict) > 0 {
		return "", preparation.ErrInvalid
	}
	return key, nil
}
func listingQuery(r *http.Request) (collection.Query, error) {
	values, parseErr := url.ParseQuery(r.URL.RawQuery)
	if parseErr != nil {
		return collection.Query{}, preparation.ErrInvalid
	}
	q := collection.Query{Limit: 50, After: values.Get("after"), Keyword: values.Get("keyword")}
	for key, list := range values {
		if len(list) != 1 || key != "limit" && key != "after" && key != "keyword" {
			return q, preparation.ErrInvalid
		}
	}
	var err error
	if values.Get("limit") != "" {
		q.Limit, err = strconv.Atoi(values.Get("limit"))
	}
	if err != nil || q.Validate() != nil {
		return q, preparation.ErrInvalid
	}
	return q, nil
}
func listingEmptyBody(r *http.Request) error {
	if len(r.Header.Values("Idempotency-Key")) > 0 || r.ContentLength > 0 {
		return preparation.ErrInvalid
	}
	if r.Body != nil {
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1))
		if err != nil || len(raw) > 0 {
			return preparation.ErrInvalid
		}
	}
	return nil
}
func listingError(c *gin.Context, err error) {
	status, code := http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE"
	switch {
	case errors.Is(err, submission.ErrExecutionInvalid), errors.Is(err, submission.ErrExecutionEvidenceRequired), errors.Is(err, collection.ErrInvalid), errors.Is(err, preparation.ErrInvalid), errors.Is(err, record.ErrInvalid), errors.Is(err, asset.ErrInvalidApproval):
		status, code = http.StatusBadRequest, "INVALID_REQUEST"
	case errors.Is(err, collection.ErrForbidden), errors.Is(err, preparation.ErrForbidden), errors.Is(err, record.ErrForbidden), errors.Is(err, asset.ErrSourceApprovalForbidden):
		status, code = http.StatusForbidden, "PERMISSION_DENIED"
	case errors.Is(err, submission.ErrExecutionNotFound), errors.Is(err, collection.ErrNotFound), errors.Is(err, preparation.ErrNotFound), errors.Is(err, record.ErrNotFound):
		status, code = http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, record.ErrSplitExists):
		status, code = http.StatusConflict, "SPLIT_ALREADY_EXISTS"
	case errors.Is(err, submission.ErrExecutionIntentConflict), errors.Is(err, preparation.ErrConflict), errors.Is(err, record.ErrConflict), errors.Is(err, asset.ErrApprovalConflict):
		status, code = http.StatusConflict, "REVISION_CONFLICT"
	case errors.Is(err, submission.ErrExecutionTargetClaimed), errors.Is(err, submission.ErrExecutionTargetSucceeded), errors.Is(err, record.ErrNotReady):
		status, code = http.StatusConflict, "NOT_READY"
	case errors.Is(err, submission.ErrExecutionOutcomeUnknown), errors.Is(err, preparation.ErrUnknown), errors.Is(err, record.ErrTargetUnknown), errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		status, code = http.StatusConflict, "OUTCOME_UNKNOWN"
	}
	c.JSON(status, gin.H{"code": code})
}

func validStage(v string) bool {
	switch v {
	case "all", "waiting", "missing", "ready", "review", "uploaded":
		return true
	}
	return false
}
