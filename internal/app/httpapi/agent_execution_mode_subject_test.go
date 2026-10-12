package httpapi

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	confighttp "task-processor/internal/agentconfig/httpapi"
	"task-processor/internal/agentcustomization"
	"task-processor/internal/authidentity"
)

type modeRegisteredProbe struct{ agentconfig.Repository }

func (modeRegisteredProbe) ReadAgent(context.Context, agent.Scope, string) (agentconfig.OrganizationAgent, error) {
	return agentconfig.OrganizationAgent{Activation: "ENABLED"}, nil
}

type modeDeliveryProbe struct {
	organization, id string
	calls            int
}

func (r *modeDeliveryProbe) Delivery(_ context.Context, s agentcustomization.Scope, id string) (agentcustomization.Delivery, error) {
	r.calls++
	if s.Platform || s.OrganizationID != r.organization || id != r.id {
		return agentcustomization.Delivery{}, agentcustomization.ErrNotFound
	}
	return agentcustomization.Delivery{ID: id, OrganizationID: s.OrganizationID, Definition: agentcustomization.QualityDefinition, Version: agentcustomization.QualityVersion}, nil
}

func TestExecutionModeSubjectUsesActualPrivateDeliveryOwner(t *testing.T) {
	h := &confighttp.Handler{Repository: modeRegisteredProbe{}, Catalog: productAgentCatalog{}, Authorize: func(context.Context, ...string) (agent.Scope, error) {
		return agent.Scope{OrganizationID: "org", ActorID: "actor"}, nil
	}}
	private := &modeDeliveryProbe{organization: "org", id: uuid.NewString()}
	resolve := agentModeSubjectResolver(h, private)
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: "actor", TenantID: "org", EffectiveOrganizationID: "org", EffectiveMemberID: "member"})
	identity, subject, err := resolve(ctx, agentconfig.ModeSubject{Kind: agentconfig.PrivateDeliverySubject, ID: private.id}, true)
	require.NoError(t, err)
	require.Equal(t, "member", identity.MemberID)
	require.Equal(t, agentcustomization.QualityVersion, subject.Definition.Version)
	require.Equal(t, agentcustomization.QualityDefinition, subject.Definition.ID)
	_, _, err = resolve(ctx, agentconfig.ModeSubject{Kind: agentconfig.PrivateDeliverySubject, ID: uuid.NewString()}, false)
	require.ErrorIs(t, err, agentconfig.ErrNotFound)
	foreign := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: "actor", TenantID: "foreign", EffectiveOrganizationID: "foreign", EffectiveMemberID: "member"})
	_, _, err = resolve(foreign, agentconfig.ModeSubject{Kind: agentconfig.PrivateDeliverySubject, ID: private.id}, false)
	require.ErrorIs(t, err, agentconfig.ErrForbidden)
	require.Equal(t, 2, private.calls)
	_, _, err = resolve(ctx, agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: "unregistered.agent"}, false)
	require.ErrorIs(t, err, agentconfig.ErrNotFound)
}
