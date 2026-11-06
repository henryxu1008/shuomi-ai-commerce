package httpapi

import (
	"context"
	"task-processor/internal/agentconfig"
	texteino "task-processor/internal/integration/agent/einomodel"
	"time"
)

func marketResearchCapabilities(ctx context.Context, a *marketResearchApplication) []agentconfig.Capability {
	now := time.Now().UTC()
	facts := agentconfig.Capability{ID: "product.research.inspect", Support: "REQUIRED", Readiness: "UNAVAILABLE", Reason: "当前环境尚未接入项目与研究资料", ObservedAt: now}
	text := agentconfig.Capability{ID: "text.generate", Support: "REQUIRED", Readiness: "NEEDS_CONFIGURATION", Reason: "需要独立的研究分析模型路由、凭据与当前预算配置", ObservedAt: now}
	if a != nil {
		if _, e := a.currentIdentity(ctx); e == nil {
			facts.Readiness, facts.Reason = "AVAILABLE", "按当前成员权限读取所有候选的真实原始版本事实"
		} else {
			facts.Readiness, facts.Reason = "REQUIRES_AUTHORIZATION", "需要当前企业的智能体使用和项目与研究资料读取权限"
		}
		if a.model != nil {
			if id, e := a.executionIdentity(ctx); e == nil {
				text.Readiness = string(a.model.RouteReadinessForVerifiedOrganization(ctx, id.TenantID))
				if text.Readiness == string(texteino.TextRouteAvailable) {
					text.Reason = "按当前预算生成有事实引用的研究分析建议"
				}
			} else {
				text.Readiness, text.Reason = "REQUIRES_AUTHORIZATION", "需要当前企业智能体使用与本地分析权限"
			}
		}
	}
	return []agentconfig.Capability{facts, text, {ID: "platform.write", Support: "NOT_SUPPORTED", Readiness: "UNAVAILABLE", Reason: "仅比较与提出建议，不采购、修改或发布商品", ObservedAt: now}}
}
