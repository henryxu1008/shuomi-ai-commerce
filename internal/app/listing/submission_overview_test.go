package listingapp

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"task-processor/internal/app/workbenchmetrics"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type overviewAuthorization struct {
	scope  collection.Scope
	calls  int
	revoke bool
}

func (a *overviewAuthorization) Authorize(context.Context, string) (collection.Scope, error) {
	a.calls++
	if a.revoke && a.calls > 1 {
		return collection.Scope{}, record.ErrForbidden
	}
	return a.scope, nil
}

func TestSubmissionOverviewDoesNotTurnUnavailableOrUnsupportedStoresIntoZero(t *testing.T) {
	now := time.Now()
	scope := collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	for _, tc := range []struct {
		name, platform, region string
		excluded               int
		rowStatus              string
	}{
		{"unavailable", "shein", "US", 0, "unavailable"},
		{"unsupported site", "shein", "GB", 1, "empty"},
		{"unsupported platform", "amazon", "US", 1, "empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			auth := &overviewAuthorization{scope: scope}
			app := &Application{Authorization: auth,
				CockpitStoreAccess: func(_ context.Context, s collection.Scope, _ []string) error {
					if s != scope {
						t.Fatal("scope changed")
					}
					return nil
				},
				CockpitSubmissionStores: func(_ context.Context, s collection.Scope, ids []string) ([]CockpitSubmissionStore, error) {
					if s != scope || len(ids) != 1 {
						t.Fatal("scope lost")
					}
					return []CockpitSubmissionStore{{ID: ids[0], Platform: tc.platform, Region: tc.region}}, nil
				},
			}
			got, err := app.SubmissionOverview(context.Background(), workbenchmetrics.Filter{Start: now.Add(-24 * time.Hour), End: now, Stores: []string{"store"}}, now)
			if err != nil {
				t.Fatal(err)
			}
			if got.UnknownTotal != nil || got.Status != "incomplete" || got.ExcludedStores != tc.excluded || got.Rows[2].Status != tc.rowStatus {
				t.Fatalf("incomplete result leaked numbers: %+v", got)
			}
			if auth.calls < 2 {
				t.Fatal("missing fresh authorization")
			}
		})
	}
}

func TestSubmissionOverviewRejectsScopeChangesAndIncompleteDirectory(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name                     string
		revoke, missing, changed bool
	}{
		{"revoked", true, false, false}, {"missing directory row", false, true, false}, {"changed directory", false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			app := &Application{Authorization: &overviewAuthorization{scope: collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}, revoke: tc.revoke},
				CockpitStoreAccess: func(context.Context, collection.Scope, []string) error { return nil },
				CockpitSubmissionStores: func(_ context.Context, _ collection.Scope, ids []string) ([]CockpitSubmissionStore, error) {
					calls++
					if tc.missing {
						return nil, nil
					}
					platform := "amazon"
					if tc.changed && calls > 1 {
						platform = "shein"
					}
					return []CockpitSubmissionStore{{ID: ids[0], Platform: platform, Region: "US"}}, nil
				},
			}
			_, err := app.SubmissionOverview(context.Background(), workbenchmetrics.Filter{Start: now.Add(-24 * time.Hour), End: now, Stores: []string{"store"}}, now)
			if err == nil {
				t.Fatal("incomplete or changed identity accepted")
			}
			if tc.revoke && !errors.Is(err, record.ErrForbidden) {
				t.Fatalf("unexpected authorization error: %v", err)
			}
		})
	}
}

var _ preparation.Authorizer = (*overviewAuthorization)(nil)

type overviewSheinFacts struct {
	*uploadOfficial
	records      map[string]record.TargetRecord
	kernels      []*uploadKernel
	limitedStore string
}

