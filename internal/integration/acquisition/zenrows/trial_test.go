package zenrows

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	jobstore "task-processor/internal/integration/persistence/product/dataacquisition"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
)

const (
	testKey = "fixture-key+/%&=do-not-log"
	testID  = "67fa4e35647515d8ad61bb3ee041e1bb"
)

var trialRequest = dataacquisition.SHEINUSTrialRequest{
	URL:  "https://us.shein.com/Women-Dresses-c-1727.html",
	Mode: "auto", Extract: "auto", ProxyCountry: "us", CardLimit: 20,
}

func approvalFixture() dataacquisition.SHEINUSTrialApproval {
	return dataacquisition.SHEINUSTrialApproval{
		ID: uuid.NewString(), ProjectID: uuid.NewString(), AccountRef: uuid.NewString(), KeyRef: uuid.NewString(),
		Scope:     collection.Scope{OrganizationID: "test-org", ActorID: "test-actor", MemberID: "test-member"},
		ExpiresAt: time.Now().Add(time.Minute), Funding: orgresource.FundingMember,
	}
}

type approvalAuthority struct {
	approval dataacquisition.SHEINUSTrialApproval
}

func (a approvalAuthority) AuthorizeSHEINUSTrial(context.Context, dataacquisition.Principal, string, dataacquisition.SHEINUSTrialAction) (dataacquisition.SHEINUSTrialApproval, error) {
	return a.approval, nil
}

func (a approvalAuthority) CheckSHEINUSTrialBudget(context.Context, dataacquisition.SHEINUSTrialApproval) (dataacquisition.SHEINUSTrialBudget, error) {
	return dataacquisition.SHEINUSTrialBudget{
		Binding: a.approval.Binding(), AvailableCredits: 25, MaximumRequestCredits: 25,
		ZeroIncrementalCash: true, ExclusiveKeyWindow: true,
	}, nil
}

