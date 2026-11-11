package setup

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/app/setupcontrol"
	"testing"
)

func TestExplicitLocalSubnetIsCanonicalPrivateAndBounded(t *testing.T) {
	for _, bad := range []string{"0.0.0.0/0", "8.8.8.0/24", "10.2.3.4/24", "10.0.0.0/8", "::/0", "10.2.3.0/31"} {
		require.ErrorIs(t, validateSubnet(bad), setupcontrol.ErrInvalid)
	}
	require.NoError(t, validateSubnet("10.251.124.0/26"))
	require.NoError(t, validateSubnet(""))
}
