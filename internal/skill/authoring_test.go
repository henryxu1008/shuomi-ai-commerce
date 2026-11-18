package skill

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOfficialAuthoringReleaseUsesOnlyAuthoringTarget(t *testing.T) {
	releases, err := OfficialPackages()
	require.NoError(t, err)
	for _, release := range releases {
		if release.Key == AuthoringReleaseKey {
			require.True(t, release.Package.Supports(AuthoringTarget))
			require.False(t, release.Package.Supports(TitleTarget))
			require.False(t, release.Package.Supports(ImageTarget))
			return
		}
	}
	t.Fatal("official authoring version was not registered")
}
