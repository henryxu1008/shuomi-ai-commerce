package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/imageagent"
	imagestore "task-processor/internal/imageagent/store"
	"task-processor/internal/product/collection"
)

type artworkAdmissionFixture struct {
	agentconfig.ArtworkConfigurationRepository
	prepares, admissions, modes int
	snapshot                    agentconfig.ArtworkConfigurationSnapshot
	receipt                     agentconfig.ArtworkRunAdmissionReceipt
}

func (f *artworkAdmissionFixture) PrepareArtworkConfiguration(context.Context, agentconfig.ArtworkStartCommand) (agentconfig.ArtworkConfigurationSnapshot, error) {
	f.prepares++
	return f.snapshot, nil
}
func (f *artworkAdmissionFixture) AdmitArtworkRun(context.Context, agentconfig.ArtworkRunAdmissionCommand, agentconfig.ImageRunLimits) (agentconfig.ArtworkRunAdmissionReceipt, error) {
	f.admissions++
	return f.receipt, nil
}
func (f *artworkAdmissionFixture) LoadArtworkConfiguration(context.Context, agent.Scope, agent.ConfigurationSnapshotRef) (agentconfig.ArtworkConfigurationSnapshot, error) {
	return f.snapshot, nil
}
func (f *artworkAdmissionFixture) ReadArtworkRunAdmission(context.Context, agent.Scope, agent.ConfigurationSnapshotRef) (agentconfig.ArtworkRunAdmissionReceipt, error) {
	return f.receipt, nil
}
func (f *artworkAdmissionFixture) CheckArtworkExecutionMode(context.Context, agent.ExecutionIdentity, agent.ConfigurationSnapshotRef, agent.ModeSnapshotRef) (agentconfig.ModeSnapshot, error) {
	f.modes++
	return agentconfig.ModeSnapshot{Mode: agent.ModeDefault}, nil
}

func TestArtworkAdmissionRequiresCollectionPermissionBeforeNewStateAndExecution(t *testing.T) {
	for _, operation := range []string{"prepare", "admit", "execute"} {
		t.Run(operation, func(t *testing.T) {
			fixture := &artworkAdmissionFixture{}
			allowed := false
			calls := 0
			configuration := artworkAdmissionConfiguration{ArtworkConfigurationRepository: fixture, authorize: func(_ context.Context, scope collection.Scope, permission string) error {
				calls++
				require.Equal(t, collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}, scope)
				require.Equal(t, collection.PermissionManage, permission)
				if !allowed {
					return collection.ErrForbidden
				}
				return nil
			}}
			invoke := func() error {
				scope := agent.Scope{OrganizationID: "org", ActorID: "actor"}
				switch operation {
				case "prepare":
					_, err := configuration.PrepareArtworkConfiguration(context.Background(), agentconfig.ArtworkStartCommand{Scope: scope, MemberID: "member"})
					return err
				case "admit":
					_, err := configuration.AdmitArtworkRun(context.Background(), agentconfig.ArtworkRunAdmissionCommand{Scope: scope, MemberID: "member"}, agentconfig.ImageRunLimits{})
					return err
				default:
					_, err := configuration.CheckArtworkExecutionMode(context.Background(), agent.ExecutionIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member"}, agent.ConfigurationSnapshotRef{}, agent.ModeSnapshotRef{})
					return err
				}
			}
			require.ErrorIs(t, invoke(), agentconfig.ErrForbidden)
			require.Equal(t, 0, fixture.prepares+fixture.admissions+fixture.modes, "no downstream state or execution work without the save permission")
			allowed = true
			require.NoError(t, invoke())
			require.Equal(t, 1, fixture.prepares+fixture.admissions+fixture.modes)
			allowed = false
			require.ErrorIs(t, invoke(), agentconfig.ErrForbidden, "authorization is checked again after revocation")
			require.Equal(t, 1, fixture.prepares+fixture.admissions+fixture.modes)
			require.Equal(t, 3, calls)
		})
	}
}

