package membership

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"task-processor/internal/authz"
)

// Decode the wire shape so the regression can run before the new protocol
// fields exist. The baseline rejects a proven zero-effect result and accepts
// a conditional dispatch without a frozen native revision.
func conditionalFixture(t *testing.T) (Operation, OperationChange, map[string]any) {
	t.Helper()
	role := authz.EnterpriseRoleKey("effective-b", 2)
	proof := map[string]any{"result": "VERSION_CONFLICT_NO_EFFECT", "instanceId": "instance", "organizationId": "effective-b", "projectId": "project", "projectGrantId": "", "authorizationId": "grant", "userId": "target", "expectedSequence": "7", "roleKey": role, "direction": "ADD", "dispatchId": "attempt"}
	payload := map[string]any{"scope": map[string]any{"projectId": "project", "organizationId": "effective-b", "actorId": "actor"}, "kind": "business_role_grant", "targetUserId": "target", "authorizationId": "grant", "role": role, "phase": "dispatched", "step": "conditional_role", "revision": 2, "dispatchId": "attempt", "conditionalRole": map[string]any{"native": map[string]any{"instanceId": "instance", "organizationId": "effective-b", "projectId": "project", "projectGrantId": "", "authorizationId": "grant", "userId": "target", "nativeSequence": "7", "roles": []string{"listingkit_admin", "platform_admin"}, "state": "active"}, "direction": "ADD", "afterRoles": []string{"listingkit_admin", "platform_admin", role}}}
	var op Operation
	data, _ := json.Marshal(payload)
	if err := json.Unmarshal(data, &op); err != nil {
		t.Fatal(err)
	}
	change := conditionalChange(t, proof)
	return op, change, proof
}

type nativeDirectoryFixture struct {
	*directoryStub
	revision NativeAuthorizationRevision
}

func (d *nativeDirectoryFixture) ReadAuthorizationRevision(context.Context, string, string) (NativeAuthorizationRevision, error) {
	return d.revision, nil
}

type conditionalWriterFixture struct {
	calls  int
	result string
	sent   Operation
}

func (w *conditionalWriterFixture) Write(_ context.Context, op Operation) (Acknowledgment, error) {
	w.calls++
	w.sent = op
	r := op.ConditionalRole.Native
	e := RoleWriteEvidence{Result: w.result, InstanceID: r.InstanceID, OrganizationID: r.OrganizationID, ProjectID: r.ProjectID, ProjectGrantID: r.ProjectGrantID, AuthorizationID: r.AuthorizationID, UserID: r.UserID, ExpectedSequence: r.NativeSequence, NativeSequence: "8", RoleKey: op.Role, Direction: op.ConditionalRole.Direction, DispatchID: op.DispatchID}
	if w.result == "lost" {
		return Acknowledgment{}, errors.New("response lost")
	}
	if w.result == RoleWriteNoEffect {
		e.NativeSequence = ""
		return Acknowledgment{}, &ProviderNoEffectError{Evidence: e}
	}
	return Acknowledgment{ID: op.AuthorizationID, At: time.Now().UTC().Format(time.RFC3339Nano), ConditionalRole: &e}, nil
}

