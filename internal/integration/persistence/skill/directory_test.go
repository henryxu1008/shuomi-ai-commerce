package skillpersistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	d "task-processor/internal/skill"
	"testing"
)

func TestUnpublishedSuccessorDoesNotHideReleasedTemplate(t *testing.T) {
	_, s := fixture(t)
	ctx := context.Background()
	p := d.Principal{UserID: "maintainer", PlatformMaintainer: true}
	owner := d.Owner{Kind: d.PlatformOwner, ID: d.PlatformID}
	resolve := func(context.Context) (d.Principal, error) { return p, nil }
	c := d.Command{Principal: p, Owner: owner, Key: uuid.NewString(), Operation: "create", Package: packageInput(t, "Released rules."), AllowCopy: true, AllowExport: true}
	r, e := s.Execute(ctx, c, resolve)
	require.NoError(t, e)
	for _, operation := range []string{"publish", "enable"} {
		r, e = s.Execute(ctx, d.Command{Principal: p, Owner: owner, Key: uuid.NewString(), ID: r.ID, Version: 1, Expected: r.Revision, Operation: operation}, resolve)
		require.NoError(t, e)
	}
	c.Key, c.ID, c.Expected, c.Operation = uuid.NewString(), r.ID, r.Revision, "save-version"
	c.Package = packageInput(t, "Private next rules.")
	_, e = s.Execute(ctx, c, resolve)
	require.NoError(t, e)
	user := d.Principal{UserID: "reader"}
	items, _, e := s.List(ctx, user, owner, "", 20)
	require.NoError(t, e)
	require.Len(t, items, 1)
	require.Equal(t, uint64(1), items[0].Number)
	require.Empty(t, items[0].Package.Files)
	versions, e := s.Versions(ctx, user, owner, r.ID)
	require.NoError(t, e)
	require.Len(t, versions, 1)
	require.Equal(t, uint64(1), versions[0].Number)
}
