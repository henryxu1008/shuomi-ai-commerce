package temu

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
)

func TestSealedTokenBindsOriginalEnterpriseMembershipApplicationAndPartition(t *testing.T) {
	p, err := NewCredentialProtection("key-1", make([]byte, 32))
	require.NoError(t, err)
	a := storecenter.OfficialConnectionAttempt{OrganizationID: "org-a", StoreID: "store-a", AttemptID: "attempt-a", ActorID: "actor-a", MemberID: "member-a", Platform: storecenter.PlatformTemu, IdentityPartition: storecenter.PartitionCN, AuthorizationMethod: storecenter.AuthorizationManualToken, AppID: "test-app", AppVersion: "v1:temu:CN:manual_token:6:r1", ConnectionVersion: 1}
	key, cipher, err := p.Seal(a, tokenCredential())
	require.NoError(t, err)
	require.NotContains(t, cipher, "test-token")
	opened, err := p.Open(a, key, cipher)
	require.NoError(t, err)
	require.Equal(t, tokenCredential(), opened)
	public, err := json.Marshal(opened)
	require.NoError(t, err)
	require.NotContains(t, string(public), "test-token")
	for _, change := range []func(*storecenter.OfficialConnectionAttempt){
		func(a *storecenter.OfficialConnectionAttempt) { a.OrganizationID = "other" },
		func(a *storecenter.OfficialConnectionAttempt) { a.StoreID = "other" },
		func(a *storecenter.OfficialConnectionAttempt) { a.AttemptID = "other" },
		func(a *storecenter.OfficialConnectionAttempt) { a.MemberID = "new-membership" },
		func(a *storecenter.OfficialConnectionAttempt) { a.ActorID = "other" },
		func(a *storecenter.OfficialConnectionAttempt) { a.Platform = storecenter.PlatformShein },
		func(a *storecenter.OfficialConnectionAttempt) { a.IdentityPartition = storecenter.PartitionUS },
		func(a *storecenter.OfficialConnectionAttempt) { a.AppID = "other" },
		func(a *storecenter.OfficialConnectionAttempt) { a.AppVersion = "changed-config" },
		func(a *storecenter.OfficialConnectionAttempt) { a.ConnectionVersion++ },
		func(a *storecenter.OfficialConnectionAttempt) {
			a.AuthorizationMethod = storecenter.AuthorizationCallback
		},
	} {
		changed := a
		change(&changed)
		_, err := p.Open(changed, key, cipher)
		require.Error(t, err)
	}
}
