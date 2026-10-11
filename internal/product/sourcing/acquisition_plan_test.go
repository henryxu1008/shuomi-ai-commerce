package sourcing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func ptrRuleText(s string) *string { return &s }

func TestAcquisitionRuleUsesOriginalStartingPriceAndMOQ(t *testing.T) {
	rule := AcquisitionRule{MinPrice: "10", MaxPrice: "10.00", MaxMOQ: 2}
	facts := []AcquisitionPrice{
		{Amount: "15", Currency: ptrRuleText("CNY"), MinQuantity: ptrRuleText("2")},
		{Amount: "10.0000", Currency: ptrRuleText("CNY"), MinQuantity: ptrRuleText("10")},
	}
	result, err := rule.Evaluate(facts)
	require.NoError(t, err)
	require.Equal(t, RuleMatched, result.State)
	require.Equal(t, "10.0000", result.StartingPrice)
	require.EqualValues(t, 2, result.MinimumQuantity)

	rule.MaxMOQ = 1
	result, err = rule.Evaluate(facts)
	require.NoError(t, err)
	require.Equal(t, RuleNotMatched, result.State)
}

func TestAcquisitionRuleMissingFactsRemainUnknown(t *testing.T) {
	for _, facts := range [][]AcquisitionPrice{nil,
		{{Amount: "10", MinQuantity: ptrRuleText("2")}},
		{{Amount: "10", Currency: ptrRuleText("USD"), MinQuantity: ptrRuleText("2")}},
		{{Amount: "10", Currency: ptrRuleText("CNY")}},
		{{Amount: "10", Currency: ptrRuleText("CNY"), MinQuantity: ptrRuleText("2")}, {Amount: "8", Currency: ptrRuleText("USD"), MinQuantity: ptrRuleText("1")}},
	} {
		result, err := (AcquisitionRule{MaxPrice: "20", MaxMOQ: 5}).Evaluate(facts)
		require.NoError(t, err)
		require.Equal(t, RuleFactsUnknown, result.State)
	}
	result, err := (AcquisitionRule{}).Evaluate(nil)
	require.NoError(t, err)
	require.Equal(t, RuleMatched, result.State)
	result, err = (AcquisitionRule{MaxMOQ: 5}).Evaluate([]AcquisitionPrice{{Amount: "10", MinQuantity: ptrRuleText("2")}})
	require.NoError(t, err)
	require.Equal(t, RuleMatched, result.State, "unused currency cannot block an MOQ-only rule")
}

func TestAcquisitionRuleExactDecimalAndInvalidBounds(t *testing.T) {
	result, err := (AcquisitionRule{MaxPrice: "0.1"}).Evaluate([]AcquisitionPrice{{Amount: "0.10000001", Currency: ptrRuleText("CNY")}})
	require.NoError(t, err)
	require.Equal(t, RuleNotMatched, result.State)
	for _, rule := range []AcquisitionRule{{MinPrice: "20", MaxPrice: "10"}, {MinPrice: "-1"}, {MaxPrice: "NaN"}, {MaxPrice: "1e3"}, {MaxMOQ: -1}} {
		require.ErrorIs(t, rule.Validate(), ErrInvalidAcquisition)
	}
}

func TestCanonicalDiscoveryInputDoesNotAdmitArbitraryNavigation(t *testing.T) {
	for _, raw := range []string{
		"https://127.0.0.1/page/offerlist.htm", "https://store.1688.com.evil.example/page/offerlist.htm",
		"https://user:pass@store.1688.com/page/offerlist.htm", "http://store.1688.com/page/offerlist.htm",
		"https://login.1688.com/page/offerlist.htm", "https://store.1688.com/member/signin.htm",
		"https://store.1688.com:444/page/offerlist.htm", "https://store.1688.com/page/offerlist.htm#x",
		"https://s.1688.com/selloffer/offer_search.htm?keywords=x&keywords=y",
		"https://s.1688.com/selloffer/offer_search.htm?keywords=x&redirect=https://evil.example",
	} {
		_, err := Canonical1688Discovery(DiscoveryPage, raw)
		require.ErrorIs(t, err, ErrInvalidAcquisition, raw)
	}
	page, err := Canonical1688Discovery(DiscoveryPage, "https://demo.1688.com/page/offerlist.htm")
	require.NoError(t, err)
	require.Equal(t, "https://demo.1688.com/page/offerlist.htm", page.URL)
	keyword, err := Canonical1688Discovery(DiscoveryKeyword, "  水杯  ")
	require.NoError(t, err)
	require.Equal(t, "水杯", keyword.Keyword)
	require.Equal(t, "https://s.1688.com/selloffer/offer_search.htm?keywords=%E6%B0%B4%E6%9D%AF", keyword.URL)
}

func TestAcquisitionPlanInputCanonicalizesAndDeduplicatesSources(t *testing.T) {
	input, err := NewAcquisitionPlanInput(DiscoveryProduct, []string{"123", "https://detail.1688.com/offer/123.html", "456"}, AcquisitionRule{}, false)
	require.NoError(t, err)
	require.Len(t, input.Inputs, 2)
	require.Equal(t, "https://detail.1688.com/offer/123.html", input.Inputs[0])
	_, err = NewAcquisitionPlanInput(DiscoveryProduct, []string{"123\n456"}, AcquisitionRule{}, false)
	require.ErrorIs(t, err, ErrInvalidAcquisition)
	_, err = NewAcquisitionPlanInput(DiscoveryKeyword, nil, AcquisitionRule{}, false)
	require.ErrorIs(t, err, ErrInvalidAcquisition)
}
