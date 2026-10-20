package agentconfigpersistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	"testing"
)

func TestArtworkSnapshotAdmissionAndModesAreSeparateFromProductImages(t *testing.T) {
	db, repo, _ := fixture(t)
	ctx := context.Background()
	require.NoError(t, VerifyArtworkConfigurationSchema(ctx, db))
	for _, mode := range []agent.ExecutionMode{agent.ModeDefault, agent.ModeReadOnly, agent.ModeAutoApprove, agent.ModeFullAccess} {
		t.Run(string(mode), func(t *testing.T) {
			scope := agent.Scope{OrganizationID: "artwork-" + string(mode), ActorID: "creator"}
			enable := command(scope, "enable", 0)
			enable.AgentID, enable.Absent = agentconfig.ArtworkAgentID, true
			_, err := repo.Execute(ctx, enable)
			require.NoError(t, err)
			parameters := agentconfig.ArtworkTemplate{Schema: agentconfig.ArtworkParameterSchema, Style: "水彩", Constraints: "不含文字"}
			create := command(scope, "create-template", 0)
			create.AgentID = agentconfig.ArtworkAgentID
			create.Input = agentconfig.TemplateInput{Name: "新图案", TargetPlatform: "sds", Artwork: &parameters}
			receipt, err := repo.Execute(ctx, create)
			require.NoError(t, err)
			identity := agent.ExecutionIdentity{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: "member"}
			subject := agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: agentconfig.ArtworkAgentID}, Definition: commercetool.AgentRef{ID: agentconfig.ArtworkAgentID, Version: agentconfig.ArtworkAgentVersion}}
			configuredMode, err := repo.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: identity, Subject: subject, Key: uuid.NewString(), Mode: mode})
			require.NoError(t, err)
			intent := digest([]byte("creative input"))
			start := agentconfig.ArtworkStartCommand{ExpectedModeRevision: configuredMode.Revision, Scope: scope, MemberID: identity.MemberID, RequestKey: uuid.NewString(), ContextID: "creative-context", RunID: uuid.NewString(), TargetPlatform: "sds", SourceDigest: intent, InputDigest: intent, Template: &agentconfig.TemplateRef{TemplateID: receipt.TemplateID, Revision: "1"}, HardLimits: agentconfig.ImageRunLimits{Images: 1, Points: 100, ElapsedSeconds: 300}}
			snapshot, err := repo.PrepareArtworkConfiguration(ctx, start)
			require.NoError(t, err)
			require.Equal(t, parameters, snapshot.ArtworkParameters)
			loaded, err := repo.LoadArtworkConfiguration(ctx, scope, snapshot.Ref())
			require.NoError(t, err)
			require.Equal(t, snapshot, loaded)
			_, err = repo.LoadImageConfiguration(ctx, scope, snapshot.Ref())
			require.Error(t, err, "POD snapshots cannot authorize the product-image entry point")
			checked, err := repo.CheckArtworkExecutionMode(ctx, identity, snapshot.Ref(), snapshot.ModeSnapshotRef)
			require.NoError(t, err)
			require.Equal(t, mode, checked.Mode)
			admission := agentconfig.ArtworkRunAdmissionCommand{Scope: scope, Snapshot: snapshot.Ref(), MemberID: identity.MemberID, RunID: start.RunID, ConfirmActionID: uuid.NewString(), SourceDigest: intent, InputDigest: intent, PlanDigest: digest([]byte("plan")), QuoteDigest: digest([]byte("quote")), Limits: agentconfig.ImageRunLimits{Images: 1, Points: 30, ElapsedSeconds: 180}}
			original, err := repo.AdmitArtworkRun(ctx, admission, start.HardLimits)
			if mode == agent.ModeReadOnly {
				require.ErrorIs(t, err, agentconfig.ErrForbidden)
				return
			}
			require.NoError(t, err)
			disable := command(scope, "disable", 1)
			disable.AgentID = agentconfig.ArtworkAgentID
			_, err = repo.Execute(ctx, disable)
			require.NoError(t, err)
			_, err = repo.CheckArtworkExecutionMode(ctx, identity, snapshot.Ref(), snapshot.ModeSnapshotRef)
			require.ErrorIs(t, err, agentconfig.ErrChanged)
			replay, err := repo.AdmitArtworkRun(ctx, admission, agentconfig.ImageRunLimits{Images: 1, Points: 1, ElapsedSeconds: 1})
			require.NoError(t, err)
			require.Equal(t, original, replay, "the original receipt is readable after disable; it is not a fresh dispatch permit")
			admission.ConfirmActionID = uuid.NewString()
			_, err = repo.AdmitArtworkRun(ctx, admission, start.HardLimits)
			require.ErrorIs(t, err, agentconfig.ErrConflict)
		})
	}
}

func TestArtworkSchemaInitializerPreservesExistingConfigurationAndDoesNotEnableAnything(t *testing.T) {
	db, repo, _ := fixture(t)
	ctx := context.Background()
	require.NoError(t, db.Exec(`ALTER TABLE agent_configuration.template_revisions
 DROP CONSTRAINT template_revisions_schema_version_check, DROP CONSTRAINT template_revisions_check, DROP CONSTRAINT template_revisions_target_platform_check,
 ADD CONSTRAINT template_revisions_schema_version_check `+materialsSchemaVersionDDL+`,
 ADD CONSTRAINT template_revisions_check `+materialsParametersDDL+`,
 ADD CONSTRAINT template_revisions_target_platform_check CHECK(target_platform IN ('product','shein','temu','amazon'))`).Error)
	require.ErrorIs(t, VerifyArtworkConfigurationSchema(ctx, db), agentconfig.ErrUnavailable)
	start := imageStartFixture(t, repo)
	before, err := repo.PrepareImageConfiguration(ctx, start)
	require.NoError(t, err)
	role := "artwork_runtime_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, db.Exec("CREATE ROLE "+role+" NOLOGIN").Error)
	t.Cleanup(func() { _ = db.Exec("DROP OWNED BY " + role).Error; _ = db.Exec("DROP ROLE " + role).Error })
	require.NoError(t, GrantRuntime(db, role))
	for i := 0; i < 2; i++ {
		require.NoError(t, UpdateArtworkConfigurationSchema(ctx, db, role))
		require.NoError(t, VerifyArtworkConfigurationSchema(ctx, db))
		after, err := repo.LoadImageConfiguration(ctx, start.Scope, before.Ref())
		require.NoError(t, err)
		require.Equal(t, before, after)
	}
	var count int64
	require.NoError(t, db.Table("agent_configuration.organization_agents").Where("agent_id=?", agentconfig.ArtworkAgentID).Count(&count).Error)
	require.Zero(t, count)
}
