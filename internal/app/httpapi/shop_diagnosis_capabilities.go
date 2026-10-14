package httpapi

import (
	"context"
	"task-processor/internal/agentconfig"
	texteino "task-processor/internal/integration/agent/einomodel"
	"time"
)

func shopDiagnosisCapabilities(ctx context.Context, a *shopDiagnosisApplication) []agentconfig.Capability {
	now := time.Now().UTC()
	facts := agentconfig.Capability{ID: "shop.operations-diagnosis.inspect", Support: "REQUIRED", Readiness: "UNAVAILABLE", Reason: "当前环境尚未接入经营财务", ObservedAt: now}
	text := agentconfig.Capability{ID: "text.generate", Support: "REQUIRED", Readiness: "NEEDS_CONFIGURATION", Reason: "需要独立的经营诊断模型路由、凭据与当前预算配置", ObservedAt: now}
	if a != nil {
		if _, e := a.currentIdentity(ctx); e == nil {
			facts.Readiness, facts.Reason = "AVAILABLE", "按当前成员权限读取所选店铺已录入财务与保存的平台观察"
		} else {
			facts.Readiness, facts.Reason = "REQUIRES_AUTHORIZATION", "需要当前企业的经营财务读取权限"
		}
		if a.model != nil {
			if id, e := a.executionIdentity(ctx); e == nil {
				text.Readiness = string(a.model.RouteReadinessForVerifiedOrganization(ctx, id.TenantID))
				if text.Readiness == string(texteino.TextRouteAvailable) {
					text.Reason = "按当前预算生成有事实引用的经营诊断建议"
				}
			} else {
				text.Readiness, text.Reason = "REQUIRES_AUTHORIZATION", "需要当前企业智能体使用与本地分析权限"
			}
		}
	}
	return []agentconfig.Capability{facts, text, {ID: "platform.write", Support: "NOT_SUPPORTED", Readiness: "UNAVAILABLE", Reason: "仅比较与提出建议，不采购、修改或发布商品", ObservedAt: now}}
}
