package httpapi

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/core/config"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/pod"
	"gorm.io/gorm"
	"strconv"
)

// Reuse the current POD-origin and immutable material owner, independently of
// the generation template, worker start or Flare route's readiness.
func configureImageAnalysisOrigin(a *imageAnalysisApplication, productDB, imageDB *gorm.DB, cfg *config.Config) {
	if a == nil || imageDB == nil || a.materials.ReadGenerated == nil {
		return
	}
	a.materials.ReadOrigin = func(ctx context.Context, b agent.Binding, img catalog.IdentifiedImage, maximum int64) ([]byte, error) {
		guard, err := buildArtworkOriginGuard(productDB, imageDB, a.parent.config.RunDB)
		if err != nil {
			return nil, err
		}
		guard.URLs = imageAgentDurableAssetPublicURLResolver(cfg)
		guard.ReadGeneratedBytes = a.materials.ReadGenerated
		i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
		if !ok {
			return nil, agent.ErrUnavailable
		}
		version, _ := strconv.ParseUint(b.CatalogVersion, 10, 63)
		ref := pod.ArtworkReference{Input: pod.InputReference{Source: collection.Source{ProductKey: b.ProductKey, PublicationID: b.PublicationID, Version: version}}}
		raw, handled, err := guard.ReadArtworkSourceBytes(ctx, collection.Scope{OrganizationID: i.TenantID, ActorID: i.UserID, MemberID: i.EffectiveMemberID}, ref, asset.ApprovedAsset{ID: img.ID, URL: img.URL, Width: img.Width, Height: img.Height}, maximum)
		if !handled && err == nil {
			return nil, agent.ErrUnavailable
		}
		return raw, err
	}
}