func TestFetchTrialPersistsOriginalUsageOverLocalTLS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := trialTestDB(t, ctx)
	store, err := jobstore.NewSHEINUSTrialStore(db)
	require.NoError(t, err)

	for _, tc := range []struct {
		name         string
		status       int
		creditHeader string
		want         int // negative means unknown
		behavior     string
	}{
		{"actual credits", 200, "25", 25, ""},
		{"known zero", 200, "0", 0, ""},
		{"paid not found", 404, "5", 5, ""},
		{"paid gone", 410, "10", 10, ""},
		{"explicit over cap", 200, "26", 26, ""},
		{"cost without credits", 200, "", -1, ""},
		{"fractional credits", 200, "0.001", -1, ""},
		{"overflow credits", 200, "2147483648", -1, ""},
		{"multiple credits", 200, "25", -1, "duplicate-credits"},
		{"multiple IDs", 200, "25", -1, "duplicate-id"},
		{"redirect is not followed", 307, "", -1, "redirect"},
		{"disconnect does not retry", 200, "25", -1, "disconnect"},
		{"partial body keeps request ID", 200, "25", -1, "truncated"},
		{"oversized body keeps request ID", 200, "25", -1, "oversized"},
		{"deadline keeps request ID", 200, "25", -1, "timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := approvalFixture()
			requestID := uuid.NewString()
			p := dataacquisition.Principal{Scope: a.Scope}
			fetch, calls, dials := localFetch(t, a, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Request-Id", requestID)
				w.Header().Set("X-Request-Cost", "0.001")
				w.Header().Set("X-Key-Credits-Remaining", "0")
				if tc.creditHeader != "" {
					w.Header().Set("X-Request-Credits", tc.creditHeader)
				}
				switch tc.behavior {
				case "duplicate-credits":
					w.Header().Add("x-request-credits", "10")
				case "duplicate-id":
					w.Header().Add("x-request-id", "another-id")
				case "redirect":
					w.Header().Set("Location", "/must-not-be-called")
				case "disconnect":
					conn, _, err := http.NewResponseController(w).Hijack()
					if err != nil {
						t.Error("fixture disconnect failed")
						return
					}
					_ = conn.Close()
					return
				case "truncated":
					w.Header().Set("Content-Length", "200")
				}
				w.WriteHeader(tc.status)
				if tc.behavior == "timeout" {
					_ = http.NewResponseController(w).Flush()
					<-r.Context().Done()
					return
				}
				if tc.behavior == "oversized" {
					_, _ = w.Write([]byte(strings.Repeat("x", maxResponseBytes+1)))
					return
				}
				_, _ = w.Write([]byte("{}"))
			})

			if tc.behavior == "timeout" {
				a.ExpiresAt = time.Now().Add(2 * time.Second)
			}
			if tc.behavior == "disconnect" {
				a.ExpiresAt = time.Now().Add(10 * time.Second)
			}
			proof := UsageEvidence{}
			lookup, err := NewLookup(func(context.Context, dataacquisition.SHEINUSTrialApproval, dataacquisition.SHEINUSTrialAttempt) (UsageEvidence, error) {
				return proof, nil
			})
			require.NoError(t, err)
			svc, err := dataacquisition.NewSHEINUSTrial(store, approvalAuthority{a}, fetch.Send, lookup)
			require.NoError(t, err)
			attempt, err := svc.Send(ctx, p, a.ID)
			if tc.want < 0 {
				require.ErrorIs(t, err, dataacquisition.ErrUnknown)
				require.Equal(t, 25, attempt.ReservedCredits())
				require.Nil(t, attempt.Credits)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, *attempt.Credits)
				require.Equal(t, tc.want > 25, attempt.OverLimit())
			}
			switch tc.behavior {
			case "duplicate-id", "disconnect":
				require.Empty(t, attempt.RequestID)
			default:
				require.Equal(t, requestID, attempt.RequestID)
			}
			persisted, err := store.Read(ctx, a)
			require.NoError(t, err)
			require.Equal(t, attempt, persisted)
			replayed, _ := svc.Send(ctx, p, a.ID)
			require.Equal(t, attempt, replayed)
			if tc.behavior == "timeout" {
				pending, err := svc.Reconcile(ctx, p, a.ID)
				require.ErrorIs(t, err, dataacquisition.ErrUnknown)
				require.Equal(t, attempt, pending)
				actual := 10
				proof = UsageEvidence{Binding: a.Binding(), Records: []UsageRecord{
					{AccountRef: a.AccountRef, RequestID: "wrong-id", Final: true, Credits: &actual},
				}}
				pending, err = svc.Reconcile(ctx, p, a.ID)
				require.ErrorIs(t, err, dataacquisition.ErrUnknown)
				require.Equal(t, attempt, pending)
				proof.Records[0].RequestID = requestID
				settled, err := svc.Reconcile(ctx, p, a.ID)
				require.NoError(t, err)
				require.Equal(t, requestID, settled.RequestID)
				require.Equal(t, actual, *settled.Credits)
				require.Zero(t, settled.ReservedCredits())
			}
			if tc.behavior == "disconnect" {
				actual := 0
				proof = UsageEvidence{
					Binding: a.Binding(),
					Window:  &AcceptanceWindow{ClaimedAt: attempt.ClaimedAt, SendBefore: a.ExpiresAt},
					Records: []UsageRecord{{
						AccountRef: a.AccountRef, KeyRef: a.KeyRef, URL: trialRequest.URL, RequestID: requestID,
						AcceptedAt: attempt.ClaimedAt.Add(time.Millisecond), Final: true, Credits: &actual,
					}},
				}
				settled, err := svc.Reconcile(ctx, p, a.ID)
				require.NoError(t, err)
				require.Equal(t, requestID, settled.RequestID)
				require.Equal(t, 0, *settled.Credits)
				require.Zero(t, settled.ReservedCredits())
			}
			require.EqualValues(t, 1, calls.Load())
			require.EqualValues(t, 1, dials.Load())
		})
	}
}

func localFetch(t *testing.T, a dataacquisition.SHEINUSTrialApproval, reply http.HandlerFunc) (*Fetch, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	calls, dials := new(atomic.Int32), new(atomic.Int32)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		q := r.URL.Query()
		if r.ProtoMajor != 1 || r.Host != "api.zenrows.com" || r.URL.Path != "/v1/" || r.Method != http.MethodGet ||
			len(q) != 5 || q.Get("apikey") != testKey || q.Get("url") != trialRequest.URL ||
			q.Get("mode") != "auto" || q.Get("extract") != "auto" || q.Get("proxy_country") != "us" {
			t.Error("request left the approved Fetch shape")
		}
		reply(w, r)
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.Close)
	resolve, err := NewDedicatedKeyResolver(a, func(_ context.Context, accountRef, keyRef string) (string, error) {
		if accountRef != a.AccountRef || keyRef != a.KeyRef {
			return "", errors.New("wrong key binding")
		}
		return testKey, nil
	})
	require.NoError(t, err)
	fetch, err := NewFetch(resolve)
	require.NoError(t, err)
	transport := fetch.http.Transport.(*http.Transport)
	require.Nil(t, transport.Proxy)
	require.True(t, transport.DisableKeepAlives)
	require.True(t, transport.Protocols.HTTP1())
	require.False(t, transport.Protocols.HTTP2())
	fixtureTransport := server.Client().Transport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{RootCAs: fixtureTransport.TLSClientConfig.RootCAs, ServerName: "example.com"}
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dials.Add(1)
		if addr != "api.zenrows.com:443" {
			return nil, errors.New("unexpected host")
		}
		return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	return fetch, calls, dials
}

