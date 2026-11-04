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
