package agentconfig_test

import (
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/contentagent"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestImageAnalysisUsesExistingOwnerContextIdentity(t *testing.T) {
	b := agent.Binding{ContextKind: contentagent.OwnProductContext, ContextID: "item", ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "product"}
	require.True(t, agentconfig.ImageAnalysisBinding(b))
	b.ContextKind = "own-product"
	require.False(t, agentconfig.ImageAnalysisBinding(b), "do not create a second alias for the existing owner")
}
