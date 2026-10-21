package temporal

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	sdkactivity "go.temporal.io/sdk/activity"

	"task-processor/internal/imageagent"
	"task-processor/internal/imageagent/objectstore"
)

// The SDK supplies activity correlation. Custom fields are fixed categories
// and an opaque run hash; errors/URLs and business payloads are never logged.
func outputRecoveryDiagnostic(ctx context.Context, runID, stage string, err error) {
	if !sdkactivity.IsActivity(ctx) {
		return
	}
	reason := "dependency_error"
	switch {
	case errors.Is(err, context.Canceled):
		reason = "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		reason = "deadline_exceeded"
	case errors.Is(err, imageagent.ErrRevisionConflict):
		reason = "revision_conflict"
	case errors.Is(err, imageagent.ErrValidation):
		reason = "validation_error"
	case errors.Is(err, objectstore.ErrArtifactUnavailable):
		reason = "artifact_unavailable"
	case errors.Is(err, objectstore.ErrObjectConflict):
		reason = "object_conflict"
	}
	sdkactivity.GetLogger(ctx).Warn("Image output recovery diagnostic",
		"event", "image_output_recovery_diagnostic", "stage", stage, "reason", reason,
		"run_id_hash", fmt.Sprintf("%x", sha256.Sum256([]byte(runID))))
}
