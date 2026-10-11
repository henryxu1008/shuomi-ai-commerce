package currentapplication

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCompletionDraftConfigDoesNotRequireStoreCollectorAssetOrWorkflow(t *testing.T) {
	cfg := runtimeTestConfig()
	product := cfg.SourceAccountDatabase
	product.Database = "completion_product"
	product.User = "source_acquisition_runtime"
	cfg.ProductAcquisitionDatabase = &product
	resource:=cfg.SourceAccountDatabase;resource.Database="completion_resources";resource.User="commercial_owner_runtime";cfg.CommercialOwnerDatabase=&resource
	cfg.ProductCollections = true
	cfg.SupplyCompletionDrafts = true
	cfg.Identity.TenantDirectoryToken = "synthetic-directory-token"
	require.NoError(t, cfg.validate())
	require.Nil(t, cfg.StoreCenter)
	require.Nil(t, cfg.BrowserCollector)
	require.Nil(t, cfg.SupplyChain)
	cfg.ProductCollections = false
	require.Error(t, cfg.validate())
	cfg.ProductCollections = true
	cfg.Identity.TenantDirectoryToken = ""
	require.Error(t, cfg.validate())
}
