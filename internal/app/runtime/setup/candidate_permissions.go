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
				result = append(result, "商品资料共用运行权限与候选功能集合不一致；停用也不会自动修改原权限，须保留原配置并由原owner维护")
				productBlocked = true
			}
			continue
		}
		if changed {
			result = append(result, m.Label+"：候选的原生运行权限检查拒绝，原实例未改变")
		}
	}
	return result
}
