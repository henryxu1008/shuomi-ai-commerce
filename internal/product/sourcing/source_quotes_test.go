package sourcing

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSourceQuoteProjectionKeepsExactFactsAndRejectsMalformedStoredPrices(t *testing.T) {
	facts := map[string]string{"other": "private", "price_fact.10": `{"amount":"27.00000001","currency":null}`, "price_fact.2": `{"amount":"28.00","currency":"CNY","minQuantity":"3"}`}
	prices, err := ProjectSourceQuotes(facts)
	require.NoError(t, err)
	require.Len(t, prices, 2)
	require.Equal(t, "28.00", prices[0].Amount)
	require.Equal(t, "27.00000001", prices[1].Amount)
	require.Nil(t, prices[1].Currency)
	require.Nil(t, prices[1].MinQuantity)
	for _, raw := range []string{`{"amount":"1","amount":"2","currency":null}`, `{"amount":"1","currency":null,"stock":3}`, `{"amount":"1","currency":"$"}`, `{"amount":"NaN","currency":null}`, `{"amount":"1","currency":null,"minQuantity":"0"}`, `{"amount":"1"}`} {
		_, err := ProjectSourceQuotes(map[string]string{"price_fact.0": raw})
		require.Error(t, err, raw)
	}
	for _, key := range []string{"price_fact.bad", "price_fact.-1", "price_fact.01", "price_fact.256"} {
		_, err := ProjectSourceQuotes(map[string]string{key: `{"amount":"1","currency":null}`})
		require.Error(t, err, key)
	}
	prices, err = ProjectSourceQuotes(nil)
	require.NoError(t, err)
	require.NotNil(t, prices)
	require.Empty(t, prices)
}
