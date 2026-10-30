package httpapi

import (
	"context"
	"errors"
	"io"
	"mime/quotedprintable"
	"net/http/httptest"
	stdmail "net/mail"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	invitationmail "task-processor/internal/integration/mail"
	memberstore "task-processor/internal/integration/persistence/organization/membership"
	"task-processor/internal/organization/membership"
	flow "task-processor/internal/organization/membership/inviteflow"
)

type invitationRoleStore struct {
	membership.RoleStore
	items         []membership.RoleDefinition
	err           error
	organizations []string
}

func (s *invitationRoleStore) Roles(_ context.Context, organization string) ([]membership.RoleDefinition, int, error) {
	s.organizations = append(s.organizations, organization)
	return s.items, 64 - len(s.items), s.err
}

func TestInvitationMailResolvesCurrentScopedRoleName(t *testing.T) {
	for _, test := range []struct{ name, role, label string }{
		{"custom", authz.EnterpriseRoleKey("B", 1), "商品运营"},
		{"administrator", "listingkit_admin", "管理员"},
	} {
		t.Run(test.name, func(t *testing.T) {
			roles := &invitationRoleStore{items: []membership.RoleDefinition{{ID: authz.EnterpriseRoleKey("B", 2), Name: "采购"}, {ID: authz.EnterpriseRoleKey("B", 1), Name: "商品运营"}}}
			service, ctx, messages := invitationMailFactoryFixture(t, roles, nil)
			inv := invitationMailIntent(test.role)
			original := inv
			require.NoError(t, service.Notify(ctx, inv))
			require.Equal(t, original, inv, "display resolution cannot mutate invitation facts")
			require.Equal(t, []string{"B"}, roles.organizations)
			select {
			case raw := <-messages:
				message, err := stdmail.ReadMessage(strings.NewReader(raw))
				require.NoError(t, err)
				body, err := io.ReadAll(quotedprintable.NewReader(message.Body))
				require.NoError(t, err)
				require.Contains(t, string(body), "邀请角色："+test.label+"\r\n")
				require.Contains(t, string(body), "https://localhost:22544/invitations/"+inv.ID)
				require.Contains(t, string(body), inv.OrganizationName)
				to, err := stdmail.ParseAddress(message.Header.Get("To"))
				require.NoError(t, err)
				require.Equal(t, inv.Contact, to.Address)
				require.NotContains(t, string(body), "sumi_role_")
			case <-time.After(time.Second):
				t.Fatal("no SMTP message")
			}
		})
	}
}

func TestInvitationMailRejectsUnresolvedOrMismatchedRole(t *testing.T) {
	for _, test := range []struct {
		name            string
		change          func(*flow.Invitation, *invitationRoleStore)
		resolutionError error
	}{
		{name: "missing", change: func(inv *flow.Invitation, _ *invitationRoleStore) { inv.Role = authz.EnterpriseRoleKey("B", 2) }},
		{name: "unbound", change: func(_ *flow.Invitation, roles *invitationRoleStore) { roles.items = nil }},
		{name: "blank", change: func(_ *flow.Invitation, roles *invitationRoleStore) { roles.items[0].Name = " " }},
		{name: "foreign-role", change: func(inv *flow.Invitation, roles *invitationRoleStore) {
			inv.Role = authz.EnterpriseRoleKey("C", 1)
			roles.items[0].ID = inv.Role
		}},
		{name: "foreign-invitation", change: func(inv *flow.Invitation, _ *invitationRoleStore) { inv.OrganizationID = "C" }},
		{name: "foreign-project", change: func(inv *flow.Invitation, _ *invitationRoleStore) { inv.ProjectID = "other" }},
		{name: "read-failure", change: func(_ *flow.Invitation, roles *invitationRoleStore) { roles.err = errors.New("role store unavailable") }},
		{name: "live-scope-failure", resolutionError: membership.ErrUnavailable},
		{name: "retired-viewer", change: func(inv *flow.Invitation, _ *invitationRoleStore) { inv.Role = "listingkit_viewer" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			roles := &invitationRoleStore{items: []membership.RoleDefinition{{ID: authz.EnterpriseRoleKey("B", 1), Name: "商品运营"}}}
			service, ctx, messages := invitationMailFactoryFixture(t, roles, test.resolutionError)
			inv := invitationMailIntent(authz.EnterpriseRoleKey("B", 1))
			if test.change != nil {
				test.change(&inv, roles)
			}
			require.Error(t, service.Notify(ctx, inv))
			select {
			case <-messages:
				t.Fatal("unresolved or wrong-scope role sent mail")
			default:
			}
		})
	}
}

