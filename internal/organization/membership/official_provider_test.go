package membership

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authz"
)

type retiredDirectory struct {
	*directoryStub
	probes int
}

// Historical method shape: official detail must never discover or call it.
func (d *retiredDirectory) ReadAuthorizationRevision(context.Context, string, string) (NativeAuthorizationRevision, error) {
	d.probes++
	return NativeAuthorizationRevision{}, ErrUnavailable
}

type countedReceipts struct {
	*memoryReceipts
	begins, changes int
}

func (s *countedReceipts) Begin(ctx context.Context, op Operation) (Operation, error) {
	s.begins++
	return s.memoryReceipts.Begin(ctx, op)
}
func (s *countedReceipts) Apply(ctx context.Context, scope OperationScope, key string, revision int64, change OperationChange) (Operation, error) {
	s.changes++
	return s.memoryReceipts.Apply(ctx, scope, key, revision, change)
}

func TestOfficialProviderProtectedDetailIsReadOnlyWithoutCustomProbe(t *testing.T) {
	for _, protected := range []string{"platform_admin", "configured-platform-admin"} {
		t.Run(protected, func(t *testing.T) {
			member := Member{ID: "grant", UserID: "target", OrganizationID: "effective-b", ProjectID: "project", Roles: []string{protected}, State: "active"}
			d := &retiredDirectory{directoryStub: &directoryStub{page: Page{Items: []Member{member}, Total: 1}}}
			a, _ := authz.NewListingKitAuthorizer(nil, []string{"configured-platform-admin"})
			s := testService(d, a, "project", "configured-platform-admin")
			s.SetRoleStore(roleStoreFixture{[]RoleDefinition{{ID: authz.EnterpriseRoleKey("effective-b", 3), Modules: []string{"store-browser"}, Version: 1}}})
			result, err := s.Read(scopedContext("listingkit_admin"), "grant")
			require.NoError(t, err)
			require.False(t, result.Items[0].CanChangeRole)
			require.False(t, result.Items[0].CanRemove)
			require.Zero(t, d.probes)
			raw, err := json.Marshal(result)
			require.NoError(t, err)
			require.NotContains(t, string(raw), "businessRoleActions")
		})
	}
}

func TestRetiredBusinessRoleCommandsCannotClaimOrWrite(t *testing.T) {
	for _, kind := range []CommandKind{CommandBusinessRoleGrant, CommandBusinessRoleRevoke} {
		t.Run(string(kind), func(t *testing.T) {
			a, _ := authz.NewListingKitAuthorizer(nil, nil)
			member := Member{ID: "grant", UserID: "target", OrganizationID: "effective-b", ProjectID: "project", Roles: []string{"platform_admin"}, State: "active"}
			s := testService(&directoryStub{page: Page{Items: []Member{member}, Total: 1}}, a, "project")
			s.SetRoleStore(roleStoreFixture{[]RoleDefinition{{ID: authz.EnterpriseRoleKey("effective-b", 3), Modules: []string{"store-browser"}, Version: 1}}})
			store := &countedReceipts{memoryReceipts: &memoryReceipts{}}
			writer := &writerStub{}
			c := NewCommands(s, store, writer, func(ctx context.Context) (context.Context, error) { return ctx, nil })
			_, err := c.Execute(scopedContext("listingkit_admin"), uuid.NewString(), CommandInput{Kind: kind, AuthorizationID: "grant", Role: authz.EnterpriseRoleKey("effective-b", 3), ExpectedVersion: strings.Repeat("a", 64)})
			require.Error(t, err)
			require.Zero(t, store.begins)
			require.Zero(t, store.changes)
			require.Zero(t, writer.calls)
		})
	}
}

func TestRetiredBusinessReceiptsRemainReadOnlyForEveryPhase(t *testing.T) {
	for _, kind := range []CommandKind{CommandBusinessRoleGrant, CommandBusinessRoleRevoke} {
		for _, phase := range []OperationPhase{PhaseReady, PhaseDispatched, PhaseCompleted, PhaseRejected} {
			t.Run(string(kind)+"/"+string(phase), func(t *testing.T) {
				a, _ := authz.NewListingKitAuthorizer(nil, nil)
				s := testService(&directoryStub{}, a, "project")
				op := Operation{Scope: OperationScope{ProjectID: "project", OrganizationID: "effective-b", ActorID: "actor"}, Key: uuid.NewString(), Kind: kind, Step: StepConditionalRole, Phase: phase, Revision: 2, AuthorizationID: "grant", TargetUserID: "target", DispatchID: "original"}
				store := &countedReceipts{memoryReceipts: &memoryReceipts{op: &op}}
				writer := &writerStub{}
				c := NewCommands(s, store, writer, func(ctx context.Context) (context.Context, error) { return ctx, nil })
				read, err := c.ReadOperation(scopedContext("listingkit_admin"), op.Key)
				require.NoError(t, err)
				require.Equal(t, op, read)
				verified, err := c.Verify(scopedContext("listingkit_admin"), op.Key)
				require.NoError(t, err)
				require.Equal(t, op, verified)
				require.Equal(t, op, *store.op)
				require.Zero(t, store.begins)
				require.Zero(t, store.changes)
				require.Zero(t, writer.calls)
				_, err = Transition(op, OperationChange{Event: EventDispatch, DispatchID: "new"})
				require.ErrorIs(t, err, ErrConflict)
			})
		}
	}
}

func TestHistoricalReceiptJSONKeepsOriginalNativeEvidence(t *testing.T) {
	const original = `{"scope":{"projectId":"project","organizationId":"effective-b","actorId":"actor"},"key":"9c767c09-041d-4792-ac06-4ea074daf09e","fingerprint":"original","kind":"business_role_grant","targetUserId":"target","authorizationId":"grant","expectedVersion":"original-version","role":"business-role","step":"conditional_role","phase":"completed","revision":3,"dispatchId":"original-attempt","userEvidence":"","conditionalRole":{"native":{"instanceId":"instance","organizationId":"effective-b","projectId":"project","projectGrantId":"native-grant","authorizationId":"grant","userId":"target","nativeSequence":"7","roles":["platform_admin"],"state":"active","changedAt":"2026-10-29T00:00:00Z"},"direction":"ADD","afterRoles":["platform_admin","business-role"]},"acknowledgment":{"id":"grant","at":"2026-10-29T00:01:00Z","conditionalRole":{"result":"APPLIED","instanceId":"instance","organizationId":"effective-b","projectId":"project","projectGrantId":"native-grant","authorizationId":"grant","userId":"target","expectedSequence":"7","nativeSequence":"8","roleKey":"business-role","direction":"ADD","dispatchId":"original-attempt"}},"providerRoleResult":{"result":"APPLIED","instanceId":"instance","organizationId":"effective-b","projectId":"project","projectGrantId":"native-grant","authorizationId":"grant","userId":"target","expectedSequence":"7","nativeSequence":"8","roleKey":"business-role","direction":"ADD","dispatchId":"original-attempt"}}`
	var op Operation
	require.NoError(t, json.Unmarshal([]byte(original), &op))
	raw, err := json.Marshal(op)
	require.NoError(t, err)
	require.JSONEq(t, original, string(raw))
	for _, event := range []OperationEvent{EventDispatch, EventAcknowledge, EventReject, OperationEvent("provider_no_effect")} {
		_, err := Transition(op, OperationChange{Event: event, DispatchID: op.DispatchID, Acknowledgment: op.Acknowledgment})
		require.ErrorIs(t, err, ErrConflict)
	}
}