func TestBusinessRoleCommandUsesNativeRevisionAndNeverResends(t *testing.T) {
	for _, outcome := range []string{RoleWriteApplied, RoleWriteNoEffect, "lost"} {
		t.Run(outcome, func(t *testing.T) {
			role := authz.EnterpriseRoleKey("effective-b", 3)
			member := Member{ID: "grant", UserID: "target", OrganizationID: "effective-b", ProjectID: "project", Roles: []string{"listingkit_admin", "platform_admin"}, State: "active", ChangedAt: "2026-10-29T00:00:00Z"}
			d := &nativeDirectoryFixture{directoryStub: &directoryStub{page: Page{Items: []Member{member}, Total: 1}}, revision: NativeAuthorizationRevision{InstanceID: "instance", OrganizationID: member.OrganizationID, ProjectID: member.ProjectID, ProjectGrantID: "native-enterprise-grant", AuthorizationID: member.ID, UserID: member.UserID, NativeSequence: "7", Roles: member.Roles, State: "active", ChangedAt: member.ChangedAt}}
			a, _ := authz.NewListingKitAuthorizer(nil, []string{"platform_admin"})
			s := NewService(d, a, "project", "platform_admin")
			s.SetRoleStore(roleStoreFixture{[]RoleDefinition{{ID: role, Modules: []string{"store-browser"}, Version: 1}}})
			store := &memoryReceipts{}
			writer := &conditionalWriterFixture{result: outcome}
			c := NewCommands(s, store, writer, func(ctx context.Context) (context.Context, error) { return ctx, nil })
			key := uuid.NewString()
			in := CommandInput{Kind: CommandBusinessRoleGrant, AuthorizationID: member.ID, Role: role, ExpectedVersion: observedVersion(member)}
			op, err := c.Execute(scopedContext("listingkit_admin"), key, in)
			if outcome == RoleWriteNoEffect {
				if !errors.Is(err, ErrConflict) || op.Phase != PhaseRejected {
					t.Fatalf("conflict=%+v %v", op, err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if writer.calls != 1 || writer.sent.ConditionalRole == nil || writer.sent.ConditionalRole.Native.NativeSequence != "7" {
				t.Fatalf("missing exact native dispatch: calls=%d", writer.calls)
			}
			if writer.sent.ConditionalRole.Native.ProjectGrantID != "native-enterprise-grant" {
				t.Fatal("dispatch lost canonical project-grant scope")
			}
			if outcome == RoleWriteApplied && op.Phase != PhaseCompleted {
				t.Fatal("committed ack not completed")
			}
			if outcome == "lost" && op.Phase != PhaseDispatched {
				t.Fatal("lost response released UNKNOWN")
			}
			_, _ = NewCommands(s, store, writer, c.refresh).Execute(scopedContext("listingkit_admin"), key, in)
			if writer.calls != 1 {
				t.Fatal("rebuilt command resent original dispatch")
			}
		})
	}
}

type afterConditionalDispatchStore struct {
	*memoryReceipts
	after func()
}

func (s *afterConditionalDispatchStore) Apply(ctx context.Context, scope OperationScope, key string, revision int64, change OperationChange) (Operation, error) {
	op, err := s.memoryReceipts.Apply(ctx, scope, key, revision, change)
	if err == nil && change.Event == EventDispatch && s.after != nil {
		s.after()
	}
	return op, err
}

func TestBusinessRoleAdmissionKeepsNarrowScopeAndRechecksActor(t *testing.T) {
	for _, scenario := range []string{"broad-role", "native-revoked", "actor-revoked-after-dispatch", "revoke-changed-definition"} {
		t.Run(scenario, func(t *testing.T) {
			role := authz.EnterpriseRoleKey("effective-b", 3)
			member := Member{ID: "grant", UserID: "target", OrganizationID: "effective-b", ProjectID: "project", Roles: []string{"listingkit_admin", "platform_admin"}, State: "active", ChangedAt: "2026-10-29T00:00:00Z"}
			modules := []string{"store-browser"}
			kind := CommandBusinessRoleGrant
			if scenario == "broad-role" {
				modules = append(modules, "members")
			}
			if scenario == "revoke-changed-definition" {
				modules = []string{"members"}
				kind = CommandBusinessRoleRevoke
				member.Roles = append(member.Roles, role)
			}
			d := &nativeDirectoryFixture{directoryStub: &directoryStub{page: Page{Items: []Member{member}, Total: 1}}, revision: NativeAuthorizationRevision{InstanceID: "instance", OrganizationID: member.OrganizationID, ProjectID: member.ProjectID, AuthorizationID: member.ID, UserID: member.UserID, NativeSequence: "7", Roles: member.Roles, State: "active", ChangedAt: member.ChangedAt}}
			if scenario == "native-revoked" {
				d.revision.Roles = []string{"listingkit_admin"}
				d.revision.NativeSequence = "8"
			}
			a, _ := authz.NewListingKitAuthorizer(nil, []string{"platform_admin"})
			s := NewService(d, a, "project", "platform_admin")
			s.SetRoleStore(roleStoreFixture{[]RoleDefinition{{ID: role, Modules: modules, Version: 2}}})
			a.SetRolePolicyReader(roleStoreFixture{[]RoleDefinition{{ID: role, Modules: modules, Version: 2}}})
			revoked := false
			store := &afterConditionalDispatchStore{memoryReceipts: &memoryReceipts{}}
			if scenario == "actor-revoked-after-dispatch" {
				store.after = func() { revoked = true }
			}
			writer := &conditionalWriterFixture{result: RoleWriteApplied}
			c := NewCommands(s, store, writer, func(ctx context.Context) (context.Context, error) {
				if revoked {
					return ctx, ErrPermission
				}
				return ctx, nil
			})
			key := uuid.NewString()
			input := CommandInput{Kind: kind, AuthorizationID: member.ID, Role: role, ExpectedVersion: observedVersion(member)}
			op, err := c.Execute(scopedContext("listingkit_admin"), key, input)
			if scenario == "revoke-changed-definition" {
				if err != nil || op.Phase != PhaseCompleted || writer.calls != 1 {
					t.Fatalf("valid scoped revoke failed: %s %v", op.Phase, err)
				}
				return
			}
			if writer.calls != 0 {
				t.Fatal("unsafe or revoked request reached provider")
			}
			if scenario == "actor-revoked-after-dispatch" {
				if op.Phase != PhaseDispatched {
					t.Fatal("post-dispatch revoke released UNKNOWN")
				}
				revoked = false
				_, _ = c.Execute(scopedContext("listingkit_admin"), key, input)
				if writer.calls != 0 {
					t.Fatal("restored actor resent old dispatch")
				}
			} else if !errors.Is(err, ErrConflict) || op.Phase != PhaseRejected {
				t.Fatalf("unsafe request not rejected: %s %v", op.Phase, err)
			}
		})
	}
}

func conditionalChange(t *testing.T, proof map[string]any) OperationChange {
	t.Helper()
	data, _ := json.Marshal(map[string]any{"Event": "provider_no_effect", "DispatchID": "attempt", "noEffect": proof})
	var change OperationChange
	if err := json.Unmarshal(data, &change); err != nil {
		t.Fatal(err)
	}
	return change
}

func TestProviderNoEffectTerminatesOnlyTheExactConditionalDispatch(t *testing.T) {
	op, change, _ := conditionalFixture(t)
	next, err := Transition(op, change)
	if err != nil || next.Phase != PhaseRejected || next.Revision != op.Revision+1 || next.DispatchID != op.DispatchID {
		t.Fatalf("proven native conflict should have a terminal receipt: phase=%s revision=%d err=%v", next.Phase, next.Revision, err)
	}
	var saved map[string]any
	raw, _ := json.Marshal(next)
	_ = json.Unmarshal(raw, &saved)
	if proof, ok := saved["providerRoleResult"].(map[string]any); !ok || proof["result"] != "VERSION_CONFLICT_NO_EFFECT" {
		t.Fatal("terminal receipt lost its zero-effect evidence")
	}
}

func TestProviderNoEffectRejectsForeignOrUnprovenResults(t *testing.T) {
	for _, field := range []string{"result", "instanceId", "organizationId", "projectId", "projectGrantId", "authorizationId", "userId", "expectedSequence", "roleKey", "direction", "dispatchId"} {
		t.Run(field, func(t *testing.T) {
			op, _, proof := conditionalFixture(t)
			proof[field] = "foreign"
			if _, err := Transition(op, conditionalChange(t, proof)); err == nil {
				t.Fatal("foreign proof released a dispatched operation")
			}
		})
	}
	for _, kind := range []CommandKind{CommandRole, CommandRemove, CommandInvite} {
		op, change, _ := conditionalFixture(t)
		op.Kind = kind
		if _, err := Transition(op, change); err == nil {
			t.Fatalf("legacy kind %s accepted zero-effect transition", kind)
		}
	}
	op, change, _ := conditionalFixture(t)
	change.DispatchID = "different-attempt"
	if _, err := Transition(op, change); err == nil {
		t.Fatal("foreign dispatch accepted")
	}
	change = OperationChange{Event: OperationEvent("provider_no_effect"), DispatchID: "attempt"}
	if _, err := Transition(op, change); err == nil {
		t.Fatal("missing proof accepted")
	}
}

func TestConditionalRoleCannotDispatchWithoutFrozenNativeRevision(t *testing.T) {
	op := Operation{Kind: CommandKind("business_role_grant"), Phase: PhaseReady, Step: OperationStep("conditional_role"), Revision: 1}
	if _, err := Transition(op, OperationChange{Event: EventDispatch, DispatchID: "attempt"}); err == nil {
		t.Fatal("conditional role dispatch accepted without immutable native revision and single-key plan")
	}
}
