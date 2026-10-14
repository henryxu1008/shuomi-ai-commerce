package ownerreconcile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type TenantDomain uint8

const (
	TenantDomainZITADELOrganization TenantDomain = iota
	TenantDomainLegacyNumeric
)

type CandidateColumn struct {
	Name   string
	Source string
}

type TableSpec struct {
	Table            string
	TenantDomain     TenantDomain
	CandidatePolicy  CandidatePolicy
	Query            string
	UpdateQuery      string
	UpdateLimitArg   int
	Columns          []string
	CandidateColumns []CandidateColumn
}

type Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Repository struct {
	Queryer    Queryer
	Inventory  []TableSpec
	Identities []LegacyIdentity
	Exceptions ExceptionStore
}

const postgresUndefinedTableSQLState = "42P01"

var ownerReconcileIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func (repository Repository) DryRun(ctx context.Context, identities []LegacyIdentity) (Report, error) {
	if repository.Queryer == nil {
		return Report{}, errors.New("owner reconciliation database is unavailable")
	}
	identityMap, err := indexLegacyIdentities(identities)
	if err != nil {
		return Report{}, err
	}
	exceptionIndex, err := repository.systemOwnedExceptionIndex(ctx)
	if err != nil {
		return Report{}, err
	}
	findings := make([]Finding, 0)
	systemOwnedFindings := make([]Finding, 0)
	resolutions := make([]Resolution, 0)
	var autoRows int64
	for _, spec := range repository.Inventory {
		if err := validateTableSpec(spec); err != nil {
			return Report{}, err
		}
		releaseSavepoint, err := beginTableSavepoint(ctx, repository.Queryer)
		if err != nil {
			return Report{}, err
		}
		rows, err := repository.Queryer.QueryContext(ctx, spec.Query)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				_ = releaseSavepoint(false)
				return Report{}, ctxErr
			}
			if isPostgresUndefinedTable(err) {
				if savepointErr := releaseSavepoint(false); savepointErr != nil {
					return Report{}, savepointErr
				}
				continue
			}
			_ = releaseSavepoint(false)
			return Report{}, fmt.Errorf("query owner reconciliation table %s failed", spec.Table)
		}
		rowsFindings, rowsSystemOwned, rowsAuto, rowsResolutions, scanErr := scanTableRows(ctx, rows, spec, identityMap, exceptionIndex)
		if scanErr != nil {
			_ = releaseSavepoint(false)
			return Report{}, scanErr
		}
		if err := releaseSavepoint(true); err != nil {
			return Report{}, err
		}
		findings = append(findings, rowsFindings...)
		systemOwnedFindings = append(systemOwnedFindings, rowsSystemOwned...)
		resolutions = append(resolutions, rowsResolutions...)
		autoRows += rowsAuto
	}
	return NewReportWithClassifiedFindings("", "", findings, systemOwnedFindings, autoRows, resolutions), nil
}

func (repository Repository) systemOwnedExceptionIndex(ctx context.Context) (map[string]int64, error) {
	if repository.Exceptions == nil {
		return map[string]int64{}, nil
	}
	items, err := repository.Exceptions.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	index := make(map[string]int64, len(items))
	for _, item := range items {
		if err := validateSystemOwnedException(item); err != nil {
			return nil, err
		}
		key := systemOwnedExceptionKey(item.Table, item.TenantFingerprint, item.CandidateFingerprint)
		if _, exists := index[key]; exists {
			return nil, errors.New("owner exception registry contains duplicate keys")
		}
		index[key] = item.Rows
	}
	return index, nil
}

const ownerReconcileTableSavepoint = "owner_reconcile_table"

