package routeconfigstore

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	"task-processor/internal/integration/aicapability/automaticfx"
)

func syntheticFXClock() (time.Time, string, string, automaticfx.Calendar) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	loc := time.FixedZone("Asia/Shanghai", 8*3600)
	local := now.In(loc)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	if local.Before(day.Add(9*time.Hour + 15*time.Minute)) {
		day = day.AddDate(0, 0, -1)
	}
	previous := day.AddDate(0, 0, -1).Format("2006-01-02")
	years := map[int]automaticfx.YearCalendar{}
	for _, y := range []int{day.Year(), local.Year(), day.AddDate(0, 0, 2).Year()} {
		years[y] = automaticfx.YearCalendar{Source: "https://www.chinamoney.com.cn/fixture", CoverageFrom: fmt.Sprintf("%d-01-01", y), CoverageUntil: fmt.Sprintf("%d-12-31", y), ExceptionsComplete: true}
	}
	y := years[day.Year()]
	y.Open = []string{day.Format("2006-01-02")}
	years[day.Year()] = y
	if day.Format("2006-01-02") != local.Format("2006-01-02") {
		y = years[local.Year()]
		y.Closed = append(y.Closed, local.Format("2006-01-02"))
		years[local.Year()] = y
	}
	return now, day.Format("2006-01-02"), previous, automaticfx.Calendar{Years: years}
}

