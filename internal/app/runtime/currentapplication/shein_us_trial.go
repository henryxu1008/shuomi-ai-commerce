package currentapplication

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"path"
	"time"

	strictjson "sigs.k8s.io/json"
	"task-processor/internal/integration/acquisition/zenrows"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
)

const (
	trialControlFileLimit = 8 << 10
	trialKeyFileLimit     = 4 << 10
	trialUsageFileLimit   = 64 << 10
)

// SHEINUSTrialFiles holds the frozen approval and the refreshable private inputs.
type SHEINUSTrialFiles struct {
	Approval   dataacquisition.SHEINUSTrialApproval
	Reconciler collection.Scope
	files      fs.FS
}

type trialScopeFile struct {
	OrganizationID string `json:"organizationId"`
	ActorID        string `json:"actorId"`
	MemberID       string `json:"memberId"`
}

type trialApprovalFile struct {
	Version    string                      `json:"version"`
	ID         string                      `json:"id"`
	ProjectID  string                      `json:"projectId"`
	AccountRef string                      `json:"accountRef"`
	KeyRef     string                      `json:"keyRef"`
	Scope      trialScopeFile              `json:"scope"`
	Reconciler trialScopeFile              `json:"reconciler"`
	Funding    orgresource.ResourceFunding `json:"funding"`
	ExpiresAt  time.Time                   `json:"expiresAt"`
}

type trialBudgetFile struct {
	AccountEvidenceRef    string    `json:"accountEvidenceRef"`
	ValidUntil            time.Time `json:"validUntil"`
	AvailableCredits      int       `json:"availableCredits"`
	MaximumRequestCredits int       `json:"maximumRequestCredits"`
	ZeroIncrementalCash   bool      `json:"zeroIncrementalCash"`
	ExclusiveKeyWindow    bool      `json:"exclusiveKeyWindow"`
}

type trialGateFile struct {
	Version             string                               `json:"version"`
	Binding             string                               `json:"binding"`
	ApprovalEvidenceRef string                               `json:"approvalEvidenceRef"`
	CheckedAt           time.Time                            `json:"checkedAt"`
	ValidUntil          time.Time                            `json:"validUntil"`
	AllowedActions      []dataacquisition.SHEINUSTrialAction `json:"allowedActions"`
	Budget              *trialBudgetFile                     `json:"budget"`
}

type trialKeyFile struct {
	Version    string `json:"version"`
	Binding    string `json:"binding"`
	AccountRef string `json:"accountRef"`
	KeyRef     string `json:"keyRef"`
	Secret     string `json:"secret"`
}

type trialUsageRecordFile struct {
	AccountRef string    `json:"accountRef"`
	KeyRef     string    `json:"keyRef"`
	URL        string    `json:"url"`
	RequestID  string    `json:"requestId"`
	AcceptedAt time.Time `json:"acceptedAt"`
	Final      bool      `json:"final"`
	Credits    *int      `json:"credits"`
}

type trialUsageFile struct {
	Version     string                 `json:"version"`
	Binding     string                 `json:"binding"`
	EvidenceRef string                 `json:"evidenceRef"`
	CheckedAt   time.Time              `json:"checkedAt"`
	Records     []trialUsageRecordFile `json:"records"`
}

func (cfg *Config) validateSHEINUSTrial() error {
	if cfg.SheinUSTrialInputDir == nil {
		return nil
	}
	dir := *cfg.SheinUSTrialInputDir
	if !path.IsAbs(dir) || dir == "/" || path.Clean(dir) != dir ||
		cfg.MarketResearch == nil || cfg.DataServices == nil || !cfg.ProductCollections || cfg.ProductAcquisitionDatabase == nil {
		return errors.New("SHEIN US trial requires an absolute private input directory, Market Research, Data Services and Product Collections")
	}
	return nil
}

func readTrialFile[T any](files fs.FS, name string, limit int64) (T, error) {
	var value *T
	var empty T
	file, err := files.Open(name)
	if err != nil {
		return empty, dataacquisition.ErrUnavailable
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(body)) > limit {
		return empty, dataacquisition.ErrUnavailable
	}
	strictErrors, err := strictjson.UnmarshalStrict(body, &value)
	if err != nil || len(strictErrors) != 0 || value == nil {
		return empty, dataacquisition.ErrUnavailable
	}
	return *value, nil
}

func loadSHEINUSTrialFiles(files fs.FS) (*SHEINUSTrialFiles, error) {
	input, err := readTrialFile[trialApprovalFile](files, "approval.json", trialControlFileLimit)
	if err != nil || input.Version != "shein-us-trial-approval@v1" {
		return nil, dataacquisition.ErrUnavailable
	}
	return &SHEINUSTrialFiles{
		Approval: dataacquisition.SHEINUSTrialApproval{
			ID: input.ID, ProjectID: input.ProjectID, AccountRef: input.AccountRef, KeyRef: input.KeyRef,
			Scope: collection.Scope(input.Scope), Funding: input.Funding, ExpiresAt: input.ExpiresAt,
		},
		Reconciler: collection.Scope(input.Reconciler), files: files,
	}, nil
}

