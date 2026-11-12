package automaticfxruntime_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	runtime "task-processor/internal/app/runtime/automaticfx"
	channelstore "task-processor/internal/integration/persistence/channelcredential"
	store "task-processor/internal/integration/persistence/routeconfig"
)

func fixture(t *testing.T) (*gorm.DB, *store.Store) {
	t.Helper()
	u, err := url.Parse(os.Getenv("ISSUE830_TEST_DSN"))
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", u.Hostname())
	root, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	name := "fxiso_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, root.Exec("DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='product_agent_owner') THEN CREATE ROLE product_agent_owner NOLOGIN; END IF; END $$").Error)
	require.NoError(t, root.Exec("CREATE DATABASE "+name+" OWNER product_agent_owner").Error)
	u.Path = "/" + name
	db, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	t.Cleanup(func() {
		r, _ := db.DB()
		_ = r.Close()
		_ = root.Exec("DROP DATABASE " + name + " WITH (FORCE)").Error
		r, _ = root.DB()
		_ = r.Close()
	})
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec("SET LOCAL ROLE product_agent_owner").Error; e != nil {
			return e
		}
		return channelstore.EnsureSchema(context.Background(), tx)
	}))
	require.NoError(t, store.InstallSchema(context.Background(), db))
	repo, err := store.New(db)
	require.NoError(t, err)
	return db, repo
}

func syntheticFXClock() (time.Time, string, string) {
	now := time.Now().UTC().Truncate(time.Microsecond)
	loc := time.FixedZone("Asia/Shanghai", 8*3600)
	local := now.In(loc)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	if local.Before(day.Add(9*time.Hour + 15*time.Minute)) {
		day = day.AddDate(0, 0, -1)
	}
	return now, day.Format("2006-01-02"), day.AddDate(0, 0, -1).Format("2006-01-02")
}

type isolatedTransport func(*http.Request) (*http.Response, error)

