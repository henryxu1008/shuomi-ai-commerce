package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	sigjson "sigs.k8s.io/json"
	recordstore "task-processor/internal/app/listingrecordstore"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/listing/record"
	"task-processor/internal/marketplace/shein/draft"
	sheinvalidator "task-processor/internal/marketplace/shein/validator"
	contract "task-processor/internal/marketplace/validator"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/review"
	"task-processor/internal/storecenter"
)

const offlineRecordsPath = "/api/v1/workbench/offline-shein-records"

func WithSheinOfflineRecords(assets *gorm.DB) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.offlineRecords++; o.offlineRecordAssets = assets }
}

type offlineCollectionReader interface {
	ReadItem(context.Context, string) (collection.ItemDetail, error)
}
type normalOfflineRecords struct {
	records     *recordstore.Repository
	create      *record.Service
	collections offlineCollectionReader
	stores      record.StoreReferenceReader
	resolver    organizationIdentityResolver
	policy      *authz.ListingKitAuthorizer
}
type offlineRecordModule struct {
	app    *normalOfflineRecords
	routes []httproute.Descriptor
}

func (offlineRecordModule) Name() string                  { return "offline-shein-records" }
func (offlineRecordModule) Enabled(c *config.Config) bool { return c != nil && c.Workbench.Enabled }
func (m offlineRecordModule) Register(r *kernelmodule.Registry) error {
	r.AddRoutes(m.routes...)
	return nil
}

func buildOfflineRecords(ctx context.Context, products, assets, stores *gorm.DB, deps routeAuthDependencies, policy *authz.ListingKitAuthorizer, collections offlineCollectionReader) (offlineRecordModule, error) {
	if products == nil || assets == nil || stores == nil || products == assets || assets == stores || stores == products || deps.organizationResolver == nil || policy == nil || collections == nil {
		return offlineRecordModule{}, record.ErrUnavailable
	}
	if err := recordstore.VerifySchema(ctx, products); err != nil {
		return offlineRecordModule{}, err
	}
	if err := assetstore.VerifySourceRuntimePermissions(ctx, assets); err != nil {
		return offlineRecordModule{}, err
	}
	repository, err := recordstore.NewPrivateRepository(products, policy, func(ctx context.Context, tx *gorm.DB) (context.Context, error) {
		// Reuse the current Collection owner on the Record transaction's connection;
		// never take another connection from the held Product pool.
		reader, err := composeProductCollectionService(ctx, tx, deps, policy)
		if err != nil {
			return nil, err
		}
		return context.WithValue(ctx, offlineTransactionCollectionKey{}, reader), nil
	})
	if err != nil {
		return offlineRecordModule{}, err
	}
	source, err := catalogstore.NewBoundedSnapshotReader(products, record.MaxPayloadBytes)
	if err != nil {
		return offlineRecordModule{}, err
	}
	inventory, err := assetstore.NewBoundedApprovedInventoryReader(assets, record.MaxPayloadBytes)
	if err != nil {
		return offlineRecordModule{}, err
	}
	storeRepo, err := storecenter.NewMemberScopedStoreRepository(stores, currentStoreMemberAuthorizer{authorizer: policy})
	if err != nil {
		return offlineRecordModule{}, err
	}
	app := &normalOfflineRecords{records: repository, collections: collections, stores: sheinRecordStoreReader{repository: storeRepo}, resolver: deps.organizationResolver, policy: policy}
	app.create, err = record.NewService(record.ServiceDependencies{Products: source, Assets: inventory, Stores: app.stores, Records: repository, Builder: draft.Builder{}, Evaluator: sheinvalidator.ExactApprovedAssetValidator{}, Authorizer: policy, Now: time.Now, RuleRevision: sheinvalidator.DiagnosticRuleVersion, PreviousRuleRevision: sheinvalidator.PreviousDiagnosticRuleVersion, PolicyRevision: sheinvalidator.BindingVersion, CreationGuard: app.guard})
	if err != nil {
		return offlineRecordModule{}, err
	}
	return offlineRecordModule{app: app, routes: offlineRecordRoutes(app)}, nil
}
func (a *normalOfflineRecords) fresh(ctx context.Context, permission string) (context.Context, record.Actor, error) {
	original, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok {
		return ctx, record.Actor{}, record.ErrForbidden
	}
	identity, err := freshWorkbenchIdentity(ctx, a.resolver, a.policy, []string{original.TenantID}, permission)
	if err != nil {
		if errors.Is(err, review.ErrUnavailable) {
			return ctx, record.Actor{}, record.ErrUnavailable
		}
		return ctx, record.Actor{}, record.ErrForbidden
	}
	if identity.EffectiveMemberID != original.EffectiveMemberID {
		return ctx, record.Actor{}, record.ErrForbidden
	}
	bound := authidentity.WithAuthenticatedIdentity(ctx, identity)
	return bound, record.Actor{OrganizationID: identity.TenantID, UserID: identity.UserID, Roles: identity.Roles}, nil
}