func (f *SHEINUSTrialFiles) gate(ctx context.Context, approved dataacquisition.SHEINUSTrialApproval, action dataacquisition.SHEINUSTrialAction) (trialGateFile, error) {
	if ctx.Err() != nil || approved.Binding() != f.Approval.Binding() {
		return trialGateFile{}, dataacquisition.ErrForbidden
	}
	gate, err := readTrialFile[trialGateFile](f.files, "gate.json", trialControlFileLimit)
	now := time.Now()
	if err != nil || ctx.Err() != nil || gate.Version != "shein-us-trial-gate@v1" || gate.Binding != approved.Binding() ||
		!collection.ValidID(gate.ApprovalEvidenceRef) || gate.CheckedAt.IsZero() || gate.CheckedAt.After(now) || !now.Before(gate.ValidUntil) {
		return trialGateFile{}, dataacquisition.ErrForbidden
	}
	allowed := make(map[dataacquisition.SHEINUSTrialAction]bool, len(gate.AllowedActions))
	for _, candidate := range gate.AllowedActions {
		switch candidate {
		case dataacquisition.SHEINUSTrialView, dataacquisition.SHEINUSTrialSend, dataacquisition.SHEINUSTrialPublish, dataacquisition.SHEINUSTrialReconcile:
		default:
			return trialGateFile{}, dataacquisition.ErrForbidden
		}
		if allowed[candidate] {
			return trialGateFile{}, dataacquisition.ErrForbidden
		}
		allowed[candidate] = true
	}
	if !allowed[action] {
		return trialGateFile{}, dataacquisition.ErrForbidden
	}
	return gate, nil
}

func (f *SHEINUSTrialFiles) Access(ctx context.Context, _ dataacquisition.Principal, approved dataacquisition.SHEINUSTrialApproval, action dataacquisition.SHEINUSTrialAction) error {
	_, err := f.gate(ctx, approved, action)
	return err
}

func (f *SHEINUSTrialFiles) Budget(ctx context.Context, approved dataacquisition.SHEINUSTrialApproval) (dataacquisition.SHEINUSTrialBudgetEvidence, error) {
	gate, err := f.gate(ctx, approved, dataacquisition.SHEINUSTrialSend)
	deadline, bounded := ctx.Deadline()
	budget := gate.Budget
	if err != nil || !bounded || budget == nil || !collection.ValidID(budget.AccountEvidenceRef) ||
		deadline.After(approved.ExpiresAt) || deadline.After(gate.ValidUntil) || deadline.After(budget.ValidUntil) || ctx.Err() != nil {
		return dataacquisition.SHEINUSTrialBudgetEvidence{}, dataacquisition.ErrUnavailable
	}
	evidence := dataacquisition.SHEINUSTrialBudgetEvidence{
		ValidUntil: budget.ValidUntil,
		SHEINUSTrialBudget: dataacquisition.SHEINUSTrialBudget{
			Binding: gate.Binding, AvailableCredits: budget.AvailableCredits, MaximumRequestCredits: budget.MaximumRequestCredits,
			ZeroIncrementalCash: budget.ZeroIncrementalCash, ExclusiveKeyWindow: budget.ExclusiveKeyWindow,
		},
	}
	if !evidence.AllowsSend(gate.Binding) {
		return dataacquisition.SHEINUSTrialBudgetEvidence{}, dataacquisition.ErrUnavailable
	}
	return evidence, nil
}

func (f *SHEINUSTrialFiles) ReadKey(ctx context.Context, accountRef, keyRef string) (string, error) {
	approval := f.Approval
	if ctx.Err() != nil || accountRef != approval.AccountRef || keyRef != approval.KeyRef {
		return "", dataacquisition.ErrUnavailable
	}
	key, err := readTrialFile[trialKeyFile](f.files, "key.json", trialKeyFileLimit)
	if err != nil || key.Version != "shein-us-trial-key@v1" || key.Binding != approval.Binding() || key.AccountRef != accountRef || key.KeyRef != keyRef {
		return "", dataacquisition.ErrUnavailable
	}
	// The resolver performs no further I/O after this final coherent gate read.
	if _, err := f.Budget(ctx, approval); err != nil {
		return "", err
	}
	return key.Secret, nil
}

func (f *SHEINUSTrialFiles) ReadUsage(ctx context.Context, approved dataacquisition.SHEINUSTrialApproval, attempt dataacquisition.SHEINUSTrialAttempt) (zenrows.UsageEvidence, error) {
	if ctx.Err() != nil || !dataacquisition.ValidSHEINUSTrialRequestID(attempt.RequestID) || approved.Binding() != f.Approval.Binding() {
		return zenrows.UsageEvidence{}, dataacquisition.ErrUnknown
	}
	usage, err := readTrialFile[trialUsageFile](f.files, "usage.json", trialUsageFileLimit)
	if err != nil || ctx.Err() != nil || usage.Version != "shein-us-trial-usage@v1" || usage.Binding != attempt.Binding ||
		!collection.ValidID(usage.EvidenceRef) || usage.CheckedAt.IsZero() || usage.CheckedAt.Before(attempt.ClaimedAt) ||
		usage.CheckedAt.After(time.Now()) || len(usage.Records) > 32 {
		return zenrows.UsageEvidence{}, dataacquisition.ErrUnknown
	}
	records := make([]zenrows.UsageRecord, 0, len(usage.Records))
	for _, r := range usage.Records {
		records = append(records, zenrows.UsageRecord(r))
	}
	return zenrows.UsageEvidence{Binding: usage.Binding, Records: records}, nil
}
