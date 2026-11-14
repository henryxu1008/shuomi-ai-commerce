package setup

import "task-processor/internal/app/runtime/currentapplication"

func productOwnerModule(id string) bool {
	switch id {
	case "productCollections", "acquisitionPlans", "supplyCompletionDrafts", "supplyChain", "controlledLocalDraft", "pod":
		return true
	}
	return false
}

// A disabled capability can still invalidate the shared Product role's deny
// contract. Never defer an explicit owner rejection until after a restart.
func candidatePermissionBlocks(modules []currentapplication.ModuleInspection, selected map[string]bool) []string {
	productChanged := false
	for id := range selected {
		productChanged = productChanged || productOwnerModule(id)
	}
	result := []string{}
	productBlocked := false
	for _, m := range modules {
		_, changed := selected[m.ID]
		if m.CheckStatus != "rejected" {
			continue
		}
		if productChanged && productOwnerModule(m.ID) {
			if !productBlocked {
				result = append(result, "所选商品功能与现有访问权限不匹配；即使停用也不能继续。请保留原配置并联系维护人员处理")
				productBlocked = true
			}
			continue
		}
		if changed {
			result = append(result, m.Label+"：新设置的访问权限检查未通过，已有系统未改变")
		}
	}
	return result
}