func TestAutomaticFXUsesPriceOwnerAndPreservesFrozenHistory(t *testing.T) {
	_, repo := fixture(t)
	ctx := context.Background()
	now, day, previous, calendar := syntheticFXClock()
	owner := &d.PlatformPricesService{Repository: repo, Now: func() time.Time { return now }}
	seed := d.PlatformPricesInput{ExpectedRevision: d.NoRevision,
		Prices: []d.PublishedModelPrice{{ID: "fixture-price", Provider: "openai", Model: "fixture-model", Channel: "fixture-channel", Region: "global", ServiceType: "standard-sync", Currency: "USD", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 750000, OutputText: 4500000}, SourceURL: "https://developers.openai.com/api/docs/pricing", VerifiedAt: now.Add(-time.Hour), EffectiveFrom: now.Add(-time.Hour), EffectiveUntil: now.Add(48 * time.Hour)}},
		FX: []d.PublishedFXRate{
			{Currency: "USD", RateDate: previous, CNYMicrosPerCurrencyUnit: 6719100, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", VerifiedAt: now.Add(-time.Hour), EffectiveFrom: now.Add(-time.Hour), EffectiveUntil: now.Add(time.Hour)},
			{Currency: "EUR", RateDate: previous, CNYMicrosPerCurrencyUnit: 7800000, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", VerifiedAt: now.Add(-time.Hour), EffectiveFrom: now.Add(-time.Hour), EffectiveUntil: now.Add(48 * time.Hour)},
		},
	}
	old, err := owner.Publish(ctx, "fixture-maintainer", "seed", seed)
	require.NoError(t, err)
	before, err := old.Payload.PlatformPrices.Resolve("fixture-price", "fixture-model", "fixture-channel", now.Add(-time.Second))
	require.NoError(t, err)
	input, changed, err := automaticfx.PrepareVerifiedPublication(old.ID, *old.Payload.PlatformPrices, automaticfx.Observation{Date: day, CNYMicrosPerUSD: 6717400}, calendar, now)
	require.NoError(t, err)
	require.True(t, changed)
	newRev, err := automaticfx.PublishFirstAttempt(ctx, automaticfx.ServiceOwner{Service: owner}, day, 1, input)
	require.NoError(t, err)
	replay, err := automaticfx.InspectPublication(ctx, automaticfx.ServiceOwner{Service: owner}, day, 1, input)
	require.NoError(t, err)
	require.Equal(t, newRev.ID, replay.ID)
	current, err := owner.Current(ctx)
	require.NoError(t, err)
	require.Equal(t, newRev.ID, current.ID)
	after, err := current.Payload.PlatformPrices.Resolve("fixture-price", "fixture-model", "fixture-channel", now)
	require.NoError(t, err)
	require.EqualValues(t, 6719100, before.CNYMicrosPerCurrencyUnit)
	require.EqualValues(t, 6717400, after.CNYMicrosPerCurrencyUnit)
	require.Equal(t, before.PriceVersion, after.PriceVersion)
	require.NotEqual(t, before.FXVersion, after.FXVersion)
	oldAgain, err := repo.Revision(ctx, d.PlatformPricesScope(), old.ID)
	require.NoError(t, err)
	frozen, err := oldAgain.Payload.PlatformPrices.Resolve("fixture-price", "fixture-model", "fixture-channel", now)
	require.NoError(t, err)
	require.Equal(t, before, frozen)
	require.Equal(t, old.Payload.PlatformPrices.FX[1], current.Payload.PlatformPrices.FX[1])
}

func TestAutomaticFXReconcilesCommittedUnknownByOriginalKey(t *testing.T) {
	_, repo := fixture(t)
	ctx := context.Background()
	now, day, _, _ := syntheticFXClock()
	service := &d.PlatformPricesService{Repository: repo, Now: func() time.Time { return now }}
	input := d.PlatformPricesInput{ExpectedRevision: d.NoRevision, FX: []d.PublishedFXRate{{Currency: "USD", RateDate: day, CNYMicrosPerCurrencyUnit: 6717400, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}}}
	repo.commitTransaction = func(tx *gorm.DB) error {
		require.NoError(t, tx.Commit().Error)
		return errors.New("synthetic lost acknowledgement")
	}
	_, err := automaticfx.PublishFirstAttempt(ctx, automaticfx.ServiceOwner{Service: service}, day, 1, input)
	require.ErrorIs(t, err, d.ErrOutcomeUnknown)
	resolved, err := automaticfx.InspectPublication(ctx, automaticfx.ServiceOwner{Service: service}, day, 1, input)
	require.NoError(t, err)
	require.Equal(t, "fx-cfets-usd-v1:"+day+":1", resolved.Key)
	var count int64
	require.NoError(t, repo.db.Table("ai_route_config_revisions").Where("capability = ?", d.PlatformPricesCapability).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestAutomaticFXKnownStaleSnapshotAllowsNewAttemptKey(t *testing.T) {
	_, repo := fixture(t)
	ctx := context.Background()
	now, day, previous, calendar := syntheticFXClock()
	service := &d.PlatformPricesService{Repository: repo, Now: func() time.Time { return now }}
	seed := d.PlatformPricesInput{ExpectedRevision: d.NoRevision, FX: []d.PublishedFXRate{{Currency: "USD", RateDate: previous, CNYMicrosPerCurrencyUnit: 6719100, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", VerifiedAt: now.Add(-time.Minute), EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now.Add(time.Hour)}}}
	first, err := service.Publish(ctx, "fixture", "seed-manual", seed)
	require.NoError(t, err)
	obs := automaticfx.Observation{Date: day, CNYMicrosPerUSD: 6717400}
	stale, changed, err := automaticfx.PrepareVerifiedPublication(first.ID, *first.Payload.PlatformPrices, obs, calendar, now)
	require.NoError(t, err)
	require.True(t, changed)
	manual := seed
	manual.ExpectedRevision = first.ID
	updated, err := service.Publish(ctx, "fixture", "manual-update", manual)
	require.NoError(t, err)
	_, err = automaticfx.PublishFirstAttempt(ctx, automaticfx.ServiceOwner{Service: service}, obs.Date, 1, stale)
	require.ErrorIs(t, err, d.ErrConflict)
	require.ErrorIs(t, err, d.ErrStaleSnapshot)
	fresh, changed, err := automaticfx.PrepareVerifiedPublication(updated.ID, *updated.Payload.PlatformPrices, obs, calendar, now)
	require.NoError(t, err)
	require.True(t, changed)
	final, err := automaticfx.PublishFirstAttempt(ctx, automaticfx.ServiceOwner{Service: service}, obs.Date, 2, fresh)
	require.NoError(t, err)
	require.Equal(t, updated.ID, final.Parent)
	require.Equal(t, "fx-cfets-usd-v1:"+day+":2", final.Key)
}

func TestAutomaticFXStaleCASAfterLockIsProvenUncommitted(t *testing.T) {
	db, repo := fixture(t)
	ctx := context.Background()
	now, day, previous, calendar := syntheticFXClock()
	service := &d.PlatformPricesService{Repository: repo, Now: func() time.Time { return now }}
	seed := d.PlatformPricesInput{ExpectedRevision: d.NoRevision, FX: []d.PublishedFXRate{{Currency: "USD", RateDate: previous, CNYMicrosPerCurrencyUnit: 6719100, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", VerifiedAt: now.Add(-time.Minute), EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now.Add(time.Hour)}}}
	first, err := service.Publish(ctx, "fixture", "seed-race", seed)
	require.NoError(t, err)
	manual := seed
	manual.ExpectedRevision = first.ID
	second, err := service.Publish(ctx, "fixture", "manual-race", manual)
	require.NoError(t, err)
	require.NoError(t, db.Exec("UPDATE public.ai_route_config_current SET revision_id=? WHERE organization_id=? AND capability=?", first.ID, d.PlatformPricesOrganization, d.PlatformPricesCapability).Error)
	input, changed, err := automaticfx.PrepareVerifiedPublication(first.ID, *first.Payload.PlatformPrices, automaticfx.Observation{Date: day, CNYMicrosPerUSD: 6717400}, calendar, now)
	require.NoError(t, err)
	require.True(t, changed)
	lock := db.Begin()
	require.NoError(t, lock.Error)
	require.NoError(t, lock.Exec("UPDATE public.ai_route_config_current SET revision_id=? WHERE organization_id=? AND capability=?", second.ID, d.PlatformPricesOrganization, d.PlatformPricesCapability).Error)
	result := make(chan error, 1)
	go func() {
		_, e := automaticfx.PublishFirstAttempt(ctx, automaticfx.ServiceOwner{Service: service}, day, 1, input)
		result <- e
	}()
	require.Eventually(t, func() bool {
		var n int64
		e := db.Raw("SELECT count(*) FROM pg_stat_activity WHERE wait_event_type='Lock' AND query LIKE '%ai_route_config_current%' AND pid<>pg_backend_pid()").Scan(&n).Error
		return e == nil && n > 0
	}, 5*time.Second, 20*time.Millisecond)
	require.NoError(t, lock.Commit().Error)
	err = <-result
	require.ErrorIs(t, err, d.ErrConflict)
	require.ErrorIs(t, err, d.ErrStaleSnapshot)
}

func TestPriceOwnerReceiptPrecedesExpiredDeadlineAndManualUnchanged(t *testing.T) {
	_, repo := fixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	service := &d.PlatformPricesService{Repository: repo}
	day := now.In(time.FixedZone("Asia/Shanghai", 8*3600)).Format("2006-01-02")
	input := d.PlatformPricesInput{ExpectedRevision: d.NoRevision, FX: []d.PublishedFXRate{{Currency: "USD", RateDate: day, CNYMicrosPerCurrencyUnit: 6717400, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Second)}}}
	deadline := now.Add(time.Second)
	first, err := service.PublishBefore(ctx, "fixture-auto", "fixture-deadline", input, deadline)
	require.NoError(t, err)
	time.Sleep(1200 * time.Millisecond)
	again, err := service.PublishBefore(ctx, "fixture-auto", "fixture-deadline", input, deadline)
	require.NoError(t, err)
	require.Equal(t, first.ID, again.ID)
	input.ExpectedRevision = first.ID
	manual, err := service.Publish(ctx, "fixture-manual", "fixture-manual", input)
	require.NoError(t, err)
	require.Equal(t, first.ID, manual.Parent)
}

func TestAutomaticFXLateCommitCannotResolveExpiredTariff(t *testing.T) {
	_, repo := fixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	day := now.In(time.FixedZone("Asia/Shanghai", 8*3600)).Format("2006-01-02")
	service := &d.PlatformPricesService{Repository: repo}
	input := d.PlatformPricesInput{ExpectedRevision: d.NoRevision, Prices: []d.PublishedModelPrice{{ID: "late", Provider: "openai", Model: "synthetic", Channel: "fixture", Region: "global", ServiceType: "standard-sync", Currency: "USD", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 750000, OutputText: 4500000}, SourceURL: "https://developers.openai.com/api/docs/pricing", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}}, FX: []d.PublishedFXRate{{Currency: "USD", RateDate: day, CNYMicrosPerCurrencyUnit: 6717400, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Second)}}}
	repo.commitTransaction = func(tx *gorm.DB) error { time.Sleep(1200 * time.Millisecond); return tx.Commit().Error }
	published, err := automaticfx.PublishFirstAttempt(ctx, automaticfx.ServiceOwner{Service: service}, day, 1, input)
	require.NoError(t, err)
	_, err = published.Payload.PlatformPrices.Resolve("late", "synthetic", "fixture", time.Now().UTC())
	require.ErrorIs(t, err, d.ErrNeedsConfiguration, "expired FX must not price a new request even if its revision committed late")
}

func TestAutomaticFXMustNotCommitAfterItsExpiryWhileWaitingOnCAS(t *testing.T) {
	db, repo := fixture(t)
	ctx := context.Background()
	now := time.Now().UTC()
	day := now.In(time.FixedZone("Asia/Shanghai", 8*3600)).Format("2006-01-02")
	previous := now.Add(-24 * time.Hour).In(time.FixedZone("Asia/Shanghai", 8*3600)).Format("2006-01-02")
	service := &d.PlatformPricesService{Repository: repo}
	seed := d.PlatformPricesInput{ExpectedRevision: d.NoRevision, FX: []d.PublishedFXRate{{Currency: "USD", RateDate: previous, CNYMicrosPerCurrencyUnit: 6719100, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", VerifiedAt: now.Add(-time.Minute), EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now.Add(time.Hour)}}}
	old, err := service.Publish(ctx, "fixture", "seed-expiry", seed)
	require.NoError(t, err)
	input := d.PlatformPricesInput{ExpectedRevision: old.ID, FX: []d.PublishedFXRate{{Currency: "USD", RateDate: previous, CNYMicrosPerCurrencyUnit: 6719100, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", VerifiedAt: now.Add(-time.Minute), EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now}, {Currency: "USD", RateDate: day, CNYMicrosPerCurrencyUnit: 6717400, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Second)}}}
	lock := db.Begin()
	require.NoError(t, lock.Error)
	var id string
	require.NoError(t, lock.Raw("SELECT revision_id FROM public.ai_route_config_current WHERE organization_id=? AND capability=? FOR UPDATE", d.PlatformPricesOrganization, d.PlatformPricesCapability).Scan(&id).Error)
	result := make(chan error, 1)
	go func() {
		_, e := automaticfx.PublishFirstAttempt(ctx, automaticfx.ServiceOwner{Service: service}, day, 1, input)
		result <- e
	}()
	time.Sleep(1500 * time.Millisecond)
	require.NoError(t, lock.Rollback().Error)
	err = <-result
	require.Error(t, err, "expired automatic publication must not be committed")
	current, err := service.Current(ctx)
	require.NoError(t, err)
	require.Equal(t, old.ID, current.ID)
}
