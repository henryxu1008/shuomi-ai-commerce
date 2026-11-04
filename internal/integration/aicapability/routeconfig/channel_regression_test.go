package routeconfiguration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	d "task-processor/internal/aicapability/routeconfig"
)

// A shared credential is not a model definition. In particular, an otherwise
// admitted model must not require its name to be persisted alongside a Key.
func TestChannelCredentialDoesNotOwnTheImageModel(t *testing.T) {
	catalog, _, ref := sharedCatalogImageFixture(t)

	qualified, err := catalog.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.True(t, qualified.Ready, "the trusted definition owns the model, not the credential row: %s", qualified.Reason)
	require.Equal(t, "qwen-image-3.0", qualified.Model)
}

func TestUndeclaredConnectionStaysUnreadyWithoutQueryingAnEmptyIdentity(t *testing.T) {
	catalog, metadata, ref := sharedCatalogImageFixture(t)
	for _, defs := range catalog.Catalog.definitions {
		def := defs[ref.ID]
		def.ConnectionID = ""
		defs[ref.ID] = def
	}
	q, e := catalog.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, e)
	require.False(t, q.Ready)
	require.Equal(t, "CREDENTIAL_NOT_ADMITTED", q.Reason)
	require.Zero(t, metadata.reads, "a missing declaration is configuration, not a database request")
}
