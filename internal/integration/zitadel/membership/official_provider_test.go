package membership

import (
	"context"
	"net/http"
	"testing"

	domain "task-processor/internal/organization/membership"
)

func TestRetiredKindsNeverFallBackToOfficialWrites(t *testing.T) {
	calls := 0
	server := authorizedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	writer, err := NewWriter(server.URL, "read", "write", "project", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []domain.CommandKind{domain.CommandBusinessRoleGrant, domain.CommandBusinessRoleRevoke} {
		for _, step := range []domain.OperationStep{domain.StepConditionalRole, domain.StepUser, domain.StepGrant, domain.StepRole, domain.StepRemove} {
			_, err := writer.Write(context.Background(), domain.Operation{Kind: kind, Scope: domain.OperationScope{ProjectID: "project", OrganizationID: "org"}, Step: step, Phase: domain.PhaseDispatched, DispatchID: "original", TargetUserID: "target", AuthorizationID: "grant", Role: "listingkit_admin", Invitation: &domain.Invitation{Email: "a@example.test", FirstName: "A", LastName: "B"}})
			if err == nil || calls != 0 {
				t.Fatalf("retired kind %s step %s: err=%v calls=%d", kind, step, err, calls)
			}
		}
	}
}
