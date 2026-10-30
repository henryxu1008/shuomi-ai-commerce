package httpapi

import (
	"context"
	"errors"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/imageagent"
	"task-processor/internal/product/collection"
)

type artworkAdmissionConfiguration struct {
	agentconfig.ArtworkConfigurationRepository
	authorize func(context.Context, collection.Scope, string) error
}

func (c artworkAdmissionConfiguration) PrepareArtworkConfiguration(ctx context.Context, command agentconfig.ArtworkStartCommand) (agentconfig.ArtworkConfigurationSnapshot, error) {
	if err := c.requireSavePermission(ctx, command.Scope, command.MemberID); err != nil {
		return agentconfig.ArtworkConfigurationSnapshot{}, err
	}
	return c.ArtworkConfigurationRepository.PrepareArtworkConfiguration(ctx, command)
}

func (c artworkAdmissionConfiguration) AdmitArtworkRun(ctx context.Context, command agentconfig.ArtworkRunAdmissionCommand, limits agentconfig.ImageRunLimits) (agentconfig.ArtworkRunAdmissionReceipt, error) {
	if err := c.requireSavePermission(ctx, command.Scope, command.MemberID); err != nil {
		return agentconfig.ArtworkRunAdmissionReceipt{}, err
	}
	return c.ArtworkConfigurationRepository.AdmitArtworkRun(ctx, command, limits)
}

func (c artworkAdmissionConfiguration) CheckArtworkExecutionMode(ctx context.Context, identity agent.ExecutionIdentity, config agent.ConfigurationSnapshotRef, mode agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error) {
	if err := c.requireSavePermission(ctx, agent.Scope{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID}, identity.MemberID); err != nil {
		return agentconfig.ModeSnapshot{}, err
	}
	return c.ArtworkConfigurationRepository.CheckArtworkExecutionMode(ctx, identity, config, mode)
}

// Only new preparation/admission and paid execution need the live save grant.
// Original configuration/admission reads keep the repository's exact ownership
// checks, so a revoked write grant cannot strand an existing receipt or result.
func (c artworkAdmissionConfiguration) requireSavePermission(ctx context.Context, scope agent.Scope, memberID string) error {
	subject := collection.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: memberID}
	if subject.Validate() != nil {
		return agentconfig.ErrForbidden
	}
	if c.ArtworkConfigurationRepository == nil || c.authorize == nil {
		return agentconfig.ErrUnavailable
	}
	err := c.authorize(ctx, subject, collection.PermissionManage)
	if errors.Is(err, collection.ErrForbidden) {
		return agentconfig.ErrForbidden
	}
	if errors.Is(err, collection.ErrUnavailable) {
		return agentconfig.ErrUnavailable
	}
	return err
}

func configuredImageAgentCapabilities(ctx context.Context, agentID string, authorize func(context.Context, ...string) (agent.Scope, error), runtime *fullImageApplication) []agentconfig.Capability {
	capabilities := productImageAgentCapabilities(ctx, authorize)
	if agentID == agentconfig.ArtworkAgentID {
		capabilities[1].Support = "NOT_SUPPORTED"
		capabilities[1].Readiness = "UNAVAILABLE"
		capabilities[1].Reason = "从主题创作全新图案，无需商品原图"
	}
	if runtime != nil && (agentID != agentconfig.ArtworkAgentID || runtime.artwork != nil) {
		capability := &capabilities[0]
		id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
		scope, authorization := authorize(ctx, authz.PermissionImageAgentWrite)
		if !ok || authorization != nil {
			capability.Readiness = "REQUIRES_AUTHORIZATION"
			capability.Reason = "需要当前企业图片生成权限"
		} else if agentID == agentconfig.ArtworkAgentID && !artworkSaveAuthorized(ctx, authorize, scope, id) {
			capability.Readiness = "REQUIRES_AUTHORIZATION"
			capability.Reason = "需要当前企业资料管理权限以保存生成图案"
		} else if !runtime.gate.AllowTenantStart(ctx, id.TenantID) {
			capability.Reason = "当前企业尚未开放图片生成"
		} else {
			_, err := runtime.quotes.ReadImageGenerationQuote(ctx, imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, TenantID: id.TenantID, UserID: id.UserID, MemberID: id.EffectiveMemberID}, nil)
			if err == nil {
				capability.Readiness = "AVAILABLE"
				capability.Reason = "可准备整套计划，确认点数后生成"
				if agentID == agentconfig.ArtworkAgentID {
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

func artworkSaveAuthorized(ctx context.Context, authorize func(context.Context, ...string) (agent.Scope, error), imageScope agent.Scope, identity authidentity.AuthenticatedIdentity) bool {
	// The handler accepts ANY of its permissions, so these grants must be read
	// separately and bound to the same authenticated organization and actor.
	saveScope, err := authorize(ctx, collection.PermissionManage)
	return err == nil && saveScope == imageScope && imageScope == (agent.Scope{OrganizationID: identity.TenantID, ActorID: identity.UserID}) && identity.EffectiveOrganizationID == identity.TenantID
}
