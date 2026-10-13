package supplychainapp

import (
	"context"
	"github.com/stretchr/testify/require"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/storecenter"
	"testing"
)

func pathPointer[T any](v T) *T { return &v }

func TestSupplierRuleReaderVerifiesEveryAncestorAndRejectsForgedRoot(t *testing.T) {
	scope, f, r := newTemuRuleFixture(t, storecenter.MerchantFullyManaged)
	f.selection = model.RuleSelection{ParentCatID: 12, CatID: 34, SupplierCategoryPath: [10]int64{12, 34}}
	f.categoryPath = []model.SupplierCategory{
		{CatID: 12, ParentCatID: pathPointer(int64(0)), CatType: pathPointer(int64(0)), Level: pathPointer(int64(1)), Leaf: pathPointer(false), Hidden: pathPointer(false), HiddenType: pathPointer(int64(0))},
		{CatID: 34, ParentCatID: pathPointer(int64(12)), CatType: pathPointer(int64(0)), Level: pathPointer(int64(2)), Leaf: pathPointer(true), Hidden: pathPointer(false), HiddenType: pathPointer(int64(0))},
	}
	_, snapshot, err := r.ReadTemuTargetRules(context.Background(), scope, f.binding.StoreID, f.selection)
	require.NoError(t, err)
	require.Equal(t, f.categoryPath, snapshot.SupplierCategoryPath)
	require.True(t, snapshot.SelectableLeaf)
	f.selection.SupplierCategoryPath[0] = 999
	_, snapshot, err = r.ReadTemuTargetRules(context.Background(), scope, f.binding.StoreID, f.selection)
	require.ErrorIs(t, err, record.ErrNotReady)
	require.Empty(t, snapshot.SupplierCategoryPath)
}