func TestArtworkAdmissionKeepsOriginalReadsAfterPermissionRevocation(t *testing.T) {
	fixture := &artworkAdmissionFixture{snapshot: agentconfig.ArtworkConfigurationSnapshot{ImageConfigurationSnapshot: agentconfig.ImageConfigurationSnapshot{ID: "original-config"}}, receipt: agentconfig.ArtworkRunAdmissionReceipt{ID: "original-admission"}}
	configuration := artworkAdmissionConfiguration{ArtworkConfigurationRepository: fixture, authorize: func(context.Context, collection.Scope, string) error {
		t.Fatal("original reads must not request a new write permission")
		return collection.ErrForbidden
	}}
	snapshot, err := configuration.LoadArtworkConfiguration(context.Background(), agent.Scope{}, agent.ConfigurationSnapshotRef{})
	require.NoError(t, err)
	require.Equal(t, fixture.snapshot, snapshot)
	receipt, err := configuration.ReadArtworkRunAdmission(context.Background(), agent.Scope{}, agent.ConfigurationSnapshotRef{})
	require.NoError(t, err)
	require.Equal(t, fixture.receipt, receipt)
}

func TestArtworkAdmissionFailsClosedWithoutHidingUnavailableAuthority(t *testing.T) {
	for _, test := range []struct {
		name      string
		member    string
		authorize func(context.Context, collection.Scope, string) error
		want      error
	}{
		{"missing authority", "member", nil, agentconfig.ErrUnavailable},
		{"invalid subject", "", func(context.Context, collection.Scope, string) error {
			t.Fatal("invalid subject reached authority")
			return nil
		}, agentconfig.ErrForbidden},
		{"unavailable authority", "member", func(context.Context, collection.Scope, string) error { return collection.ErrUnavailable }, agentconfig.ErrUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := &artworkAdmissionFixture{}
			configuration := artworkAdmissionConfiguration{ArtworkConfigurationRepository: fixture, authorize: test.authorize}
			_, err := configuration.PrepareArtworkConfiguration(context.Background(), agentconfig.ArtworkStartCommand{Scope: agent.Scope{OrganizationID: "org", ActorID: "actor"}, MemberID: test.member})
			require.ErrorIs(t, err, test.want)
			require.Zero(t, fixture.prepares)
		})
	}
}

type artworkAdmissionServiceConfiguration struct{ artworkAdmissionFixture }

func (f *artworkAdmissionServiceConfiguration) PrepareArtworkConfiguration(ctx context.Context, command agentconfig.ArtworkStartCommand) (agentconfig.ArtworkConfigurationSnapshot, error) {
	// Reuse the existing image configuration fixture's exact identity binding.
	in := agentconfig.ImageStartCommand(command)
	in.Template = &agentconfig.TemplateRef{}
	snapshot, err := (imageSetPrepareACKConfiguration{}).PrepareImageConfiguration(ctx, in)
	snapshot.AgentVersion = agentconfig.ArtworkAgentVersion
	f.prepares++
	f.snapshot = agentconfig.ArtworkConfigurationSnapshot{ImageConfigurationSnapshot: snapshot, ArtworkParameters: agentconfig.ArtworkTemplate{Schema: agentconfig.ArtworkParameterSchema, Style: "扁平插画"}}
	return f.snapshot, err
}

func (f *artworkAdmissionServiceConfiguration) AdmitArtworkRun(_ context.Context, command agentconfig.ArtworkRunAdmissionCommand, _ agentconfig.ImageRunLimits) (agentconfig.ArtworkRunAdmissionReceipt, error) {
	f.admissions++
	now := time.Now().UTC().Truncate(time.Microsecond)
	f.receipt = agentconfig.ArtworkRunAdmissionReceipt{ID: uuid.NewString(), Command: agentconfig.ImageRunAdmissionCommand(command), AdmittedAt: now, Deadline: now.Add(time.Duration(command.Limits.ElapsedSeconds) * time.Second)}
	raw, err := json.Marshal(f.receipt)
	if err != nil {
		return agentconfig.ArtworkRunAdmissionReceipt{}, err
	}
	sum := sha256.Sum256(raw)
	f.receipt.Digest = hex.EncodeToString(sum[:])
	return f.receipt, nil
}

