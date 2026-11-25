package skillpersistence

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	configstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/agentconfig"
	d "github.com/henryxu1008/shuomi-ai-commerce/internal/skill"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestImageMethodsRequireTheirOwnActivationTargetAndSharedOwner(t *testing.T) {
	db, s := fixture(t)
	ctx := context.Background()
	admin := d.Principal{UserID: "maintainer", PlatformMaintainer: true}
	installed, err := s.InstallOfficial(ctx, admin, func(context.Context) (d.Principal, error) { return admin, nil })
	require.NoError(t, err)
	p := d.Principal{UserID: "alice", OrganizationID: "org-a", MemberID: "member-a", UseOrganization: true, ReadOrganization: true, ConfigureOrganization: true}
	cfg, err := configstore.New(db)
	require.NoError(t, err)
	for _, id := range []string{agentconfig.ContentAgentID, agentconfig.ImageAgentID} {
		_, err = cfg.Execute(ctx, agentconfig.Command{Scope: agent.Scope{OrganizationID: p.OrganizationID, ActorID: p.UserID}, Key: uuid.NewString(), AgentID: id, Operation: "enable", Absent: true})
		require.NoError(t, err)
	}
	for _, target := range []string{d.TitleTarget, d.ImageTarget} {
		items, err := s.CandidatesForTarget(ctx, p, target)
		require.NoError(t, err)
		require.Len(t, items, 3)
		for _, v := range items {
			require.True(t, v.Package.Supports(target))
			require.Empty(t, v.Package.Files)
			if target == d.ImageTarget {
				require.EqualValues(t, 2, v.Number)
				old, err := s.Read(ctx, admin, d.Owner{Kind: d.PlatformOwner, ID: d.PlatformID}, v.Metadata.ID, 1)
				require.NoError(t, err)
				_, err = s.ResolveSelectionForTarget(ctx, p, old.Metadata.ID, 1, old.Package.Digest, d.ImageTarget)
				require.ErrorIs(t, err, d.ErrInvalid)
				_, held, err := s.AcquireUseForTarget(ctx, p, old.Ref(), d.ImageTarget)
				require.NoError(t, err)
				held()
			}
		}
	}
	var image agent.SkillVersionRef
	for _, ref := range installed.Installed {
		v, e := s.Read(ctx, admin, d.Owner{Kind: d.PlatformOwner, ID: d.PlatformID}, ref.ID, 1)
		require.NoError(t, e)
		if v.Package.Target == d.ImageTarget {
			image = ref
			break
		}
	}
	_, err = s.ReadForUse(ctx, p, image)
	require.ErrorIs(t, err, d.ErrForbidden)
	pkg, release, err := s.AcquireUseForTarget(ctx, p, image, d.ImageTarget)
	require.NoError(t, err)
	require.Equal(t, d.ImageTarget, pkg.Target)
	release()
	resolve := func(context.Context) (d.Principal, error) { return p, nil }
	for _, owner := range []d.Owner{{Kind: d.UserOwner, ID: p.UserID}, {Kind: d.OrganizationOwner, ID: p.OrganizationID}} {
		r, e := s.Execute(ctx, d.Command{Principal: p, Owner: owner, Key: uuid.NewString(), Operation: "copy", Source: &image}, resolve)
		require.NoError(t, e)
		for _, action := range []string{"publish", "enable"} {
			r, e = s.Execute(ctx, d.Command{Principal: p, Owner: owner, Key: uuid.NewString(), ID: r.ID, Version: 1, Expected: r.Revision, Operation: action}, resolve)
			require.NoError(t, e)
		}
		v, e := s.Read(ctx, p, owner, r.ID, 1)
		require.NoError(t, e)
		_, release, e := s.AcquireUseForTarget(ctx, p, v.Ref(), d.ImageTarget)
		if owner.Kind == d.UserOwner {
			require.ErrorIs(t, e, d.ErrForbidden)
			require.Nil(t, release)
		} else {
			require.NoError(t, e)
			release()
			other := p
			other.OrganizationID = "org-b"
			_, _, e = s.AcquireUseForTarget(ctx, other, v.Ref(), d.ImageTarget)
			require.Error(t, e)
		}
	}
	_, err = s.CandidatesForTarget(ctx, p, "arbitrary-target")
	require.ErrorIs(t, err, d.ErrInvalid)
}

