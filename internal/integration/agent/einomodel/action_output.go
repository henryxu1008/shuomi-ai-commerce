package einomodel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/sirupsen/logrus"
	sigjson "sigs.k8s.io/json"
	"task-processor/internal/agent"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

// Only fixed protocol metadata may leave the decoder. JSON error messages and
// unknown field names can contain model/source text, so never log those.
type actionOutputError struct{ Code, Field string }

func (e *actionOutputError) Error() string { return "invalid agent action: " + e.Code + " " + e.Field }
func (e *actionOutputError) Unwrap() error { return governed.ErrInvalid }

func decodeModelAction(raw string) (agent.Action, error) {
	var action agent.Action
	if len(raw) > agent.MaxModelOutputBytes {
		return action, &actionOutputError{Code: "output_too_large"}
	}
	strict, err := sigjson.UnmarshalStrict([]byte(raw), &action)
	if err != nil {
		rejection := &actionOutputError{Code: "json_syntax"}
		// The strict decoder uses its own JSON error types. The standard
		// decoder is used only on the rejected response to classify a type
		// error; its result is never admitted as an alternate parse.
		var diagnostic agent.Action
		var typeError *json.UnmarshalTypeError
		if errors.As(json.Unmarshal([]byte(raw), &diagnostic), &typeError) {
			rejection.Code = "json_type"
			switch typeError.Field {
			case "Kind", "Tool.ID", "Tool.Version", "Candidate.Changes.Field", "Candidate.Changes.Value", "Candidate.Changes.EvidenceIDs", "Confidence.Field", "Confidence.Value", "Confidence.Known", "Unresolved", "ContextCitationIDs":
				rejection.Field = typeError.Field
			}
		}
		return agent.Action{}, rejection
	}
	if len(strict) != 0 {
		return agent.Action{}, &actionOutputError{Code: "unknown_or_duplicate_field"}
	}
	if action.Kind != "tool" && action.Kind != "propose" && action.Kind != "interrupt" {
		return agent.Action{}, &actionOutputError{Code: "invalid_action_kind", Field: "Kind"}
	}
	return action, nil
}

func reportActionOutputFailure(ctx context.Context, invocationID, raw string, err error) {
	var rejection *actionOutputError
	if !errors.As(err, &rejection) {
		return
	}
	invocationHash, outputHash := sha256.Sum256([]byte(invocationID)), sha256.Sum256([]byte(raw))
	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"component": "product-agent", "event": "agent_action_output_rejected",
		"category": rejection.Code, "field": rejection.Field,
		"invocation_id_hash": hex.EncodeToString(invocationHash[:]), "output_hash": hex.EncodeToString(outputHash[:]),
	}).Warn("agent action output rejected")
}