type offlineSourceContextKey struct{}
type offlineTransactionCollectionKey struct{}
type offlineSourceContext struct{ itemID, org, user, member string }

func (a *normalOfflineRecords) guard(ctx context.Context, actor record.Actor, input record.Input) error {
	source, ok := ctx.Value(offlineSourceContextKey{}).(offlineSourceContext)
	if !ok || !collection.ValidID(source.itemID) {
		return record.ErrForbidden
	}
	fresh, current, err := a.fresh(ctx, authz.PermissionListingKitAdminWrite)
	if err != nil {
		return err
	}
	id, _ := authidentity.AuthenticatedIdentityFromContext(fresh)
	if source.org != current.OrganizationID || source.user != current.UserID || source.member != id.EffectiveMemberID || actor.OrganizationID != current.OrganizationID || actor.UserID != current.UserID {
		return record.ErrForbidden
	}
	reader := a.collections
	if transactionReader, ok := ctx.Value(offlineTransactionCollectionKey{}).(offlineCollectionReader); ok {
		reader = transactionReader
	}
	item, err := reader.ReadItem(fresh, source.itemID)
	if err != nil {
		if fresh.Err() != nil {
			return fresh.Err()
		}
		if errors.Is(err, collection.ErrNotFound) {
			return record.ErrNotFound
		}
		if errors.Is(err, collection.ErrUnavailable) {
			return record.ErrUnavailable
		}
		return record.ErrForbidden
	}
	if item.Item.ID != source.itemID || item.Item.ArchivedAt != nil || item.Item.Source.ProductKey != input.ProductKey || item.Item.Source.Version != input.SnapshotVersion {
		return record.ErrNotFound
	}
	ref, err := a.stores.GetStoreReference(fresh, current.OrganizationID, input.StoreID)
	if err != nil {
		return err
	}
	if ref.OrganizationID != current.OrganizationID || ref.StoreID != input.StoreID || ref.Platform != "shein" {
		return record.ErrNotFound
	}
	return fresh.Err()
}
func (a *normalOfflineRecords) ReadOfflinePackage(ctx context.Context, actor record.Actor, id string) (record.Record, error) {
	fresh, current, err := a.fresh(ctx, authz.PermissionListingKitAdminRead)
	if err != nil {
		return record.Record{}, err
	}
	if actor.OrganizationID != current.OrganizationID || actor.UserID != current.UserID {
		return record.Record{}, record.ErrForbidden
	}
	value, err := a.records.ReadOfflinePackage(fresh, current, id)
	if err != nil {
		return record.Record{}, err
	}
	_, _, err = a.fresh(fresh, authz.PermissionListingKitAdminRead)
	if err != nil {
		return record.Record{}, err
	}
	return value, nil
}
func offlineRecordRoutes(app *normalOfflineRecords) []httproute.Descriptor {
	defs := []struct{ method, path, permission string }{
		{http.MethodPost, offlineRecordsPath, authz.PermissionListingKitAdminWrite},
		{http.MethodGet, offlineRecordsPath, authz.PermissionListingKitAdminRead},
		{http.MethodGet, offlineRecordsPath + "/:record_id", authz.PermissionListingKitAdminRead},
		{http.MethodGet, offlineRecordsPath + "/commands/:operation_id", authz.PermissionListingKitAdminRead},
	}
	routes := make([]httproute.Descriptor, 0, len(defs))
	for _, def := range defs {
		routes = append(routes, httproute.Descriptor{Method: def.method, Path: def.path, Module: "offline-shein-records", Permission: def.permission, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: record.Timeout + 2*time.Second, RejectUnreadRequestBody: def.method == http.MethodGet, Handler: func(c *gin.Context) { handleOfflineRecord(c, app) }})
	}
	return routes
}

type offlineInputDTO struct {
	ProductKey      string          `json:"product_key"`
	SnapshotVersion string          `json:"snapshot_version"`
	StoreID         string          `json:"store_id"`
	Country         string          `json:"country"`
	Language        string          `json:"language"`
	Action          contract.Action `json:"action"`
}

