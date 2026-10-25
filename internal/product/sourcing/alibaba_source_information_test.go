package sourcing

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test1688SourceInformationSurvivesExistingV2MappingWithoutOwnedStock(t *testing.T) {
	evidence := acquisitionEvidenceFixture()
	evidence.SchemaVersion, evidence.ParserVersion = 2, "1688-browser-dom/v3"
	evidence.PriceFacts = []AcquisitionPrice{}
	evidence.Warnings = []SourceWarning{}
	evidence.MissingFacts = []MissingFact{}
	evidence.Images[0].Role = "source"
	evidence.Attributes = []AcquisitionAttribute{
		{Name: "来源计价单位", Value: "件"},
		{Name: "来源供应商名称", Value: "浦江福衣特服饰有限公司"},
		{Name: "来源发货地", Value: "浙江省金华市"},
		{Name: "来源可订数量（规格ID provider-red）", Value: "0"},
		{Name: "来源可订数量（规格ID provider-blue）", Value: "9007199254740993"},
	}
	evidence.Variants = []AcquisitionVariant{
		{SourceID: acquisitionString("provider-red"), Attributes: []AcquisitionAttribute{{Name: "颜色", Value: "红色"}},
			Price: &AcquisitionPrice{Amount: "27.00"}, Images: []AcquisitionImage{{URL: "https://cbu01.alicdn.com/red.jpg", Role: "source"}}},
		{SourceID: acquisitionString("provider-blue"), Attributes: []AcquisitionAttribute{{Name: "颜色", Value: "蓝色"}},
			Price: &AcquisitionPrice{Amount: "27.00"}, Images: []AcquisitionImage{{URL: "https://cbu01.alicdn.com/blue.jpg", Role: "source"}}},
	}

	raw, err := json.Marshal(map[string]any{"captureVersion": 2, "evidence": evidence})
	require.NoError(t, err)
	capture, err := ParseBrowserCapture(raw)
	require.NoError(t, err)
	envelope, err := MapBrowserCaptureEvidence(capture.Source, capture.Evidence, "source-information-test")
	require.NoError(t, err)
	snapshot, err := ToSnapshot(envelope)
	require.NoError(t, err)

	attributes := map[string]string{}
	for _, attribute := range snapshot.Attributes {
		attributes[attribute.Name] = attribute.Value
	}
	for _, fact := range evidence.Attributes {
		require.Equal(t, fact.Value, envelope.ProductCandidate.Attributes[fact.Name])
		require.Equal(t, fact.Value, attributes[fact.Name])
	}
	require.Len(t, snapshot.Variants, 2)
	for index, variant := range snapshot.Variants {
		require.Equal(t, *evidence.Variants[index].SourceID, variant.SourceID)
		require.Zero(t, variant.Stock, "source availability is not owned inventory")
		require.Empty(t, variant.SKU)
		require.Nil(t, variant.Price, "missing source currency must not create a Catalog price")
		require.Equal(t, evidence.Variants[index].Images[0].URL, variant.Images[0].URL)
		for _, attribute := range variant.Attributes {
			require.NotContains(t, attribute.Name, "可订数量", "source availability is not a dimension")
		}
	}

	// Metadata must neither reject sourceID-only variants with equal counts nor
	// disguise duplicate dimensions by adding different availability values.
	for index := range capture.Evidence.Variants {
		capture.Evidence.Variants[index].Attributes = nil
	}
	capture.Evidence.Attributes[4].Value = "0"
	_, err = MapBrowserCaptureEvidence(capture.Source, capture.Evidence, "source-id-only")
	require.NoError(t, err)
	for index := range capture.Evidence.Variants {
		capture.Evidence.Variants[index].Attributes = []AcquisitionAttribute{{Name: "颜色", Value: "红色"}}
	}
	capture.Evidence.Attributes[4].Value = "9007199254740993"
	_, err = MapBrowserCaptureEvidence(capture.Source, capture.Evidence, "duplicate-dimensions")
	require.ErrorIs(t, err, ErrInvalidAcquisition)
}
