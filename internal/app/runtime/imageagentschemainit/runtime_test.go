package imageagentschemainit

import (
	"context"
	"testing"

	"task-processor/internal/app/runtime/currentapplication"
	"task-processor/internal/core/config"
)

func TestImageAgentGrantRequiresExplicitMatchingOwnerDatabase(t *testing.T) {
	if err := GrantImageAgentRuntime(context.Background(), "config/config-dev.yaml", ""); err == nil {
		t.Fatal("grant mode accepted an implicit legacy database config")
	}
	owner := &config.Config{Database: &config.DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "owner", Database: "image_owner"}}
	current := &currentapplication.Config{ImageAgent: &currentapplication.ImageAgentConfig{Database: currentapplication.DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "image_agent_runtime", Database: "image_owner"}}}
	if err := validateImageAgentGrantTarget(owner, current); err != nil {
		t.Fatalf("matching owner rejected: %v", err)
	}
	for _, change := range []func(){
		func() { current.ImageAgent.Database.Database = "other" },
		func() { current.ImageAgent.Database.Host = "other" },
		func() { current.ImageAgent.Database.Port = 5433 },
	} {
		original := current.ImageAgent.Database
		change()
		if err := validateImageAgentGrantTarget(owner, current); err == nil {
			t.Fatal("grant accepted a mismatched owner database")
		}
		current.ImageAgent.Database = original
	}
	owner.Database.User = "image_agent_runtime"
	if err := validateImageAgentGrantTarget(owner, current); err == nil {
		t.Fatal("runtime credential used as owner installer")
	}
	owner.Database.User = "owner"
	current.ImageAgent = nil
	if err := validateImageAgentGrantTarget(owner, current); err == nil {
		t.Fatal("grant accepted manifest without ImageAgent owner")
	}
}

func TestImageAgentWorkerGrantRequiresExplicitMatchingOwnerDatabase(t *testing.T) {
	if err := GrantImageAgentWorkerRuntime(context.Background(), "config/config-dev.yaml", ""); err == nil {
		t.Fatal("worker grant accepted an implicit legacy database config")
	}
	owner := &config.Config{Database: &config.DatabaseConfig{Host: "127.0.0.1", Port: 5437, User: "postgres", Database: "image_agent"}}
	current := &currentapplication.Config{ImageAgent: &currentapplication.ImageAgentConfig{Database: currentapplication.DatabaseConfig{Host: "127.0.0.1", Port: 5437, User: "image_agent_runtime", Database: "image_agent"}}}
	if err := validateImageAgentGrantTarget(owner, current); err != nil {
		t.Fatalf("matching worker owner rejected: %v", err)
	}
	owner.Database.Database = "legacy_product"
	if err := validateImageAgentGrantTarget(owner, current); err == nil {
		t.Fatal("worker grant accepted wrong owner database")
	}
}

func TestImageAgentOrganizationInitRequiresExplicitManifestAndOwner(t *testing.T) {
	if err := InitializeOrganizationImageAgent(context.Background(), "config/config-dev.yaml", ""); err == nil {
		t.Fatal("organization image agent init accepted legacy default config")
	}
	if err := InitializeOrganizationImageAgent(context.Background(), "C:/missing-owner.yaml", ""); err == nil {
		t.Fatal("organization image agent init accepted missing current owner manifest")
	}
}
