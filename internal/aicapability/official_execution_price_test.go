package aicapability

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTextProfileRequiresExactFrozenOfficialPrice(t *testing.T) {
	p := testTextProfile()
	price := officialMiniPrice()
	price.Currency, price.CNYMicrosPerCurrencyUnit = "CNY", 1_000_000
	price.Model = p.ModelID
	p.OfficialTariff = price
	require.NoError(t, p.Validate())
	maximum, err := p.OfficialTariff.ReserveTextMicropoints(100, 10)
	require.NoError(t, err)
	require.Positive(t, maximum)
	p.OfficialTariff.Model = "another-model"
	require.Error(t, p.Validate(), "a valid but unrelated tariff cannot price this model")
	p.OfficialTariff = OfficialModelTariff{}
	require.Error(t, p.Validate(), "there is no independent point-price fallback")
}
