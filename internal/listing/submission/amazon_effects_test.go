package submission

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	amazonmarket "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/amazonus"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
)

func TestAmazonWritePermitSharesConsumptionAcrossCopies(t *testing.T) {
	scope := collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	binding := storecenter.ProductMerchantBinding{StoreID: "11111111-1111-4111-8111-111111111111"}
	count := 0
	input := amazonmarket.DraftInput{ProductType: "LUGGAGE", Title: "Synthetic pouch", Brand: "Synthetic", GTIN: "036000291452",
		PriceCents: 1200, Quantity: &count, MainImageID: "test-image"}
	const schema = `{"$schema":"https://schemas.amazon.com/selling-partners/definitions/product-types/meta-schema/v1","type":"object"}`
	draft, err := amazonmarket.BuildDraft(input, binding.StoreID, "product", "https://images.example.org/main.jpg", []byte(schema))
	require.NoError(t, err)
	input.PriceCents++
	changed, err := amazonmarket.BuildDraft(input, binding.StoreID, "product", "https://images.example.org/main.jpg", []byte(schema))
	require.NoError(t, err)
	now := time.Now()
	base := AmazonWritePermit{
		intent: OfficialIntent{Owner: scope, Binding: binding, Amazon: &AmazonIntent{SKU: draft.SKU, WireHash: draft.WireHash},
			PayloadFingerprint: digestExecutionParts("amazon", binding.StoreID, draft.SKU, AmazonSubmitAction, string(draft.Attributes))},
		expiresAt: now.Add(time.Minute), closed: new(atomic.Bool),
	}
	require.ErrorIs(t, base.Consume(scope, binding, changed, now), ErrExecutionClaimRejected)
	differentStore := storecenter.ProductMerchantBinding{StoreID: "22222222-2222-4222-8222-222222222222"}
	require.ErrorIs(t, base.Consume(scope, differentStore, draft, now), ErrExecutionClaimRejected)
	require.ErrorIs(t, base.Consume(scope, binding, draft, base.expiresAt), ErrExecutionClaimRejected)
	var successes atomic.Int32
	var group sync.WaitGroup
	for range 8 {
		copyOfPermit := base
		group.Go(func() {
			if copyOfPermit.Consume(scope, binding, draft, now) == nil {
				successes.Add(1)
			}
		})
	}
	group.Wait()
	require.Equal(t, int32(1), successes.Load())

	t.Run("send and local stop cannot both win", func(t *testing.T) {
		ctx := context.Background()
		for range 20 {
			k, command := noSendKernelFixture(t)
			k.repository.(*noSendMemoryRepository).loseStopAck = true
			command.Target = ExecutionTarget{Platform: "amazon", StoreID: binding.StoreID, SubjectID: draft.SKU}
			command.Action, command.Payload = AmazonSubmitAction, draft.Attributes
			intent := base.intent
			intent.Key, intent.Target = command.IntentKey, command.Target
			command.Amazon = &AmazonSendAuthorization{intent: intent}
			acquired, err := k.Acquire(ctx, command)
			require.NoError(t, err)
			require.NotNil(t, acquired.AmazonPermit)
			copied := *acquired.AmazonPermit
			start := make(chan struct{})
			sent, stopped := make(chan error, 1), make(chan error, 1)
			go func() {
				<-start
				sent <- acquired.AmazonPermit.Consume(scope, binding, draft, k.now())
			}()
			go func() {
				<-start
				_, e := k.StopAmazonBeforeSend(ctx, &copied)
				stopped <- e
			}()
			close(start)
			sendErr, stopErr := <-sent, <-stopped
			require.NotEqual(t, sendErr == nil, stopErr == nil)
			persisted, err := k.Get(ctx, command.Scope, acquired.Attempt.AttemptID)
			require.NoError(t, err)
			if stopErr == nil {
				require.Equal(t, ExecutionNotSent, persisted.Status, "original receipt must survive a lost commit acknowledgement")
				require.Equal(t, NoSendAccessRecheckFailed, persisted.Evidence.Reason)
			} else {
				require.ErrorIs(t, stopErr, ErrExecutionClaimRejected)
				require.Equal(t, ExecutionClaimed, persisted.Status)
			}
		}
	})
}
