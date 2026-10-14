package httpapi

import (
	"context"
	"gorm.io/gorm"
	"sort"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	confighttp "task-processor/internal/agentconfig/httpapi"
	"task-processor/internal/agentcustomization"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/commercetool"
	"task-processor/internal/httproute"
	"task-processor/internal/imageagent"
	texteino "task-processor/internal/integration/agent/einomodel"
	"task-processor/internal/integration/knowledgeauth"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/knowledge"
	"task-processor/internal/listing/materials/tools/materialsinspect"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/listing/readiness/tools/readinessinspect"
	"task-processor/internal/product/asset/tools/assetinspect"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
	"task-processor/internal/product/contentagent"
	"task-processor/internal/product/selection/tools/selectioninspect"
	"task-processor/internal/product/sourcing/tools/sourceevidenceinspect"
	"task-processor/internal/workbenchcontext"
	"time"
)

// productTitleRegistration is the single code-owned registration consumed by
// both the market catalog and the executable runtime. No database allowlist.
func productTitleRegistration() (commercetool.AgentDefinition, []commercetool.Definition) {
	tools := []commercetool.Definition{canonicalinspect.Definition(), sourceevidenceinspect.Definition(), assetinspect.Definition(), readinessinspect.Definition()}
	d := commercetool.AgentDefinition{ID: "product.title.agent", Version: "v1.0.0"}
	for _, t := range tools {
		d.AllowedTools = append(d.AllowedTools, t.Ref)
	}
	return d, tools
}

type productAgentCatalog struct{}

func productImageRegistration() commercetool.AgentDefinition {
	return commercetool.AgentDefinition{ID: agentconfig.ImageAgentID, Version: agentconfig.ImageAgentVersion}
}

