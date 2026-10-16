package supplychainapp

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
	"testing"
	"time"
)

type readaptImportRecords struct{ *uploadRecords }

func (r readaptImportRecords) ReadTargetCommand(context.Context, collection.Scope, string) (record.TargetReceipt, error) {
	return record.TargetReceipt{}, record.ErrNotFound
}

type captureReadaptation struct{ input record.TargetInput }

func (c *captureReadaptation) CreateForExecution(_ context.Context, _ collection.Scope, _ string, in record.TargetInput) (record.TargetReceipt, error) {
	c.input = in
	return record.TargetReceipt{Record: record.TargetRecord{Input: in}}, nil
}
func TestReadaptationPreservesImportedBusinessFieldsWithoutReusingImportCommand(t *testing.T) {
	upload, scope, records, _, _, _, _ := uploadFixture(t)
	binding := &record.SheinImport{BaseRecordID: uuid.NewString(), CompletionDraft: record.CompletionDraftSelection{RecordID: uuid.NewString(), Revision: 3}, Selection: record.SheinImportSelection{Title: true}}
	records.saved.Input.SheinImport = binding
	creator := &captureReadaptation{}
	activity := OperationActivities{Sources: upload.dependencies.Sources, Products: upload.dependencies.Products, Targets: readaptImportRecords{records}, Creator: creator}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, err := activity.adapt(ctx, preparation.Operation{Owner: scope, Input: preparation.OperationInput{PreparationID: records.saved.Source.PreparationID, StoreID: records.saved.Input.StoreID}}, preparation.OperationItem{SourceID: records.saved.Source.ID}, uuid.NewString())
	require.NoError(t, err)
	expected := records.saved.Input
	expected.ExpectedRevision = records.saved.Revision
	expected.SheinImport = nil
	require.Equal(t, expected, creator.input)
	require.Same(t, binding, records.saved.Input.SheinImport, "immutable historical provenance remains")
}