func (i offlineInputDTO) domain() (record.Input, error) {
	version, err := strconv.ParseUint(i.SnapshotVersion, 10, 63)
	input := record.Input{ProductKey: i.ProductKey, SnapshotVersion: version, StoreID: i.StoreID, Country: i.Country, Language: i.Language, Action: i.Action}
	if err != nil || strconv.FormatUint(version, 10) != i.SnapshotVersion || input.Validate() != nil {
		return record.Input{}, record.ErrInvalid
	}
	return input, nil
}
func offlineInput(input record.Input) offlineInputDTO {
	return offlineInputDTO{input.ProductKey, strconv.FormatUint(input.SnapshotVersion, 10), input.StoreID, input.Country, input.Language, input.Action}
}

type offlineSavedDTO struct {
	Origin      string             `json:"origin"`
	RecordID    string             `json:"recordId"`
	OperationID string             `json:"operationId"`
	Input       offlineInputDTO    `json:"input"`
	Version     string             `json:"sourceVersion"`
	CreatedAt   time.Time          `json:"createdAt"`
	Diagnostic  sheinDiagnosticDTO `json:"diagnostic"`
}

func offlineSaved(value record.Record) (offlineSavedDTO, error) {
	var d contract.DiagnosticResult
	if json.Unmarshal(value.Diagnostic, &d) != nil || !d.DiagnosticOnly || d.Scope != "shein.offline_package" || d.Target.Marketplace != "shein" || d.Target.Site != "" || d.Action != value.Input.Action || d.RuleVersion != value.RuleRevision || d.Input.BindingVersion != value.PolicyRevision || d.OfflineChecks.Status != value.DiagnosticStatus {
		return offlineSavedDTO{}, record.ErrUnavailable
	}
	expected, err := sheinvalidator.DiagnosticInputDigest(contract.BoundRequest[[]byte]{Input: value.Payload, Target: d.Target, Action: value.Input.Action, RuleVersion: value.RuleRevision, BindingVersion: value.PolicyRevision})
	if err != nil || contract.ValidateBoundInput(d.Input, expected) != nil || d.Freshness.Validate(expected, d.Input.EvaluatedAt) != nil {
		return offlineSavedDTO{}, record.ErrUnavailable
	}
	return offlineSavedDTO{Origin: "saved", RecordID: value.ID, OperationID: value.OperationID, Input: offlineInput(value.Input), Version: value.InputHash, CreatedAt: value.CreatedAt, Diagnostic: projectSheinDiagnostic(d)}, nil
}
func handleOfflineRecord(c *gin.Context, a *normalOfflineRecords) {
	if a == nil {
		writeOfflineRecordError(c, record.ErrUnavailable)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), record.Timeout)
	defer cancel()
	ctx, err := (productReviewCapabilityBinder{now: time.Now}).Bind(ctx, c.GetHeader("Authorization"))
	if err != nil {
		writeOfflineRecordError(c, record.ErrForbidden)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	permission := authz.PermissionListingKitAdminRead
	if c.Request.Method == http.MethodPost {
		permission = authz.PermissionListingKitAdminWrite
	}
	ctx, actor, err := a.fresh(ctx, permission)
	if err != nil {
		writeOfflineRecordError(c, err)
		return
	}
	if c.Request.Method == http.MethodPost {
		if c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery || len(c.Request.Header.Values("Idempotency-Key")) != 1 {
			writeOfflineRecordError(c, record.ErrInvalid)
			return
		}
		var cmd struct {
			CollectionItemID string          `json:"collectionItemId"`
			Input            offlineInputDTO `json:"input"`
		}
		if deadline, ok := ctx.Deadline(); ok {
			_ = http.NewResponseController(c.Writer).SetReadDeadline(deadline)
		}
		raw, e := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 2048))
		var transport net.Error
		if ctx.Err() != nil || errors.Is(e, context.Canceled) || errors.Is(e, context.DeadlineExceeded) || errors.As(e, &transport) && transport.Timeout() {
			writeOfflineRecordError(c, context.DeadlineExceeded)
			return
		}
		if e != nil {
			writeOfflineRecordError(c, record.ErrInvalid)
			return
		}
		strict, e := sigjson.UnmarshalStrict(raw, &cmd, sigjson.DisallowUnknownFields, sigjson.DisallowDuplicateFields)
		input, inputErr := cmd.Input.domain()
		if e != nil || len(strict) > 0 || !collection.ValidID(cmd.CollectionItemID) || inputErr != nil {
			writeOfflineRecordError(c, record.ErrInvalid)
			return
		}
		id, _ := authidentity.AuthenticatedIdentityFromContext(ctx)
		ctx = context.WithValue(ctx, offlineSourceContextKey{}, offlineSourceContext{cmd.CollectionItemID, actor.OrganizationID, actor.UserID, id.EffectiveMemberID})
		receipt, e := a.create.Create(ctx, c.GetHeader("Idempotency-Key"), input)
		if e != nil {
			writeOfflineRecordError(c, e)
			return
		}
		c.JSON(http.StatusCreated, receipt)
		return
	}
	if c.Request.Body != nil {
		one, e := io.ReadAll(io.LimitReader(c.Request.Body, 1))
		if e != nil || len(one) > 0 {
			writeOfflineRecordError(c, record.ErrInvalid)
			return
		}
	}
	var saved record.Record
	if operation := c.Param("operation_id"); operation != "" {
		if c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
			writeOfflineRecordError(c, record.ErrInvalid)
			return
		}
		saved, err = a.records.FindOperation(ctx, actor, operation)
	} else if id := c.Param("record_id"); id != "" {
		if c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
			writeOfflineRecordError(c, record.ErrInvalid)
			return
		}
		saved, err = a.ReadOfflinePackage(ctx, actor, id)
	} else {
		q, e := url.ParseQuery(c.Request.URL.RawQuery)
		if e != nil || len(c.Request.URL.RawQuery) > maxSheinRecordCollectionQueryBytes {
			writeOfflineRecordError(c, record.ErrInvalid)
			return
		}
		query, e := parseSheinRecordPageQuery(q)
		if e != nil {
			writeOfflineRecordError(c, e)
			return
		}
		page, e := a.records.List(ctx, actor, query)
		if e != nil {
			writeOfflineRecordError(c, e)
			return
		}
		if _, _, e = a.fresh(ctx, permission); e != nil {
			writeOfflineRecordError(c, e)
			return
		}
		body, e := marshalSheinRecordPage(page)
		if e != nil || len(body) > maxSheinRecordCollectionResponseBytes {
			writeOfflineRecordError(c, record.ErrUnavailable)
			return
		}
		c.Data(200, "application/json", body)
		return
	}
	if err != nil {
		writeOfflineRecordError(c, err)
		return
	}
	dto, err := offlineSaved(saved)
	if err != nil {
		writeOfflineRecordError(c, err)
		return
	}
	if _, _, err = a.fresh(ctx, permission); err != nil {
		writeOfflineRecordError(c, err)
		return
	}
	wire, err := json.Marshal(dto)
	if err != nil || len(wire) > (2<<20)+4096 {
		writeOfflineRecordError(c, record.ErrUnavailable)
		return
	}
	c.Data(200, "application/json", wire)
}
func writeOfflineRecordError(c *gin.Context, err error) {
	status, code := 503, "DEPENDENCY_UNAVAILABLE"
	switch {
	case errors.Is(err, record.ErrForbidden):
		status, code = 403, "FORBIDDEN"
	case errors.Is(err, record.ErrNotFound):
		status, code = 404, "NOT_FOUND"
	case errors.Is(err, record.ErrInvalid):
		status, code = 400, "INVALID_REQUEST"
	case errors.Is(err, record.ErrConflict):
		status, code = 409, "COMMAND_CONFLICT"
	case errors.Is(err, record.ErrNotReady):
		status, code = 422, "NOT_READY"
	case errors.Is(err, record.ErrTooLarge):
		status, code = 413, "TOO_LARGE"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		status, code = 504, "DEPENDENCY_UNAVAILABLE"
	}
	c.JSON(status, gin.H{"code": code})
}
func validateOfflineRecordDescriptor(route httproute.Descriptor) error {
	for _, expected := range offlineRecordRoutes(nil) {
		if expected.Method == route.Method && expected.Path == route.Path {
			if route.Module != expected.Module || route.Permission != expected.Permission || route.AuthPolicy != expected.AuthPolicy || route.OrganizationAccessPolicy != expected.OrganizationAccessPolicy || route.OrganizationTargetResolver != nil || route.RequestTimeout != expected.RequestTimeout || route.RejectUnreadRequestBody != expected.RejectUnreadRequestBody || route.Handler == nil {
				return errors.New("offline record descriptor boundary changed")
			}
			return nil
		}
	}
	return errors.New("offline record route not admitted")
}