func invitationMailIntent(role string) flow.Invitation {
	return flow.Invitation{ID: uuid.NewString(), ProjectID: "project", OrganizationID: "B", OrganizationName: "测试企业", CreatorID: "actor", Contact: "recipient@example.test", Role: role, State: flow.Pending, ExpiresAt: time.Now().Add(time.Hour)}
}

type invitationMailAttemptStore struct {
	flow.Store
	inv                                                        flow.Invitation
	claimedID, finishedID, claimAttempt, finishAttempt, status string
}

func (s *invitationMailAttemptStore) Read(context.Context, string) (flow.Invitation, error) {
	return s.inv, nil
}

func (s *invitationMailAttemptStore) ClaimDelivery(_ context.Context, id, attempt string, _ time.Time) (flow.Invitation, error) {
	s.claimedID, s.claimAttempt = id, attempt
	return s.inv, nil
}

func (s *invitationMailAttemptStore) FinishDelivery(_ context.Context, id, attempt, status string, _ time.Time) (flow.Invitation, error) {
	s.finishedID, s.finishAttempt, s.status = id, attempt, status
	return s.inv, nil
}

func TestInvitationMailRoleDependencyFailureKeepsOriginalDeliveryUnknown(t *testing.T) {
	for name, roles := range map[string]membership.RoleStore{"read-failure": &invitationRoleStore{err: membership.ErrUnavailable}, "missing-store": nil} {
		t.Run(name, func(t *testing.T) {
			service, ctx, messages := invitationMailFactoryFixture(t, roles, nil)
			inv := invitationMailIntent(authz.EnterpriseRoleKey("B", 1))
			store := &invitationMailAttemptStore{inv: inv}
			service.Store = store
			result, err := service.Resend(ctx, inv.ID)
			require.NoError(t, err, "durable invitation remains readable after notification failure")
			require.Equal(t, inv, result)
			require.Equal(t, inv.ID, store.claimedID)
			require.Equal(t, inv.ID, store.finishedID)
			require.NotEmpty(t, store.claimAttempt)
			require.Equal(t, store.claimAttempt, store.finishAttempt)
			require.Equal(t, "delivery_unknown", store.status)
			select {
			case <-messages:
				t.Fatal("failed role read sent mail")
			default:
			}
		})
	}
}

func invitationMailFactoryFixture(t *testing.T, roles membership.RoleStore, resolutionError error) (*flow.Service, context.Context, <-chan string) {
	t.Helper()
	messages := make(chan string, 1)
	port := membershipSMTP(t, messages)
	authorizer, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	identity := authidentity.AuthenticatedIdentity{UserID: "actor", EffectiveOrganizationID: "B", Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Hour), OrganizationGrants: []authidentity.OrganizationGrant{{ProjectID: "project", OrganizationID: "B", Roles: []string{"listingkit_admin"}}}}
	auth := routeAuthDependencies{workbenchVerifier: mountedVerifierStub{identity: identity}, organizationResolver: &productReviewResolverStub{value: identity, err: resolutionError}}
	members := membership.NewService(&pointLimitDirectory{}, authorizer, "project")
	members.SetRoleStore(roles)
	factory, err := buildInvitationFactory(MembershipDependencies{ProviderOrigin: "http://127.0.0.1:9000", InvitationMail: &invitationmail.Config{Host: "127.0.0.1", Port: port, LocalPlaintext: true, From: "invitations@example.test", PublicOrigin: "https://localhost:22544"}}, "project", auth, members, &memberstore.Repository{}, nil, authorizer)
	require.NoError(t, err)
	req := httptest.NewRequest("POST", "/api/v1/account/invitations", nil)
	req.Header.Set("Authorization", "Bearer fixture-token")
	req = req.WithContext(authidentity.WithAuthenticatedIdentity(req.Context(), identity))
	service, err := factory(req)
	require.NoError(t, err)
	return service, req.Context(), messages
}
