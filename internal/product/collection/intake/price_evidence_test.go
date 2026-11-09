package intake

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPriceEvidenceRejectsUnconfirmedAndPartialAmounts(t *testing.T) {
	cases := []struct {
		name, notes, currency string
		price                 float64
	}{
		{"range", "黑 M，CNY 10-20", "CNY", 10},
		{"contiguous field conflict", "黑 M，CNY 10，颜色黑价格20", "CNY", 10},
		{"contiguous brand pending", "黑 M，CNY 10，品牌 NIKE价格待定", "CNY", 10},
		{"currency is attribute", "黑 M，价格 10，品牌 CNY", "CNY", 10},
		{"symbol currency is attribute", "黑 M，$10，品牌 USD", "USD", 10},
		{"range end", "黑 M，CNY 10-20", "CNY", 20},
		{"range spaced", "黑 M，CNY 10 - 20", "CNY", 10},
		{"range tilde", "黑 M，价格 10～20，币种 CNY", "CNY", 10},
		{"range chinese", "黑 M，CNY 10 到 20", "CNY", 10},
		{"grouped prefix", "黑 M，USD 1,299.00", "USD", 1},
		{"grouped unsupported", "黑 M，USD 1,299.00", "USD", 1299},
		{"decimal comma", "黑 M，EUR 10,50", "EUR", 10},
		{"negated", "黑 M，不是 CNY 10，价格待定", "CNY", 10},
		{"negated only", "黑 M，并非 CNY 10", "CNY", 10},
		{"negated english", "黑 M，not USD 10", "USD", 10},
		{"unknown", "黑 M，CNY 10，价格待定", "CNY", 10},
		{"bare pending", "黑 M，CNY 10，尚未确认", "CNY", 10},
		{"postfix pending", "黑 M，CNY 10 pending", "CNY", 10},
		{"conflict", "黑 M，CNY 10 或 CNY 20，尚未确认", "CNY", 10},
		{"separate conflict", "黑 M，CNY 10，价格 20", "CNY", 10},
		{"currency conflict", "黑 M，CNY 10，USD 10", "CNY", 10},
		{"currency detached", "黑 M，CNY，USD 10", "CNY", 10},
		{"symbol currency mismatch", "黑 M，CNY，$10", "CNY", 10},
		{"approximate", "黑 M，约 CNY 10", "CNY", 10},
		{"approximate suffix", "黑 M，CNY 10 左右", "CNY", 10},
		{"invalid decimal", "黑 M，CNY 10.5.5", "CNY", 10.5},
		{"exponent prefix", "黑 M，CNY 10e2", "CNY", 10},
		{"identifier prefix", "黑 M，SKU PRICE-10，CNY", "CNY", 10},
		{"sku currency", "黑 M，SKU USD-10，CNY", "CNY", 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Candidate{Rows: []Row{{Attributes: map[string]string{"颜色": "黑", "尺码": "M"}, Evidence: tc.notes, Currency: tc.currency, Price: &tc.price}}}
			raw, err := json.Marshal(c)
			require.NoError(t, err)
			_, err = Decode(raw, Input{Notes: tc.notes})
			require.Equal(t, ValidationIssue{"rows[0].price", "price_evidence"}, err)
			c.Rows[0].Price = nil
			raw, err = json.Marshal(c)
			require.NoError(t, err)
			_, err = Decode(raw, Input{Notes: tc.notes})
			require.NoError(t, err, "unknown price remains representable")
		})
	}
}

