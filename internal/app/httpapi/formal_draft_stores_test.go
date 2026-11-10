package httpapi

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	listinghttp "task-processor/internal/app/listing/httpapi"
	"task-processor/internal/httproute"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/collection"
	collectionhttp "task-processor/internal/product/collection/httpapi"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type formalStoreSpy struct {
	calls   int
	failure error
}

func (s *formalStoreSpy) ReadProductExecution(context.Context, storecenter.ProductExecutionSubject, string, storecenter.ProductExecutionAuthorizer, time.Time) (storecenter.ProductExecutionMaterial, error) {
	s.calls++
	return storecenter.ProductExecutionMaterial{}, s.failure
}

func TestFormalStorePurposeGuardPrecedesCapturedConsumersAndPreservesOtherFailures(t *testing.T) {
	spy := &formalStoreSpy{failure: errors.New("original dependency failure")}
	guarded := formalDraftStores{spy, "org", "test-store"}
	for _, purpose := range []string{storecenter.ProductPurposeRules, storecenter.ProductPurposeStatus, storecenter.ProductPurposePublish, storecenter.ProductPurposeImage, storecenter.ProductPurposeShelfOn} {
		_, e := guarded.ReadProductExecution(context.Background(), storecenter.ProductExecutionSubject{OrganizationID: "org", Purpose: purpose}, "test-store", nil, time.Now())
		require.ErrorIs(t, e, storecenter.ErrNotFound)
	}
	require.Zero(t, spy.calls)
	_, e := guarded.ReadProductExecution(context.Background(), storecenter.ProductExecutionSubject{OrganizationID: "org", Purpose: storecenter.ProductPurposeRules}, "ordinary-store", nil, time.Now())
	require.ErrorIs(t, e, spy.failure)
	require.Equal(t, 1, spy.calls)
	require.NoError(t, formalDraftRead(context.Background(), collection.Scope{}, record.TargetRecord{}), "unmarked history keeps its existing contract")
	require.ErrorIs(t, formalDraftRead(context.Background(), collection.Scope{}, record.TargetRecord{LocalTrial: &record.LocalTrial{}}), record.ErrForbidden)
	require.ErrorIs(t, formalDraftRead(context.Background(), collection.Scope{}, record.TargetRecord{Input: record.TargetInput{LocalTrial: &record.LocalTrial{}}}), record.ErrForbidden)
}

func TestDraftCoexistenceRouteComposition(t *testing.T) {
	for _, tc := range []struct{ local, formal, writes bool }{{true, false, true}, {false, true, false}, {true, true, true}, {true, true, false}} {
		routes := []httproute.Descriptor{}
		for _, r := range currentWorkbenchApplicationRoutes {
			routes = append(routes, httproute.Descriptor{Method: r.Method, Path: r.Path})
		}
		routes = append(routes, collectionhttp.Routes(nil, nil)...)
		if tc.formal {
			routes = append(routes, listinghttp.ListingRoutes(nil, nil)...)
		} else {
			routes = append(routes, listinghttp.PreparationRoutes(nil, nil)...)
		}
		routes = append(routes, listinghttp.CompletionDraftRoutes(nil, nil)...)
		local := listinghttp.ControlledLocalDraftRoutes(nil, nil, tc.writes)
		if tc.local {
			routes = append(routes, local...)
		}
		flags := currentApplicationOptionalRoutes{Collections: true, CompletionDrafts: true, SupplyChain: tc.formal, ControlledLocalDraft: tc.local, ControlledLocalWrites: tc.writes}
		require.NoError(t, validateCurrentApplicationRoutesInternal(routes, false, false, false, false, false, false, false, flags))
		if tc.local {
			changed := append([]httproute.Descriptor{}, routes...)
			changed[len(changed)-1].Module = "supply-chain"
			require.Error(t, validateCurrentApplicationRoutesInternal(changed, false, false, false, false, false, false, false, flags))
			flags.ControlledLocalDraft = false
			require.Error(t, validateCurrentApplicationRoutesInternal(routes, false, false, false, false, false, false, false, flags))
		}
	}
}
