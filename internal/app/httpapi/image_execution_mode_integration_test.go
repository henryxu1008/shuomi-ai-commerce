package httpapi

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	"task-processor/internal/imageagent"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	"task-processor/internal/product/asset"
	"testing"
)

func TestImageGenerationAndAdoptionUseActualFrozenModeOwner(t *testing.T) {
	f := newAcquisitionHTTPFixture(t)
	ctx := context.Background()
	require.NoError(t, configstore.InstallSchema(f.owner))
	configuration, err := configstore.New(f.owner)
	require.NoError(t, err)
	scope := agent.Scope{OrganizationID: "B", ActorID: "admin"}
	identity := agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}
	_, err = configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.ImageAgentID, Operation: "enable", Absent: true})
	require.NoError(t, err)
	parameters := agentconfig.SetTemplate{Schema: agentconfig.ImageConfigurationSchema, Mode: "standard", ShareOriginals: true, Background: "white", Language: "en", Carousel: []agentconfig.ContentTask{{ID: "identity", Purpose: "product_identity"}}}
	template, err := configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.ImageAgentID, Operation: "create-template", Input: agentconfig.TemplateInput{Name: "Mode fixture", TargetPlatform: "product", Image: &parameters}})
	require.NoError(t, err)
	subject := agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: agentconfig.ImageAgentID}, Definition: commercetool.AgentRef{ID: agentconfig.ImageAgentID, Version: agentconfig.ImageAgentVersion}}
	for index, value := range []agent.ExecutionMode{agent.ModeDefault, agent.ModeAutoApprove, agent.ModeReadOnly, agent.ModeFullAccess} {
		t.Run(string(value), func(t *testing.T) {
			receipt, err := configuration.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: identity, Subject: subject, Key: uuid.NewString(), Mode: value, ExpectedRevision: uint64(index)})
			require.NoError(t, err)
			start := agentconfig.ImageStartCommand{ExpectedModeRevision: receipt.Revision, Scope: scope, MemberID: identity.MemberID, RequestKey: uuid.NewString(), ContextID: uuid.NewString(), RunID: uuid.NewString(), TargetPlatform: "product", SourceDigest: strings.Repeat("a", 64), InputDigest: strings.Repeat("b", 64), Template: &agentconfig.TemplateRef{TemplateID: template.TemplateID, Revision: "1"}, HardLimits: agentconfig.ImageRunLimits{Images: 32, Points: 100, ElapsedSeconds: 300}}
			snapshot, err := configuration.PrepareImageConfiguration(ctx, start)
			require.NoError(t, err)
			p := imageagent.RunProjection{Run: imageagent.Run{ID: start.RunID, TenantID: scope.OrganizationID, UserID: scope.ActorID, MemberID: identity.MemberID, BusinessTaskID: start.ContextID}, Plan: imageagent.Plan{Revision: 1, Set: &imageagent.ImageSetPlan{Configuration: snapshot.Ref(), ModeSnapshotRef: snapshot.ModeSnapshotRef, InputDigest: snapshot.InputDigest}, Slots: []imageagent.Slot{{ID: "main", Recipe: &imageagent.ImageSlotRecipe{Placement: imageagent.ImagePlacement{Group: "carousel", Order: 1}}}}}, ResultDigest: strings.Repeat("c", 64), Slots: []imageagent.SlotProjection{{Slot: imageagent.Slot{ID: "main", Status: imageagent.SlotStatusAccepted}, Attempt: 1, Candidates: []imageagent.AssetCandidate{{AssetID: "generated"}}}}}
			execution := imageagent.ExecutionIdentity{RunID: p.Run.ID, TenantID: p.Run.TenantID, UserID: p.Run.UserID, MemberID: p.Run.MemberID}
			old := p
			oldSet := *p.Plan.Set
			oldSet.ModeSnapshotRef = agent.ModeSnapshotRef{}
			old.Plan.Set = &oldSet
			_, err = checkImageGenerationMode(ctx, configuration, execution, old, false)
			require.ErrorIs(t, err, imageagent.ErrCommandBlocked, "readable history is never authority for a new generation")
			mode, err := checkImageGenerationMode(ctx, configuration, execution, p, false)
			if value == agent.ModeReadOnly {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, value, mode.Mode)
			}
			command := asset.ImageSetCommand{ApprovingResult: asset.ImageSetResultBinding{RunID: p.Run.ID, PlanRevision: 1, ResultDigest: p.ResultDigest}, Choices: []asset.ImageSetChoice{{Kind: "generated", RunID: p.Run.ID, PlanRevision: 1, ResultDigest: p.ResultDigest, SlotID: "main", Attempt: 1, AssetID: "generated", Presentation: asset.ImagePresentation{Group: "carousel", Order: 1}}}}
			source := asset.SourceSelection{TenantID: p.Run.TenantID, ActorID: p.Run.UserID, MemberID: p.Run.MemberID}
			require.ErrorIs(t, imageSelectionModeAdmission(imageSetHTTPRepository{projection: old}, configuration)(ctx, source, command), imageagent.ErrCommandBlocked, "missing mode ref cannot authorize a new adoption")
			admit := imageSelectionModeAdmission(imageSetHTTPRepository{projection: p}, configuration)
			err = admit(ctx, source, command)
			if value == agent.ModeReadOnly {
				require.Error(t, err)
			} else {
				require.NoError(t, err, "genuine human selection remains admitted")
			}
			if value == agent.ModeFullAccess {
				proof := imagePolicyProvenance(mode, p)
				command.PolicyAdoption = &proof
				require.NoError(t, admit(ctx, source, command), "complete native policy selection")
				command.Choices[0].AssetID = "foreign"
				require.Error(t, admit(ctx, source, command))
				command.Choices[0].AssetID = "generated"
			}
			foreign := source
			foreign.MemberID = "replacement"
			require.Error(t, admit(ctx, foreign, command))
			if index == 3 {
				_, err = configuration.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: identity, Subject: subject, Key: uuid.NewString(), Mode: agent.ModeReadOnly, ExpectedRevision: 4})
				require.NoError(t, err)
				_, err = checkImageGenerationMode(ctx, configuration, execution, p, false)
				require.ErrorIs(t, err, agentconfig.ErrChanged)
				require.ErrorIs(t, admit(ctx, source, command), agentconfig.ErrChanged)
			}
		})
	}
}
