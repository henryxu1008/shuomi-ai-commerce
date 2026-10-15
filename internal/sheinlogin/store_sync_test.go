package sheinlogin

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"task-processor/internal/listingadmin"
)

func TestStoreSyncRejectsIncompleteOrCrossTenantLookupWithoutMutationOrDisclosure(t *testing.T) {
	var logs bytes.Buffer
	localLogger := logrus.New()
	localLogger.SetOutput(&logs)
	originalLogger := sheinLoginServiceLogger
	sheinLoginServiceLogger = logrus.NewEntry(localLogger)
	t.Cleanup(func() { sheinLoginServiceLogger = originalLogger })
	for _, tc := range []struct {
		name      string
		duplicate *listingadmin.StoreRespDTO
		err       error
		missing   bool
	}{
		{"foreign tenant", &listingadmin.StoreRespDTO{ID: 987654, TenantID: 876543, Name: "foreign-private-name", StoreID: "123", Platform: "SHEIN"}, nil, false},
		{"zero tenant", &listingadmin.StoreRespDTO{ID: 987654, Name: "foreign-private-name", StoreID: "123", Platform: "SHEIN"}, nil, false},
		{"invalid row", &listingadmin.StoreRespDTO{TenantID: 7, StoreID: "123", Platform: "SHEIN"}, nil, false},
		{"self", &listingadmin.StoreRespDTO{ID: 5, TenantID: 7, StoreID: "123", Platform: "SHEIN"}, nil, false},
		{"disabled", &listingadmin.StoreRespDTO{ID: 99, TenantID: 7, StoreID: "123", Platform: "SHEIN", Status: 1}, nil, false},
		{"other platform", &listingadmin.StoreRespDTO{ID: 99, TenantID: 7, StoreID: "123", Platform: "TEMU"}, nil, false},
		{"other external id", &listingadmin.StoreRespDTO{ID: 99, TenantID: 7, StoreID: "456", Platform: "SHEIN"}, nil, false},
		{"query failure", nil, errors.New("foreign-private-name tenant=876543 row=987654"), false},
		{"partial query failure", &listingadmin.StoreRespDTO{TenantID: 876543, Name: "foreign-private-name"}, errors.New("foreign-private-name"), false},
		{"missing lookup", nil, nil, true},
	} {
		for _, recorded := range []string{"", "456"} {
			t.Run(tc.name+"/recorded="+recorded, func(t *testing.T) {
				logs.Reset()
				client := &stubStoreClient{}
				svc := &Service{
					resolveStoreID: func(context.Context, Account) (int64, error) { return 123, nil },
					storeClientFor: func(int64) StoreSyncClient { return client },
				}
				if !tc.missing {
					svc.findDuplicateStore = func(context.Context, Account, string) (*listingadmin.StoreRespDTO, error) {
						return tc.duplicate, tc.err
					}
				}
				svc.syncStoreIDAfterLogin(context.Background(), Account{TenantID: 7, StoreID: 5, StoreName: recorded})
				require.Nil(t, client.updateStoreIDReq, "incomplete lookup must not persist StoreID")
				require.Nil(t, client.updateStoreStatusReq, "incomplete lookup must not disable current store")
				for _, private := range []string{"foreign-private-name", "876543", "987654"} {
					require.NotContains(t, logs.String(), private)
				}
			})
		}
	}
}

func TestStoreSyncRejectsInvalidAccountBeforeResolvingOrQuerying(t *testing.T) {
	for _, account := range []Account{{StoreID: 5}, {TenantID: -1, StoreID: 5}, {TenantID: 7}, {TenantID: 7, StoreID: -1}} {
		svc := &Service{resolveStoreID: func(context.Context, Account) (int64, error) {
			t.Error("invalid account must not resolve an external store id")
			return 123, nil
		}, storeClientFor: func(int64) StoreSyncClient {
			t.Error("invalid account must not obtain a mutation client")
			return &stubStoreClient{}
		}}
		svc.syncStoreIDAfterLogin(context.Background(), account)
	}
}

func TestWorkerLookupFailurePreservesCommittedCookieAndSuccessfulAttempt(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, &stubAutomation{result: &AutomationResult{
		BrowserState: map[string]any{"cookies": []any{map[string]any{"name": "sid", "value": "ok"}}},
	}})
	client := &stubStoreClient{}
	svc.resolveStoreID = func(context.Context, Account) (int64, error) { return 123, nil }
	svc.storeClientFor = func(int64) StoreSyncClient { return client }
	svc.findDuplicateStore = func(context.Context, Account, string) (*listingadmin.StoreRespDTO, error) {
		return nil, errors.New("lookup unavailable")
	}
	attempt, created, err := svc.store.EnqueueLoginAttempt(ctx, 1, 2, LoginRequest{ForceLogin: true})
	require.NoError(t, err)
	require.True(t, created)
	require.NoError(t, svc.processLoginAttemptMessage(ctx, "test-worker", "", map[string]any{"attempt_id": attempt.ID}))
	finished, err := svc.store.LoadLoginAttempt(ctx, attempt.ID)
	require.NoError(t, err)
	require.Equal(t, LoginAttemptSucceeded, finished.Status)
	hasCookie, err := svc.store.HasCookie(ctx, 1, 2)
	require.NoError(t, err)
	require.True(t, hasCookie)
	require.Nil(t, client.updateStoreIDReq)
	require.Nil(t, client.updateStoreStatusReq)
}