func (f *artworkAdmissionServiceConfiguration) ReadArtworkRunAdmission(context.Context, agent.Scope, agent.ConfigurationSnapshotRef) (agentconfig.ArtworkRunAdmissionReceipt, error) {
	if f.receipt.ID == "" {
		return agentconfig.ArtworkRunAdmissionReceipt{}, agentconfig.ErrNotFound
	}
	return f.receipt, nil
}

type artworkAdmissionWorkflow struct {
	imageagent.WorkflowClient
	starts int
	err    error
}

func (w *artworkAdmissionWorkflow) StartManual(context.Context, imageagent.WorkflowStart) error {
	w.starts++
	return w.err
}

func TestArtworkAdmissionBlocksNewPaidWorkButPreservesOriginalRecovery(t *testing.T) {
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member"})
	fixture := &artworkAdmissionServiceConfiguration{}
	allowed := false
	configuration := artworkAdmissionConfiguration{ArtworkConfigurationRepository: fixture, authorize: func(_ context.Context, scope collection.Scope, permission string) error {
		require.Equal(t, collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}, scope)
		require.Equal(t, collection.PermissionManage, permission)
		if !allowed {
			return collection.ErrForbidden
		}
		return nil
	}}
	quotes := &artworkReadinessQuote{}
	workflows := &artworkAdmissionWorkflow{}
	repository := &imageSetPrepareACKRepository{Repository: imagestore.NewMemoryRepository()}
	service, err := imageagent.NewService(repository, workflows, closedImageSetCatalog{}, imageagent.WithOrganizationScope(), imageagent.WithArtworkDependencies(imageagent.ArtworkDependencies{Configuration: configuration, Quotes: quotes, HardLimits: agentconfig.ImageRunLimits{Images: 1, Points: 100, ElapsedSeconds: 300}}))
	require.NoError(t, err)
	input := imageagent.PrepareArtworkInput{RequestID: uuid.NewString(), ContextID: "creative", ExpectedModeRevision: "0", Theme: "原创森林动物"}
	_, err = service.PrepareArtwork(ctx, input)
	require.ErrorIs(t, err, agentconfig.ErrForbidden)
	require.Zero(t, quotes.calls)
	require.Zero(t, repository.initializations)
	allowed = true
	prepared, err := service.PrepareArtwork(ctx, input)
	require.NoError(t, err)
	require.Equal(t, 1, repository.initializations)
	confirm := imageagent.ConfirmArtworkInput{RunID: prepared.Projection.Run.ID, ActionID: uuid.NewString(), ExpectedRevision: 1, PlanDigest: prepared.PlanDigest, QuoteDigest: prepared.QuoteDigest}
	allowed = false
	_, err = service.ConfirmArtwork(ctx, confirm)
	require.ErrorIs(t, err, agentconfig.ErrForbidden)
	require.Zero(t, fixture.admissions)
	require.Zero(t, workflows.starts)
	allowed = true
	workflows.err = errors.New("workflow ACK lost")
	confirmed, err := service.ConfirmArtwork(ctx, confirm)
	require.ErrorIs(t, err, workflows.err)
	require.NoError(t, imageagent.ValidateArtworkAdmission(confirmed.Run, confirmed.Plan))
	allowed = false
	readback, err := service.GetPreparedArtworkByKey(ctx, input.ContextID, input.RequestID)
	require.NoError(t, err)
	require.Equal(t, confirmed.Run.ID, readback.Projection.Run.ID)
	_, err = service.PrepareArtwork(ctx, input)
	require.NoError(t, err, "same-key preparation reads the original without creating new state")
	workflows.err = nil
	replayed, err := service.ConfirmArtwork(ctx, confirm)
	require.NoError(t, err, "original workflow ACK recovery does not issue a new admission")
	require.Equal(t, confirmed.Run.ImageAdmission, replayed.Run.ImageAdmission)
	require.Equal(t, 1, fixture.admissions)
	require.Equal(t, 2, workflows.starts)
	guard := modeBoundImageContexts{artworkConfiguration: configuration}
	identity := imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, TenantID: "org", UserID: "actor", MemberID: "member", RunID: confirmed.Run.ID, BusinessTaskID: input.ContextID}
	require.ErrorIs(t, guard.AuthorizeArtworkExecution(ctx, identity, replayed), agentconfig.ErrForbidden, "restored execution still needs a fresh save grant before paid provider work")
	require.Zero(t, fixture.modes)
	allowed = true
	require.NoError(t, guard.AuthorizeArtworkExecution(ctx, identity, replayed))
	require.Equal(t, 1, fixture.modes)
	changed := replayed
	changed.Run.MemberID = "other-member"
	require.ErrorIs(t, guard.AuthorizeArtworkExecution(ctx, identity, changed), imageagent.ErrCommandBlocked)
	require.Equal(t, 1, fixture.modes, "changed durable subject never reaches the mode check")
	changed = replayed
	changed.Plan.Artwork = imageagent.CloneArtworkPlan(replayed.Plan.Artwork)
	changed.Plan.Artwork.InputDigest = strings.Repeat("f", 64)
	require.Error(t, guard.AuthorizeArtworkExecution(ctx, identity, changed))
}

