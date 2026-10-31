package currentapplication

import (
	"path/filepath"
	"testing"
)

func TestFulfillmentRequiresSeparateEnterpriseSitesAndPrivateCredentials(t *testing.T) {
	cfg := supplyRuntimeConfig(t)
	cfg.Fulfillment = &FulfillmentConfig{TemporalAddress: "127.0.0.1:7233", TemporalNamespace: "default", Sites: []FulfillmentSiteConfig{{Alias: "site-a", Organization: "org-a", Origin: "https://erp-a.example.invalid", APIKey: "test-key", APISecretFile: filepath.Join(t.TempDir(), "private-secret"), SupplyMovement: "owned_transfer"}}}
	if err := cfg.validateFulfillment(); err != nil {
		t.Fatal(err)
	}
	original := cfg.Fulfillment.Sites[0]
	for _, bad := range []FulfillmentSiteConfig{
		{Alias: "site-b", Organization: "org-a", Origin: "https://erp-b.example.invalid", APIKey: "test-key-b", APISecretFile: filepath.Join(t.TempDir(), "other"), SupplyMovement: "owned_transfer"},
		{Alias: "site-b", Organization: "org-b", Origin: original.Origin, APIKey: "test-key-b", APISecretFile: filepath.Join(t.TempDir(), "other"), SupplyMovement: "owned_transfer"},
		{Alias: "site-b", Organization: "org-b", Origin: "https://erp-b.example.invalid", APIKey: "test-key-b", APISecretFile: original.APISecretFile, SupplyMovement: "owned_transfer"},
	} {
		cfg.Fulfillment.Sites = []FulfillmentSiteConfig{original, bad}
		if cfg.validateFulfillment() == nil {
			t.Fatal("shared enterprise binding accepted", bad.Alias)
		}
	}
	cfg.Fulfillment.Sites = []FulfillmentSiteConfig{original}
	cfg.Fulfillment.Sites[0].Origin = "http://127.0.0.1:46680"
	if cfg.validateFulfillment() == nil {
		t.Fatal("spike HTTP origin entered serving configuration")
	}
	cfg.Fulfillment.Sites[0] = original
	cfg.Identity.TenantDirectoryToken = ""
	if cfg.validateFulfillment() == nil {
		t.Fatal("missing exact membership authority accepted")
	}
}
