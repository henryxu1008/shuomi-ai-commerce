package agentconfig_test

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/product/contentagent"
	"testing"
)

func TestImageAnalysisUsesExistingOwnerContextIdentity(t *testing.T) {
	b := agent.Binding{ContextKind: contentagent.OwnProductContext, ContextID: "item", ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "product"}
	require.True(t, agentconfig.ImageAnalysisBinding(b))
	b.ContextKind = "own-product"
	require.False(t, agentconfig.ImageAnalysisBinding(b), "do not create a second alias for the existing owner")
}
