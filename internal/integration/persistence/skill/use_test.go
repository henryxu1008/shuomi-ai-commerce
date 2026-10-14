package skillpersistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	d "task-processor/internal/skill"
	"testing"
	"time"
)

func enabledMethod(t *testing.T, s *Store, p d.Principal) d.Version {
	t.Helper()
	v := createPersonal(t, s, p)
	resolve := func(context.Context) (d.Principal, error) { return p, nil }
	for _, operation := range []string{"publish", "enable"} {
		r, e := s.Execute(context.Background(), d.Command{Principal: p, Owner: v.Metadata.Owner, Key: uuid.NewString(), ID: v.Metadata.ID, Version: 1, Expected: v.Metadata.Revision, Operation: operation}, resolve)
		require.NoError(t, e)
		v.Metadata.Revision = r.Revision
	}
	v, e := s.Read(context.Background(), p, v.Metadata.Owner, v.Metadata.ID, 1)
	require.NoError(t, e)
	return v
}
func TestUseGateSerializesRevocationAndChecksEnterpriseAdmission(t *testing.T) {
	db, s := fixture(t)
	ctx := context.Background()
	p := d.Principal{UserID: "alice", OrganizationID: "org-a", MemberID: "member", UseOrganization: true, ConfigureOrganization: true}
	v := enabledMethod(t, s, p)
	config, e := configstore.New(db)
	require.NoError(t, e)
	_, e = config.Execute(ctx, agentconfig.Command{Scope: agent.Scope{OrganizationID: p.OrganizationID, ActorID: p.UserID}, Key: uuid.NewString(), AgentID: agentconfig.ContentAgentID, Operation: "enable", Absent: true})
	require.NoError(t, e)
	_, release, e := s.AcquireUse(ctx, p, v.Ref())
	require.ErrorIs(t, e, d.ErrForbidden)
	require.Nil(t, release)
	require.NoError(t, db.Exec("INSERT INTO agent_configuration.skill_admissions(organization_id,agent_id,allow_personal,revision,updated_by,updated_at) VALUES(?,'product.content.agent',true,1,?,now())", p.OrganizationID, p.UserID).Error)
	material, release, e := s.AcquireUse(ctx, p, v.Ref())
	require.NoError(t, e)
	require.Equal(t, v.Package.Digest, material.Digest)
	defer release()
	wrong := v.Ref()
	wrong.Digest = "0000000000000000000000000000000000000000000000000000000000000000"
	_, wrongRelease, e := s.AcquireUse(ctx, p, wrong)
	require.Error(t, e)
	require.Nil(t, wrongRelease)
	finished := make(chan error, 1)
	go func() {
		_, e := s.Execute(ctx, d.Command{Principal: p, Owner: v.Metadata.Owner, Key: uuid.NewString(), ID: v.Metadata.ID, Version: 1, Expected: v.Metadata.Revision, Operation: "revoke"}, func(context.Context) (d.Principal, error) { return p, nil })
		finished <- e
	}()
	select {
	case e := <-finished:
		t.Fatalf("revocation passed an active use gate: %v", e)
	case <-time.After(100 * time.Millisecond):
	}
	release()
	require.NoError(t, <-finished)
	_, release, e = s.AcquireUse(ctx, p, v.Ref())
	require.ErrorIs(t, e, d.ErrNotEnabled)
	require.Nil(t, release)
}
