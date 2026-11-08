package intake

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

// ValidationIssue contains owner-defined identifiers only, never model text.
type ValidationIssue struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

func (e ValidationIssue) Error() string { return ErrInvalid.Error() + ": " + e.Field + ": " + e.Code }
func (e ValidationIssue) Unwrap() error { return ErrInvalid }

var issueField = regexp.MustCompile(`^(candidate|title|description|brand|attributes|questions|rows|rows\[([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\]\.(attributes|evidence|sku|price|stock|image))$`)

func (e ValidationIssue) valid() bool {
	if !issueField.MatchString(e.Field) {
		return false
	}
	switch e.Code {
	case "invalid_json", "invalid_schema", "invalid_value", "empty_candidate", "source_evidence", "duplicate_variant", "duplicate_sku", "price_evidence", "stock_evidence", "image_reference":
		return true
	}
	return false
}

// IssueToken uses the Run's existing unresolved metadata, not a second store.
func IssueToken(err error) string {
	var issue ValidationIssue
	if !errors.As(err, &issue) || !issue.valid() {
		issue = ValidationIssue{"candidate", "invalid_schema"}
	}
	raw, _ := json.Marshal(issue)
	return "own-intake-validation:" + string(raw)
}

func IssueFromToken(token string) *ValidationIssue {
	const prefix = "own-intake-validation:"
	if len(token) > 192 || !strings.HasPrefix(token, prefix) {
		return nil
	}
	var issue ValidationIssue
	raw := strings.TrimPrefix(token, prefix)
	if json.Unmarshal([]byte(raw), &issue) != nil || !issue.valid() {
		return nil
	}
	canonical, _ := json.Marshal(issue)
	if raw != string(canonical) {
		return nil
	}
	return &issue
}