func (productAgentCatalog) ReadCatalog(context.Context) ([]agentconfig.CatalogEntry, error) {
	d, _ := productTitleRegistration()
	content, _ := contentagent.Registration()
	entries := []agentconfig.CatalogEntry{
		{Definition: selectioninspect.AgentDefinition(), Name: "选品对比智能体", Description: "比较2–5个真实已保存商品，给出有事实引用的选品建议和缺失信息，不采购或修改商品。", ParameterSchema: agentconfig.SelectionParameterSchema},
		{Definition: commercetool.AgentDefinition{ID: agentconfig.ArtworkAgentID, Version: agentconfig.ArtworkAgentVersion}, Name: "POD设计智能体", Description: "从主题生成全新图案，确认保存后进入模板布局与效果图流程。", ParameterSchema: agentconfig.ArtworkParameterSchema},
		{Definition: materialsinspect.AgentDefinition(), Name: "平台上架资料智能体", Description: "使用标准商品和当前店铺规则完善类目、属性、SKU与已批准图片资料，保存可核对的本地平台草稿。", ParameterSchema: agentconfig.MaterialsParameterSchema},
		{Definition: content, Name: "商品内容完善与本地化智能体", Description: "按明确目标语言和字段生成标题、描述、卖点与SEO关键词建议，按当前权限核对并采用到本地商品。", ParameterSchema: agentconfig.ContentParameterSchema},
		{Definition: productImageRegistration(), Name: "商品图片智能体", Description: "基于真实商品素材规划整套主图和详情图，确认点数后分别生成，人工选择并批准商品图片。", ParameterSchema: agentconfig.ImageParameterSchema},
		{Definition: d, Name: "商品标题优化智能体", Description: "读取商品证据生成标题建议，经过人工审核后显式应用。", ParameterSchema: agentconfig.ParameterSchema},
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Definition.ID < entries[j].Definition.ID })
	return entries, nil
}
func WithAgentConfiguration(db *gorm.DB) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.agentConfigurationDB = db }
}
func buildAgentConfigurationModule(ctx context.Context, db *gorm.DB, resolver organizationIdentityResolver, auth *authz.ListingKitAuthorizer, k *knowledge.Service, runtime *productAgentApplication, private privateModeDeliveryReader, selection *productSelectionApplication, images ...*fullImageApplication) (kernelmodule.Module, error) {
	if len(images) > 1 {
		return nil, agentconfig.ErrUnavailable
	}
	var imageRuntime *fullImageApplication
	if len(images) == 1 {
		imageRuntime = images[0]
	}
	if runtime != nil && (runtime.config.RunDB != db || runtime.store == nil || !runtime.store.UsesPool(db)) {
		return nil, agentconfig.ErrUnavailable
	}
	if e := configstore.VerifySchema(ctx, db); e != nil {
		return nil, e
	}
	repo, e := configstore.New(db)
	if e != nil {
		return nil, e
	}
	knowledgeAuth, e := knowledgeauth.NewAuthorizer(resolver, auth)
	if e != nil {
		return nil, e
	}
	h := &confighttp.Handler{Repository: repo, ModeRepository: repo, Catalog: productAgentCatalog{}, Bind: (productReviewCapabilityBinder{now: time.Now}).Bind}
	h.Authorize = func(ctx context.Context, permissions ...string) (agent.Scope, error) {
		id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
		if !ok || !agent.ValidID(id.EffectiveOrganizationID) || !agent.ValidID(id.UserID) || !agent.ValidID(id.EffectiveMemberID) || id.TenantID != id.EffectiveOrganizationID {
			return agent.Scope{}, agentconfig.ErrForbidden
		}
		for _, p := range permissions {
			if authz.AllowedOrganization(ctx, auth, id.UserID, id.EffectiveOrganizationID, id.Roles, p) {
				return agent.Scope{OrganizationID: id.EffectiveOrganizationID, ActorID: id.UserID}, nil
			}
		}
		return agent.Scope{}, agentconfig.ErrForbidden
	}
	h.ResolveModeSubject = agentModeSubjectResolver(h, private)
	if service, ok := private.(*agentcustomization.Service); ok {
		service.WithExecutionModes(privateQualityMode{repository: repo, resolve: h.ResolveModeSubject, identity: func(ctx context.Context) (agent.ExecutionIdentity, error) {
			return privateQualityExecutionIdentity(ctx, h)
		}})
	}
	h.Knowledge = func(ctx context.Context, scope agent.Scope, id string) error {
		if k == nil {
			return agentconfig.ErrUnavailable
		}
		ctx, e := knowledgeRequestContext(ctx)
		if e != nil {
			return agentconfig.ErrForbidden
		}
		fresh, e := knowledgeAuth.AuthorizeKnowledge(ctx)
		if e != nil || fresh.OrganizationID != scope.OrganizationID || fresh.ActorID != scope.ActorID {
			return agentconfig.ErrForbidden
		}
		base, e := k.GetBase(ctx, fresh, id)
		if e != nil {
			return agentconfig.ErrUnavailable
		}
		if base.OrganizationID != scope.OrganizationID || base.State != knowledge.Active {
			return agentconfig.ErrUnavailable
		}
		return nil
	}
	h.Capabilities = func(ctx context.Context, entry agentconfig.CatalogEntry) []agentconfig.Capability {
		if entry.Definition.ID == agentconfig.SelectionAgentID {
			return productSelectionCapabilities(ctx, selection)
		}
		if entry.Definition.ID == agentconfig.MaterialsAgentID {
			var listing *listingMaterialsApplication
			if runtime != nil {
				listing = runtime.materials
			}
			return listingMaterialsAgentCapabilities(ctx, listing)
		}
		if entry.Definition.ID == agentconfig.ContentAgentID {
			var content *productAgentApplication
			if runtime != nil {
				content = runtime.content
			}
			capabilities := productAgentCapabilities(ctx, resolver, h.Authorize, k, content)
			if capabilities[0].Readiness != "AVAILABLE" {
				capabilities[0].Reason = "内容执行需要独立的内容模型路由与凭据配置"
			}
			if content == nil {
				capabilities[0].Readiness = "NEEDS_CONFIGURATION"
			}
			if err := configstore.VerifyContentConfigurationSchema(ctx, db); err != nil {
				capabilities[0].Readiness, capabilities[0].Reason = "NEEDS_CONFIGURATION", "当前环境尚未安装内容参数载体"
			}
			capabilities[1].Reason = "读取当前企业保存商品的真实来源，执行时重新核实权限与版本"
			return capabilities
		}
		if entry.Definition.ID == agentconfig.ImageAgentID || entry.Definition.ID == agentconfig.ArtworkAgentID {
			capabilities := productImageAgentCapabilities(ctx, h.Authorize)
			if entry.Definition.ID == agentconfig.ArtworkAgentID {
				capabilities[1].Support = "NOT_SUPPORTED"
				capabilities[1].Readiness = "UNAVAILABLE"
				capabilities[1].Reason = "从主题创作全新图案，无需商品原图"
			}
			if imageRuntime != nil && (entry.Definition.ID != agentconfig.ArtworkAgentID || imageRuntime.artwork != nil) {
				capability := &capabilities[0]
				id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
				_, authorization := h.Authorize(ctx, authz.PermissionImageAgentWrite)
				if !ok || authorization != nil {
					capability.Readiness = "REQUIRES_AUTHORIZATION"
					capability.Reason = "需要当前企业图片生成权限"
				} else if !imageRuntime.gate.AllowTenantStart(ctx, id.TenantID) {
					capability.Reason = "当前企业尚未开放图片生成"
				} else {
					_, err := imageRuntime.quotes.ReadImageGenerationQuote(ctx, imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, TenantID: id.TenantID, UserID: id.UserID, MemberID: id.EffectiveMemberID})
					if err == nil {
						capability.Readiness = "AVAILABLE"
						capability.Reason = "可准备整套计划，确认点数后生成"
						if entry.Definition.ID == agentconfig.ArtworkAgentID {
							capability.Reason = "可输入主题准备原创图案报价，确认后生成"
						}
					} else {
						capability.Readiness = "NEEDS_CONFIGURATION"
						capability.Reason = "需要有效的企业图片模型配置与凭据"
					}
				}
			}
			return capabilities
		}
		return productAgentCapabilities(ctx, resolver, h.Authorize, k, runtime)
	}
	h.RunsAvailable = func(_ context.Context, entry agentconfig.CatalogEntry) bool {
		if entry.Definition.ID == agentconfig.SelectionAgentID {
			return selection != nil && selection.store != nil && selection.configuration != nil
		}
		if entry.Definition.ID == agentconfig.MaterialsAgentID {
			return runtime != nil && runtime.materials != nil
		}
		if entry.Definition.ID == agentconfig.ContentAgentID {
			return runtime != nil && runtime.content != nil
		}
		if entry.Definition.ID == agentconfig.ArtworkAgentID {
			return imageRuntime != nil && imageRuntime.artwork != nil
		}
		if entry.Definition.ID == agentconfig.ImageAgentID {
			return imageRuntime != nil && imageRuntime.recent != nil
		}
		return runtime != nil
	}
	h.Recent = func(ctx context.Context, scope agent.Scope, id, cursor string, size int) (any, string, error) {
		if id == agentconfig.SelectionAgentID {
			if _, err := h.Authorize(ctx, authz.PermissionLocalAgentWrite); err != nil {
				return nil, "", err
			}
			if selection == nil || selection.store == nil {
				return nil, "", agentconfig.ErrUnavailable
			}
			fresh, e := selection.currentIdentity(ctx)
			if e != nil || fresh.ActorID != scope.ActorID || fresh.OrganizationID != scope.OrganizationID {
				return nil, "", agentconfig.ErrForbidden
			}
			return repo.Recent(ctx, scope, fresh.MemberID, id, cursor, size, selection.store, func(b agent.Binding) error {
				current, e := selection.inspector.AuthorizeHistory(ctx, b)
				if e != nil {
					return e
				}
				if current.OrganizationID != fresh.OrganizationID || current.ActorID != fresh.ActorID || current.MemberID != fresh.MemberID {
					return agentconfig.ErrForbidden
				}
				return nil
			})
		}
		if id == agentconfig.MaterialsAgentID {
			if runtime == nil || runtime.materials == nil {
				return nil, "", agentconfig.ErrUnavailable
			}
			listing := runtime.materials
			fresh, e := listing.currentIdentity(ctx)
			if e != nil || fresh.ActorID != scope.ActorID || fresh.OrganizationID != scope.OrganizationID {
				return nil, "", agentconfig.ErrForbidden
			}
			return repo.Recent(ctx, scope, fresh.MemberID, id, cursor, size, runtime.store, func(b agent.Binding) error {
				_, _, exact, e := listing.readBase(ctx, b.ContextID)
				if e != nil {
					return e
				}
				if b != exact {
					return agentconfig.ErrConflict
				}
				return nil
			})
		}
		if id == agentconfig.ArtworkAgentID {
			if imageRuntime == nil {
				return nil, "", agentconfig.ErrUnavailable
			}
			reader, ok := imageRuntime.recent.(imageagent.ArtworkRunReader)
			identity, valid := authidentity.AuthenticatedIdentityFromContext(ctx)
			if !ok || !valid || identity.UserID != scope.ActorID || identity.TenantID != scope.OrganizationID {
				return nil, "", agentconfig.ErrForbidden
			}
			return reader.ListArtworkRuns(ctx, imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, TenantID: identity.TenantID, UserID: identity.UserID, MemberID: identity.EffectiveMemberID}, cursor, size)
		}
		if id == agentconfig.ImageAgentID {
			if imageRuntime == nil {
				return nil, "", agentconfig.ErrUnavailable
			}
			return imageRuntime.readRecent(ctx, "", cursor, size)
		}
		selected := runtime
		if id == agentconfig.ContentAgentID && selected != nil {
			selected = runtime.content
		}
		if selected == nil {
			return nil, "", agentconfig.ErrUnavailable
		}
		ctx, e := knowledgeRequestContext(ctx)
		if e != nil {
			return nil, "", agentconfig.ErrForbidden
		}
		fresh, e := selected.freshIdentity(ctx)
		if e != nil || fresh.UserID != scope.ActorID || fresh.TenantID != scope.OrganizationID {
			return nil, "", agentconfig.ErrForbidden
		}
		return repo.Recent(ctx, scope, fresh.EffectiveMemberID, id, cursor, size, selected.store, func(b agent.Binding) error { _, e := selected.Authorize(ctx, b); return e })
	}
	return confighttp.NewModule(h), nil
}