func TestPriceEvidenceKeepsExplicitAmountsAndUnrelatedUnknownFields(t *testing.T) {
	for _, tc := range []struct {
		notes, currency string
		price           float64
	}{
		{"黑 M，CNY 10", "CNY", 10},
		{"黑 M，CNY 0", "CNY", 0},
		{"黑 M，CNY 10，品牌 NIKE", "CNY", 10},
		{"黑 M，CNY 10，品牌 GAP", "CNY", 10},
		{"黑 M，PVC，CNY 10", "CNY", 10},
		{"黑 M，ABS 20，CNY 10", "CNY", 10},
		{"黑 M，CNY 10，尺码 XXXL", "CNY", 10},
		{"黑 M，CNY 10，尺码 XXX", "CNY", 10},
		{"黑 M，USD 1299.00", "USD", 1299},
		{"黑 M，价格 10", "", 10},
		{"黑 M，币种 CNY，价格 10", "CNY", 10},
		{"黑 M，CNY 10，价格 10.00", "CNY", 10},
		{"黑 M，CNY 10，库存未填写", "CNY", 10},
		{"黑 M，CNY 10，图片尚未确认", "CNY", 10},
		{"黑 M，CNY 10 库存 0", "CNY", 10},
		{"黑 M，价格 CNY 10", "CNY", 10},
		{"黑 M，币种 USD，$10", "USD", 10},
		{"黑 M，币种 EUR，€10.25", "EUR", 10.25},
		{"黑 M，价格明确为 10，币种 CNY", "CNY", 10},
	} {
		t.Run(tc.notes, func(t *testing.T) {
			c := Candidate{Rows: []Row{{Attributes: map[string]string{"颜色": "黑", "尺码": "M"}, Evidence: tc.notes, Currency: tc.currency, Price: &tc.price}}}
			raw, err := json.Marshal(c)
			require.NoError(t, err)
			out, err := Decode(raw, Input{Notes: tc.notes})
			require.NoError(t, err)
			require.Equal(t, tc.price, *out.Rows[0].Price)
		})
	}
}

func TestPriceEvidenceCannotHideQualifiersInOriginalStatement(t *testing.T) {
	for _, notes := range []string{"不是黑 M，CNY 10，价格待定", "黑 M，CNY 10-20", "黑 M，CNY 10，CNY 20", "黑 M，CNY 10，待确认", "黑 M，CNY 10，颜色黑价格20", "黑 M，CNY 10，品牌 NIKE价格待定"} {
		c := Candidate{Rows: []Row{{Attributes: map[string]string{"颜色": "黑", "尺码": "M"}, Evidence: "黑 M，CNY 10", Currency: "CNY", Price: new(10.0)}}}
		raw, err := json.Marshal(c)
		require.NoError(t, err)
		_, err = Decode(raw, Input{Notes: notes})
		require.Equal(t, ValidationIssue{"rows[0].price", "price_evidence"}, err, notes)
	}
}

func TestPriceEvidenceIgnoresProductAbbreviations(t *testing.T) {
	for _, term := range []string{"PVC", "ABS", "TPU", "LED", "ABS 10", "LED 20W"} {
		for _, evidence := range []string{"黑 M，CNY 10", "材质 " + term + "，黑 M，CNY 10"} {
			t.Run(term+"/"+evidence, func(t *testing.T) {
				row := Row{Attributes: map[string]string{"颜色": "黑", "尺码": "M"}, Evidence: evidence, Currency: "CNY", Price: new(10.0)}
				raw, err := json.Marshal(Candidate{Rows: []Row{row}})
				require.NoError(t, err)
				_, err = Decode(raw, Input{Notes: "材质 " + term + "，黑 M，CNY 10"})
				require.NoError(t, err)
			})
		}
	}
}

func TestPriceEvidenceVariantScopesKeepQualifiersAndConflicts(t *testing.T) {
	for _, notes := range []string{
		"黑 M，CNY 10-20，白 L，CNY 30",
		"黑 M，不是 CNY 10，白 L，CNY 30",
		"黑 M，CNY 10，待确认，白 L，CNY 30",
		"黑 M，CNY 10，价格 20，白 L，CNY 30",
		"黑 M，CNY 10 或 白 L，CNY 30",
		"黑 M，CNY 10，或 白 L，CNY 30",
		"黑 M，CNY 10，黑 M，CNY 20，白 L，CNY 30",
		"不是黑 M，CNY 10，白 L，CNY 30",
		"黑 M，CNY 10，白 L，CNY 30，黑 M，CNY 20",
		"黑 M，CNY 10，白 L，CNY 30，待确认",
	} {
		t.Run(notes, func(t *testing.T) {
			rows := []Row{
				{Attributes: map[string]string{"颜色": "黑", "尺码": "M"}, Evidence: "黑 M，CNY 10", Currency: "CNY", Price: new(10.0)},
				{Attributes: map[string]string{"颜色": "白", "尺码": "L"}, Evidence: "白 L，CNY 30", Currency: "CNY", Price: new(30.0)},
			}
			if notes == "黑 M，不是 CNY 10，白 L，CNY 30" {
				rows[0].Evidence = "黑 M，不是 CNY 10"
			}
			raw, err := json.Marshal(Candidate{Rows: rows})
			require.NoError(t, err)
			_, err = Decode(raw, Input{Notes: notes})
			require.Error(t, err)
			require.Contains(t, err.Error(), "price_evidence")
		})
	}
}

