package skillpersistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"sync"
	d "task-processor/internal/skill"
	"testing"
)

func createPersonal(t *testing.T, s *Store, p d.Principal) d.Version {
	t.Helper()
	ctx := context.Background()
	o := d.Owner{Kind: d.UserOwner, ID: p.UserID}
	c := d.Command{Principal: p, Owner: o, Key: uuid.NewString(), Operation: "create", Package: packageInput(t, "Only verified product facts."), AllowCopy: true, AllowExport: true}
	receipt, e := s.Execute(ctx, c, func(context.Context) (d.Principal, error) { return p, nil })
	require.NoError(t, e)
	v, e := s.Read(ctx, p, o, receipt.ID, receipt.Version)
	require.NoError(t, e)
	return v
}
func TestOrganizationCopyNeedsExactConsentAndIsIndependent(t *testing.T) {
	db, s := fixture(t)
	ctx := context.Background()
	person := d.Principal{UserID: "alice", OrganizationID: "org-a", UseOrganization: true}
	source := createPersonal(t, s, person)
	resolvePerson := func(context.Context) (d.Principal, error) { return person, nil }
	_, e := s.Share(ctx, person, uuid.NewString(), source.Metadata.ID, 1, false, resolvePerson)
	require.ErrorIs(t, e, d.ErrInvalid)
	shared, e := s.Share(ctx, person, uuid.NewString(), source.Metadata.ID, 1, true, resolvePerson)
	require.NoError(t, e)
	admin := d.Principal{UserID: "admin", OrganizationID: "org-a", ReadOrganization: true, ConfigureOrganization: true}
	foreign := admin
	foreign.OrganizationID = "org-b"
	_, e = s.ReadShared(ctx, foreign, shared.ID)
	require.Error(t, e)
	designated, e := s.ReadShared(ctx, admin, shared.ID)
	require.NoError(t, e)
	require.Equal(t, source.Ref(), designated.Ref())
	_, e = s.Read(ctx, admin, source.Metadata.Owner, source.Metadata.ID, 1)
	require.Error(t, e)
	resolveAdmin := func(context.Context) (d.Principal, error) { return admin, nil }
	var copies [2]d.ShareRequest
	var errs [2]error
	var group sync.WaitGroup
	for i := range 2 {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			copies[i], errs[i] = s.DecideShare(ctx, admin, uuid.NewString(), shared.ID, "approve", resolveAdmin)
		}(i)
	}
	group.Wait()
	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	require.Equal(t, copies[0].ResultID, copies[1].ResultID)
	org, e := s.Read(ctx, admin, d.Owner{Kind: d.OrganizationOwner, ID: admin.OrganizationID}, copies[0].ResultID, 1)
	require.NoError(t, e)
	require.Equal(t, source.Package.Digest, org.Package.Digest)
	require.Equal(t, source.Ref(), *org.Source)
	require.Equal(t, d.Draft, org.State)
	update := d.Command{Principal: person, Owner: source.Metadata.Owner, Key: uuid.NewString(), ID: source.Metadata.ID, Operation: "save-version", Expected: 1, Package: packageInput(t, "New personal rules."), AllowCopy: true, AllowExport: true}
	_, e = s.Execute(ctx, update, resolvePerson)
	require.NoError(t, e)
	retained, e := s.Read(ctx, admin, org.Metadata.Owner, org.Metadata.ID, 1)
	require.NoError(t, e)
	require.Equal(t, org.Package, retained.Package)
	var count int64
	require.NoError(t, db.Raw("SELECT count(*) FROM skill_capability.skills WHERE owner_kind='ORGANIZATION'").Scan(&count).Error)
	require.EqualValues(t, 1, count)
}
func TestWithdrawnConsentCannotCreateOrganizationAssets(t *testing.T) {
	db, s := fixture(t)
	ctx := context.Background()
	p := d.Principal{UserID: "alice", OrganizationID: "org-a", UseOrganization: true}
	source := createPersonal(t, s, p)
	resolve := func(context.Context) (d.Principal, error) { return p, nil }
	shared, e := s.Share(ctx, p, uuid.NewString(), source.Metadata.ID, 1, true, resolve)
	require.NoError(t, e)
	identityOnly := d.Principal{UserID: p.UserID}
	_, e = s.DecideShare(ctx, identityOnly, uuid.NewString(), shared.ID, "withdraw", func(context.Context) (d.Principal, error) { return identityOnly, nil })
	require.NoError(t, e)
	admin := d.Principal{UserID: "admin", OrganizationID: "org-a", ConfigureOrganization: true}
	_, e = s.DecideShare(ctx, admin, uuid.NewString(), shared.ID, "approve", func(context.Context) (d.Principal, error) { return admin, nil })
	require.Error(t, e)
	_, e = s.ReadShared(ctx, admin, shared.ID)
	require.Error(t, e)
	var count int64
	require.NoError(t, db.Raw("SELECT count(*) FROM skill_capability.skills WHERE owner_kind='ORGANIZATION'").Scan(&count).Error)
	require.Zero(t, count)
}
