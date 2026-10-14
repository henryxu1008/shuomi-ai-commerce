package skill

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOwnerAndActionsAreIndependent(t *testing.T) {
	owner := Principal{UserID: "alice", OrganizationID: "org-a", UseOrganization: true}
	personal := Version{Metadata: Metadata{Owner: Owner{UserOwner, "alice"}, Creator: "alice"}, State: Enabled, Visibility: Private, AllowCopy: true, AllowExport: true}
	require.True(t, Authorized(owner, personal, Edit))
	require.True(t, Authorized(owner, personal, Read))
	admin := Principal{UserID: "bob", OrganizationID: "org-a", ReadOrganization: true, UseOrganization: true, ConfigureOrganization: true, PlatformMaintainer: true}
	require.False(t, Authorized(admin, personal, Read))
	require.False(t, Authorized(admin, personal, Edit))
	org := personal
	org.Metadata.Owner = Owner{OrganizationOwner, "org-a"}
	org.Visibility = Shared
	require.True(t, Authorized(owner, org, Use))
	require.False(t, Authorized(owner, org, Publish))
	require.True(t, Authorized(admin, org, Publish))
	org.AllowCopy = false
	require.False(t, Authorized(admin, org, Copy))
	require.True(t, Authorized(admin, org, Read))
	revokedCreator := Principal{UserID: org.Metadata.Creator, OrganizationID: "org-a"}
	require.False(t, Authorized(revokedCreator, org, Read))
	org.AllowExport = false
	require.False(t, Authorized(admin, org, Export))
	foreign := admin
	foreign.OrganizationID = "org-b"
	require.False(t, Authorized(foreign, org, Read))
	require.False(t, Authorized(foreign, org, Edit))
	org.Visibility = Private
	require.False(t, Authorized(Principal{UserID: "c", OrganizationID: "org-a", ReadOrganization: true, UseOrganization: true}, org, Read))
	require.True(t, Authorized(admin, org, Read))
	official := personal
	official.Metadata.Owner = Owner{PlatformOwner, PlatformID}
	official.Visibility = Shared
	require.True(t, Authorized(Principal{UserID: "signed-in"}, official, Read))
	require.False(t, Authorized(owner, official, Edit))
	require.True(t, Authorized(admin, official, Edit))
	official.State = Draft
	require.False(t, Authorized(owner, official, Read))
	official.State = Revoked
	require.False(t, Authorized(owner, official, Use))
	for _, state := range []string{Draft, Ready, Disabled, Revoked} {
		personal.State = state
		require.False(t, Authorized(owner, personal, Use), state)
	}
	require.False(t, Authorized(Principal{}, personal, Read))
}
