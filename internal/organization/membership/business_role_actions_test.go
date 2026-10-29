package membership

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
)

type actionDirectory struct {
	*nativeDirectoryFixture
	calls int
	err   error
	after func()
}

func TestDetailDiscardsActionsAfterNativeReadDeadlineOrTokenExpiry(t *testing.T) {
	for _, expired := range []bool{false, true} {
		role := authz.EnterpriseRoleKey("effective-b", 3)
		member := Member{ID: "grant", UserID: "target", OrganizationID: "effective-b", ProjectID: "project", Roles: []string{"listingkit_admin"}, State: "active", ChangedAt: "2026-10-29T00:00:00Z"}
		d := &actionDirectory{nativeDirectoryFixture: &nativeDirectoryFixture{directoryStub: &directoryStub{page: Page{Items: []Member{member}, Total: 1}}, revision: NativeAuthorizationRevision{InstanceID: "instance", OrganizationID: member.OrganizationID, ProjectID: member.ProjectID, AuthorizationID: member.ID, UserID: member.UserID, NativeSequence: "7", Roles: member.Roles, State: member.State, ChangedAt: member.ChangedAt}}}
		a, _ := authz.NewListingKitAuthorizer(nil, nil)
		s := NewService(d, a, "project")
		s.SetRoleStore(roleStoreFixture{[]RoleDefinition{{ID: role, Modules: []string{"store-browser"}, Version: 1}}})
		ctx, cancel := context.WithCancel(scopedContext("listingkit_admin"))
		expected := ErrUnavailable
		if expired {
			identity, _ := authidentity.AuthenticatedIdentityFromContext(ctx)
			identity.TokenExpiresAt = time.Now().Add(50 * time.Millisecond)
			ctx = authidentity.WithAuthenticatedIdentity(ctx, identity)
			d.after = func() { time.Sleep(time.Until(identity.TokenExpiresAt) + time.Millisecond) }
			expected = ErrAuthentication
		} else {
			d.after = cancel
		}
		result, err := s.Read(ctx, "grant")
		cancel()
		require.ErrorIs(t, err, expected)
		require.Empty(t, result.Items)
	}
}

func (d *actionDirectory) ReadAuthorizationRevision(ctx context.Context, org, id string) (NativeAuthorizationRevision, error) {
	d.calls++
	if d.after != nil {
		d.after()
	}
	return d.revision, d.err
}

func TestDetailBusinessRoleActionsUseNativeStateAndCommandEligibility(t *testing.T) {
	for _, scenario := range []string{"ready", "builtin-platform-role", "read-only", "stale-native", "unavailable", "ordinary", "unknown-role", "inactive"} {
		t.Run(scenario, func(t *testing.T) {
			add, remove := authz.EnterpriseRoleKey("effective-b", 3), authz.EnterpriseRoleKey("effective-b", 4)
			member := Member{ID: "grant", UserID: "target", OrganizationID: "effective-b", ProjectID: "project", Roles: []string{"listingkit_admin", "platform_admin", remove}, State: "active", ChangedAt: "2026-10-29T00:00:00Z"}
			if scenario == "ordinary" {
				member.Roles = []string{remove}
			}
			if scenario == "unknown-role" {
				member.Roles = append(member.Roles, "unknown")
			}
			if scenario == "inactive" {
				member.State = "inactive"
			}
			d := &actionDirectory{nativeDirectoryFixture: &nativeDirectoryFixture{directoryStub: &directoryStub{page: Page{Items: []Member{member}, Total: 1}}, revision: NativeAuthorizationRevision{InstanceID: "instance", OrganizationID: member.OrganizationID, ProjectID: member.ProjectID, ProjectGrantID: "enterprise-grant", AuthorizationID: member.ID, UserID: member.UserID, NativeSequence: "7", Roles: member.Roles, State: member.State, ChangedAt: member.ChangedAt}}}
			if scenario == "stale-native" {
				d.revision.Roles = []string{"listingkit_admin"}
			}
			if scenario == "unavailable" {
				d.err = ErrUnavailable
			}
			a, _ := authz.NewListingKitAuthorizer(nil, []string{"platform_admin"})
			s := testService(d, a, "project", "platform_admin")
			if scenario == "builtin-platform-role" {
				a, _ = authz.NewListingKitAuthorizer(nil, nil)
				s = testService(d, a, "project")
			}
			definitions := roleStoreFixture{[]RoleDefinition{{ID: testReadRole, Modules: []string{"members"}, Version: 1}, {ID: add, Modules: []string{"store-browser"}, Version: 1}, {ID: remove, Modules: []string{"members"}, Version: 2}}}
			s.SetRoleStore(definitions)
			a.SetRolePolicyReader(definitions)
			actorRole := "listingkit_admin"
			if scenario == "read-only" {
				actorRole = testReadRole
			}
			result, err := s.Read(scopedContext(actorRole), "grant")
			require.NoError(t, err)
			data, err := json.Marshal(result.Items[0])
			require.NoError(t, err)
			var wire struct {
				Actions *struct {
					Grantable []string `json:"grantableRoleIds"`
					Revocable []string `json:"revocableRoleIds"`
				} `json:"businessRoleActions"`
			}
			require.NoError(t, json.Unmarshal(data, &wire))
			require.NotNil(t, wire.Actions, "detail must explicitly project business-role availability")
			if scenario == "ready" || scenario == "builtin-platform-role" {
				require.Equal(t, []string{add}, wire.Actions.Grantable)
				require.Equal(t, []string{remove}, wire.Actions.Revocable, "a changed business role can still be revoked")
				require.True(t, s.businessRoleEligible(scopedContext(actorRole), member, CommandBusinessRoleGrant, add))
			} else {
				require.Empty(t, wire.Actions.Grantable)
				require.Empty(t, wire.Actions.Revocable)
			}
			probes := d.calls
			_, err = s.List(scopedContext(actorRole), PageRequest{Limit: 20})
			require.NoError(t, err)
			require.Equal(t, probes, d.calls, "list must not probe native state for every member")
		})
	}
}
