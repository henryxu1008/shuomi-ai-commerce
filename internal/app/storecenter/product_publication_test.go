package storecenterapp

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/shein/model"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

func TestPublicationHandleCannotUseReadAuthorityForShelfMutation(t *testing.T) {
	now := time.Now().UTC()
	reader := &productMaterialReader{material: storecenter.ProductExecutionMaterial{StoreVersion: 1, Platform: "shein", ServiceExpiresAt: now.Add(time.Hour), Connection: storecenter.OfficialConnectionView{AttemptID: "connection-a", Version: 1, Status: "connected", State: "verified"}, Attempt: storecenter.OfficialConnectionAttempt{AppID: "app-a", AppVersion: BoundOfficialRevision("v1", storecenter.ApplicationSelfOperated), State: "verified", KeyID: "key-a", Ciphertext: "sealed", AttemptID: "connection-a", OrganizationID: "org-a", StoreID: "store-a"}}}
	provider := &publicationTestProvider{productProvider: productProvider{appVersion: BoundOfficialRevision("v1", storecenter.ApplicationSelfOperated)}}
	registry, e := NewOfficialApplicationRegistry([]OfficialApplicationRegistration{{Provider: provider, Protection: productSecretProtection{}, Type: storecenter.ApplicationSelfOperated}})
	require.NoError(t, e)
	access, e := NewOfficialProductAccess(reader, productLiveAccess{}, registry)
	require.NoError(t, e)
	subject := storecenter.ProductExecutionSubject{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a", Purpose: storecenter.ProductPurposeStatus}
	h, e := access.Authorize(context.Background(), subject, "store-a", nil)
	require.NoError(t, e)
	_, e = h.QueryDocumentState(context.Background(), "spu", "v1")
	require.NoError(t, e)
	_, e = h.ShelfOn(context.Background(), "skc")
	require.ErrorIs(t, e, ErrProductAccessChanged)
	require.Zero(t, provider.mutations)
	subject.Purpose = storecenter.ProductPurposeShelfOn
	h, e = access.Authorize(context.Background(), subject, "store-a", nil)
	require.NoError(t, e)
	_, e = h.ShelfOn(context.Background(), "skc")
	require.NoError(t, e)
	_, e = h.ShelfOn(context.Background(), "skc")
	require.ErrorIs(t, e, ErrProductAccessChanged)
	require.Equal(t, 1, provider.mutations)
	h, e = access.Authorize(context.Background(), subject, "store-a", nil)
	require.NoError(t, e)
	reader.err = errors.New("revoked")
	_, e = h.ShelfOn(context.Background(), "skc")
	require.ErrorIs(t, e, ErrProductAccessChanged)
	require.Equal(t, 1, provider.mutations)
}

type publicationTestProvider struct {
	productProvider
	mutations int
}

func (p *publicationTestProvider) QueryDocumentState(context.Context, storecenter.OfficialMerchantCredential, string, string) (model.DocumentAudit, error) {
	return model.DocumentAudit{}, nil
}
func (p *publicationTestProvider) QueryShelfProduct(context.Context, storecenter.OfficialMerchantCredential, string) (model.ShelfProduct, error) {
	return model.ShelfProduct{}, nil
}
func (p *publicationTestProvider) QuerySKUInventory(context.Context, storecenter.OfficialMerchantCredential, []string, string) (model.InventorySnapshot, error) {
	return model.InventorySnapshot{}, nil
}
func (p *publicationTestProvider) ShelfOn(context.Context, storecenter.OfficialMerchantCredential, string) (model.ShelfResult, error) {
	p.mutations++
	return model.ShelfResult{}, nil
}