func (f *overviewSheinFacts) ListSheinPublishAttemptIDs(_ context.Context, _ collection.Scope, cohort submission.SheinPublishReadScope, start, end time.Time, limit int) ([]string, error) {
	if cohort.StoreID == f.limitedStore {
		return make([]string, 251), nil
	}
	ids := []string{}
	for _, kernel := range f.kernels {
		for _, attempt := range kernel.attempts {
			if attempt.Target.StoreID == cohort.StoreID && attempt.Action == submission.OfficialPublishAction && !attempt.CreatedAt.Before(start) && attempt.CreatedAt.Before(end) {
				ids = append(ids, attempt.AttemptID, attempt.AttemptID) // Replayed original key must still count once.
			}
		}
	}
	return ids, nil
}
func (f *overviewSheinFacts) ReadCockpitSubmissionAttempt(_ context.Context, _ collection.Scope, id string) (submission.ExecutionAttempt, error) {
	for _, kernel := range f.kernels {
		for _, attempt := range kernel.attempts {
			if attempt.AttemptID == id {
				return attempt, nil
			}
		}
	}
	return submission.ExecutionAttempt{}, record.ErrUnavailable
}

type overviewRecordReader struct {
	record.TargetRepository
	records map[string]record.TargetRecord
}

func (r overviewRecordReader) ReadTargetRecord(_ context.Context, _ collection.Scope, id string) (record.TargetRecord, error) {
	v, ok := r.records[id]
	if !ok {
		return v, record.ErrNotFound
	}
	return v, nil
}
func TestSubmissionOverviewSumsNativeOriginalAttemptsAcrossStoresAndDropsLimitedPlatform(t *testing.T) {
	ctx := context.Background()
	one, scope, first, _, kernel, receipts, auth := uploadFixture(t)
	two, _, second, _, secondKernel, secondReceipts, _ := uploadFixture(t)
	for _, seed := range []struct {
		service *UploadService
		saved   record.TargetRecord
	}{{one, first.saved}, {two, second.saved}} {
		key := uuid.NewString()
		if _, err := seed.service.Upload(ctx, scope, key, seed.saved.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := seed.service.Upload(ctx, scope, key, seed.saved.ID); err != nil {
			t.Fatal(err)
		}
	}
	for k, v := range secondReceipts.intents {
		receipts.intents[k] = v
	}
	for k, v := range secondReceipts.receipts {
		receipts.receipts[k] = v
	}
	facts := &overviewSheinFacts{uploadOfficial: receipts, kernels: []*uploadKernel{kernel, secondKernel}, records: map[string]record.TargetRecord{first.saved.ID: first.saved, second.saved.ID: second.saved}}
	bindings := map[string]storecenter.ProductMerchantBinding{first.saved.Merchant.StoreID: first.saved.Merchant, second.saved.Merchant.StoreID: second.saved.Merchant}
	app := &Application{Authorization: auth, Records: overviewRecordReader{records: facts.records}, SheinPublishSource: facts,
		CockpitStoreAccess: func(context.Context, collection.Scope, []string) error { return nil },
		CockpitSubmissionStores: func(_ context.Context, _ collection.Scope, ids []string) ([]CockpitSubmissionStore, error) {
			out := []CockpitSubmissionStore{}
			for _, id := range ids {
				out = append(out, CockpitSubmissionStore{ID: id, Platform: "shein", Region: "US"})
			}
			return out, nil
		},
		SheinPublishMerchant: func(_ context.Context, _ collection.Scope, id string) (storecenter.ProductMerchantBinding, error) {
			return bindings[id], nil
		},
	}
	now := time.Now()
	filter := workbenchmetrics.Filter{Start: now.Add(-time.Hour), End: now.Add(time.Second), Stores: []string{first.saved.Merchant.StoreID, second.saved.Merchant.StoreID}}
	got, err := app.SubmissionOverview(ctx, filter, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "complete" || got.UnknownTotal == nil || *got.UnknownTotal != 0 || got.Rows[2].Counts == nil || got.Rows[2].Counts.Confirmed != 2 || got.Rows[2].Stores != 2 {
		t.Fatalf("wrong cross-store result: %+v", got)
	}
	facts.limitedStore = filter.Stores[1]
	got, err = app.SubmissionOverview(ctx, filter, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "incomplete" || got.UnknownTotal != nil || got.Rows[2].Status != "limited" || got.Rows[2].Counts != nil {
		t.Fatalf("limited platform leaked partial counts: %+v", got)
	}
}
