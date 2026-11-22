package target

import (
	"context"
	"testing"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	temumodel "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/temu/model"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type temuRuleQueryUnusedSources struct{}

func (temuRuleQueryUnusedSources) Select(context.Context, string) (preparation.AuthorizedSource, error) {
	return preparation.AuthorizedSource{}, ErrForbidden
}

func TestTemuMetadataQueryCannotBeSavedAsSheinTarget(t *testing.T) {
	scope := collection.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}
	service := &TargetService{dependencies: TargetDependencies{Authorizer: targetAuth{scope}, Sources: temuRuleQueryUnusedSources{}}}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
	_, err := service.Create(ctx, uuid.NewString(), TargetInput{SourceID: uuid.NewString(), StoreID: uuid.NewString(), EffectiveVersion: 1, TemuRules: &temumodel.RuleSelection{CatID: 34}})
	require.ErrorIs(t, err, ErrNotReady, "query-only TEMU selection must not enter the current SHEIN draft/record pipeline")
}