func TestDedicatedKeyResolverRejectsAnotherReferenceAndExpiry(t *testing.T) {
	a := approvalFixture()
	reads := 0
	read := func(context.Context, string, string) (string, error) {
		reads++
		return testKey, nil
	}
	resolve, err := NewDedicatedKeyResolver(a, read)
	require.NoError(t, err)
	_, err = resolve(context.Background(), "another-account", a.KeyRef)
	require.ErrorIs(t, err, dataacquisition.ErrUnavailable)
	_, err = resolve(context.Background(), a.AccountRef, "primary-key")
	require.ErrorIs(t, err, dataacquisition.ErrUnavailable)
	expired := a
	expired.ExpiresAt = time.Now().Add(-time.Second)
	stale, err := NewDedicatedKeyResolver(expired, read)
	require.NoError(t, err)
	_, err = stale(context.Background(), a.AccountRef, a.KeyRef)
	require.ErrorIs(t, err, dataacquisition.ErrUnavailable)
	require.Zero(t, reads)
	key, err := resolve(context.Background(), a.AccountRef, a.KeyRef)
	require.NoError(t, err)
	require.Equal(t, testKey, key)
	require.Equal(t, 1, reads)
}

func TestFetchRedactsKeyAndRejectsDifferentRequest(t *testing.T) {
	ctx := context.Background()
	a := approvalFixture()
	fetch, err := NewFetch(func(context.Context, string, string) (string, error) { return testKey, nil })
	require.NoError(t, err)
	transport := fetch.http.Transport.(*http.Transport)
	var calls int
	transport.DialContext = func(context.Context, string, string) (net.Conn, error) {
		calls++
		return nil, errors.New("private dial detail " + testKey)
	}
	bad := trialRequest
	bad.URL += "?page=2"
	_, err = fetch.Send(ctx, a, bad)
	require.ErrorIs(t, err, dataacquisition.ErrInvalid)
	require.Zero(t, calls)
	_, err = fetch.Send(ctx, a, trialRequest)
	require.ErrorIs(t, err, dataacquisition.ErrUnknown)
	require.Equal(t, 1, calls)
	require.NotContains(t, err.Error(), testKey)
	require.NotContains(t, err.Error(), url.QueryEscape(testKey))
	require.NotContains(t, err.Error(), "apikey")
	require.NotContains(t, err.Error(), "private dial detail")
	require.NotErrorAs(t, err, new(*url.Error))
	var logs bytes.Buffer
	slog.New(slog.NewTextHandler(&logs, nil)).Error("Fetch failed", "err", err)
	require.NotContains(t, logs.String(), testKey)
	require.NotContains(t, logs.String(), url.QueryEscape(testKey))
	require.NotContains(t, logs.String(), "apikey")
}

