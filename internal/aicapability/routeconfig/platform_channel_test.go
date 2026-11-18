package routeconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlatformChannelGrantAndBindingIdentity(t *testing.T) {
	connection := "uc-5006eec4-1bf9-88a3-9bd0-ab2bfdcc3f4d"
	release := PlatformChannel{ConnectionID: connection, PriceID: "official-a", Organizations: []string{"org-a"}}
	require.NoError(t, release.Validate(connection))
	ref := release.Reference()
	require.Equal(t, connection, PlatformConnection(ref.ID))
	require.True(t, release.Allows("org-a", ref))
	require.False(t, release.Allows("org-b", ref))
	release.Organizations = []string{"org-b"}
	require.Equal(t, ref, release.Reference())
	require.False(t, release.Allows("org-a", ref))
	release.PriceID = "official-b"
	require.False(t, release.Allows("org-b", ref))
	release.Organizations = []string{"org-b", "org-b"}
	require.ErrorIs(t, release.Validate(connection), ErrInvalid)
}
