package aicapability

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"sort"
	"time"
)

// OfficialTextPriceWindow is an input/output value from the published price
// owner. An arithmetic value cannot admit a provider or its billing clock.
type OfficialTextPriceWindow struct {
	EntryVersion string             `json:"entryVersion"`
	FromUTC      string             `json:"fromUTC"`
	UntilUTC     string             `json:"untilUTC"`
	Rates        OfficialModelRates `json:"rates"`
}

// All storage is private, copied and comparable inside tariff/ModelProfile.
// Unused array tails have exactly the zero value.
type OfficialTextPriceWindows struct {
	priceID, revision, version, clockPolicy, boundEvidence string
	count                                                  uint8
	entries                                                [32]OfficialTextPriceWindow
}
type textPriceWindowsWire struct {
	SchemaVersion        int                       `json:"schemaVersion"`
	PriceID              string                    `json:"priceId"`
	Revision             string                    `json:"revision"`
	GroupVersion         string                    `json:"groupVersion,omitempty"`
	ClockPolicyVersion   string                    `json:"clockPolicyVersion"`
	BoundEvidenceVersion string                    `json:"boundEvidenceVersion"`
	Windows              []OfficialTextPriceWindow `json:"windows"`
}

func NewOfficialTextPriceWindows(priceID, revision, clockPolicy, boundEvidence string, entries []OfficialTextPriceWindow) (OfficialTextPriceWindows, error) {
	var result OfficialTextPriceWindows
	if !validTextFact(priceID, 64) || !validTextFact(revision, 128) || !validTextFact(clockPolicy, 128) || !validTextFact(boundEvidence, 128) || len(entries) == 0 || len(entries) > 32 {
		return result, ErrOfficialModelPrice
	}
	entries = append([]OfficialTextPriceWindow(nil), entries...)
	for i := range entries {
		entry := &entries[i]
		from, e1 := time.Parse(time.RFC3339Nano, entry.FromUTC)
		until, e2 := time.Parse(time.RFC3339Nano, entry.UntilUTC)
		if e1 != nil || e2 != nil || from.IsZero() || !until.After(from) || !validTextFact(entry.EntryVersion, 128) || !validTextWindowRates(entry.Rates) {
			return result, ErrOfficialModelPrice
		}
		entry.FromUTC, entry.UntilUTC = from.UTC().Format(time.RFC3339Nano), until.UTC().Format(time.RFC3339Nano)
	}
	sort.Slice(entries, func(i, j int) bool {
		a, _ := time.Parse(time.RFC3339Nano, entries[i].FromUTC)
		b, _ := time.Parse(time.RFC3339Nano, entries[j].FromUTC)
		return a.Before(b)
	})
	for i := 1; i < len(entries); i++ {
		previous, _ := time.Parse(time.RFC3339Nano, entries[i-1].UntilUTC)
		next, _ := time.Parse(time.RFC3339Nano, entries[i].FromUTC)
		if next.Before(previous) {
			return result, ErrOfficialModelPrice
		}
	}
	result.priceID, result.revision, result.clockPolicy, result.boundEvidence = priceID, revision, clockPolicy, boundEvidence
	result.count = uint8(len(entries))
	copy(result.entries[:], entries)
	raw, err := json.Marshal(result.wire())
	if err != nil || len(raw) > 8192 {
		return OfficialTextPriceWindows{}, ErrOfficialModelPrice
	}
	hash := sha256.Sum256(raw)
	result.version = "text-windows-v1:" + hex.EncodeToString(hash[:])
	raw, err = json.Marshal(result.wire())
	if err != nil || len(raw) > 8192 {
		return OfficialTextPriceWindows{}, ErrOfficialModelPrice
	}
	return result, nil
}

