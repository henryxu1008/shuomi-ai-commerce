package assetinspect

import (
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/commercetool"
	"time"
)

// Definition retains the current Catalog read policy, with no execution or
// approval capability. Fresh principal resolution is required at assembly.
func Definition() commercetool.Definition {
	return commercetool.Definition{
		Ref: commercetool.ToolRef{ID: "product.asset.inspect", Version: "v1.0.0"}, Capability: "product.asset", Owner: "product.asset",
		Description: "Read exact product and approved asset facts for one explicitly selected platform.",
		InputSchema: InputSchema(), OutputSchema: OutputSchema(), Risk: commercetool.RiskRead,
		Permission:  commercetool.PermissionRequirement{Permission: authz.PermissionLocalAgentWrite},
		SideEffects: commercetool.SideEffectPolicy{Mode: commercetool.SideEffectNone}, Idempotency: commercetool.IdempotencyPolicy{Mode: commercetool.IdempotencyDeterministic},
		Timeout: commercetool.TimeoutPolicy{Duration: 3 * time.Second}, Retry: commercetool.RetryPolicy{Owner: commercetool.RetryOwnerCaller}, Usage: commercetool.UsagePolicy{Owner: commercetool.UsageOwnerUnmetered},
	}
}
