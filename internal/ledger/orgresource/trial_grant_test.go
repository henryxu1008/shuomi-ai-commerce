package orgresource

import (
	"context"
	"errors"
	"testing"
)

type trialTestAuthorizer struct{ approval TrialAIGrantApproval }

func (a trialTestAuthorizer) AuthorizeTrialAIGrant(context.Context, Principal, string) (TrialAIGrantApproval, error) {
	return a.approval, nil
}

type trialTestExecutor struct {
	reads, writes int
	execution     TrialAIGrantExecution
}

func (e *trialTestExecutor) ReplayTrialAIGrant(_ context.Context, input TrialAIGrantExecution) (TrialAIGrantResult, bool, error) {
	e.reads++
	e.execution = input
	return TrialAIGrantResult{}, false, nil
}

func (e *trialTestExecutor) ExecuteTrialAIGrant(_ context.Context, input TrialAIGrantExecution) (TrialAIGrantResult, error) {
	e.writes++
	e.execution = input
	return TrialAIGrantResult{}, nil
}

func trialDomainFixture() (TrialAIGrantInput, TrialAIGrantApproval) {
	return TrialAIGrantInput{OrganizationID: "org-local", OperationID: "trial-op-1", Principal: Principal{ID: "operator", Kind: PrincipalTrustedTrialOperator}},
		TrialAIGrantApproval{OrganizationID: "org-local", OperatorID: "operator", EvidenceID: "human-approved-trial", GrantEnabled: true}
}

func TestTrialAIGrantRejectsInvalidAuthorityBeforeAnyRepositoryAccess(t *testing.T) {
	for _, name := range []string{"cross-org", "missing-evidence", "wrong-approval-org", "wrong-operator", "tenant-human", "disabled", "empty-operation", "control-character"} {
		t.Run(name, func(t *testing.T) {
			input, approval := trialDomainFixture()
			switch name {
			case "cross-org":
				input.OrganizationID = "org-other"
			case "missing-evidence":
				approval.EvidenceID = ""
			case "wrong-approval-org":
				approval.OrganizationID = "org-other"
			case "wrong-operator":
				approval.OperatorID = "another"
			case "tenant-human":
				input.Principal.Kind = PrincipalTenantHuman
			case "disabled":
				approval.GrantEnabled = false
			case "empty-operation":
				input.OperationID = ""
			case "control-character":
				input.OperationID = "op\x00next"
			}
			executor := &trialTestExecutor{}
			service, err := NewTrialAIGrantService(executor, trialTestAuthorizer{approval})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = service.Grant(context.Background(), input); err == nil {
				t.Fatal("unauthorized grant accepted")
			}
			if executor.reads != 0 || executor.writes != 0 {
				t.Fatal("invalid grant reached repository")
			}
		})
	}
}

func TestTrialAIGrantSourceAndFingerprintIgnoreOperationButBindEvidence(t *testing.T) {
	input, approval := trialDomainFixture()
	executor := &trialTestExecutor{}
	service, err := NewTrialAIGrantService(executor, trialTestAuthorizer{approval})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Grant(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	first := executor.execution
	input.OperationID = "after-restart"
	if _, err = service.Grant(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if first.SourceIdentity != executor.execution.SourceIdentity || first.RequestFingerprint != executor.execution.RequestFingerprint {
		t.Fatal("new request can change source identity")
	}
	approval.EvidenceID = "different-approval"
	service, _ = NewTrialAIGrantService(executor, trialTestAuthorizer{approval})
	if _, err = service.Grant(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if first.SourceIdentity != executor.execution.SourceIdentity || first.RequestFingerprint == executor.execution.RequestFingerprint {
		t.Fatal("evidence is not bound independently of source")
	}
	if first.Validate() != nil {
		t.Fatal("canonical execution rejected")
	}
	first.SourceIdentity = "caller-chosen-source"
	if !errors.Is(first.Validate(), ErrInvalidInput) {
		t.Fatal("noncanonical source accepted")
	}
}

func TestTrialAIGrantInspectRemainsReadOnlyWhenGrantDisabled(t *testing.T) {
	input, approval := trialDomainFixture()
	approval.GrantEnabled = false
	executor := &trialTestExecutor{}
	service, _ := NewTrialAIGrantService(executor, trialTestAuthorizer{approval})
	_, found, err := service.Inspect(context.Background(), input)
	if err != nil || found || executor.reads != 1 || executor.writes != 0 {
		t.Fatalf("inspect: found=%v err=%v executor=%+v", found, err, executor)
	}
	if _, err = service.Grant(context.Background(), input); !errors.Is(err, ErrForbidden) {
		t.Fatalf("disabled grant: %v", err)
	}
}
