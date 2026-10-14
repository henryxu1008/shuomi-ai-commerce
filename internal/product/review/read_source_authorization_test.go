package review

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/product/sourcing"
	"testing"
	"time"
)

type protectedViewStore struct {
	Store
	record Record
	reads  int
}

func (s *protectedViewStore) Read(context.Context, Scope, string) (Record, error) {
	s.reads++
	return s.record, nil
}
func (s *protectedViewStore) List(context.Context, Scope, PageRequest) (Page, error) {
	s.reads++
	return Page{Items: []CollectionItem{{ID: s.record.ID, ProductKey: s.record.Input.ProductKey, BaseVersion: 1, Revision: 1, State: "pending"}}}, nil
}
func (s *protectedViewStore) FindAgentReviewID(context.Context, Scope, string) (string, bool, error) {
	s.reads++
	return s.record.ID, true, nil
}

func TestReviewReadEntrypointsRecheckSourceAuthorizationBeforeProtectedRead(t *testing.T) {
	for _, entry := range []string{"get", "agent-view", "list"} {
		t.Run(entry, func(t *testing.T) {
			record := contentRecordFixture(t)
			store := &protectedViewStore{record: record}
			auth, err := authz.NewListingKitAuthorizer(nil, nil)
			require.NoError(t, err)
			source := &sourcePublicationReaderStub{err: sourcing.ErrPublicationForbidden}
			service := &Service{store: store, auth: auth, sourceReader: source}
			ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: record.Owner, TenantID: record.Org, EffectiveOrganizationID: record.Org, Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Hour)})
			switch entry {
			case "get":
				_, err = service.Get(ctx, record.ID)
			case "agent-view":
				_, _, err = service.FindAgentReview(ctx, record.Execution.RunID)
			case "list":
				_, err = service.List(ctx, PageRequest{Limit: 20})
			}
			require.ErrorIs(t, err, ErrForbidden)
			require.Zero(t, store.reads, "source revocation must prevent disclosure of both content and metadata")
		})
	}
}