func (f isolatedTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type racingPriceRepository struct {
	d.Repository
	reads atomic.Int32
	race  func(context.Context) error
}

func (r *racingPriceRepository) Current(ctx context.Context, scope d.Scope) (d.Revision, error) {
	if r.reads.Add(1) == 2 {
		if err := r.race(ctx); err != nil {
			return d.Revision{}, err
		}
	}
	return r.Repository.Current(ctx, scope)
}

func TestAutomaticFXIsolatedTemporalExistingRateChecksAtExpiry(t *testing.T) {
	addr := os.Getenv("FX_ISOLATED_TEMPORAL")
	if addr == "" || os.Getenv("ISSUE830_TEST_DSN") == "" {
		t.Skip("requires isolated Temporal and PostgreSQL")
	}
	require.Equal(t, "127.0.0.1:7233", addr)
	_, repo := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: addr, Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	now, day, previous := syntheticFXClock()
	owner := &d.PlatformPricesService{Repository: repo}
	seed := d.PlatformPricesInput{ExpectedRevision: d.NoRevision, FX: []d.PublishedFXRate{{Currency: "USD", RateDate: previous, CNYMicrosPerCurrencyUnit: 6719100, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", VerifiedAt: now.Add(-time.Minute), EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now.Add(4 * time.Second)}}}
	original, err := owner.Publish(ctx, "fixture-admin", "same-rate", seed)
	require.NoError(t, err)
	var fetches atomic.Int32
	transport := isolatedTransport(func(req *http.Request) (*http.Response, error) {
		dte := previous
		rate := "6.7191"
		if fetches.Add(1) > 1 {
			dte = day
			rate = "6.7174"
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"date":%q,"base":"USD","quote":"CNY","rate":%s}`, dte, rate))), Request: req}, nil
	})
	w := worker.New(c, runtime.TaskQueue, worker.Options{})
	require.NoError(t, runtime.RegisterWorker(w, owner, transport))
	require.NoError(t, w.Start())
	defer w.Stop()
	_, err = c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: runtime.CoordinatorID, TaskQueue: runtime.TaskQueue}, runtime.CoordinatorWorkflowName, runtime.CoordinatorState{})
	require.NoError(t, err)
	require.NoError(t, c.SignalWorkflow(ctx, runtime.CoordinatorID, "", runtime.TickSignal, struct{}{}))
	require.Eventually(t, func() bool { return fetches.Load() >= 1 }, 3*time.Second, 50*time.Millisecond)
	before, err := owner.Current(ctx)
	require.NoError(t, err)
	require.Equal(t, original.ID, before.ID)
	require.Eventually(t, func() bool { r, e := owner.Current(ctx); return e == nil && r.ID != original.ID }, 12*time.Second, 100*time.Millisecond)
	current, err := owner.Current(ctx)
	require.NoError(t, err)
	require.Equal(t, "fx-cfets-usd-v1:"+day+":1", current.Key)
	require.NoError(t, c.TerminateWorkflow(ctx, runtime.CoordinatorID, "", "isolated fixture complete"))
}

func TestAutomaticFXIsolatedTemporalManualRace(t *testing.T) {
	addr := os.Getenv("FX_ISOLATED_TEMPORAL")
	if addr == "" || os.Getenv("ISSUE830_TEST_DSN") == "" {
		t.Skip("requires isolated Temporal and PostgreSQL")
	}
	require.Equal(t, "127.0.0.1:7233", addr)
	for _, tc := range []struct {
		name      string
		changeUSD bool
	}{{"manual-usd", true}, {"manual-eur", false}} {
		t.Run(tc.name, func(t *testing.T) {
			_, repo := fixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			c, err := client.Dial(client.Options{HostPort: addr, Namespace: "default"})
			require.NoError(t, err)
			defer c.Close()
			now, day, previous := syntheticFXClock()
			seed := d.PlatformPricesInput{ExpectedRevision: d.NoRevision, FX: []d.PublishedFXRate{
				{Currency: "USD", RateDate: previous, CNYMicrosPerCurrencyUnit: 6719100, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", VerifiedAt: now.Add(-time.Minute), EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now.Add(time.Hour)},
				{Currency: "EUR", RateDate: previous, CNYMicrosPerCurrencyUnit: 7800000, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", VerifiedAt: now.Add(-time.Minute), EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now.Add(time.Hour)},
			}}
			manual := &d.PlatformPricesService{Repository: repo}
			original, err := manual.Publish(ctx, "fixture-admin", "seed", seed)
			require.NoError(t, err)
			wrapped := &racingPriceRepository{Repository: repo, race: func(ctx context.Context) error {
				next := seed
				next.ExpectedRevision = original.ID
				next.FX = append([]d.PublishedFXRate(nil), seed.FX...)
				if tc.changeUSD {
					next.FX[0].CNYMicrosPerCurrencyUnit = 6900000
				} else {
					next.FX[1].CNYMicrosPerCurrencyUnit = 7900000
				}
				_, e := manual.Publish(ctx, "fixture-admin", "manual-change", next)
				return e
			}}
			transport := isolatedTransport(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{"date":%q,"base":"USD","quote":"CNY","rate":6.7174}`, day))), Request: req}, nil
			})
			w := worker.New(c, runtime.TaskQueue, worker.Options{})
			require.NoError(t, runtime.RegisterWorker(w, &d.PlatformPricesService{Repository: wrapped}, transport))
			require.NoError(t, w.Start())
			defer w.Stop()
			_, err = c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: runtime.CoordinatorID, TaskQueue: runtime.TaskQueue}, runtime.CoordinatorWorkflowName, runtime.CoordinatorState{})
			require.NoError(t, err)
			require.NoError(t, c.SignalWorkflow(ctx, runtime.CoordinatorID, "", runtime.TickSignal, struct{}{}))
			want := "PUBLISHED"
			if tc.changeUSD {
				want = "USD_CONFLICT"
			}
			require.Eventually(t, func() bool {
				if !tc.changeUSD {
					r, e := manual.Current(ctx)
					return e == nil && r.Key == "fx-cfets-usd-v1:"+day+":2"
				}
				v, e := c.QueryWorkflow(ctx, runtime.CoordinatorID, "", "status")
				if e != nil {
					return false
				}
				var s runtime.StatusSnapshot
				return v.Get(&s) == nil && s.State == want
			}, 20*time.Second, 100*time.Millisecond)
			if tc.changeUSD {
				require.NoError(t, c.SignalWorkflow(ctx, runtime.CoordinatorID, "", runtime.TickSignal, struct{}{}))
				time.Sleep(200 * time.Millisecond)
			}
			current, err := manual.Current(ctx)
			require.NoError(t, err)
			if tc.changeUSD {
				require.Equal(t, "manual-change", current.Key)
				require.EqualValues(t, 6900000, current.Payload.PlatformPrices.FX[0].CNYMicrosPerCurrencyUnit)
			} else {
				require.Equal(t, "fx-cfets-usd-v1:"+day+":2", current.Key)
				found := false
				for _, rate := range current.Payload.PlatformPrices.FX {
					if rate.Currency == "EUR" && rate.CNYMicrosPerCurrencyUnit == 7900000 {
						found = true
					}
				}
				require.True(t, found)
			}
			require.NoError(t, c.TerminateWorkflow(ctx, runtime.CoordinatorID, "", "isolated fixture complete"))
		})
	}
}

