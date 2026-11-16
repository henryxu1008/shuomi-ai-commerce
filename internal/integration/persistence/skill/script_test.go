package skillpersistence

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	d "task-processor/internal/skill"
)

func TestScriptCanonicalVersionSurvivesShareAndOrganizationCopy(t *testing.T) {
	_, store := fixture(t)
	ctx := context.Background()
	principal := d.Principal{UserID: "alice", OrganizationID: "org-a", UseOrganization: true}
	pkg, err := d.ValidatePackage([]d.File{{Path: "SKILL.md", Text: "---\nname: saved-title-script\ndescription: Format saved title\nmetadata:\n  listingkit-method-v1: product-title\n---\nApply the saved formatter.\n"}, {Path: "execution.json", Text: `{"schemaVersion":"skill-script-v1","profile":"python-title-json-v1","entrypoint":"scripts/main.py","phase":"title-candidate-postprocess","inputContract":"title-candidate-v1","outputContract":"title-result-v1"}`}, {Path: "scripts/main.py", Text: "print('saved source')\n"}})
	require.NoError(t, err)
	receipt, err := store.Execute(ctx, d.Command{Principal: principal, Owner: d.Owner{Kind: d.UserOwner, ID: principal.UserID}, Key: uuid.NewString(), Operation: "create", Package: pkg, AllowCopy: true, AllowExport: true}, func(context.Context) (d.Principal, error) { return principal, nil })
	require.NoError(t, err)
	source, err := store.Read(ctx, principal, d.Owner{Kind: d.UserOwner, ID: principal.UserID}, receipt.ID, 1)
	require.NoError(t, err)
	require.Equal(t, d.ScriptPolicyVersion, source.Ref().PolicyVersion)
	request, err := store.Share(ctx, principal, uuid.NewString(), source.Metadata.ID, 1, true, func(context.Context) (d.Principal, error) { return principal, nil })
	require.NoError(t, err)
	require.Equal(t, source.Ref(), request.Source)
	shares, err := store.Shares(ctx, principal, false)
	require.NoError(t, err)
	require.Len(t, shares, 1)
	require.Equal(t, source.Ref(), shares[0].Source)
	administrator := d.Principal{UserID: "admin", OrganizationID: "org-a", ReadOrganization: true, ConfigureOrganization: true}
	shared, err := store.ReadShared(ctx, administrator, request.ID)
	require.NoError(t, err)
	require.Equal(t, source.Ref(), shared.Ref())
	accepted, err := store.DecideShare(ctx, administrator, uuid.NewString(), request.ID, "approve", func(context.Context) (d.Principal, error) { return administrator, nil })
	require.NoError(t, err)
	copied, err := store.Read(ctx, administrator, d.Owner{Kind: d.OrganizationOwner, ID: "org-a"}, accepted.ResultID, 1)
	require.NoError(t, err)
	require.Equal(t, pkg.Digest, copied.Package.Digest)
	require.Equal(t, pkg.Files, copied.Package.Files)
	require.NotNil(t, copied.Source)
	require.Equal(t, source.Ref(), *copied.Source)
	require.Equal(t, d.ScriptPolicyVersion, copied.Ref().PolicyVersion)
	require.False(t, copied.Ref() == (agent.SkillVersionRef{}))
}
