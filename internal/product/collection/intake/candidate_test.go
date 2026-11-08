package intake

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func naturalStockFixture(stock string) (Input, Candidate) {
	lines := []string{"黑 M，SKU BLK-M，CNY 99，" + stock, "黑 L，SKU BLK-L，CNY 99，库存未填写", "白 M，SKU WHT-M，CNY 99，库存为 5，图片2"}
	in := Input{Notes: strings.Join(lines, "\n"), GalleryCount: 2, SelectedImage: 1}
	c := Candidate{Title: "黑白T恤", Rows: []Row{
		{Attributes: map[string]string{"颜色": "黑", "尺码": "M"}, Evidence: lines[0], SKU: "BLK-M", Currency: "CNY", Price: new(99.0), Stock: new(0)},
		{Attributes: map[string]string{"颜色": "黑", "尺码": "L"}, Evidence: lines[1], SKU: "BLK-L", Currency: "CNY", Price: new(99.0)},
		{Attributes: map[string]string{"颜色": "白", "尺码": "M"}, Evidence: lines[2], SKU: "WHT-M", Currency: "CNY", Price: new(99.0), Stock: new(5), Image: 2},
	}}
	return in, c
}

func TestCandidateNaturalStockKeepsThreeRowsAndUnknown(t *testing.T) {
	for _, phrase := range []string{"库存明确为 0", "库存明确为0", "库存为 0", "库存是 0", "库存数量为 0", "库存：0", "stock is 0", "库存明确为 0 件", "库存明确为0，图片尚未确认"} {
		t.Run(phrase, func(t *testing.T) {
			in, c := naturalStockFixture(phrase)
			raw, err := json.Marshal(c)
			require.NoError(t, err)
			out, err := Decode(raw, in)
			require.NoError(t, err)
			require.Len(t, out.Rows, 3)
			require.Equal(t, new(0), out.Rows[0].Stock)
			require.Nil(t, out.Rows[1].Stock)
			require.Equal(t, new(5), out.Rows[2].Stock)
			require.Equal(t, 2, out.Rows[2].Image)
			encoded, err := json.Marshal(out.Rows)
			require.NoError(t, err)
			var rows []map[string]any
			require.NoError(t, json.Unmarshal(encoded, &rows))
			require.Contains(t, rows[0], "stock")
			require.NotContains(t, rows[1], "stock")
		})
	}
}

func TestCandidateStockDoesNotTurnUnknownOrAmbiguousIntoZero(t *testing.T) {
	for _, phrase := range []string{"库存未填写", "库存未知，不能填0", "库存不是0", "库存不为0", "库存为-1", "库存为0.5", "库存为0或5", "库存约为0", "库存为0左右", "库存为0，库存为5", "库存为0，尚未确认", "库存为0.0", "库存为0，库存大约5", "库存为0，库存未知", "并非库存为0", "未确认库存为0", "not stock is 0", "大约库存为0"} {
		t.Run(phrase, func(t *testing.T) {
			in, c := naturalStockFixture(phrase)
			err := c.Validate(in)
			require.ErrorIs(t, err, ErrInvalid)
			require.ErrorContains(t, err, "rows[0].stock")
		})
	}
	in, c := naturalStockFixture("库存明确为0")
	c.Rows[1].Stock = new(0)
	require.ErrorContains(t, c.Validate(in), "rows[1].stock")
	c.Rows[1].Stock = nil
	c.Rows[0].Evidence += "\n" + c.Rows[2].Evidence
	require.ErrorContains(t, c.Validate(in), "rows[0].evidence")
}

func TestStockLabelsIgnoreSKUFragmentsAndKeepUnconfirmedUnknown(t *testing.T) {
	for _, sku := range []string{"STOCK-M", "INVENTORY-M", "STOCK123", "SKU_STOCK_M", "stock"} {
		in, c := naturalStockFixture("库存明确为0")
		in.Notes = strings.ReplaceAll(in.Notes, "BLK-M", sku)
		c.Rows[0].Evidence = strings.ReplaceAll(c.Rows[0].Evidence, "BLK-M", sku)
		c.Rows[0].SKU = sku
		require.NoError(t, c.Validate(in), sku)
	}
	for _, phrase := range []string{"库存为0，待确认", "库存为0，未确认", "库存为0，未确定", "库存为0，待核实"} {
		in, c := naturalStockFixture(phrase)
		require.ErrorContains(t, c.Validate(in), "rows[0].stock", phrase)
	}
	for _, line := range []string{"黑 M 库存 5 SKU BLK-M", "黑 M 库存为5 CNY 99 SKU BLK-M", "黑 M 库存 5 SKU stock", "黑 M 库存 5 SKU STOCK-123"} {
		in := Input{Notes: line}
		sku := strings.Fields(line)[len(strings.Fields(line))-1]
		c := Candidate{Rows: []Row{{Attributes: map[string]string{"颜色": "黑", "尺码": "M"}, Evidence: line, SKU: sku, Stock: new(5)}}}
		if strings.Contains(line, "CNY 99") {
			c.Rows[0].Currency = "CNY"
			c.Rows[0].Price = new(99.0)
		}
		require.NoError(t, c.Validate(in), line)
	}
}

func TestStockChecksEveryDeclarationWithoutGuessingCurrency(t *testing.T) {
	for _, phrase := range []string{"库存为0，库存待核实", "库存为0，库存尚未确认", "库存为0，库存明确未知", "库存为0，库存数量未填写", "库存 0 not confirmed", "库存 0 not 5"} {
		t.Run(phrase, func(t *testing.T) {
			in, c := naturalStockFixture(phrase)
			require.ErrorContains(t, c.Validate(in), "rows[0].stock")
		})
	}
}

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
