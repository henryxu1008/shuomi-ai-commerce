package sourcing

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const MaxSourceQuoteBytes = 64 << 10

// SourcePrice is a read projection of source evidence, never a selling price.
type SourcePrice struct {
	Amount        string           `json:"amount"`
	Currency      *string          `json:"currency"`
	MinQuantity   *string          `json:"minQuantity"`
	Condition     *string          `json:"condition,omitempty"`
	ConditionNote *string          `json:"conditionNote,omitempty"`
	Alternative   *amazonOfferWire `json:"alternative,omitempty"`
}

// ProjectSourceQuotes reads only the existing sourcing owner's price facts.
// Malformed facts fail the supplementary read rather than becoming empty facts.
func ProjectSourceQuotes(facts map[string]string) ([]SourcePrice, error) {
	indexed := map[int]string{}
	for key, raw := range facts {
		if !strings.HasPrefix(key, "price_fact.") {
			continue
		}
		suffix := strings.TrimPrefix(key, "price_fact.")
		index, err := strconv.Atoi(suffix)
		if err != nil || index < 0 || index >= browserCaptureMaxCollection || strconv.Itoa(index) != suffix {
			return nil, ErrInvalidAcquisition
		}
		indexed[index] = raw
	}
	keys := make([]int, 0, len(indexed))
	for key := range indexed {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	prices := make([]SourcePrice, 0, len(keys))
	for _, key := range keys {
		raw := []byte(indexed[key])
		if len(raw) > MaxSourceQuoteBytes || !utf8.Valid(raw) || !browserUnicodeEscapesValid(raw) {
			return nil, ErrInvalidAcquisition
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		aggregate := 0
		if checkBrowserWire(decoder, reflect.TypeFor[amazonPriceWire](), &aggregate) != nil {
			return nil, ErrInvalidAcquisition
		}
		if _, err := decoder.Token(); err != io.EOF {
			return nil, ErrInvalidAcquisition
		}
		var price amazonPriceWire
		if json.Unmarshal(raw, &price) != nil {
			return nil, ErrInvalidAcquisition
		}
		if err := price.validate(); err != nil {
			return nil, err
		}
		prices = append(prices, SourcePrice{Amount: price.Amount, Currency: price.Currency, MinQuantity: price.MinQuantity, Condition: price.Condition, ConditionNote: price.ConditionNote, Alternative: price.Alternative})
	}
	encoded, err := json.Marshal(prices)
	if err != nil || len(encoded) > MaxSourceQuoteBytes {
		return nil, ErrInvalidAcquisition
	}
	return prices, nil
}
