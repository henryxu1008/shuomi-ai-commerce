package httpapi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/product/catalog"
)

func TestWorkbenchProductProjectionIsBoundedAndTextOnly(t *testing.T) {
	p := projectWorkbenchProduct(catalog.ProductSnapshot{Title: strings.Repeat("衣", 300), Description: "描述", SellingPoints: []string{"卖点", "二", "三", "四", "五", "六"}, Images: []catalog.Image{{URL: "https://secret.invalid/image"}}})
	require.LessOrEqual(t, len(p.Title), 512)
	require.True(t, p.Truncated)
	require.Equal(t, "描述", p.Description)
	require.Len(t, p.SellingPoints, 5)
	require.NotContains(t, p.Title, "\ufffd")
}
