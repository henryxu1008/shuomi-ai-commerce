package openai

import (
	"context"
	"errors"
	"task-processor/internal/shared/aiidentity"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

func TestOrganizationOnlyCredentialResolverNeverUsesMemberOrGlobalCredential(t *testing.T) {
	db := openTestCredentialDB(t)
	writer := NewGormCredentialResolver(db)
	for _, credential := range []AIClientCredential{
		{TenantID: "tenant-a", ClientName: "text", APIKey: "organization-key", BaseURL: "https://organization.example.test/v1", Model: "organization-model", APIStyle: "openai-compatible", Enabled: true},
		{TenantID: "tenant-a", UserID: "member-a", ClientName: "text", APIKey: "member-key", BaseURL: "https://member.example.test/v1", Model: "member-model", APIStyle: "openai-compatible", Enabled: true},
		{TenantID: "tenant-b", UserID: "member-b", ClientName: "text", APIKey: "other-member-key", BaseURL: "https://other.example.test/v1", Model: "other-model", APIStyle: "openai-compatible", Enabled: true},
	} {
		if err := writer.SaveCredential(context.Background(), credential); err != nil {
			t.Fatal(err)
		}
	}
	resolver := NewOrganizationOnlyCredentialResolver(db)
	selected, err := resolver.ResolveClientConfig(aiidentity.WithIdentity(context.Background(), aiidentity.Identity{TenantID: "tenant-a", UserID: "member-a"}), "text")
	if err != nil || selected == nil || selected.Config.APIKey != "organization-key" || selected.Config.Model != "organization-model" {
		t.Fatalf("organization resolution = %#v, %v", selected, err)
	}
	selected, err = resolver.ResolveClientConfig(aiidentity.WithIdentity(context.Background(), aiidentity.Identity{TenantID: "tenant-a"}), "text")
	if err != nil || selected == nil || selected.Config.APIKey != "organization-key" {
		t.Fatalf("operator organization resolution = %#v, %v", selected, err)
	}
	for _, identity := range []aiidentity.Identity{{TenantID: "tenant-b", UserID: "member-b"}, {TenantID: "tenant-c", UserID: "member-c"}} {
		selected, err = resolver.ResolveClientConfig(aiidentity.WithIdentity(context.Background(), identity), "text")
		if !errors.Is(err, ErrClientConfigurationUnavailable) || selected != nil {
			t.Fatalf("missing organization credential for %s = %#v, %v", identity.TenantID, selected, err)
		}
	}
}

func openTestCredentialDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&AIClientCredential{}); err != nil {
		t.Fatalf("migrate credentials: %v", err)
	}
	return db
}
