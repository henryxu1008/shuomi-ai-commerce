package listingapp

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	record "task-processor/internal/listing/record/target"
	"testing"
)

func TestLocalTrialTargetDeniedBeforeUploadEffects(t *testing.T) {
	for _, location := range []string{"record", "input"} {
		t.Run(location, func(t *testing.T) {
			service, scope, records, merchant, kernel, official, _ := uploadFixture(t)
			if location == "record" {
				records.saved.LocalTrial = &record.LocalTrial{ContractVersion: 1}
			} else {
				records.saved.Input.LocalTrial = &record.LocalTrial{ContractVersion: 1}
			}
			records.saved.Result.ReadyForUpload = true
			_, err := service.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
			require.ErrorIs(t, err, record.ErrForbidden)
			require.Zero(t, merchant.transforms)
			require.Zero(t, merchant.publishes)
			require.Empty(t, kernel.attempts)
			require.Empty(t, official.intents)
			require.Empty(t, official.receipts)
		})
	}
}