func listingMaterialsAgentCapabilities(ctx context.Context, runtime *listingMaterialsApplication) []agentconfig.Capability {
	now := time.Now().UTC()
	text := agentconfig.Capability{ID: "text.generate", Support: "REQUIRED", Readiness: "NEEDS_CONFIGURATION", Reason: "需要安装平台资料配置约束并配置本企业平台资料模型路由与凭据", ObservedAt: now}
	evidence := agentconfig.Capability{ID: "listing.materials.inspect", Support: "REQUIRED", Readiness: "UNAVAILABLE", Reason: "需要正常供应链、真实店铺及已保存平台草稿", ObservedAt: now}
	if runtime != nil {
		if _, e := runtime.supply.Authorization.Authorize(ctx, preparation.PermissionRead); e == nil {
			evidence.Readiness, evidence.Reason = "AVAILABLE", "运行时核对所选草稿、标准商品、店铺规则和已批准资产"
		} else {
			evidence.Readiness, evidence.Reason = "REQUIRES_AUTHORIZATION", "需要当前企业供应链读取权限"
		}
		if runtime.model != nil && runtime.runtime != nil {
			id, e := runtime.currentIdentity(ctx)
			if e != nil {
				text.Readiness, text.Reason = "REQUIRES_AUTHORIZATION", "需要当前企业智能体使用权限"
			} else {
				switch runtime.model.RouteReadinessForVerifiedOrganization(ctx, id.OrganizationID) {
				case texteino.TextRouteAvailable:
					text.Readiness, text.Reason = "AVAILABLE", "平台资料模型配置已接入，执行时重新核对当前权限和预算"
				case texteino.TextRouteNeedsConfiguration:
					text.Reason = "当前企业平台资料模型凭据尚未配置"
				default:
					text.Readiness, text.Reason = "UNAVAILABLE", "当前企业平台资料执行尚未开放"
				}
			}
		}
	}
	return []agentconfig.Capability{text, evidence, {ID: "knowledge.context", Support: "NOT_SUPPORTED", Readiness: "UNAVAILABLE", Reason: "平台资料依据标准商品与店铺规则，无知识选择入口", ObservedAt: now}, {ID: "platform.write", Support: "NOT_SUPPORTED", Readiness: "UNAVAILABLE", Reason: "只保存本地草稿，真实平台发布另行授权", ObservedAt: now}}
}

