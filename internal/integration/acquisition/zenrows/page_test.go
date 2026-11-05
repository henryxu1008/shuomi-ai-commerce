package zenrows

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"task-processor/internal/app/dataservices"
	"task-processor/internal/authidentity"
	keystore "task-processor/internal/integration/persistence/dataservice"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	jobstore "task-processor/internal/integration/persistence/product/dataacquisition"
	sourcestore "task-processor/internal/integration/persistence/product/sourcing"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
)

type pageLiveAccess struct{}

func (pageLiveAccess) CheckRead(context.Context, dataacquisition.Principal) error { return nil }
func (pageLiveAccess) CheckExecution(context.Context, dataacquisition.Principal, orgresource.ResourceFunding) error {
	return nil
}

type pageTrialAuthority struct {
	approvalAuthority
	check func(dataacquisition.SHEINUSTrialAction) error
}

func (a pageTrialAuthority) AuthorizeSHEINUSTrial(ctx context.Context, p dataacquisition.Principal, id string, action dataacquisition.SHEINUSTrialAction) (dataacquisition.SHEINUSTrialApproval, error) {
	if err := a.check(action); err != nil {
		return dataacquisition.SHEINUSTrialApproval{}, err
	}
	return a.approvalAuthority.AuthorizeSHEINUSTrial(ctx, p, id, action)
}

