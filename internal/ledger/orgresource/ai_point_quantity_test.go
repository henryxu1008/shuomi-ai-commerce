package orgresource

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAIPointQuantityExactDisplayAndCommercialConversion(t *testing.T) {
	for _, tc := range []struct {
		text  string
		micro int64
	}{{"0", 0}, {"0.000001", 1}, {"0.7455", 745500}, {"1", 1000000}, {"9223372036854.775807", math.MaxInt64}} {
		quantity, err := ParseAIPointQuantity(tc.text)
		require.NoError(t, err)
		require.EqualValues(t, tc.micro, quantity.Micropoints)
		require.Equal(t, AIPointQuantityUnitV1, quantity.UnitVersion)
		text, err := quantity.PointsString()
		require.NoError(t, err)
		require.Equal(t, tc.text, text)
	}
	quantity, err := WholeAIPointQuantity(100)
	require.NoError(t, err)
	require.EqualValues(t, 100_000_000, quantity.Micropoints)
	_, err = WholeAIPointQuantity(math.MaxInt64)
	require.Error(t, err)
	_, err = WholeAIPointQuantity(-1)
	require.Error(t, err)
}

func TestAIPointQuantityRejectsAmbiguousOrDifferentUnits(t *testing.T) {
	for _, text := range []string{"", " 1", "1 ", "01", "+1", "-1", "1e2", ".1", "1.", "1.0000001", "9223372036854.775808"} {
		_, err := ParseAIPointQuantity(text)
		require.Error(t, err, text)
	}
	for _, unit := range []string{"", "whole-point-v1", "money-cent", "data-row"} {
		_, err := (AIPointQuantity{Micropoints: 1, UnitVersion: unit}).PointsString()
		require.Error(t, err, "a number alone cannot reinterpret another resource unit")
	}
}

func TestSignedMicropointDisplayNeverUsesFloatOrLosesIntegers(t *testing.T) {
	for _, tc := range []struct {
		value int64
		text  string
	}{{0, "0"}, {1, "0.000001"}, {-1, "-0.000001"}, {9007199254740993, "9007199254.740993"}, {-9223372036854775808, "-9223372036854.775808"}} {
		require.Equal(t, tc.text, FormatAIPointMicropoints(tc.value))
	}
}
