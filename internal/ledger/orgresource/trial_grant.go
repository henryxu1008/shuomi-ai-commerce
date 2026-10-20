package orgresource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	OperationGrantLocalAITrial                  = "grant_local_ai_trial"
	SourceLocalAITrial                          = "local_ai_trial_v1"
	TrialAIGrantQuantity          int64         = 10_000
	PrincipalTrustedTrialOperator PrincipalKind = "trusted_trial_operator"
)

type TrialAIGrantApproval struct {
	OrganizationID string
	OperatorID     string
	EvidenceID     string
	GrantEnabled   bool
}

// Runtime assembly verifies the protected operator identity and exact approved
// organization. Caller-supplied principal kinds alone never grant authority.
type TrialAIGrantAuthorizer interface {
	AuthorizeTrialAIGrant(context.Context, Principal, string) (TrialAIGrantApproval, error)
}

type TrialAIGrantInput struct {
	OrganizationID string
	OperationID    string
	Principal      Principal
}

// There is deliberately no caller-selectable resource, amount or campaign.
type TrialAIGrantExecution struct {
	OrganizationID     string
	OperationID        string
	ActorID            string
	ApprovalEvidenceID string
	SourceIdentity     string
	RequestFingerprint string
}

type TrialAIGrantSnapshot struct {
	OperationID        string       `json:"operation_id"`
	OrganizationID     string       `json:"organization_id"`
	ResourceType       ResourceType `json:"resource_type"`
	Quantity           string       `json:"quantity"`
	GrossCredit        string       `json:"gross_credit"`
	DebtRepaid         string       `json:"debt_repaid"`
	NetCredit          string       `json:"net_credit"`
	BalanceAfter       string       `json:"balance_after"`
	AllocatedAfter     string       `json:"allocated_after"`
	ReservedAfter      string       `json:"reserved_after"`
	ConsumedAfter      string       `json:"consumed_after"`
	SourceType         string       `json:"source_type"`
	SourceIdentity     string       `json:"source_identity"`
	ApprovalEvidenceID string       `json:"approval_evidence_id"`
	ActorID            string       `json:"actor_id"`
	EventID            string       `json:"event_id"`
}

type TrialAIGrantResult struct {
	Snapshot TrialAIGrantSnapshot `json:"snapshot"`
	Replayed bool                 `json:"replayed"`
}

type TrialAIGrantExecutor interface {
	ReplayTrialAIGrant(context.Context, TrialAIGrantExecution) (TrialAIGrantResult, bool, error)
	ExecuteTrialAIGrant(context.Context, TrialAIGrantExecution) (TrialAIGrantResult, error)
}

type TrialAIGrantService struct {
	executor   TrialAIGrantExecutor
	authorizer TrialAIGrantAuthorizer
}

func NewTrialAIGrantService(executor TrialAIGrantExecutor, authorizer TrialAIGrantAuthorizer) (*TrialAIGrantService, error) {
	if executor == nil || authorizer == nil {
		return nil, errors.New("trial grant executor and authorizer are required")
	}
	return &TrialAIGrantService{executor: executor, authorizer: authorizer}, nil
}

func (s *TrialAIGrantService) Inspect(ctx context.Context, input TrialAIGrantInput) (TrialAIGrantResult, bool, error) {
	execution, err := s.prepare(ctx, input, false)
	if err != nil {
		return TrialAIGrantResult{}, false, err
	}
	return s.executor.ReplayTrialAIGrant(ctx, execution)
}

func (s *TrialAIGrantService) Grant(ctx context.Context, input TrialAIGrantInput) (TrialAIGrantResult, error) {
	execution, err := s.prepare(ctx, input, true)
	if err != nil {
		return TrialAIGrantResult{}, err
	}
	if result, found, err := s.executor.ReplayTrialAIGrant(ctx, execution); err != nil || found {
		return result, err
	}
	return s.executor.ExecuteTrialAIGrant(ctx, execution)
}

func (s *TrialAIGrantService) prepare(ctx context.Context, input TrialAIGrantInput, grant bool) (TrialAIGrantExecution, error) {
	if ctx == nil {
		return TrialAIGrantExecution{}, ErrInvalidInput
	}
	if err := input.Validate(); err != nil {
		return TrialAIGrantExecution{}, err
	}
	if ctx.Err() != nil {
		return TrialAIGrantExecution{}, ctx.Err()
	}
	approval, err := s.authorizer.AuthorizeTrialAIGrant(ctx, input.Principal, input.OrganizationID)
	if err != nil || approval.OrganizationID != input.OrganizationID || approval.OperatorID != input.Principal.ID || !validTrialID(approval.EvidenceID, 192) || (grant && !approval.GrantEnabled) {
		return TrialAIGrantExecution{}, ErrForbidden
	}
	execution := TrialAIGrantExecution{OrganizationID: input.OrganizationID, OperationID: input.OperationID, ActorID: input.Principal.ID, ApprovalEvidenceID: approval.EvidenceID, SourceIdentity: TrialAIGrantSourceIdentity(input.OrganizationID)}
	execution.RequestFingerprint = trialAIGrantFingerprint(execution)
	return execution, nil
}

func (input TrialAIGrantInput) Validate() error {
	if !validTrialID(input.OrganizationID, 128) || !validTrialID(input.OperationID, 128) {
		return ErrInvalidInput
	}
	if !validTrialID(input.Principal.ID, 192) || input.Principal.Kind != PrincipalTrustedTrialOperator {
		return ErrForbidden
	}
	return nil
}

// Source identity never includes operation/deployment/time or replaceable
// approval evidence. A new approval cannot mint a second first-trial grant.
func TrialAIGrantSourceIdentity(organizationID string) string {
	return trialDigest([]string{SourceLocalAITrial, organizationID, string(ResourceAIPoint)})
}

func (input TrialAIGrantExecution) Validate() error {
	if !validTrialID(input.OrganizationID, 128) || !validTrialID(input.OperationID, 128) || !validTrialID(input.ActorID, 192) || !validTrialID(input.ApprovalEvidenceID, 192) || input.SourceIdentity != TrialAIGrantSourceIdentity(input.OrganizationID) || input.RequestFingerprint != trialAIGrantFingerprint(input) {
		return ErrInvalidInput
	}
	return nil
}

func trialAIGrantFingerprint(input TrialAIGrantExecution) string {
	return trialDigest([]string{OperationGrantLocalAITrial, input.OrganizationID, string(ResourceAIPoint), "10000", SourceLocalAITrial, input.SourceIdentity, input.ApprovalEvidenceID})
}

func trialDigest(value []string) string {
	encoded, _ := json.Marshal(value) // A string slice is always JSON encodable.
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func validTrialID(value string, maximum int) bool {
	return value != "" && value == strings.TrimSpace(value) && len(value) <= maximum && utf8.ValidString(value) && !strings.ContainsFunc(value, unicode.IsControl)
}
