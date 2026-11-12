package automaticfx

import (
	"errors"
	"time"

	d "task-processor/internal/aicapability/routeconfig"
)

const officialCFETSSource = "https://www.chinamoney.com.cn/chinese/bkccpr/"

var ErrPublicationConflict = errors.New("automatic FX publication requires review")

// PreparePublication constructs an input for the existing platform owner.
// It does not write data or resolve an unknown earlier publication.
func PreparePublication(revision string, snapshot d.PlatformPricesSnapshot, observation Observation, now, expiry time.Time) (d.PlatformPricesInput, bool, error) {
	var invalid d.PlatformPricesInput
	day, err := time.Parse("2006-01-02", observation.Date)
	local := now.In(time.FixedZone("Asia/Shanghai", 8*3600))
	if err != nil || day.Format("2006-01-02") != observation.Date || observation.Date > local.Format("2006-01-02") || observation.CNYMicrosPerUSD <= 0 || observation.CNYMicrosPerUSD > 1_000_000_000_000 || revision == "" || now.IsZero() || !expiry.After(now) {
		return invalid, false, ErrPublicationConflict
	}
	out := d.PlatformPricesInput{ExpectedRevision: revision, Prices: append([]d.PublishedModelPrice(nil), snapshot.Prices...), FX: append([]d.PublishedFXRate(nil), snapshot.FX...)}
	for i := range out.Prices {
		out.Prices[i].Version = ""
	}
	latest := ""
	var latestValue int64
	for _, rate := range out.FX {
		if rate.Currency == "USD" && rate.RateDate > latest {
			latest, latestValue = rate.RateDate, rate.CNYMicrosPerCurrencyUnit
		}
	}
	if latest > observation.Date || (latest == observation.Date && latestValue != observation.CNYMicrosPerUSD) {
		return invalid, false, ErrPublicationConflict
	}
	if latest == observation.Date {
		return invalid, false, nil
	}
	for i := range out.FX {
		r := &out.FX[i]
		r.Version = ""
		if r.Currency != "USD" {
			continue
		}
		if !r.EffectiveFrom.Before(expiry) || !now.Before(r.EffectiveUntil) {
			continue
		}
		if !r.EffectiveFrom.Before(now) {
			return invalid, false, ErrPublicationConflict
		}
		r.EffectiveUntil = now
	}
	// Keep other currencies untouched. Only already ended USD windows can be
	// removed from the current snapshot; immutable revisions are never modified.
	for len(out.FX) >= 8 {
		idx := -1
		for i, r := range out.FX {
			if r.Currency == "USD" && !r.EffectiveUntil.After(now) && (idx == -1 || r.EffectiveUntil.Before(out.FX[idx].EffectiveUntil)) {
				idx = i
			}
		}
		if idx < 0 {
			return invalid, false, ErrPublicationConflict
		}
		out.FX = append(out.FX[:idx], out.FX[idx+1:]...)
	}
	out.FX = append(out.FX, d.PublishedFXRate{Currency: "USD", RateDate: observation.Date, CNYMicrosPerCurrencyUnit: observation.CNYMicrosPerUSD, SourceURL: officialCFETSSource, VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: expiry})
	return out, true, nil
}