func validTextWindowRates(r OfficialModelRates) bool {
	if r.InputImage != 0 || r.CachedInputImage != 0 || r.OutputImage != 0 || r.OutputImagePerItem != 0 || r.CachedInputText > r.InputText {
		return false
	}
	return r.InputText > 0 && r.InputText <= 1_000_000_000_000 && r.CachedInputText >= 0 && r.OutputText > 0 && r.OutputText <= 1_000_000_000_000
}
func (g OfficialTextPriceWindows) wire() textPriceWindowsWire {
	return textPriceWindowsWire{SchemaVersion: 1, PriceID: g.priceID, Revision: g.revision, GroupVersion: g.version, ClockPolicyVersion: g.clockPolicy, BoundEvidenceVersion: g.boundEvidence, Windows: g.Entries()}
}
func (g OfficialTextPriceWindows) Empty() bool                { return g == (OfficialTextPriceWindows{}) }
func (g OfficialTextPriceWindows) Version() string            { return g.version }
func (g OfficialTextPriceWindows) ClockPolicyVersion() string { return g.clockPolicy }
func (g OfficialTextPriceWindows) Entries() []OfficialTextPriceWindow {
	if g.count == 0 || g.count > 32 {
		return nil
	}
	return append([]OfficialTextPriceWindow(nil), g.entries[:g.count]...)
}
func (g OfficialTextPriceWindows) Validate() error {
	canonical, err := NewOfficialTextPriceWindows(g.priceID, g.revision, g.clockPolicy, g.boundEvidence, g.Entries())
	if err != nil || canonical != g {
		return ErrOfficialModelPrice
	}
	return nil
}
func (g OfficialTextPriceWindows) MarshalJSON() ([]byte, error) {
	if g.Validate() != nil {
		return nil, ErrOfficialModelPrice
	}
	raw, err := json.Marshal(g.wire())
	if err != nil || len(raw) > 8192 {
		return nil, ErrOfficialModelPrice
	}
	return raw, nil
}
func (g *OfficialTextPriceWindows) UnmarshalJSON(raw []byte) error {
	if len(raw) == 0 || len(raw) > 8192 {
		return ErrOfficialModelPrice
	}
	var wire textPriceWindowsWire
	// Keep the domain's standard-library-only boundary. The standard decoder
	// parses syntax/types; the bounded token walk retains duplicate/null checks.
	tokens := json.NewDecoder(bytes.NewReader(raw))
	tokens.UseNumber()
	if !uniqueTextWindowJSON(tokens, 0) {
		return ErrOfficialModelPrice
	}
	if _, err := tokens.Token(); err != io.EOF {
		return ErrOfficialModelPrice
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&wire) != nil || wire.SchemaVersion != 1 {
		return ErrOfficialModelPrice
	}
	// A null scalar would otherwise decode as zero and could turn an unknown
	// cache rate into a free one without changing the canonical value/hash.
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || !exactTextWindowFields(fields, "schemaVersion", "priceId", "revision", "groupVersion", "clockPolicyVersion", "boundEvidenceVersion", "windows") {
		return ErrOfficialModelPrice
	}
	var presence []map[string]json.RawMessage
	if json.Unmarshal(fields["windows"], &presence) != nil {
		return ErrOfficialModelPrice
	}
	for _, entry := range presence {
		if !exactTextWindowFields(entry, "entryVersion", "fromUTC", "untilUTC", "rates") {
			return ErrOfficialModelPrice
		}
		var rates map[string]json.RawMessage
		if json.Unmarshal(entry["rates"], &rates) != nil {
			return ErrOfficialModelPrice
		}
		expected := 6
		if _, ok := rates["outputImagePerItem"]; ok {
			expected++
		}
		if len(rates) != expected {
			return ErrOfficialModelPrice
		}
		for _, key := range []string{"inputText", "cachedInputText", "inputImage", "cachedInputImage", "outputText", "outputImage"} {
			value, ok := rates[key]
			if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return ErrOfficialModelPrice
			}
		}
		if value, ok := rates["outputImagePerItem"]; ok && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return ErrOfficialModelPrice
		}
	}
	value, err := NewOfficialTextPriceWindows(wire.PriceID, wire.Revision, wire.ClockPolicyVersion, wire.BoundEvidenceVersion, wire.Windows)
	if err != nil || value.version != wire.GroupVersion {
		return ErrOfficialModelPrice
	}
	*g = value
	return nil
}

func exactTextWindowFields(fields map[string]json.RawMessage, keys ...string) bool {
	if len(fields) != len(keys) {
		return false
	}
	for _, key := range keys {
		if _, ok := fields[key]; !ok {
			return false
		}
	}
	return true
}

// This finite price object has at most four nested containers. No separate
// parser or generic JSON service is introduced; all syntax remains stdlib-owned.
func uniqueTextWindowJSON(decoder *json.Decoder, depth int) bool {
	if depth > 5 {
		return false
	}
	token, err := decoder.Token()
	if err != nil || token == nil {
		return false
	}
	delimiter, container := token.(json.Delim)
	if !container {
		return true
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			token, err := decoder.Token()
			key, ok := token.(string)
			if err != nil || !ok || seen[key] {
				return false
			}
			seen[key] = true
			if !uniqueTextWindowJSON(decoder, depth+1) {
				return false
			}
		}
	case '[':
		for decoder.More() {
			if !uniqueTextWindowJSON(decoder, depth+1) {
				return false
			}
		}
	default:
		return false
	}
	_, err = decoder.Token()
	return err == nil
}
func (g OfficialTextPriceWindows) MaximumRates() OfficialModelRates {
	var rates OfficialModelRates
	for _, entry := range g.Entries() {
		rates.InputText = max(rates.InputText, entry.Rates.InputText)
		rates.CachedInputText = max(rates.CachedInputText, entry.Rates.CachedInputText)
		rates.OutputText = max(rates.OutputText, entry.Rates.OutputText)
	}
	return rates
}
func (g OfficialTextPriceWindows) RatesAt(at, policy string) (OfficialModelRates, error) {
	instant, err := time.Parse(time.RFC3339Nano, at)
	if err != nil || instant.IsZero() || instant.UTC().Format(time.RFC3339Nano) != at || policy != g.clockPolicy || g.Validate() != nil {
		return OfficialModelRates{}, ErrOfficialModelPrice
	}
	for _, entry := range g.Entries() {
		from, _ := time.Parse(time.RFC3339Nano, entry.FromUTC)
		until, _ := time.Parse(time.RFC3339Nano, entry.UntilUTC)
		if !instant.Before(from) && instant.Before(until) {
			return entry.Rates, nil
		}
	}
	return OfficialModelRates{}, ErrOfficialModelPrice
}

// Covers checks continuous send/deadline coverage. A response must separately
// prove the actual billing instant; local time cannot stand in for that proof.
func (g OfficialTextPriceWindows) Covers(from, until time.Time) bool {
	if from.IsZero() || !until.After(from) || g.Validate() != nil {
		return false
	}
	cursor := from
	for _, entry := range g.Entries() {
		start, _ := time.Parse(time.RFC3339Nano, entry.FromUTC)
		end, _ := time.Parse(time.RFC3339Nano, entry.UntilUTC)
		if !end.After(cursor) {
			continue
		}
		if start.After(cursor) {
			return false
		}
		cursor = end
		if !cursor.Before(until) {
			return true
		}
	}
	return false
}
