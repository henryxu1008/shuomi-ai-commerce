package skillimage

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/imageagent"
	d "task-processor/internal/skill"
	"testing"
)

type useFixture struct {
	p        d.Package
	target   string
	released bool
}

func (f *useFixture) AcquireUseForTarget(_ context.Context, p d.Principal, _ agent.SkillVersionRef, target string) (d.Package, func(), error) {
	f.target = target
	return f.p, func() { f.released = true }, nil
}

type authFixture struct{ denied bool }

func (a *authFixture) AuthorizeExecution(context.Context, imageagent.ExecutionIdentity) error {
	if a.denied {
		return imageagent.ErrIdentityRequired
	}
	return nil
}

func TestImageMethodAdapterUsesLiveImageAuthorityAndCanonicalExactPackage(t *testing.T) {
	packages, err := d.OfficialPackages()
	require.NoError(t, err)
	var p d.Package
	for _, r := range packages {
		if r.Package.Target == d.ImageTarget {
			p = r.Package
			break
		}
	}
	store := &useFixture{p: p}
	auth := &authFixture{}
	adapter, err := New(store, auth)
	require.NoError(t, err)
	ref := agent.SkillVersionRef{Kind: "skill-version-v1", ID: "2bb819bd-af13-4e68-95e0-1cc86130fb1f", OwnerKind: d.PlatformOwner, OwnerID: d.PlatformID, Version: 1, Digest: p.Digest, PolicyVersion: d.MethodPolicyVersion}
	id := imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, RunID: "run", TenantID: "org", UserID: "actor", MemberID: "member", BusinessTaskID: "product"}
	text, release, err := adapter.AcquireImageMethod(context.Background(), id, ref)
	require.NoError(t, err)
	require.Contains(t, text, p.Files[0].Text[:30])
	require.Equal(t, d.ImageTarget, store.target)
	require.False(t, store.released)
	release()
	require.True(t, store.released)
	store.released = false
	store.p.Files[0].Text += "changed"
	_, _, err = adapter.AcquireImageMethod(context.Background(), id, ref)
	require.Error(t, err)
	require.True(t, store.released)
	auth.denied = true
	_, _, err = adapter.AcquireImageMethod(context.Background(), id, ref)
	require.ErrorIs(t, err, imageagent.ErrIdentityRequired)
}
