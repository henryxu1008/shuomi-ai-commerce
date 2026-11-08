// Package modelinvocationrecovery assembles one explicitly approved recovery.
// It has no provider client, schema installer, HTTP endpoint or product writer.
package modelinvocationrecovery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	sigjson "sigs.k8s.io/json"
	"task-processor/internal/aicapability"
	aistore "task-processor/internal/aicapability/store"
	"task-processor/internal/app/schema/productagentruntime"
	coreconfig "task-processor/internal/core/config"
	resourceadapter "task-processor/internal/integration/orgresource"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/platform/database"
)

var ErrRecoveryUnavailable = errors.New("original invocation recovery unavailable; inspect the same configured identity")
var ErrRecoveryNotApproved = errors.New("original invocation recovery is not explicitly approved")

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type Config struct {
	OrganizationID              string                                 `json:"organizationId"`
	MemberID                    string                                 `json:"memberId"`
	InvocationID                string                                 `json:"invocationId"`
	ReservationID               string                                 `json:"reservationId"`
	ExpectedDispatchFingerprint string                                 `json:"expectedDispatchFingerprint"`
	ExpectedProofDigest         string                                 `json:"expectedProofDigest"`
	OperatorID                  string                                 `json:"operatorId"`
	ApprovalEvidenceID          string                                 `json:"approvalEvidenceId"`
	ObservedAt                  time.Time                              `json:"observedAt"`
	Evidence                    aicapability.ProviderRejectionEvidence `json:"evidence"`
	ResolveEnabled              bool                                   `json:"resolveEnabled"`
	InvocationDatabase          DatabaseConfig                         `json:"invocationDatabase"`
	ResourceDatabase            DatabaseConfig                         `json:"resourceDatabase"`
}

type Output struct {
	OperationKey       string                                     `json:"operationKey"`
	InvocationID       string                                     `json:"invocationId"`
	Outcome            aicapability.InvocationOutcome             `json:"outcome"`
	BindingDigest      string                                     `json:"bindingDigest"`
	Reservation        resourceadapter.ModelInvocationReservation `json:"reservation"`
	ProofReady         bool                                       `json:"proofReady"`
	ProofDigest        string                                     `json:"proofDigest,omitempty"`
	Settled            bool                                       `json:"settled"`
	SettledMicropoints *int64                                     `json:"settledMicropoints,omitempty"`
}

func LoadConfig(ctx context.Context, path string) (Config, error) {
	if coreconfig.VerifyPrivateFiles(ctx, []string{path}) != nil {
		return Config{}, fmt.Errorf("%w: protected file permissions", ErrRecoveryUnavailable)
	}
	file, err := os.Open(path)
	if err != nil {
		return Config{}, ErrRecoveryUnavailable
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, (16<<10)+1))
	defer clear(raw)
	if err != nil || len(raw) > 16<<10 {
		return Config{}, ErrRecoveryUnavailable
	}
	var cfg Config
	strict, err := sigjson.UnmarshalStrict(raw, &cfg, sigjson.DisallowDuplicateFields, sigjson.DisallowUnknownFields)
	if err != nil || len(strict) != 0 {
		return Config{}, fmt.Errorf("%w: protected configuration format", ErrRecoveryUnavailable)
	}
	return cfg, nil
}

func (c DatabaseConfig) connection() *database.Config {
	return &database.Config{Host: c.Host, Port: c.Port, User: c.User, Password: c.Password, Database: c.Database, MaxConnections: 1, MaxIdleConnections: 1, ConnectionMaxLifetime: time.Minute}
}

