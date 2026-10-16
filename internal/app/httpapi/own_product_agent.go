package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"task-processor/internal/agent"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/contentagent"
)

const ownContentAgentRoot = "/api/v1/workbench/collections/items/:item_id/content-agent"
const ownContentAgentSource = ownContentAgentRoot + "/source"
const ownContentAgentBase = ownContentAgentRoot + "/runs"

type ownProductReader interface {
	ReadItem(context.Context, string) (collection.ItemDetail, error)
}
type ownSourceSelection struct {
	CatalogVersion string `json:"catalogVersion"`
	PublicationID  string `json:"publicationId"`
}

func (a *productAgentApplication) ownSource(ctx context.Context, i authidentity.AuthenticatedIdentity, itemID string, selected *ownSourceSelection) (contentagent.OwnSource, error) {
	if a == nil || a.ownProducts == nil {
		return contentagent.OwnSource{}, agent.ErrUnavailable
	}
	ctx = authidentity.WithAuthenticatedIdentity(ctx, i)
	item, err := a.ownProducts.ReadItem(ctx, itemID)
	if err != nil {
		return contentagent.OwnSource{}, err
	}
	if item.Item.ID != itemID {
		return contentagent.OwnSource{}, agent.ErrConflict
	}
	version, publication := "", ""
	if selected != nil {
		version, publication = selected.CatalogVersion, selected.PublicationID
		if version == "" || publication == "" {
			return contentagent.OwnSource{}, agent.ErrInvalid
		}
	}
	return contentagent.ResolveOwnSource(ctx, collection.Scope{OrganizationID: i.TenantID, ActorID: i.UserID, MemberID: i.EffectiveMemberID}, item, a.snapshots, a.reviewLookup, version, publication)
}

func (a *productAgentApplication) authorizeOwnBinding(ctx context.Context, i authidentity.AuthenticatedIdentity, binding agent.Binding) error {
	if !binding.Valid() || binding.ContextKind != contentagent.OwnProductContext || binding.TargetPlatform != "product" {
		return agent.ErrInvalid
	}
	source, err := a.ownSource(ctx, i, binding.ContextID, &ownSourceSelection{binding.CatalogVersion, binding.PublicationID})
	if err != nil {
		return err
	}
	if source.Binding != binding {
		return agent.ErrConflict
	}
	return nil
}

// The receipt namespace is read before selecting today's Catalog head. This is
// essential when the first request succeeded but its response was lost.
func (a *productAgentApplication) ownStartBinding(ctx context.Context, i authidentity.AuthenticatedIdentity, itemID, key string, body productAgentRequestBody) (agent.Binding, bool, error) {
	if body.TargetPlatform != "product" || body.OwnSource == nil {
		return agent.Binding{}, false, agent.ErrInvalid
	}
	existing, found, err := a.store.LookupContext(ctx, agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}, contentagent.OwnProductContext, itemID, key)
	if err != nil {
		return agent.Binding{}, false, err
	}
	if found {
		b := existing.State.Request.Binding
		if b.CatalogVersion != body.OwnSource.CatalogVersion || b.PublicationID != body.OwnSource.PublicationID || b.TargetPlatform != body.TargetPlatform {
			return agent.Binding{}, true, agent.ErrConflict
		}
		return b, true, a.authorizeOwnBinding(ctx, i, b)
	}
	source, err := a.ownSource(ctx, i, itemID, nil)
	if err == nil && (source.Binding.CatalogVersion != body.OwnSource.CatalogVersion || source.Binding.PublicationID != body.OwnSource.PublicationID) {
		err = agent.ErrConflict
	}
	return source.Binding, false, err
}

func ownContentAgentRoutes(a *productAgentApplication) []httproute.Descriptor {
	routes := productAgentRoutesForSource(a, true, true)
	return append(routes, httproute.Descriptor{Method: http.MethodGet, Path: ownContentAgentSource, Module: "product-agent", Permission: authz.PermissionWorkbenchCollectionRead, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 15 * time.Second, Handler: func(c *gin.Context) {
		if a == nil {
			writeProductAgentError(c, agent.ErrUnavailable)
			return
		}
		if !acquisitionHTTPUUID(c.Param("item_id")) || c.Request.URL.RawQuery != "" || readProductReviewGETBody(c) != nil {
			writeProductAgentError(c, agent.ErrInvalid)
			return
		}
		ctx, err := (productReviewCapabilityBinder{now: time.Now}).Bind(c.Request.Context(), c.GetHeader("Authorization"))
		if err != nil {
			writeProductAgentError(c, err)
			return
		}
		i, err := a.freshIdentity(ctx)
		if err != nil {
			writeProductAgentError(c, err)
			return
		}
		source, err := a.ownSource(ctx, i, c.Param("item_id"), nil)
		if err != nil {
			writeProductAgentError(c, err)
			return
		}
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, gin.H{"itemId": source.Binding.ContextID, "productKey": source.Binding.ProductKey, "catalogVersion": source.Binding.CatalogVersion, "publicationId": source.Binding.PublicationID, "originalVersion": source.Original.Version, "originalPublicationId": source.Original.PublicationID, "title": source.Effective.Snapshot.Title})
	}})
}