func beginTableSavepoint(ctx context.Context, queryer Queryer) (func(bool) error, error) {
	tx, ok := queryer.(*sql.Tx)
	if !ok {
		return func(bool) error { return nil }, nil
	}
	if _, err := tx.ExecContext(ctx, "SAVEPOINT "+ownerReconcileTableSavepoint); err != nil {
		return nil, errors.New("owner reconciliation savepoint failed")
	}
	return func(keep bool) error {
		if !keep {
			if _, err := tx.ExecContext(ctx, "ROLLBACK TO SAVEPOINT "+ownerReconcileTableSavepoint); err != nil {
				return errors.New("owner reconciliation savepoint rollback failed")
			}
		}
		if _, err := tx.ExecContext(ctx, "RELEASE SAVEPOINT "+ownerReconcileTableSavepoint); err != nil {
			return errors.New("owner reconciliation savepoint release failed")
		}
		return nil
	}, nil
}

type sqlStateError interface {
	SQLState() string
}

func isPostgresUndefinedTable(err error) bool {
	var stateError sqlStateError
	return errors.As(err, &stateError) && stateError.SQLState() == postgresUndefinedTableSQLState
}

func validateTableSpec(spec TableSpec) error {
	if !ownerReconcileIdentifier.MatchString(spec.Table) {
		return errors.New("owner reconciliation inventory contains an invalid table identifier")
	}
	query := strings.TrimSpace(spec.Query)
	if query == "" || !strings.EqualFold(strings.TrimSpace(strings.SplitN(query, " ", 2)[0]), "select") {
		return fmt.Errorf("owner reconciliation table %s must use a SELECT query", spec.Table)
	}
	if len(spec.Columns) == 0 {
		return fmt.Errorf("owner reconciliation table %s has no result columns", spec.Table)
	}
	for _, column := range spec.Columns {
		if !ownerReconcileIdentifier.MatchString(column) {
			return errors.New("owner reconciliation inventory contains an invalid result identifier")
		}
	}
	for _, candidate := range spec.CandidateColumns {
		if !ownerReconcileIdentifier.MatchString(candidate.Name) || strings.TrimSpace(candidate.Source) == "" {
			return errors.New("owner reconciliation inventory contains an invalid candidate identifier")
		}
		found := false
		for _, column := range spec.Columns {
			if column == candidate.Name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("owner reconciliation table %s omitted candidate column %s", spec.Table, candidate.Name)
		}
	}
	if update := strings.TrimSpace(spec.UpdateQuery); update != "" {
		firstToken := strings.ToUpper(strings.TrimSpace(strings.SplitN(update, " ", 2)[0]))
		if firstToken != "UPDATE" && firstToken != "WITH" || strings.Contains(update, ";") {
			return fmt.Errorf("owner reconciliation table %s has an invalid update query", spec.Table)
		}
		if spec.UpdateLimitArg != len(spec.CandidateColumns)+3 {
			return fmt.Errorf("owner reconciliation table %s has an invalid update limit parameter", spec.Table)
		}
	}
	return nil
}

func indexLegacyIdentities(identities []LegacyIdentity) (map[string]string, error) {
	result := make(map[string]string, len(identities))
	for _, identity := range identities {
		tenantID := strings.TrimSpace(identity.TenantID)
		legacyUserID := strings.TrimSpace(identity.LegacyUserID)
		subject := strings.TrimSpace(identity.Subject)
		if tenantID == "" || legacyUserID == "" || subject == "" {
			return nil, errors.New("legacy identity metadata contains an incomplete mapping")
		}
		key := legacyIdentityKey(tenantID, legacyUserID)
		if _, exists := result[key]; exists {
			return nil, errors.New("legacy identity metadata contains a duplicate mapping")
		}
		result[key] = subject
	}
	return result, nil
}

func legacyIdentityKey(tenantID, legacyUserID string) string {
	return tenantID + "\x00" + legacyUserID
}

