package httpapi

import (
	"context"
	"strconv"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	imageapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/imageagent"
	listingapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/listing"
	imageworker "github.com/henryxu1008/shuomi-ai-commerce/internal/app/worker/imageagent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/commercetool"
	einoruntime "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/agent/eino"
	model "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/agent/einomodel"
	governed "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/aicapability/einomodel"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/commercetoolauth"
	configstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/agentconfig"
	assetstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset/tools/assetinspect"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog/tools/canonicalinspect"
	"go.opentelemetry.io/otel"
)

type imageAnalysisApplication struct {
	parent    *productAgentApplication
	supply    *listingapp.Application
	assets    asset.ApprovedInventoryReader
	sources   *imageapp.AnalysisSources
	materials imageapp.AnalysisMaterialReader
	canonical *canonicalinspect.Invoker
	approved  *assetinspect.Invoker
	profile   func(context.Context, string) (aicapability.ModelProfile, error)
	limits    agent.Limits
}

func buildImageAnalysisApplication(parent *productAgentApplication, supply *listingapp.Application, images *FullImageSetDependencies, media *collectionSourceMediaDependencies) (*imageAnalysisApplication, error) {
	if parent == nil || parent.config.ContentOnly || parent.config.ContentTitleOnly || parent.config.AssetDB == nil {
		return nil, nil
	}
	a := &imageAnalysisApplication{parent: parent, supply: supply, limits: parent.config.Limits}
	a.limits.ModelCalls = 1
	assets, err := assetstore.NewBoundedApprovedInventoryReader(parent.config.AssetDB, 1<<20)
	if err != nil {
		return nil, err
	}
	a.assets = assets
	a.materials = imageapp.AnalysisMaterialReader{Snapshots: parent.snapshots, ReadPublic: newImageSetSourceByteReader(), ReadOwned: func(ctx context.Context, b agent.Binding, url, owner string, max int64) ([]byte, bool, error) {
		return media.readAuthorizedPublished(ctx, owner, url, max)
	}}
	// The API history configuration carries public URLs, not object storage access.
	if images != nil && images.WorkerConfig != nil && images.WorkerConfig.ImageAgent.ArtifactStore.Enabled {
		materials, e := imageworker.NewImageSetMaterialReader(images.WorkerConfig)
		if e != nil {
			return nil, e
		}
		a.materials.ReadGenerated = materials.Read
	}
	a.sources, err = imageapp.NewAnalysisSources(parent.snapshots, assets, a, &a.materials)
	if err != nil {
		return nil, err
	}
	routes, err := newProductTextRouteResolver(parent.config, nil)
	if err != nil {
		return nil, err
	}
	a.profile = func(ctx context.Context, org string) (aicapability.ModelProfile, error) {
		if _, ok := parent.config.ImageAnalysisPolicies[org]; !ok {
			return aicapability.ModelProfile{}, agent.ErrUnavailable
		}
		r, e := routes.Resolve(ctx, aicapability.TextInputIdentity{OrganizationID: org, Operation: aicapability.OperationProductImageAnalysis})
		if e != nil {
			return aicapability.ModelProfile{}, e
		}
		if !agentconfig.ImageAnalysisProfileAdmissible(r.Profile, a.limits) {
			return aicapability.ModelProfile{}, agent.ErrUnavailable
		}
		return r.Profile, nil
	}
	fresh := analysisPrincipal{a}
	permissions, err := commercetoolauth.NewCasbinAuthorizer(parent.authorizer)
	if err != nil {
		return nil, err
	}
	d, _ := imageapp.ImageAnalysisRegistration(true)
	deps := commercetool.InvocationDependencies{PrincipalResolver: fresh, Authorizer: permissions, Recorder: parent.store, Tracer: otel.Tracer("product-image-analysis"), Now: time.Now, AuditTimeout: 2 * time.Second}
	a.canonical, err = canonicalinspect.NewInvoker(parent.snapshots, d, deps)
	if err != nil {
		return nil, err
	}
	a.approved, err = assetinspect.NewInvoker(parent.snapshots, assets, fresh, d, deps)
	return a, err
}

