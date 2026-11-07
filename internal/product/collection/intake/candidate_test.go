package intake

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCandidatePreservesSparseVariantsAndZero(t *testing.T) {
	input := Input{Notes: "黑 M，SKU BLK-M，CNY 29，库存 0\n黑 L，SKU BLK-L，CNY 29\n白 M，SKU WHT-M，CNY 29，图片2", GalleryCount: 2}
	raw := []byte(`{"title":"三规格上衣","rows":[{"attributes":{"颜色":"黑","尺码":"M"},"evidence":"黑 M，SKU BLK-M，CNY 29，库存 0","sku":"BLK-M","currency":"CNY","price":29,"stock":0},{"attributes":{"颜色":"黑","尺码":"L"},"evidence":"黑 L，SKU BLK-L，CNY 29","sku":"BLK-L","currency":"CNY","price":29},{"attributes":{"颜色":"白","尺码":"M"},"evidence":"白 M，SKU WHT-M，CNY 29，图片2","sku":"WHT-M","currency":"CNY","price":29,"image":2}]}`)
	proposal, err := Decode(raw, input)
	require.NoError(t, err)
	require.Len(t, proposal.Rows, 3)
	require.Equal(t, new(0), proposal.Rows[0].Stock)
	require.Nil(t, proposal.Rows[1].Stock)
	require.Equal(t, 2, proposal.Rows[2].Image)
	proposal.Rows[2].Attributes["尺码"] = "L"
	require.Error(t, proposal.Validate(input))
	proposal.Rows[2].Attributes["尺码"] = "M"
	proposal.Rows[2].Image = 3
	require.Error(t, proposal.Validate(input))
	proposal.Rows[2].Image = 2
	proposal.Rows[1].Stock = new(0)
	require.Error(t, proposal.Validate(input))
}