func productImageAgentCapabilities(ctx context.Context, authorize func(context.Context, ...string) (agent.Scope, error)) []agentconfig.Capability {
	now := time.Now().UTC()
	evidence := agentconfig.Capability{ID: "product.source-evidence", Support: "REQUIRED", Readiness: "REQUIRES_AUTHORIZATION", Reason: "生成需要当前商品素材的读取权限", ObservedAt: now}
	if _, err := authorize(ctx, authz.PermissionProductSourcingWrite); err == nil {
		evidence.Readiness, evidence.Reason = "AVAILABLE", "可读取当前企业已保存的采集素材，执行时重新核实权限与素材版本"
	}
	return []agentconfig.Capability{
		{ID: "image.generate", Support: "REQUIRED", Readiness: "UNAVAILABLE", Reason: "当前环境尚未接入完整图片执行", ObservedAt: now},
		evidence,
		{ID: "text.generate", Support: "NOT_SUPPORTED", Readiness: "UNAVAILABLE", Reason: "标题优化由商品标题智能体执行", ObservedAt: now},
		{ID: "knowledge.context", Support: "NOT_SUPPORTED", Readiness: "UNAVAILABLE", Reason: "图片配置使用真实商品事实与原始素材", ObservedAt: now},
		{ID: "platform.write", Support: "NOT_SUPPORTED", Readiness: "UNAVAILABLE", Reason: "批准图片保存为商品资产，远端发布由刊登流程执行", ObservedAt: now},
	}
}

