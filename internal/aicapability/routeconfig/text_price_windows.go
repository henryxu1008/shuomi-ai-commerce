package routeconfig

import (
	"task-processor/internal/aicapability"
	"time"
)

// TextPriceWindowRequest contains only identities from the original Catalog
// and published revision. Resolving arithmetic does not admit the clock policy.
type TextPriceWindowRequest struct {
	ID, Revision, Model, Channel, Region, ServiceType string
	ClockPolicyVersion, BoundEvidenceVersion          string
	Now, Until                                        time.Time
}

func (s PlatformPricesSnapshot) ResolveTextWindows(r TextPriceWindowRequest) (aicapability.OfficialModelTariff, error) {
	invalid := aicapability.OfficialModelTariff{}
	if s.Validate(r.Now) != nil || r.Model != "deepseek-v4.1-flash" || r.Region != "cn-beijing" || r.ServiceType != "standard-sync" || !ValidFact(r.Channel, 128) || !r.Until.After(r.Now) {
		return invalid, ErrNeedsConfiguration
	}
	entries := []aicapability.OfficialTextPriceWindow{}
	peak := aicapability.OfficialModelRates{InputText: 2_000_000, CachedInputText: 40_000, OutputText: 8_000_000}
	idle := aicapability.OfficialModelRates{InputText: 1_000_000, CachedInputText: 20_000, OutputText: 4_000_000}
	peakProven, idleProven := false, false
	for _, p := range s.Prices {
		if p.ID != r.ID {
			continue
		}
		if p.Provider != "deepseek" || p.Model != r.Model || p.Channel != r.Channel || p.Region != r.Region || p.ServiceType != r.ServiceType || p.Image != nil || p.Currency != "CNY" || p.UsageVersion != "openai-text-v1" {
			return invalid, ErrNeedsConfiguration
		}
		switch p.Rates {
		case peak:
			peakProven = true
		case idle:
			idleProven = true
		default:
			return invalid, ErrNeedsConfiguration
		}
		entries = append(entries, aicapability.OfficialTextPriceWindow{EntryVersion: p.Version, FromUTC: p.EffectiveFrom.UTC().Format(time.RFC3339Nano), UntilUTC: p.EffectiveUntil.UTC().Format(time.RFC3339Nano), Rates: p.Rates})
	}
	if !peakProven || !idleProven {
		return invalid, ErrNeedsConfiguration
	}
	windows, err := aicapability.NewOfficialTextPriceWindows(r.ID, r.Revision, r.ClockPolicyVersion, r.BoundEvidenceVersion, entries)
	if err != nil || !windows.Covers(r.Now, r.Until) {
		return invalid, ErrNeedsConfiguration
	}
	price := aicapability.OfficialModelTariff{UnitVersion: s.UnitVersion, PriceVersion: windows.Version(), FXVersion: "cny-identity-v1", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1_000_000, Model: r.Model, Channel: r.Channel, UsageVersion: "openai-text-v1", Rates: windows.MaximumRates(), TextPriceWindows: windows}
	if price.Validate() != nil {
		return invalid, ErrNeedsConfiguration
	}
	return price, nil
}
