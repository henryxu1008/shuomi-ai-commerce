package httpapi

import (
	"context"
	"gorm.io/gorm"
	listingapp "task-processor/internal/app/listing"
	listinghttp "task-processor/internal/app/listing/httpapi"
	"task-processor/internal/authz"
	completionstore "task-processor/internal/integration/persistence/listing/completiondraft"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"time"
)

type ControlledLocalDraftDependencies struct {
	Policy  record.LocalTrialPolicy
	AssetDB *gorm.DB
}

func WithControlledLocalDraft(d ControlledLocalDraftDependencies) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.controlledLocalDrafts++; o.controlledLocalDraft = &d }
}

type controlledSources struct {
	base   record.TargetSourceSelector
	policy record.LocalTrialPolicy
}

func (s controlledSources) Select(ctx context.Context, id string) (preparation.AuthorizedSource, error) {
	if id != s.policy.Binding.SourceID {
		return preparation.AuthorizedSource{}, record.ErrForbidden
	}
	v, err := s.base.Select(ctx, id)
	if err != nil {
		return v, err
	}
	owner, source, _, err := v.Read(ctx)
	if err != nil || owner != s.policy.Scope || source.Source.ProductKey != s.policy.ProductKey || source.Source.Version != s.policy.ProductVersion {
		return preparation.AuthorizedSource{}, record.ErrForbidden
	}
	return v, nil
}

type controlledRules struct {
	policy record.LocalTrialPolicy
	stores interface {
		ReadControlledLocalStore(context.Context, string, string) (*storecenter.Store, error)
	}
}

func (r controlledRules) ReadTargetRules(ctx context.Context, scope collection.Scope, id string, _ goods.OfficialDraftInput) (storecenter.ProductMerchantBinding, goods.OfficialRuleSnapshot, error) {
	if scope != r.policy.Scope || id != r.policy.Binding.StoreID {
		return storecenter.ProductMerchantBinding{}, goods.OfficialRuleSnapshot{}, record.ErrForbidden
	}
	store, err := r.stores.ReadControlledLocalStore(ctx, scope.OrganizationID, id)
	if err != nil || store == nil {
		return storecenter.ProductMerchantBinding{}, goods.OfficialRuleSnapshot{}, record.ErrForbidden
	}
	// Current native Store facts, never an artificial official binding.
	binding := storecenter.ProductMerchantBinding{Platform: storecenter.PlatformShein, OrganizationID: scope.OrganizationID, StoreID: id, Site: "shein-us", StoreVersion: store.Version(), ApplicationType: storecenter.ApplicationSelfOperated, SupplierIdentityHash: collection.Digest([]string{"local_trial", scope.OrganizationID, id})}
	return binding, listingapp.ControlledLocalRules(), nil
}

type controlledNoImages struct{}

func (controlledNoImages) Probe(context.Context, asset.ApprovedAsset, int) (goods.OfficialImageObservation, error) {
	return goods.OfficialImageObservation{}, record.ErrForbidden
}

func buildControlledLocalDraftModule(ctx context.Context, productDB, storeDB *gorm.DB, d ControlledLocalDraftDependencies, deps routeAuthDependencies, permissions *authz.ListingKitAuthorizer, features ...bool) (listingModule, error) {
	var empty listingModule
	p := d.Policy
	if p.Scope.Validate() != nil || !p.Binding.Valid() || d.AssetDB == nil || storeDB == nil {
		return empty, record.ErrForbidden
	}
	if err := assetstore.VerifySourceRuntimePermissions(ctx, d.AssetDB); err != nil {
		return empty, err
	}
	core, err := buildNativeDraftReadCore(ctx, productDB, deps, permissions, nil, features...)
	if err != nil {
		return empty, err
	}
	stores, err := storecenter.NewMemberScopedStoreRepository(storeDB, currentStoreMemberAuthorizer{authorizer: permissions})
	if err != nil {
		return empty, err
	}
	core.app.ValidateDraftRead = func(ctx context.Context, scope collection.Scope, v record.TargetRecord) error {
		if scope != p.Scope || v.Source.ID != p.Binding.SourceID || v.Input.StoreID != p.Binding.StoreID || !record.SameLocalTrial(v.LocalTrial, &p.Binding) || !record.ValidPlatformRecord(v) {
			return record.ErrForbidden
		}
		_, err := stores.Get(ctx, scope.OrganizationID, p.Binding.StoreID)
		return err
	}
	if p.Enabled {
		rules := controlledRules{p, stores}
		assets, e := assetstore.NewBoundedApprovedInventoryReader(d.AssetDB, record.MaxPayloadBytes)
		if e != nil {
			return empty, e
		}
		versions, e := completionstore.NewRepository(ctx, productDB)
		if e != nil {
			return empty, e
		}
		repository, ok := core.app.Records.(record.TargetRepository)
		if !ok {
			return empty, record.ErrUnavailable
		}
		targets, e := record.NewLocalTrialTargetService(record.TargetDependencies{Sources: controlledSources{core.app.Sources, p}, Products: core.app.Products, Assets: assets, Rules: rules, CompletionVersions: versions, Records: repository, Images: controlledNoImages{}, Authorizer: core.app.Authorization}, p)
		if e != nil {
			return empty, e
		}
		core.app.Targets = targets
		core.app.Rules = rules
		core.app.ValidateTargetInput = func(scope collection.Scope, in record.TargetInput) error {
			if scope != p.Scope || in.SourceID != p.Binding.SourceID || in.StoreID != p.Binding.StoreID || in.EffectiveVersion != p.ProductVersion || in.ApplyReceiptID != "" || in.ColorPartition != nil || in.Temu != nil || in.TikTok != nil || in.Walmart != nil || in.TemuRules != nil || in.TikTokRules != nil || in.WalmartRules != nil {
				return record.ErrForbidden
			}
			return nil
		}
	}
	binder := productReviewCapabilityBinder{now: time.Now}
	return listingModule{app: core.app, routes: listinghttp.ControlledLocalDraftRoutes(core.app, binder.Bind, p.Enabled)}, nil
}
