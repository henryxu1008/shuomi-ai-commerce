package aicapability

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode"
)

const ProviderRejectionProofV1 = "own-intake-openai-unbilled-v1"
const ProviderRejectedUnbilledCode = "provider_rejected_unbilled"

var ErrProviderRejectionProof = errors.New("original invocation requires verified provider rejection and no-charge evidence")

// The commercial owner verifies its original receipt before a rejection can
// become immutable in the separate invocation database. This port never writes.
type ProviderRejectionReservationVerifier interface {
	VerifyProviderRejectionReservation(context.Context, InvocationRecord) error
}

// ProviderRejectionEvidence contains only bounded, non-secret references. A
// protected operator verifies the underlying provider material before sealing it.
// An HTTP status or aggregate usage total alone cannot populate this evidence.
type ProviderRejectionEvidence struct {
	Source                 string    `json:"source"`
	Reason                 string    `json:"reason"`
	HTTPStatus             int       `json:"httpStatus"`
	RejectedAt             time.Time `json:"rejectedAt"`
	RequestIDHash          string    `json:"requestIdHash"`
	ProviderRecordDigest   string    `json:"providerRecordDigest"`
	NoChargeReceiptHash    string    `json:"noChargeReceiptHash"`
	NoChargeEvidenceDigest string    `json:"noChargeEvidenceDigest"`
	ProjectIdentityHash    string    `json:"projectIdentityHash"`
	KeyIdentityHash        string    `json:"keyIdentityHash"`
	CorrelationDigest      string    `json:"correlationDigest,omitempty"`
	MatchingRequests       int       `json:"matchingRequests"`
	Handoffs               int       `json:"handoffs"`
}

type ProviderRejectionProof struct {
	Version             string                    `json:"version"`
	BindingDigest       string                    `json:"bindingDigest"`
	DispatchFingerprint string                    `json:"dispatchFingerprint"`
	ReservationID       string                    `json:"reservationId"`
	ReservedMicropoints int64                     `json:"reservedMicropoints"`
	MemberMonth         time.Time                 `json:"memberMonth"`
	OperatorID          string                    `json:"operatorId"`
	ApprovalEvidenceID  string                    `json:"approvalEvidenceId"`
	OperationKey        string                    `json:"operationKey"`
	ObservedAt          time.Time                 `json:"observedAt"`
	Evidence            ProviderRejectionEvidence `json:"evidence"`
	Digest              string                    `json:"digest"`
}

// InvocationDispatchIdentity removes only terminal fields. Commercial accounting
// retains its existing fixed JSON encoding around this original dispatch value.
func InvocationDispatchIdentity(fact InvocationRecord) InvocationRecord {
	fact.ProviderRejectionProof = nil
	fact.BillableUsage = nil
	fact.StartedAt = fact.StartedAt.UTC().Truncate(time.Microsecond)
	fact.FinishedAt = time.Time{}
	fact.LatencyMilliseconds = 0
	fact.PromptTokens, fact.CompletionTokens, fact.TotalTokens, fact.ImageCount = 0, 0, 0, 0
	fact.EstimatedCostMicros, fact.EstimatedCostKnown, fact.UsageKnown = 0, false, false
	fact.Outcome = InvocationDispatched
	fact.ErrorCategory, fact.ErrorCode, fact.RouteErrorCategory = "", "", ""
	fact.ProviderRequestID, fact.UpstreamJobID, fact.OutputHash = "", "", ""
	return fact
}

func rejectionHash(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// InvocationBindingDigest binds the native fact independently of the commercial
// ledger's fixed encoding. Both digests must match their respective owners.
func InvocationBindingDigest(fact InvocationRecord) string {
	return rejectionHash(InvocationDispatchIdentity(fact))
}

func ProviderRejectionOperationKey(invocationID string) string {
	return "unbilled-recovery-v1:" + rejectionHash(invocationID)
}

func ProviderRejectionDigest(proof ProviderRejectionProof) string {
	proof.Digest = ""
	proof.MemberMonth = proof.MemberMonth.UTC()
	proof.ObservedAt = proof.ObservedAt.UTC().Truncate(time.Microsecond)
	proof.Evidence.RejectedAt = proof.Evidence.RejectedAt.UTC().Truncate(time.Microsecond)
	return rejectionHash(proof)
}

func validRejectionHash(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && value == strings.ToLower(value)
}

func validRejectionIdentity(value string) bool {
	return value != "" && len(value) <= 192 && strings.TrimSpace(value) == value && !strings.ContainsFunc(value, unicode.IsControl)
}

func ValidateProviderRejection(fact InvocationRecord) error {
	p := fact.ProviderRejectionProof
	if p == nil || p.Version != ProviderRejectionProofV1 || fact.Outcome != InvocationFailed || fact.ErrorCategory != ErrorProviderRejected || fact.ErrorCode != ProviderRejectedUnbilledCode ||
		fact.Operation != OperationOwnProductIntake || fact.Capability != CapabilityProductEnrichText || fact.ProviderID != "openai" || fact.ModelID != SingleImageMiniModel ||
		fact.OfficialTariff.Channel != "openai-direct" || fact.OfficialTariff.UsageVersion != "openai-text-v1" || !fact.OfficialTariff.MatchesWireModel(fact.ModelID) ||
		fact.UsageKnown || fact.BillableUsage != nil || !fact.EstimatedCostKnown || fact.EstimatedCostMicros != 0 || fact.PromptTokens != 0 || fact.CompletionTokens != 0 || fact.TotalTokens != 0 || fact.ImageCount != 0 {
		return ErrProviderRejectionProof
	}
	for _, value := range []string{fact.TenantID, fact.UserID, fact.MemberID, fact.InvocationID, fact.AgentRunID, fact.InputHash, p.ReservationID, p.OperatorID, p.ApprovalEvidenceID} {
		if !validRejectionIdentity(value) {
			return ErrProviderRejectionProof
		}
	}
	e := p.Evidence
	for _, value := range []string{p.BindingDigest, p.DispatchFingerprint, p.Digest, e.RequestIDHash, e.ProviderRecordDigest, e.NoChargeReceiptHash, e.NoChargeEvidenceDigest, e.ProjectIdentityHash, e.KeyIdentityHash} {
		if !validRejectionHash(value) {
			return ErrProviderRejectionProof
		}
	}
	month := time.Date(fact.StartedAt.UTC().Year(), fact.StartedAt.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	reserved, err := fact.OfficialTariff.ReserveTextMicropoints(fact.MaximumPromptTokens, fact.MaximumCompletionTokens)
	if err != nil || reserved <= 0 || reserved != p.ReservedMicropoints || !p.MemberMonth.Equal(month) || fact.StartedAt.IsZero() || e.RejectedAt.Before(fact.StartedAt) ||
		p.ObservedAt.Before(e.RejectedAt) || !fact.FinishedAt.Equal(e.RejectedAt) || e.HTTPStatus != 429 || e.Reason != "billing_inactive" || e.MatchingRequests != 1 || e.Handoffs != 1 ||
		p.OperationKey != ProviderRejectionOperationKey(fact.InvocationID) || p.BindingDigest != InvocationBindingDigest(fact) || p.Digest != ProviderRejectionDigest(*p) {
		return ErrProviderRejectionProof
	}
	if e.Source != "provider_request_receipt" && e.Source != "verified_control_plane" || e.Source == "verified_control_plane" && !validRejectionHash(e.CorrelationDigest) {
		return ErrProviderRejectionProof
	}
	raw, err := json.Marshal(p)
	if err != nil || len(raw) > 8<<10 {
		return ErrProviderRejectionProof
	}
	return nil
}