func scanTableRows(ctx context.Context, rows *sql.Rows, spec TableSpec, identities map[string]string, exceptionIndex map[string]int64) (findings []Finding, systemOwnedFindings []Finding, autoRows int64, resolutions []Resolution, resultErr error) {
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && resultErr == nil {
			resultErr = fmt.Errorf("close owner reconciliation rows for %s failed", spec.Table)
		}
	}()

	columnIndex := make(map[string]int, len(spec.Columns))
	for index, column := range spec.Columns {
		columnIndex[column] = index
	}
	tenantIndex, tenantOK := columnIndex["tenant_id"]
	countIndex, countOK := columnIndex["row_count"]
	if !tenantOK || !countOK {
		return nil, nil, 0, nil, fmt.Errorf("owner reconciliation table %s omitted tenant_id or row_count", spec.Table)
	}
	for rows.Next() {
		values := make([]any, len(spec.Columns))
		pointers := make([]any, len(values))
		for index := range values {
			pointers[index] = &values[index]
		}
		if err := rows.Scan(pointers...); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, nil, 0, nil, ctxErr
			}
			return nil, nil, 0, nil, fmt.Errorf("scan owner reconciliation rows for %s failed", spec.Table)
		}
		tenantID := sqlText(values[tenantIndex])
		rowCount, err := sqlInt64(values[countIndex])
		if err != nil {
			return nil, nil, 0, nil, fmt.Errorf("scan row count for %s failed", spec.Table)
		}
		fingerprintParts := make([]string, 0, len(spec.CandidateColumns))
		rawCandidates := make([]Candidate, 0, len(spec.CandidateColumns))
		for _, candidateColumn := range spec.CandidateColumns {
			value := sqlText(values[columnIndex[candidateColumn.Name]])
			rawCandidates = append(rawCandidates, Candidate{Source: candidateColumn.Source, Subject: value})
			if value == "" {
				continue
			}
			fingerprintParts = append(fingerprintParts, candidateColumn.Source+"="+value)
		}
		candidateFingerprint := shortFingerprint(strings.Join(fingerprintParts, ";"))
		if expectedRows, exempted := exceptionIndex[systemOwnedExceptionKey(spec.Table, shortFingerprint(tenantID), candidateFingerprint)]; exempted && expectedRows == rowCount {
			systemOwnedFindings = append(systemOwnedFindings, Finding{
				Table: spec.Table, TenantFingerprint: shortFingerprint(tenantID), OwnerFingerprint: candidateFingerprint,
				Rows: rowCount, Reason: "system_owned",
			})
			continue
		}
		subject, reason := resolveCandidateValues(spec.CandidatePolicy, tenantID, rawCandidates, identities)
		if subject != "" {
			autoRows += rowCount
			resolutions = append(resolutions, Resolution{
				Table:                spec.Table,
				TenantFingerprint:    shortFingerprint(tenantID),
				CandidateFingerprint: shortFingerprint(strings.Join(fingerprintParts, ";")),
				SubjectFingerprint:   shortFingerprint(subject),
				Rows:                 rowCount,
			})
			continue
		}
		finding := Finding{
			Table:             spec.Table,
			TenantFingerprint: shortFingerprint(tenantID),
			OwnerFingerprint:  shortFingerprint(strings.Join(fingerprintParts, ";")),
			Rows:              rowCount,
			Reason:            reason,
		}
		if reason == "system_owned" || reason == "no_candidate" {
			if reason == "no_candidate" {
				finding.Reason = "system_owned"
			}
			systemOwnedFindings = append(systemOwnedFindings, finding)
		} else {
			findings = append(findings, finding)
		}
	}
	if err := rows.Err(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, nil, 0, nil, ctxErr
		}
		return nil, nil, 0, nil, fmt.Errorf("iterate owner reconciliation rows for %s failed", spec.Table)
	}
	return findings, systemOwnedFindings, autoRows, resolutions, nil
}

func sqlText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case []byte:
		return strings.TrimSpace(string(typed))
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func sqlInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case int32:
		return int64(typed), nil
	case int:
		return int64(typed), nil
	case []byte:
		return strconv.ParseInt(string(typed), 10, 64)
	case string:
		return strconv.ParseInt(typed, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported count type %T", value)
	}
}