// This test is explicitly opt-in and must target a task-owned Temporal and
// PostgreSQL pair. It has no model credential or provider network connection.
func TestAutomaticFXIsolatedTemporalScheduleAndOwner(t *testing.T) {
	addr := os.Getenv("FX_ISOLATED_TEMPORAL")
	if addr == "" || os.Getenv("ISSUE830_TEST_DSN") == "" {
		t.Skip("requires explicitly isolated Temporal and PostgreSQL")
	}
	require.Equal(t, "127.0.0.1:7233", addr)
	db, repo := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: addr, Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	now, day, previous := syntheticFXClock()
	owner := &d.PlatformPricesService{Repository: repo}
	seed := d.PlatformPricesInput{ExpectedRevision: d.NoRevision, Prices: []d.PublishedModelPrice{{ID: "fx-iso", Provider: "openai", Model: "synthetic-model", Channel: "synthetic-channel", Region: "global", ServiceType: "standard-sync", Currency: "USD", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 750000, OutputText: 4500000}, SourceURL: "https://developers.openai.com/api/docs/pricing", VerifiedAt: now.Add(-time.Hour), EffectiveFrom: now.Add(-time.Hour), EffectiveUntil: now.Add(48 * time.Hour)}}, FX: []d.PublishedFXRate{{Currency: "USD", RateDate: previous, CNYMicrosPerCurrencyUnit: 6719100, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/", VerifiedAt: now.Add(-time.Hour), EffectiveFrom: now.Add(-time.Hour), EffectiveUntil: now.Add(time.Hour)}}}
	initial, err := owner.Publish(ctx, "synthetic-operator", "synthetic-seed", seed)
	require.NoError(t, err)
	lock := db.Begin()
	require.NoError(t, lock.Error)
	defer lock.Rollback()
	require.NoError(t, lock.Exec("SELECT 1 FROM public.ai_route_config_current WHERE organization_id=? AND capability=? FOR UPDATE", d.PlatformPricesOrganization, d.PlatformPricesCapability).Error)
	transport := isolatedTransport(func(req *http.Request) (*http.Response, error) {
		body := fmt.Sprintf(`{"date":%q,"base":"USD","quote":"CNY","rate":6.7174}`, day)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})
	w := worker.New(c, runtime.TaskQueue, worker.Options{})
	require.NoError(t, runtime.RegisterWorker(w, owner, transport))
	require.NoError(t, w.Start())
	defer w.Stop()
	w2 := worker.New(c, runtime.TaskQueue, worker.Options{})
	require.NoError(t, runtime.RegisterWorker(w2, owner, transport))
	require.NoError(t, w2.Start())
	defer w2.Stop()
	require.NoError(t, runtime.PrepareFresh(ctx, c, "default"))
	require.NoError(t, runtime.BootstrapFresh(ctx, c, "default"))
	schedule, err := c.ScheduleClient().GetHandle(ctx, runtime.ScheduleID).Describe(ctx)
	require.NoError(t, err)
	require.NotNil(t, schedule.Schedule.Spec)
	require.Len(t, schedule.Schedule.Spec.Intervals, 1)
	require.Equal(t, 7*24*time.Hour, schedule.Schedule.Spec.Intervals[0].Every)
	require.Eventually(t, func() bool {
		var n int64
		e := db.Raw("SELECT count(*) FROM pg_stat_activity WHERE wait_event_type='Lock' AND query LIKE '%ai_route_config_current%' AND pid<>pg_backend_pid()").Scan(&n).Error
		return e == nil && n > 0
	}, 10*time.Second, 50*time.Millisecond)
	require.NoError(t, c.ScheduleClient().GetHandle(ctx, runtime.ScheduleID).Trigger(ctx, client.ScheduleTriggerOptions{}))
	require.ErrorIs(t, runtime.RequestStop(ctx, c), runtime.ErrControlUnknown)
	stopped, stopErr := runtime.CheckStop(ctx, c)
	require.False(t, stopped)
	require.ErrorIs(t, stopErr, runtime.ErrControlUnknown)
	time.Sleep(500 * time.Millisecond)
	beforeRelease, err := owner.Current(ctx)
	require.NoError(t, err)
	require.Equal(t, initial.ID, beforeRelease.ID)
	require.NoError(t, lock.Commit().Error)
	require.Eventually(t, func() bool { current, e := owner.Current(ctx); return e == nil && current.ID != initial.ID }, 25*time.Second, 250*time.Millisecond)
	current, err := owner.Current(ctx)
	require.NoError(t, err)
	tariff, err := current.Payload.PlatformPrices.Resolve("fx-iso", "synthetic-model", "synthetic-channel", time.Now())
	require.NoError(t, err)
	require.EqualValues(t, 6717400, tariff.CNYMicrosPerCurrencyUnit)
	require.Eventually(t, func() bool { stopped, e := runtime.CheckStop(ctx, c); return e == nil && stopped }, 10*time.Second, 100*time.Millisecond)
	time.Sleep(500 * time.Millisecond)
	rows, err := repo.History(ctx, d.PlatformPricesScope(), "", 50)
	require.NoError(t, err)
	require.Len(t, rows, 2, "queued tick cannot duplicate the publication")
	require.Error(t, runtime.BootstrapFresh(ctx, c, "default"), "existing marker must block reinitialization")
	require.NoError(t, c.TerminateWorkflow(ctx, runtime.CoordinatorID, "", "synthetic termination"))
	stopped, stopErr = runtime.CheckStop(ctx, c)
	require.False(t, stopped)
	require.ErrorIs(t, stopErr, runtime.ErrControlUnknown)
	require.NoError(t, c.ScheduleClient().GetHandle(ctx, runtime.ScheduleID).Trigger(ctx, client.ScheduleTriggerOptions{}))
	time.Sleep(1500 * time.Millisecond)
	desc, err := c.DescribeWorkflowExecution(ctx, runtime.CoordinatorID, "")
	require.NoError(t, err)
	require.Equal(t, enums.WORKFLOW_EXECUTION_STATUS_TERMINATED, desc.WorkflowExecutionInfo.Status)
}
