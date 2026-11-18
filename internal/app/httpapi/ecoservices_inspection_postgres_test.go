//go:build integration

package httpapi

import (
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/app/runtime/currentapplication"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	estore "task-processor/internal/integration/persistence/ecoservices"
)

func TestEcoservicesQualificationInspectionChecksRetainedTradeFacts(t *testing.T) {
	ctx := t.Context()
	admin, connection := commercialPostgres(t, "postgres:16-alpine")
	require.NoError(t, admin.Exec("CREATE DATABASE ecoservices").Error)
	require.NoError(t, admin.Exec("CREATE ROLE ecoservices_runtime LOGIN PASSWORD 'fixture-eco-role'").Error)
	require.NoError(t, admin.Exec("REVOKE CREATE,TEMP ON DATABASE ecoservices FROM PUBLIC").Error)

	ownerConfig := *connection
	ownerConfig.Database, ownerConfig.User, ownerConfig.Password = "ecoservices", "issue347", "issue347-fixture"
	owner := openCommercialFixtureReader(t, &ownerConfig)
	require.NoError(t, owner.Exec("REVOKE ALL ON SCHEMA public FROM PUBLIC").Error)
	require.NoError(t, estore.Install(ctx, owner))
	require.NoError(t, estore.GrantRuntime(ctx, owner))

	roleConfig := ownerConfig
	roleConfig.User, roleConfig.Password = "ecoservices_runtime", "fixture-eco-role"
	role := openCommercialFixtureReader(t, &roleConfig)
	cfg := &currentapplication.Config{
		Ecoservices: &currentapplication.EcoservicesConfig{
			Enabled: true, NonPaymentOnly: true,
			Database: currentapplication.DatabaseConfig{
				Host: roleConfig.Host, Port: roleConfig.Port, Database: roleConfig.Database,
				User: roleConfig.User, Password: roleConfig.Password, MaxConnections: 2,
			},
		},
	}
	check := func() currentapplication.ModuleInspection {
		for _, m := range currentapplication.InspectModules(ctx, cfg, nil) {
			if m.ID == "ecoservices" {
				return m
			}
		}
		t.Fatal("ecoservices missing from native inspection")
		return currentapplication.ModuleInspection{}
	}
	buildQualification := func() error {
		handler, recovery, err := buildEcoservices(ctx, EcoservicesDependencies{NonPaymentOnly: true, DB: role, Objects: &combinationObjects{}}, nil, nil, authz.DefaultListingKitAuthorizer(), &config.Config{})
		if err == nil {
			require.NotNil(t, handler)
			require.Nil(t, recovery, "qualification mode has no original-command recovery")
		}
		return err
	}

	control := check()
	require.Equal(t, "passed", control.CheckStatus)
	require.True(t, control.Ready)
	require.NoError(t, buildQualification())
	t.Log("empty qualification database passed inspection and startup checks")

	require.NoError(t, owner.Exec("INSERT INTO public.ecoservices_operations (organization_id,kind,key) VALUES (?,?,?)", "synthetic-organization", "confirm_quote", "synthetic-retained-command").Error)
	after := commercialOwnerTableSnapshot(t, owner)
	startupErr := buildQualification()
	require.ErrorContains(t, startupErr, "ecoservices qualification mode cannot consume retained merchant or service facts")

	report := check()
	require.Equal(t, "rejected", report.CheckStatus, "qualification inspection must reject the same database as startup")
	require.NotEqual(t, "verified", report.Installed)
	require.False(t, report.Ready)

	cfg.Ecoservices.Enabled = false
	report = check()
	require.Equal(t, "passed", report.CheckStatus, "the qualification guard must not block disabling the module")
	require.False(t, report.Ready)

	cfg.Ecoservices.Enabled = true
	cfg.Ecoservices.NonPaymentOnly = false
	require.Equal(t, "passed", check().CheckStatus, "full-mode inspection keeps the existing owner-permission check")
	require.Equal(t, after, commercialOwnerTableSnapshot(t, owner))
	t.Log("retained trade facts rejected by qualification inspection and startup; read-only checks preserved database contents")
}
