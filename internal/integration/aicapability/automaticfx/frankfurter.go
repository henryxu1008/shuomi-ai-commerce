// Package automaticfx implements bounded, opt-in automatic FX preparation.
package automaticfx

import (
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"time"

	strictjson "sigs.k8s.io/json"
)

var ErrInvalidObservation = errors.New("invalid CFETS observation")

type Observation struct {
	Date            string
	CNYMicrosPerUSD int64
}

// ParseCFETSObservation parses the pinned CFETS provider's USD/CNY response.
// This validates the response, not the upstream provider's own calculations.
func ParseCFETSObservation(body io.Reader, now time.Time) (Observation, error) {
	var out Observation
	if body == nil || now.IsZero() {
		return out, ErrInvalidObservation
	}
	b, err := io.ReadAll(io.LimitReader(body, 65537))
	if err != nil || len(b) == 0 || len(b) > 65536 {
		return out, ErrInvalidObservation
	}
	var response struct {
		Date  string          `json:"date"`
		Base  string          `json:"base"`
		Quote string          `json:"quote"`
		Rate  json.RawMessage `json:"rate"`
	}
	violations, err := strictjson.UnmarshalStrict(b, &response, strictjson.DisallowDuplicateFields, strictjson.DisallowUnknownFields)
	if err != nil || len(violations) != 0 || response.Base != "USD" || response.Quote != "CNY" {
		return out, ErrInvalidObservation
	}
	day, err := time.Parse("2006-01-02", response.Date)
	if err != nil || day.Format("2006-01-02") != response.Date || response.Date > now.In(time.FixedZone("Asia/Shanghai", 8*3600)).Format("2006-01-02") {
		return out, ErrInvalidObservation
	}
	rate := string(response.Rate)
	decimal := false
	digits, fractional := 0, 0
	for _, ch := range rate {
		if ch == '.' && !decimal && digits > 0 {
			decimal = true
			continue
		}
		if ch < '0' || ch > '9' {
			return out, ErrInvalidObservation
		}
		digits++
		if decimal {
			fractional++
		}
	}
	if digits == 0 || (decimal && fractional == 0) || fractional > 6 {
		return out, ErrInvalidObservation
	}
	value, ok := new(big.Rat).SetString(rate)
	if !ok || value.Sign() <= 0 {
		return out, ErrInvalidObservation
	}
	value.Mul(value, big.NewRat(1_000_000, 1))
	if !value.IsInt() || !value.Num().IsInt64() || value.Num().Int64() > 1_000_000_000_000 {
		return out, ErrInvalidObservation
	}
	return Observation{Date: response.Date, CNYMicrosPerUSD: value.Num().Int64()}, nil
}