func TestFetchToSHEINUSPagePublication(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db := trialTestDB(t, ctx)
	product, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, install := range []func(*gorm.DB) error{keystore.InstallSchema, catalogstore.AutoMigrate, sourcestore.InstallSchema, collectionstore.InstallSchema, jobstore.InstallSchema} {
		require.NoError(t, install(product))
	}
	a := approvalFixture()
	p := dataacquisition.Principal{Scope: a.Scope}
	ctx = authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{
		TenantID: p.Scope.OrganizationID, EffectiveOrganizationID: p.Scope.OrganizationID,
		UserID: p.Scope.ActorID, EffectiveMemberID: p.Scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute),
	})
	for _, kind := range []string{"page", "wrong final URL", "timeout", "prepare rollback", "publish revoked", "same receipt race", "conflicting receipt race"} {
		t.Run(kind, func(t *testing.T) {
			a := approvalFixture()
			products := make([]map[string]any, 21)
			for i := range products {
				id := fmt.Sprint(100 + i)
				products[i] = map[string]any{"goods_id": id, "title": "Dress " + id, "product_url": "https://us.shein.com/Dress-p-" + id + ".html?track=example", "price": 12.34}
			}
			products[0]["price_note"] = "after coupon"
			products[1]["product_url"] = "https://us.shein.com/Dress-p-999.html"
			delete(products[2], "title")
			delete(products[3], "price")
			products[4]["product_url"] = "https://www.shein.com/Dress-p-104.html"
			requestID := a.ID
			fetch, calls, _ := localFetch(t, a, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Request-Id", requestID)
				w.Header().Set("X-Request-Credits", "10")
				finalURL := trialRequest.URL
				if kind == "wrong final URL" {
					finalURL = "https://www.shein.com/Women-Dresses-c-1727.html"
				}
				w.Header().Set("Zr-Final-Url", finalURL)
				if kind == "timeout" {
					w.WriteHeader(http.StatusOK)
					_ = http.NewResponseController(w).Flush()
					<-r.Context().Done()
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"parsed": map[string]any{
					"category_context": map[string]any{"currency": "USD"},
					"products":         products,
				}})
			})
			if kind == "timeout" {
				a.ExpiresAt = time.Now().Add(2 * time.Second)
			}
			lookup, err := NewLookup(func(context.Context, dataacquisition.SHEINUSTrialApproval, dataacquisition.SHEINUSTrialAttempt) (UsageEvidence, error) {
				credits := 10
				if kind == "conflicting receipt race" {
					credits = 7
				}
				return UsageEvidence{Binding: a.Binding(), Records: []UsageRecord{{AccountRef: a.AccountRef, RequestID: requestID, Final: true, Credits: &credits}}}, nil
			})
			require.NoError(t, err)
			var probe *dataservicesapp.SHEINUSPageTrial
			revoked := kind == "publish revoked"
			publishChecks := 0
			authority := pageTrialAuthority{approvalAuthority: approvalAuthority{a}, check: func(action dataacquisition.SHEINUSTrialAction) error {
				if action != dataacquisition.SHEINUSTrialPublish {
					return nil
				}
				if revoked {
					return dataacquisition.ErrForbidden
				}
				publishChecks++
				if publishChecks == 2 && (kind == "same receipt race" || kind == "conflicting receipt race") {
					_, err := probe.Reconcile(ctx, p, a.ID)
					require.NoError(t, err)
				}
				return nil
			}}
			probe, err = dataservicesapp.NewSHEINUSPageTrial(ctx, product, pageLiveAccess{}, authority, fetch.Send, lookup)
			require.NoError(t, err)
			if kind == "prepare rollback" {
				require.NoError(t, product.Exec("ALTER TABLE data_acquisition_items ADD CONSTRAINT reject_fixture_capture CHECK (source_kind <> 'shein_us_data') NOT VALID").Error)
				t.Cleanup(func() {
					require.NoError(t, product.Exec("ALTER TABLE data_acquisition_items DROP CONSTRAINT reject_fixture_capture").Error)
				})
			}
			attempt, saved, err := probe.Run(ctx, p, a.ID)
			switch kind {
			case "timeout", "prepare rollback":
				require.ErrorIs(t, err, dataacquisition.ErrUnknown)
				require.Equal(t, requestID, attempt.RequestID)
				require.Equal(t, 25, attempt.ReservedCredits())
				require.Empty(t, saved)
				attempt, err = probe.Reconcile(ctx, p, a.ID)
				require.NoError(t, err)
				_, saved, err = probe.Run(ctx, p, a.ID)
				require.NoError(t, err)
			case "publish revoked":
				require.ErrorIs(t, err, dataacquisition.ErrForbidden)
				revoked = false
				attempt, saved, err = probe.Run(ctx, p, a.ID)
				require.NoError(t, err)
			case "conflicting receipt race":
				require.ErrorIs(t, err, dataacquisition.ErrConflict)
				attempt, saved, err = probe.Run(ctx, p, a.ID)
				require.NoError(t, err)
			default:
				require.NoError(t, err)
			}
			require.Equal(t, requestID, attempt.RequestID)
			expectedCredits := 10
			if kind == "conflicting receipt race" {
				expectedCredits = 7
			}
			require.Equal(t, expectedCredits, *attempt.Credits)
			if kind == "page" || kind == "same receipt race" {
				require.Len(t, saved, 17)
				require.Equal(t, collection.StableID("shein-us-trial-item", a.ID, "100"), saved[0].PublicationID)
				require.Equal(t, collection.StableID("shein-us-trial-item", a.ID, "103"), saved[1].PublicationID)
				for _, i := range saved {
					require.NotEqual(t, collection.StableID("shein-us-trial-item", a.ID, "120"), i.PublicationID)
				}
				_, replayed, err := probe.Run(ctx, p, a.ID)
				require.NoError(t, err)
				require.Equal(t, saved, replayed)
				var counts struct{ Items, Reserved int }
				require.NoError(t, product.Raw("SELECT count(*) AS items, count(reservation_id) AS reserved FROM data_acquisition_items WHERE job_id=? AND state='SAVED'", a.ID).Scan(&counts).Error)
				require.Equal(t, 17, counts.Items)
				require.Zero(t, counts.Reserved)
				var state string
				require.NoError(t, product.Raw("SELECT state FROM data_acquisition_jobs WHERE id=?", a.ID).Row().Scan(&state))
				require.Equal(t, "SUCCEEDED", state)
				var raw []byte
				require.NoError(t, product.Raw("SELECT evidence_json FROM data_acquisition_items WHERE id=?", saved[0].PublicationID).Row().Scan(&raw))
				var observation dataacquisition.SHEINUSObservation
				require.NoError(t, json.Unmarshal(raw, &observation))
				require.Equal(t, "after coupon", *observation.Price.PromotionNote)
				require.NotContains(t, observation.SourceURL, "track")
				require.NoError(t, product.Raw("SELECT evidence_json FROM data_acquisition_items WHERE id=?", saved[1].PublicationID).Row().Scan(&raw))
				require.NoError(t, json.Unmarshal(raw, &observation))
				require.Nil(t, observation.Price)
				require.Contains(t, observation.Missing, "price")
			} else {
				require.Empty(t, saved)
				var count int
				require.NoError(t, product.Raw("SELECT count(*) FROM data_acquisition_jobs WHERE id=?", a.ID).Row().Scan(&count))
				require.Zero(t, count)
			}
			require.EqualValues(t, 1, calls.Load())
		})
	}
}
