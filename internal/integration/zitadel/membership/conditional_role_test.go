package membership

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"task-processor/internal/authz"
	domain "task-processor/internal/organization/membership"
)

func conditionalOperation() domain.Operation {
	role := authz.EnterpriseRoleKey("org", 2)
	return domain.Operation{Scope: domain.OperationScope{OrganizationID: "org", ProjectID: "project", ActorID: "actor"}, Kind: domain.CommandBusinessRoleGrant, TargetUserID: "target", AuthorizationID: "grant", Role: role, Phase: domain.PhaseDispatched, Step: domain.StepConditionalRole, DispatchID: "attempt", ConditionalRole: &domain.ConditionalRolePlan{Native: domain.NativeAuthorizationRevision{InstanceID: "instance", OrganizationID: "org", ProjectID: "project", AuthorizationID: "grant", UserID: "target", NativeSequence: "7", State: "active", Roles: []string{"listingkit_admin", "platform_admin"}}, Direction: "ADD", AfterRoles: []string{"listingkit_admin", "platform_admin", role}}}
}

func TestConditionalWriterUsesOneNativeDeltaAndOnlyTrustsBoundNoEffect(t *testing.T) {
	for _, outcome := range []string{"APPLIED", "VERSION_CONFLICT_NO_EFFECT", "foreign-proof", "http-409"} {
		t.Run(outcome, func(t *testing.T) {
			op := conditionalOperation()
			calls := 0
			server := authorizedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/zitadel.authorization.v2.AuthorizationService/ChangeAuthorizationRoleConditional" {
					t.Errorf("unexpected native path: %s", r.URL.Path)
				}
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body["expectedSequence"] != "7" || body["roleKey"] != op.Role || body["direction"] != "ADD" || body["dispatchId"] != "attempt" || body["roleKeys"] != nil {
					t.Errorf("unsafe delta: %v", body)
				}
				if outcome == "http-409" {
					w.WriteHeader(409)
					return
				}
				e := domain.RoleWriteEvidence{Result: outcome, InstanceID: "instance", OrganizationID: "org", ProjectID: "project", AuthorizationID: "grant", UserID: "target", ExpectedSequence: "7", RoleKey: op.Role, Direction: "ADD", DispatchID: "attempt"}
				response := map[string]any{"evidence": e}
				if outcome == "APPLIED" {
					e.NativeSequence = "8"
					response["evidence"] = e
					response["changeDate"] = "2026-10-29T00:00:00Z"
				}
				if outcome == "foreign-proof" {
					e.Result = domain.RoleWriteNoEffect
					e.OrganizationID = "foreign"
					response["evidence"] = e
				}
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()
			writer, _ := NewWriter(server.URL, "read", "write", "project", server.Client())
			ack, err := writer.Write(context.Background(), op)
			var noEffect *domain.ProviderNoEffectError
			if outcome == "APPLIED" {
				if err != nil || ack.ConditionalRole == nil {
					t.Fatalf("applied=%+v %v", ack, err)
				}
			} else if outcome == "VERSION_CONFLICT_NO_EFFECT" {
				if !errors.As(err, &noEffect) {
					t.Fatalf("lost proven no-effect: %v", err)
				}
			} else if err == nil || errors.As(err, &noEffect) {
				t.Fatalf("untrusted error became known no effect: %v", err)
			}
			if calls != 1 {
				t.Fatalf("provider calls=%d", calls)
			}
		})
	}
}

func TestNativeRevisionReadRequiresExactScopeAndNativeSequence(t *testing.T) {
	for _, invalid := range []string{"", "enterprise", "organization", "semver"} {
		t.Run(invalid, func(t *testing.T) {
			revision := conditionalOperation().ConditionalRole.Native
			revision.ChangedAt = "2026-10-29T00:00:00Z"
			if invalid == "enterprise" {
				revision.ProjectGrantID = "native-enterprise-grant"
			}
			if invalid == "organization" {
				revision.OrganizationID = "foreign"
			}
			if invalid == "semver" {
				revision.NativeSequence = "v1"
			}
			server := authorizedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if invalid == "enterprise" {
					var request map[string]any
					_ = json.NewDecoder(r.Body).Decode(&request)
					if _, present := request["projectGrantId"]; present {
						t.Error("native grant discovery was replaced with an empty grant assertion")
					}
				}
				if r.URL.Path != "/zitadel.authorization.v2.AuthorizationService/GetAuthorizationRevision" {
					t.Errorf("wrong revision route %s", r.URL.Path)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"revision": revision})
			}))
			defer server.Close()
			client, _ := NewClient(server.URL, "read", "project", server.Client())
			got, err := client.ReadAuthorizationRevision(context.Background(), "org", "grant")
			if invalid == "" || invalid == "enterprise" {
				if err != nil || got.NativeSequence != "7" {
					t.Fatalf("revision=%+v %v", got, err)
				}
				if invalid == "enterprise" && got.ProjectGrantID != "native-enterprise-grant" {
					t.Fatal("lost discovered grant scope")
				}
			} else if err == nil {
				t.Fatal("invalid native revision was accepted")
			}
		})
	}
}