func TestImageMethodBatchUsesOneConnectionAndReturnsItOnCancel(t *testing.T) {
	db, s := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, s.PrepareOfficial(ctx))
	p := d.Principal{UserID: "alice", OrganizationID: "org-a", MemberID: "member-a", UseOrganization: true}
	cfg, err := configstore.New(db)
	require.NoError(t, err)
	_, err = cfg.Execute(ctx, agentconfig.Command{
		Scope: agent.Scope{OrganizationID: p.OrganizationID, ActorID: p.UserID}, Key: uuid.NewString(),
		AgentID: agentconfig.ImageAgentID, Operation: "enable", Absent: true,
	})
	require.NoError(t, err)
	methods, err := s.CandidatesForTarget(ctx, p, d.ImageTarget)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(methods), 2)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	release, err := s.AcquireImageUses(ctx, p, []agent.SkillVersionRef{methods[1].Ref(), methods[0].Ref()})
	require.NoError(t, err)
	defer release()
	require.Equal(t, 1, pool.Stats().InUse)
	cancel()
	require.Eventually(t, func() bool { return pool.Stats().InUse == 0 }, time.Second, 5*time.Millisecond)
}

func TestImageMethodSnapshotRechecksExactSkillWithOneConnection(t *testing.T) {
	db, skills := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, skills.PrepareOfficial(ctx))
	principal := d.Principal{UserID: "alice", OrganizationID: "org-a", MemberID: "member-a", UseOrganization: true}
	scope := agent.Scope{OrganizationID: principal.OrganizationID, ActorID: principal.UserID}
	cfg, err := configstore.New(db)
	require.NoError(t, err)
	_, err = cfg.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.ImageAgentID, Operation: "enable", Absent: true})
	require.NoError(t, err)
	parameters := agentconfig.SetTemplate{Schema: agentconfig.ImageParameterSchema, Mode: "standard", ShareOriginals: true, Background: "white", Language: "en", Carousel: []agentconfig.ContentTask{{ID: "identity", Purpose: "product_identity"}}}
	template, err := cfg.Execute(ctx, agentconfig.Command{
		Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.ImageAgentID, Operation: "create-template",
		Input: agentconfig.TemplateInput{Name: "Test image template", TargetPlatform: "product", Image: &parameters},
	})
	require.NoError(t, err)
	methods, err := skills.CandidatesForTarget(ctx, principal, d.ImageTarget)
	require.NoError(t, err)
	require.NotEmpty(t, methods)
	ref := methods[0].Ref()
	start := agentconfig.ImageStartCommand{
		ExpectedModeRevision: "0", Scope: scope, MemberID: principal.MemberID,
		RequestKey: uuid.NewString(), ContextID: "product", RunID: uuid.NewString(), TargetPlatform: "product",
		SourceDigest: strings.Repeat("a", 64), InputDigest: strings.Repeat("b", 64),
		Template:      &agentconfig.TemplateRef{TemplateID: template.TemplateID, Revision: "1"},
		HardLimits:    agentconfig.ImageRunLimits{UnitVersion: "ai-micropoint-v1", Images: 1, Points: 10, ElapsedSeconds: 60},
		ImageSkillRef: &ref,
	}
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	cfg = cfg.WithImageSkillCheck(func(ctx context.Context, tx *gorm.DB, command agentconfig.ImageStartCommand) error {
		return skills.CheckImageUseInTransaction(ctx, tx, principal, *command.ImageSkillRef)
	})
	snapshot, err := cfg.PrepareImageConfiguration(ctx, start)
	require.NoError(t, err)
	replayed, err := cfg.PrepareImageConfiguration(ctx, start)
	require.NoError(t, err)
	require.Equal(t, snapshot, replayed)
	ref.Digest = strings.Repeat("c", 64)
	_, err = cfg.PrepareImageConfiguration(ctx, start)
	require.ErrorIs(t, err, d.ErrConflict)
	require.Zero(t, pool.Stats().InUse)
}