type artworkReadinessPublisher struct{ imageagent.ArtworkPublisher }
type artworkReadinessQuote struct{ calls int }

func (q *artworkReadinessQuote) ReadImageGenerationQuote(ctx context.Context, identity imageagent.ExecutionIdentity, route *imageagent.ImageRouteSelection) (imageagent.ImageGenerationQuote, error) {
	q.calls++
	return (imageSetPrepareACKQuote{}).ReadImageGenerationQuote(ctx, identity, route)
}

func TestArtworkReadinessRequiresBothPermissionsInTheSameScope(t *testing.T) {
	for _, test := range []struct {
		name                             string
		image, save, otherScope, generic bool
		want                             string
	}{
		{"missing save", true, false, false, false, "REQUIRES_AUTHORIZATION"},
		{"missing image", false, true, false, false, "REQUIRES_AUTHORIZATION"},
		{"both", true, true, false, false, "AVAILABLE"},
		{"different scope", true, true, true, false, "REQUIRES_AUTHORIZATION"},
		{"generic images unchanged", true, false, false, true, "AVAILABLE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member"})
			authorize := func(_ context.Context, permissions ...string) (agent.Scope, error) {
				// Match the real handler's ANY semantics, so passing two permissions cannot masquerade as AND.
				for _, permission := range permissions {
					if permission == authz.PermissionImageAgentWrite && test.image {
						return agent.Scope{OrganizationID: "org", ActorID: "actor"}, nil
					}
					if permission == collection.PermissionManage && test.save {
						org := "org"
						if test.otherScope {
							org = "other"
						}
						return agent.Scope{OrganizationID: org, ActorID: "actor"}, nil
					}
				}
				return agent.Scope{}, agentconfig.ErrForbidden
			}
			quote := &artworkReadinessQuote{}
			runtime := &fullImageApplication{artwork: artworkReadinessPublisher{}, quotes: quote, gate: imageagent.TenantAllowlistStartGate{Enabled: true, AllowedTenantIDs: []string{"org"}}}
			id := agentconfig.ArtworkAgentID
			if test.generic {
				id = agentconfig.ImageAgentID
			}
			capabilities := configuredImageAgentCapabilities(ctx, id, authorize, runtime)
			require.Equal(t, test.want, capabilities[0].Readiness)
			if test.want != "AVAILABLE" {
				require.Zero(t, quote.calls, "missing permission must be rejected before quote/provider dependencies")
			}
		})
	}
}