func Run(ctx context.Context, cfg Config, action string) (Output, error) {
	if ctx == nil || action != "inspect" && action != "resolve" || cfg.OrganizationID == "" || cfg.MemberID == "" || cfg.InvocationID == "" || cfg.ReservationID == "" ||
		cfg.InvocationDatabase.User != "product_agent_runtime" || cfg.ResourceDatabase.User != resourceadapter.RuntimeRole {
		return Output{}, ErrRecoveryUnavailable
	}
	if action == "resolve" && (!cfg.ResolveEnabled || cfg.OperatorID == "" || cfg.ApprovalEvidenceID == "" || cfg.ExpectedDispatchFingerprint == "" || cfg.ExpectedProofDigest == "") {
		return Output{}, ErrRecoveryNotApproved
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	open := database.OpenExistingReadOnlyContext
	if action == "resolve" {
		open = database.OpenExistingWritableContext
	}
	native, err := open(ctx, cfg.InvocationDatabase.connection())
	if err != nil {
		return Output{}, ErrRecoveryUnavailable
	}
	defer database.Close(native)
	if productagentruntime.VerifyInvocationRecoveryRuntime(ctx, native) != nil {
		return Output{}, fmt.Errorf("%w: invocation runtime qualification", ErrRecoveryUnavailable)
	}
	commercial, err := open(ctx, cfg.ResourceDatabase.connection())
	if err != nil {
		return Output{}, ErrRecoveryUnavailable
	}
	defer database.Close(commercial)
	if resourceadapter.VerifyRuntimePermissions(ctx, commercial) != nil {
		return Output{}, fmt.Errorf("%w: resource runtime qualification", ErrRecoveryUnavailable)
	}
	recorder := aistore.NewGormInvocationRecorder(native)
	repo, err := resourceadapter.NewGormModelInvocationRepository(commercial, resourceadapter.TransactionConfig{}, recorder, recoveryAuthorizer{cfg: cfg})
	if err != nil {
		return Output{}, ErrRecoveryUnavailable
	}
	recorder.SetUsageSettler(repo)
	return inspectOrResolve(ctx, cfg, action, recorder, repo)
}

type recoveryAuthorizer struct{ cfg Config }

func (a recoveryAuthorizer) AuthorizeModelInvocation(_ context.Context, fact aicapability.InvocationRecord) error {
	if fact.TenantID != a.cfg.OrganizationID || fact.MemberID != a.cfg.MemberID || fact.InvocationID != a.cfg.InvocationID || fact.Operation != aicapability.OperationOwnProductIntake || fact.ProviderID != "openai" || fact.ModelID != aicapability.SingleImageMiniModel {
		return orgresource.ErrForbidden
	}
	return nil
}

func inspectOrResolve(ctx context.Context, cfg Config, action string, recorder *aistore.GormInvocationRecorder, repo *resourceadapter.GormModelInvocationRepository) (Output, error) {
	fact, found, err := recorder.FindInvocation(ctx, cfg.OrganizationID, cfg.MemberID, cfg.InvocationID, "")
	if err != nil || !found || (recoveryAuthorizer{cfg: cfg}).AuthorizeModelInvocation(ctx, fact) != nil {
		return Output{}, ErrRecoveryUnavailable
	}
	reservation, err := repo.InspectModelInvocationReservation(ctx, cfg.OrganizationID, cfg.InvocationID)
	if err != nil || reservation.ReservationID != cfg.ReservationID || cfg.ExpectedDispatchFingerprint != "" && reservation.DispatchFingerprint != cfg.ExpectedDispatchFingerprint {
		return Output{}, ErrRecoveryUnavailable
	}
	proof := &aicapability.ProviderRejectionProof{
		Version: aicapability.ProviderRejectionProofV1, BindingDigest: aicapability.InvocationBindingDigest(fact), DispatchFingerprint: reservation.DispatchFingerprint,
		ReservationID: reservation.ReservationID, ReservedMicropoints: reservation.Quantity, MemberMonth: reservation.MemberMonth,
		OperatorID: cfg.OperatorID, ApprovalEvidenceID: cfg.ApprovalEvidenceID, OperationKey: aicapability.ProviderRejectionOperationKey(fact.InvocationID), ObservedAt: cfg.ObservedAt, Evidence: cfg.Evidence,
	}
	proof.Digest = aicapability.ProviderRejectionDigest(*proof)
	terminal := fact
	terminal.ProviderRejectionProof = proof
	terminal.Outcome, terminal.FinishedAt = aicapability.InvocationFailed, proof.Evidence.RejectedAt
	terminal.UsageKnown, terminal.BillableUsage = false, nil
	terminal.PromptTokens, terminal.CompletionTokens, terminal.TotalTokens, terminal.ImageCount = 0, 0, 0, 0
	terminal.EstimatedCostKnown, terminal.EstimatedCostMicros = true, 0
	terminal.ErrorCategory, terminal.ErrorCode = aicapability.ErrorProviderRejected, aicapability.ProviderRejectedUnbilledCode
	ready := aicapability.ValidateProviderRejection(terminal) == nil && (fact.Outcome == aicapability.InvocationDispatched || fact.ProviderRejectionProof != nil && fact.ProviderRejectionProof.Digest == proof.Digest)
	out := Output{OperationKey: proof.OperationKey, InvocationID: fact.InvocationID, Outcome: fact.Outcome, BindingDigest: proof.BindingDigest, Reservation: reservation, ProofReady: ready}
	if ready {
		out.ProofDigest = proof.Digest
	}
	if action == "resolve" {
		if !ready || cfg.ExpectedProofDigest != proof.Digest {
			return Output{}, aicapability.ErrProviderRejectionProof
		}
		if err = recorder.ResolveDispatchedInvocation(ctx, terminal); err != nil {
			return Output{}, ErrRecoveryUnavailable
		}
		return inspectOrResolve(ctx, cfg, "inspect", recorder, repo)
	}
	if fact.ProviderRejectionProof != nil {
		out.Settled = repo.ConfirmModelInvocationSettlement(ctx, fact.TenantID, fact.InvocationID, fact.AgentRunID) == nil
		if out.Settled {
			zero := int64(0)
			out.SettledMicropoints = &zero
		}
	}
	return out, nil
}