func productAgentCapabilities(ctx context.Context, resolver organizationIdentityResolver, authorize func(context.Context, ...string) (agent.Scope, error), k *knowledge.Service, runtime *productAgentApplication) []agentconfig.Capability {
	now := time.Now().UTC()
	text := agentconfig.Capability{ID: "text.generate", Support: "REQUIRED", Readiness: "UNAVAILABLE", Reason: "当前环境尚未开放标题执行", ObservedAt: now}
	if runtime != nil && runtime.model != nil {
		original, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
		capability, bound := ctx.Value(productReviewCapabilityContextKey{}).(productReviewRequestCapability)
		if ok && bound && capability.actorID == original.UserID && capability.effectiveOrganizationID == original.EffectiveOrganizationID {
			fresh, err := resolver.Resolve(ctx, httproute.OrganizationAccessPolicyLiveWrite, workbenchcontext.ResolveInput{Identity: authidentity.AuthenticatedIdentity{UserID: capability.actorID, HomeOrganizationID: capability.homeOrganizationID, TokenExpiresAt: capability.tokenExpiresAt}, BearerToken: capability.bearerToken, RequestedOrganizationID: capability.effectiveOrganizationID})
			allowed := false
			for _, organizationID := range runtime.config.AllowedOrganizationIDs {
				allowed = allowed || organizationID == fresh.EffectiveOrganizationID
			}
			if err == nil && allowed && fresh.UserID == original.UserID && fresh.TenantID == original.EffectiveOrganizationID && fresh.EffectiveOrganizationID == fresh.TenantID {
				status := runtime.model.RouteReadinessForVerifiedOrganization(authidentity.WithAuthenticatedIdentity(ctx, fresh), fresh.EffectiveOrganizationID)
				_, configured := runtime.config.TextPolicies[fresh.EffectiveOrganizationID]
				if !configured && runtime.definition.ID == "product.title.agent" {
					status = texteino.TextRouteNeedsConfiguration
				}
				switch status {
				case texteino.TextRouteAvailable:
					text.Readiness = "AVAILABLE"
					text.Reason = "文本配置已接入，执行时重新确认权限、点数与预算"
				case texteino.TextRouteNeedsConfiguration:
					text.Readiness = "NEEDS_CONFIGURATION"
					text.Reason = "当前企业标题模型凭据需由部署者配置"
				default:
					text.Reason = "当前企业标题执行尚未开放"
				}
			}
		}
	}
	read := "UNAVAILABLE"
	reason := "当前环境未接入企业知识引用"
	if k != nil {
		read = "REQUIRES_AUTHORIZATION"
		reason = "选用知识时重新确认知识权限与可读版本"
		if _, e := authorize(ctx, authz.PermissionWorkbenchKnowledgeRead); e == nil {
			read = "AVAILABLE"
			reason = "可选企业知识引用，执行前显式确认"
		}
	}
	evidence := agentconfig.Capability{ID: "product.source-evidence", Support: "REQUIRED", Readiness: "REQUIRES_AUTHORIZATION", Reason: "标题执行需要1688采集模块权限", ObservedAt: now}
	if _, err := authorize(ctx, sourceevidenceinspect.Definition().Permission.Permission); err == nil {
		evidence.Readiness = "AVAILABLE"
		evidence.Reason = "可读取当前企业已保存的采集证据，执行时重新确认权限"
	}
	return []agentconfig.Capability{text, evidence, {ID: "knowledge.context", Support: "OPTIONAL", Readiness: read, Reason: reason, ObservedAt: now}, {ID: "image.generate", Support: "NOT_SUPPORTED", Readiness: "UNAVAILABLE", Reason: "本智能体未提供图片执行", ObservedAt: now}, {ID: "platform.write", Support: "NOT_SUPPORTED", Readiness: "UNAVAILABLE", Reason: "本智能体不执行远端平台写入", ObservedAt: now}}
}