func TestLookupEvidenceRequiresOneOriginalFinalRecord(t *testing.T) {
	ctx := context.Background()
	a := approvalFixture()
	claimed := time.Now().Add(-time.Second)
	original := dataacquisition.SHEINUSTrialAttempt{
		SHEINUSTrialUsage: dataacquisition.SHEINUSTrialUsage{ApprovalID: a.ID, Binding: a.Binding()},
		ClaimedAt:         claimed,
	}
	credit := 10
	matched := UsageRecord{AccountRef: a.AccountRef, KeyRef: a.KeyRef, URL: trialRequest.URL, RequestID: testID, AcceptedAt: claimed.Add(time.Millisecond), Final: true, Credits: &credit}
	for _, tc := range []struct {
		name     string
		knownID  string
		verified bool
		alter    func(*UsageEvidence)
		want     bool
	}{
		{"known ID", testID, false, nil, true},
		{"verified unique window", "", true, nil, true},
		{"unverified window", "", false, nil, false},
		{"no evidence", testID, false, func(p *UsageEvidence) { p.Records = nil }, false},
		{"wrong request ID", "not-original", true, nil, false},
		{"wrong binding", testID, false, func(p *UsageEvidence) { p.Binding = "wrong" }, false},
		{"other account", testID, false, func(p *UsageEvidence) { p.Records[0].AccountRef = "wrong" }, false},
		{"other key", "", true, func(p *UsageEvidence) { p.Records[0].KeyRef = "wrong" }, false},
		{"wrong URL", "", true, func(p *UsageEvidence) { p.Records[0].URL += "?page=2" }, false},
		{"after caller cutoff", "", true, func(p *UsageEvidence) {
			p.Records[0].AcceptedAt = claimed.Add(11 * time.Second)
		}, false},
		{"unproven claim binding", "", true, func(p *UsageEvidence) {
			p.Window.ClaimedAt = claimed.Add(-time.Second)
		}, false},
		{"asserted cutoff beyond maximum", "", true, func(p *UsageEvidence) {
			p.Window.SendBefore = claimed.Add(dataacquisition.SHEINUSTrialTimeout + time.Second)
		}, false},
		{"competing record missing account", "", true, func(p *UsageEvidence) {
			ambiguous := matched
			ambiguous.AccountRef = ""
			p.Records = append(p.Records, ambiguous)
		}, false},
		{"after maximum send duration before approval expiry", "", true, func(p *UsageEvidence) {
			p.Records[0].AcceptedAt = claimed.Add(45 * time.Second)
		}, false},
		{"competing record missing key", "", true, func(p *UsageEvidence) {
			ambiguous := matched
			ambiguous.KeyRef = ""
			p.Records = append(p.Records, ambiguous)
		}, false},
		{"competing record missing URL", "", true, func(p *UsageEvidence) {
			ambiguous := matched
			ambiguous.URL = ""
			p.Records = append(p.Records, ambiguous)
		}, false},
		{"after window", "", true, func(p *UsageEvidence) { p.Records[0].AcceptedAt = a.ExpiresAt }, false},
		{"unfinished", testID, false, func(p *UsageEvidence) { p.Records[0].Final = false }, false},
		{"unknown credits", testID, false, func(p *UsageEvidence) { p.Records[0].Credits = nil }, false},
		{"contradictory records for original ID", testID, false, func(p *UsageEvidence) {
			conflict := matched
			conflict.KeyRef = "wrong"
			p.Records = append(p.Records, conflict)
		}, false},
		{"candidate without acceptance time", "", true, func(p *UsageEvidence) {
			unverified := matched
			unverified.AcceptedAt = time.Time{}
			p.Records = append(p.Records, unverified)
		}, false},
		{"multiple matching records", "", true, func(p *UsageEvidence) { p.Records = append(p.Records, matched); p.Records[1].RequestID = "another" }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attempt := original
			attempt.RequestID = tc.knownID
			proof := UsageEvidence{Binding: a.Binding(), Records: []UsageRecord{matched}}
			if tc.verified {
				proof.Window = &AcceptanceWindow{ClaimedAt: claimed, SendBefore: claimed.Add(10 * time.Second)}
			}
			if tc.alter != nil {
				tc.alter(&proof)
			}
			lookup, err := NewLookup(func(context.Context, dataacquisition.SHEINUSTrialApproval, dataacquisition.SHEINUSTrialAttempt) (UsageEvidence, error) {
				return proof, nil
			})
			require.NoError(t, err)
			u, err := lookup(ctx, a, trialRequest, attempt)
			if !tc.want {
				require.ErrorIs(t, err, dataacquisition.ErrUnknown)
				require.Nil(t, u.Credits)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testID, u.RequestID)
			require.Equal(t, credit, *u.Credits)
			require.Equal(t, a.Binding(), u.Binding)
		})
	}
}

func trialTestDB(t *testing.T, ctx context.Context) *sql.DB {
	t.Helper()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("zenrows_trial"), tcpostgres.WithUsername("test_owner"), tcpostgres.WithPassword("isolated-zenrows"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pg.Terminate(context.Background())) })
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, jobstore.InstallSHEINUSTrialSchema(ctx, db))
	return db
}
