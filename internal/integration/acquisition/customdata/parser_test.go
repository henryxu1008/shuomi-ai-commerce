package customdata

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"strings"
	"task-processor/internal/dataservice"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
	"testing"
)

func TestSpecialistSpreadsheetZeroDoesNotDeclareOperatorStock(t *testing.T) {
	workbook := excelize.NewFile()
	defer workbook.Close()
	require.NoError(t, workbook.SetSheetRow("Sheet1", "A1", &collection.ImportHeaders))
	require.NoError(t, workbook.SetSheetRow("Sheet1", "A2", &[]any{"fixture", "", "", "", "SKU", "USD", 10, 0}))
	xlsx, err := workbook.WriteToBuffer()
	require.NoError(t, err)
	for format, raw := range map[string][]byte{
		"csv":  []byte(strings.Join(collection.ImportHeaders, ",") + "\nfixture,,,,SKU,USD,10,0\n"),
		"xlsx": xlsx.Bytes(),
	} {
		t.Run(format, func(t *testing.T) {
			rows, err := ParseDelivery(context.Background(), format, raw)
			require.NoError(t, err)
			require.Len(t, rows[0].Variants, 1)
			require.Zero(t, rows[0].Variants[0].Stock)
			envelope, err := collection.OwnEnvelope(collection.StableID("custom", format), rows[0])
			require.NoError(t, err)
			snapshot, err := sourcing.ToSnapshot(envelope)
			require.NoError(t, err)
			require.False(t, snapshot.Variants[0].OwnStockDeclared)
			require.Nil(t, snapshot.Variants[0].Stock)
			envelope.ProductCandidate.Variants[0].OwnStockDeclared = true
			require.False(t, sourcing.ValidOwnStockDeclarations(sourcing.ProducerDescriptor{Kind: "custom_dataset", Version: "v1"}, envelope))
		})
	}
}

func TestDeliveryJSONCSVAreStrictBoundedAndUseProductValidation(t *testing.T) {
	rows, err := ParseDelivery(context.Background(), "json", []byte(`[{"title":"fixture","description":"","images":[],"attributes":{"asin":"B000123456"}}]`))
	require.NoError(t, err)
	require.Equal(t, "B000123456", rows[0].Attributes["asin"])
	rows, err = ParseDelivery(context.Background(), "csv", []byte("title,description,brand,images,sku,currency,price,stock\nfixture,,,,,,,,\n"))
	require.ErrorIs(t, err, dataservice.ErrInvalid, "extra column is rejected")
	rows, err = ParseDelivery(context.Background(), "csv", []byte("title,description,brand,images,sku,currency,price,stock\nfixture,,,,SKU,USD,10,0\n"))
	require.NoError(t, err)
	require.Len(t, rows[0].Variants, 1)
	for _, raw := range []string{`[{"title":"one","title":"two","images":[]}]`, `[{"title":"one","organizationId":"foreign","images":[]}]`, `[{"title":"one","variants":[{"price":-1}],"images":[]}]`, strings.Repeat("x", 2<<20+1)} {
		_, err := ParseDelivery(context.Background(), "json", []byte(raw))
		require.ErrorIs(t, err, dataservice.ErrInvalid)
	}
	_, err = ParseDelivery(context.Background(), "csv", []byte("title,description,brand,images,sku,currency,price,stock\n"+strings.Repeat("fixture,,,,,,,\n", 201)))
	require.ErrorIs(t, err, dataservice.ErrInvalid)
}
