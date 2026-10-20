package skillimage

import (
	"context"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/imageagent"
	d "task-processor/internal/skill"
	"testing"
)

func TestImageAdapterSendsOnlyExactVisualResourceAndRejectsMissingOrOversize(t *testing.T) {
	releases, err := d.OfficialPackages()
	require.NoError(t, err)
	for _, release := range releases {
		if release.Package.Target != d.ImageTarget {
			continue
		}
		for _, mode := range []string{"visual", "missing", "oversize"} {
			t.Run(release.Key+"/"+mode, func(t *testing.T) {
				files := []d.File{{Path: "SKILL.md", Text: release.Package.Files[0].Text + "\nPRIVATE_SKILL_DOCUMENT_CANARY\n"}}
				brief := "A finished product photograph with a clear subject, soft light and balanced negative space."
				if mode == "oversize" {
					brief = strings.Repeat("x", (4<<10)+1)
				}
				if mode != "missing" {
					files = append(files, d.File{Path: "references/visual-brief.md", Text: brief})
				}
				p, err := d.ValidatePackage(files)
				require.NoError(t, err)
				store := &useFixture{p: p}
				adapter, err := New(store, &authFixture{})
				require.NoError(t, err)
				ref := agent.SkillVersionRef{Kind: "skill-version-v1", ID: "2bb819bd-af13-4e68-95e0-1cc86130fb1f", OwnerKind: d.PlatformOwner, OwnerID: d.PlatformID, Version: 2, Digest: p.Digest, PolicyVersion: d.MethodPolicyVersion}
				id := imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, RunID: "run", TenantID: "org", UserID: "actor", MemberID: "member", BusinessTaskID: "product"}
				text, held, err := adapter.AcquireImageMethod(context.Background(), id, ref, imageagent.ImageMethodPolicyVersion)
				if mode != "visual" {
					require.Error(t, err)
					require.True(t, store.released)
					return
				}
				require.NoError(t, err)
				defer held()
				require.Equal(t, brief, text)
				require.NotContains(t, text, "PRIVATE_SKILL_DOCUMENT_CANARY")
				require.NotContains(t, text, "Method file:")
			})
		}
	}
}

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
	text, release, err := adapter.AcquireImageMethod(context.Background(), id, ref, imageagent.ImageMethodPolicyVersion)
	require.NoError(t, err)
	brief, err := d.ImageVisualBrief(p)
	require.NoError(t, err)
	require.Equal(t, brief, text)
	require.Equal(t, d.ImageTarget, store.target)
	require.False(t, store.released)
	release()
	require.True(t, store.released)
	store.released = false
	store.p.Files[0].Text += "changed"
	_, _, err = adapter.AcquireImageMethod(context.Background(), id, ref, imageagent.ImageMethodPolicyVersion)
	require.Error(t, err)
	require.True(t, store.released)
	auth.denied = true
	_, _, err = adapter.AcquireImageMethod(context.Background(), id, ref, imageagent.ImageMethodPolicyVersion)
	require.ErrorIs(t, err, imageagent.ErrIdentityRequired)
}

func TestHistoricalMethodKeepsAvailabilityFenceWithoutImageMaterial(t *testing.T) {
	releases, err := d.OfficialPackages()
	require.NoError(t, err)
	for _, release := range releases {
		if release.Previous == nil {
			continue
		}
		p := *release.Previous
		store := &useFixture{p: p}
		adapter, err := New(store, &authFixture{})
		require.NoError(t, err)
		ref := agent.SkillVersionRef{Kind: "skill-version-v1", ID: "2bb819bd-af13-4e68-95e0-1cc86130fb1f", OwnerKind: d.PlatformOwner, OwnerID: d.PlatformID, Version: 1, Digest: p.Digest, PolicyVersion: d.MethodPolicyVersion}
		id := imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, RunID: "run", TenantID: "org", UserID: "actor", MemberID: "member", BusinessTaskID: "product"}
		text, held, err := imageagent.AcquireImageMethod(context.Background(), adapter, id, &imageagent.ImageMethodBinding{Skill: ref, PolicyVersion: imageagent.HistoricalImageMethodPolicyVersion})
		require.NoError(t, err)
		require.Empty(t, text)
		require.False(t, store.released)
		held()
		require.True(t, store.released)
	}
}