func TestPriceEvidenceBindsSameLineVariants(t *testing.T) {
	for _, separator := range []string{"，", ", ", "；", "\n"} {
		t.Run(separator, func(t *testing.T) {
			rows := []Row{
				{Attributes: map[string]string{"颜色": "黑", "尺码": "M"}, Evidence: "黑 M，CNY 10", Currency: "CNY", Price: new(10.0)},
				{Attributes: map[string]string{"颜色": "白", "尺码": "L"}, Evidence: "白 L，CNY 20", Currency: "CNY", Price: new(20.0)},
			}
			raw, err := json.Marshal(Candidate{Rows: rows})
			require.NoError(t, err)
			_, err = Decode(raw, Input{Notes: rows[0].Evidence + separator + rows[1].Evidence})
			require.NoError(t, err)
		})
	}
}

func TestPriceEvidenceCannotUseAnAmountAsAVariantBoundary(t *testing.T) {
	rows := []Row{
		{Attributes: map[string]string{"颜色": "黑", "尺码": "M"}, Evidence: "黑 M，CNY 10", Currency: "CNY", Price: new(10.0)},
		{Attributes: map[string]string{"颜色": "CNY", "尺码": "20"}, Evidence: "CNY 20"},
	}
	raw, err := json.Marshal(Candidate{Rows: rows})
	require.NoError(t, err)
	_, err = Decode(raw, Input{Notes: "黑 M，CNY 10，CNY 20"})
	require.Equal(t, ValidationIssue{"rows[0].price", "price_evidence"}, err)
}

func TestPriceEvidenceCannotInventVariantBoundaries(t *testing.T) {
	for _, suffix := range []string{"20", "待确认", "20 M", "待确认 M"} {
		attributes := map[string]string{"尺码": suffix}
		if suffix == "20 M" || suffix == "待确认 M" {
			attributes = map[string]string{"颜色": strings.TrimSuffix(suffix, " M"), "尺码": "M"}
		}
		rows := []Row{
			{Attributes: map[string]string{"颜色": "黑", "尺码": "M"}, Evidence: "黑 M，CNY 10", Currency: "CNY", Price: new(10.0)},
			{Attributes: attributes, Evidence: suffix},
		}
		raw, err := json.Marshal(Candidate{Rows: rows})
		require.NoError(t, err)
		_, err = Decode(raw, Input{Notes: "黑 M，CNY 10，" + suffix})
		require.Equal(t, ValidationIssue{"rows[0].price", "price_evidence"}, err, suffix)
	}
}

func TestPriceEvidenceBindsTypedNumericVariantHeaders(t *testing.T) {
	rows := []Row{
		{Attributes: map[string]string{"尺码": "20"}, Evidence: "尺码 20，CNY 10", Currency: "CNY", Price: new(10.0)},
		{Attributes: map[string]string{"尺码": "30"}, Evidence: "尺码 30，CNY 20", Currency: "CNY", Price: new(20.0)},
	}
	raw, err := json.Marshal(Candidate{Rows: rows})
	require.NoError(t, err)
	_, err = Decode(raw, Input{Notes: rows[0].Evidence + "，" + rows[1].Evidence})
	require.NoError(t, err)
}
