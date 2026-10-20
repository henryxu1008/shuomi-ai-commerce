// Package localaitrial assembles the bounded native trial grant. It is never
// mounted as an HTTP endpoint or invoked during serving startup.
package localaitrial

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	sigjson "sigs.k8s.io/json"
	coreconfig "task-processor/internal/core/config"
	orgresourceadapter "task-processor/internal/integration/orgresource"
	"task-processor/internal/ledger/orgresource"
	platformdatabase "task-processor/internal/platform/database"
)

var ErrOutcomeUnconfirmed = errors.New("trial grant result is unconfirmed; inspect the same operation ID before retrying")

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type Config struct {
	OrganizationID     string         `json:"organizationId"`
	OperatorID         string         `json:"operatorId"`
	ApprovalEvidenceID string         `json:"approvalEvidenceId"`
	GrantEnabled       bool           `json:"grantEnabled"`
	Database           DatabaseConfig `json:"database"`
}

type Output struct {
	Action         string                          `json:"action"`
	OrganizationID string                          `json:"organizationId"`
	SourceIdentity string                          `json:"sourceIdentity"`
	Found          bool                            `json:"found"`
	Receipt        *orgresource.TrialAIGrantResult `json:"receipt,omitempty"`
}

func LoadConfig(ctx context.Context, path string) (Config, error) {
	invalid := errors.New("private trial configuration is invalid or unavailable")
	if coreconfig.VerifyPrivateFiles(ctx, []string{path}) != nil {
		return Config{}, invalid
	}
	file, err := os.Open(path)
	if err != nil {
		return Config{}, invalid
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, (16<<10)+1))
	defer clear(raw)
	if err != nil || len(raw) > 16<<10 {
		return Config{}, invalid
	}
	var cfg Config
	strict, err := sigjson.UnmarshalStrict(raw, &cfg, sigjson.DisallowDuplicateFields, sigjson.DisallowUnknownFields)
	if err != nil || len(strict) != 0 {
		return Config{}, invalid
	}
	return cfg, nil
}

func (cfg Config) input(operationID string) orgresource.TrialAIGrantInput {
	return orgresource.TrialAIGrantInput{OrganizationID: cfg.OrganizationID, OperationID: operationID, Principal: orgresource.Principal{ID: cfg.OperatorID, Kind: orgresource.PrincipalTrustedTrialOperator}}
}

func (cfg Config) validate(operationID, action string) error {
	if action != "inspect" && action != "grant" {
		return orgresource.ErrInvalidInput
	}
	if err := cfg.input(operationID).Validate(); err != nil {
		return err
	}
	if cfg.ApprovalEvidenceID == "" || len(cfg.ApprovalEvidenceID) > 192 || cfg.ApprovalEvidenceID != strings.TrimSpace(cfg.ApprovalEvidenceID) || !utf8.ValidString(cfg.ApprovalEvidenceID) || strings.ContainsFunc(cfg.ApprovalEvidenceID, unicode.IsControl) {
		return orgresource.ErrInvalidInput
	}
	if cfg.Database.User != orgresourceadapter.RuntimeRole || cfg.Database.Host == "" || cfg.Database.Database == "" || cfg.Database.Password == "" || cfg.Database.Port < 1 || cfg.Database.Port > 65535 {
		return orgresource.ErrInvalidInput
	}
	if action == "grant" && !cfg.GrantEnabled {
		return orgresource.ErrForbidden
	}
	return nil
}

// Run only opens the existing resource database. Inspect additionally uses a
// read-only connection; no installer, grant, migration or provider is reachable.
func Run(ctx context.Context, cfg Config, operationID, action string) (Output, error) {
	if ctx == nil {
		return Output{}, orgresource.ErrInvalidInput
	}
	if err := cfg.validate(operationID, action); err != nil {
		return Output{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	database := &platformdatabase.Config{Host: cfg.Database.Host, Port: cfg.Database.Port, User: cfg.Database.User, Password: cfg.Database.Password, Database: cfg.Database.Database, MaxConnections: 1, MaxIdleConnections: 1, ConnectionMaxLifetime: time.Minute}
	open := platformdatabase.OpenExistingReadOnlyContext
	if action == "grant" {
		open = platformdatabase.OpenExistingWritableContext
	}
	db, err := open(ctx, database)
	if err != nil {
		return Output{}, errors.New("trial resource database is unavailable")
	}
	defer platformdatabase.Close(db)
	if orgresourceadapter.VerifyRuntimePermissions(ctx, db) != nil {
		return Output{}, errors.New("trial resource runtime permissions unavailable")
	}
	repository, err := orgresourceadapter.NewGormRepository(db, orgresourceadapter.TransactionConfig{})
	if err != nil {
		return Output{}, errors.New("trial resource repository unavailable")
	}
	return execute(ctx, cfg, operationID, action, repository)
}

type operatorAuthorizer struct{ cfg Config }

func (a operatorAuthorizer) AuthorizeTrialAIGrant(_ context.Context, principal orgresource.Principal, organizationID string) (orgresource.TrialAIGrantApproval, error) {
	if principal.Kind != orgresource.PrincipalTrustedTrialOperator || principal.ID != a.cfg.OperatorID || organizationID != a.cfg.OrganizationID {
		return orgresource.TrialAIGrantApproval{}, orgresource.ErrForbidden
	}
	return orgresource.TrialAIGrantApproval{OrganizationID: a.cfg.OrganizationID, OperatorID: a.cfg.OperatorID, EvidenceID: a.cfg.ApprovalEvidenceID, GrantEnabled: a.cfg.GrantEnabled}, nil
}

func execute(ctx context.Context, cfg Config, operationID, action string, executor orgresource.TrialAIGrantExecutor) (Output, error) {
	if err := cfg.validate(operationID, action); err != nil {
		return Output{}, err
	}
	service, err := orgresource.NewTrialAIGrantService(executor, operatorAuthorizer{cfg})
	if err != nil {
		return Output{}, errors.New("trial resource service unavailable")
	}
	output := Output{Action: action, OrganizationID: cfg.OrganizationID, SourceIdentity: orgresource.TrialAIGrantSourceIdentity(cfg.OrganizationID)}
	if action == "inspect" {
		result, found, err := service.Inspect(ctx, cfg.input(operationID))
		if err != nil {
			return Output{}, sanitizedTrialError(err, false)
		}
		output.Found = found
		if found {
			output.Receipt = &result
		}
		return output, nil
	}
	result, err := service.Grant(ctx, cfg.input(operationID))
	if err != nil {
		return Output{}, sanitizedTrialError(err, true)
	}
	output.Found, output.Receipt = true, &result
	return output, nil
}

func sanitizedTrialError(err error, grant bool) error {
	for _, known := range []error{orgresource.ErrForbidden, orgresource.ErrInvalidInput, orgresource.ErrIdempotencyKeyConflict} {
		if errors.Is(err, known) {
			return known
		}
	}
	if grant {
		return ErrOutcomeUnconfirmed
	}
	return errors.New("trial grant receipt is unavailable")
}
