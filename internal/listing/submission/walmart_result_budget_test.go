package submission

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/product/collection"
)

func TestBoundWalmartResultsPreservesAllReceiptIdentitiesAndCanonicalFacts(t *testing.T) {
	records := []WalmartResultRecord{}
	for n := 0; n < 3; n++ {
		stamp, hash := time.Now().UTC(), strings.Repeat("a", 64)
		item := WalmartResultRecord{ID: uuid.NewString(), ReceiptID: uuid.NewString(), Generation: 1, Latest: true, RepairableSKUs: []string{"sku-0"}, Observation: marketplace.FeedObservation{FeedID: fmt.Sprint("feed-", n), FeedStatus: "PROCESSED", Received: 50, Failed: 50, ObservedAt: stamp, ResponseHashes: []string{hash}}}
		expected := []string{}
		for i := range 50 {
			sku := fmt.Sprint("sku-", i)
			expected = append(expected, sku)
			item.Observation.Items = append(item.Observation.Items, marketplace.ItemResult{SKU: sku, IngestionStatus: "DATA_ERROR", Errors: []marketplace.ItemError{{Code: "DATA_ERROR", Description: strings.Repeat("x", 3000)}}})
		}
		item.ReadProgress = &WalmartReadProgress{Round: 1, ExpectedHash: collection.Digest(expected), NextOffset: 50, Pages: []WalmartReadPage{{Count: 50, ResponseHash: hash, ObservedAt: stamp}}}
		records = append(records, item)
	}
	before, err := json.Marshal(records)
	require.NoError(t, err)
	view, err := BoundWalmartResults(records, 12<<10)
	require.NoError(t, err)
	raw, err := json.Marshal(view)
	require.NoError(t, err)
	require.LessOrEqual(t, len(raw), 12<<10)
	require.Len(t, view, len(records))
	for n, item := range view {
		require.Equal(t, records[n].ReceiptID, item.ReceiptID)
		require.True(t, item.Limited)
		require.True(t, item.Observation.Partial)
		require.Empty(t, item.RepairableSKUs)
		require.NotNil(t, item.ReadOffset)
		require.Equal(t, 50, *item.ReadOffset)
	}
	after, err := json.Marshal(records)
	require.NoError(t, err)
	require.Equal(t, string(before), string(after), "display limits must never rewrite native facts or cursors")
}