func (a *imageAnalysisApplication) Authorize(ctx context.Context, b agent.Binding) (agent.Scope, error) {
	if a == nil || !agentconfig.ImageAnalysisBinding(b) {
		return agent.Scope{}, agent.ErrInvalid
	}
	i, err := a.parent.freshWorkbenchIdentity(ctx, authz.PermissionWorkbenchAgentUse)
	if err != nil {
		return agent.Scope{}, err
	}
	ctx = authidentity.WithAuthenticatedIdentity(ctx, i)
	if b.ContextKind == "own_product" {
		err = a.parent.authorizeOwnBinding(ctx, i, b)
	} else {
		var original catalog.PublishedSnapshot
		if b.ContextKind == "acquisition" {
			original, err = a.parent.acquisitionSnapshot(ctx, i, b.ContextID)
		} else {
			if a.supply == nil || a.supply.Sources == nil {
				return agent.Scope{}, agent.ErrUnavailable
			}
			selected, e := a.supply.Sources.Select(ctx, b.ContextID)
			err = e
			if e == nil {
				var ownerOrg string
				owner, source, p, e := selected.Read(ctx)
				err = e
				ownerOrg = owner.OrganizationID
				if err == nil && (ownerOrg != i.TenantID || owner.ActorID != i.UserID || owner.MemberID != i.EffectiveMemberID || source.ID != b.ContextID) {
					err = agent.ErrConflict
				}
				original = p
			}
		}
		if err == nil {
			version, e := strconv.ParseUint(b.CatalogVersion, 10, 63)
			err = e
			if e == nil {
				var exact catalog.PublishedSnapshot
				exact, err = a.parent.snapshots.GetSnapshot(ctx, catalog.SnapshotIdentity{TenantID: i.TenantID, ProductKey: b.ProductKey}, version)
				if err == nil && exact.PublicationID != b.PublicationID {
					err = agent.ErrConflict
				}
				if err == nil {
					err = a.parent.checkAcquisitionLineage(ctx, i, original, exact)
				}
			}
		}
	}
	if err != nil {
		return agent.Scope{}, err
	}
	return agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}, nil
}

type analysisPrincipal struct{ a *imageAnalysisApplication }

func (p analysisPrincipal) ResolvePrincipal(ctx context.Context) (commercetool.Principal, error) {
	return p.ResolveFreshPrincipal(ctx)
}
func (p analysisPrincipal) ResolveFreshPrincipal(ctx context.Context) (commercetool.Principal, error) {
	i, e := p.a.parent.freshIdentity(ctx)
	return commercetool.Principal{TenantID: i.TenantID, UserID: i.UserID, Roles: i.Roles}, e
}

func (a *imageAnalysisApplication) Invoke(ctx context.Context, ref commercetool.ToolRef, meta commercetool.CallMetadata, b agent.Binding) (commercetool.Result, error) {
	if _, err := a.Authorize(ctx, b); err != nil {
		return commercetool.Result{}, err
	}
	switch ref {
	case canonicalinspect.Definition().Ref:
		return a.canonical.Invoke(ctx, meta, canonicalinspect.Input{ProductKey: b.ProductKey, CatalogVersion: b.CatalogVersion})
	case assetinspect.Definition().Ref:
		return a.approved.Invoke(ctx, meta, assetinspect.Input{ProductKey: b.ProductKey, CatalogVersion: b.CatalogVersion, TargetPlatform: "product"})
	}
	return commercetool.Result{}, agent.ErrInvalid
}

func (a *imageAnalysisApplication) runtime(approved bool) (*einoruntime.Runtime, error) {
	d, tools := imageapp.ImageAnalysisRegistration(approved)
	routes, err := newProductTextRouteResolver(a.parent.config, nil)
	if err != nil {
		return nil, err
	}
	executor := &governed.Executor{Ledger: a.parent.config.Ledger, Logger: a.parent.config.TextLogger, Admission: a.parent.textAdmission, Resolve: routes.Resolve, BaseTransport: a.parent.config.TextTransport, Authorize: func(ctx context.Context, in aicapability.TextInputIdentity) error {
		i, e := a.parent.freshIdentity(ctx)
		if e != nil {
			return e
		}
		if i.TenantID != in.OrganizationID || i.UserID != in.ActorID || i.EffectiveMemberID != in.MemberID {
			return agent.ErrUnavailable
		}
		return nil
	}}
	m, err := model.NewImageAnalysisModel(executor, a.parent.configuration, a.profile, d.AllowedTools, a.parent.freshIdentity, a.sources)
	if err != nil {
		return nil, err
	}
	validator, err := imageapp.NewAnalysisResultValidator(a.parent.configuration, a.parent.store, a.parent.config.Ledger, a.sources)
	if err != nil {
		return nil, err
	}
	guard, err := configstore.NewGuard(a.parent.config.RunDB, a.parent.configuration, a.parent.store, func(id, version string) (agent.Limits, error) {
		if id != agentconfig.ImageAgentID || version != agentconfig.ImageAgentVersion {
			return agent.Limits{}, agent.ErrInvalid
		}
		return a.limits, nil
	}, a.parent.currentModeIdentity)
	if err != nil {
		return nil, err
	}
	return einoruntime.New(einoruntime.Config{SingleCallReadOnly: true, PreReadTools: d.AllowedTools, Definition: d, Tools: tools, Model: m, Gateway: a, DomainProposalKind: asset.ImageVisualAnalysisSchema, DomainValidator: validator, Authorizer: a, Store: guard})
}
