package httpapi

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"

	credential "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/channelcredential"
	d "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/routeconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/websearch"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aiworkbench"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
)

func (m modelRoutePricingModule) searchSettingsRoutes() []httproute.Descriptor {
	type spec struct{ method, path, permission, action string }
	rows := []spec{
		{"GET", "/search/enterprise", authz.PermissionWorkbenchAgentConfigure, "list"},
		{"POST", "/search/enterprise", authz.PermissionWorkbenchAgentConfigure, "select"},
		{"GET", "/search/enterprise/receipts/:key", authz.PermissionWorkbenchAgentConfigure, "select-receipt"},
		{"GET", "/search/owned-key", authz.PermissionWorkbenchChannelsManage, "owned-key"},
		{"POST", "/search/owned-key", authz.PermissionWorkbenchChannelsManage, "owned-key"},
		{"GET", "/search/owned-key/receipts/:key", authz.PermissionWorkbenchChannelsManage, "owned-key-receipt"},
		{"GET", "/search/platform", authz.PermissionListingKitPlatformAdm, "platform"},
		{"POST", "/search/platform", authz.PermissionListingKitPlatformAdm, "publish"},
		{"GET", "/search/platform/receipts/:key", authz.PermissionListingKitPlatformAdm, "publish-receipt"},
		{"GET", "/search/platform-key", authz.PermissionListingKitPlatformAdm, "platform-key"},
		{"POST", "/search/platform-key", authz.PermissionListingKitPlatformAdm, "platform-key"},
		{"GET", "/search/platform-key/receipts/:key", authz.PermissionListingKitPlatformAdm, "platform-key-receipt"},
	}
	routes := make([]httproute.Descriptor, 0, len(rows))
	for _, row := range rows {
		routes = append(routes, httproute.Descriptor{
			Method: row.method, Path: modelRouteBase + row.path, Module: m.Name(), Permission: row.permission, RequestTimeout: 30 * time.Second,
			AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RejectUnreadRequestBody: row.method == "GET",
			Handler: func(c *gin.Context) { m.searchSettings(c, row.action, row.permission) },
		})
	}
	return routes
}

func writeSearchError(c *gin.Context, err error) {
	status, code := http.StatusServiceUnavailable, "UNAVAILABLE"
	switch {
	case errors.Is(err, websearch.ErrOutcomeUnknown):
		status, code = http.StatusServiceUnavailable, "OUTCOME_UNKNOWN"
	case errors.Is(err, websearch.ErrInvalid), errors.Is(err, credential.ErrInvalid):
		status, code = http.StatusBadRequest, "INVALID"
	case errors.Is(err, websearch.ErrNotFound), errors.Is(err, credential.ErrNotFound), errors.Is(err, aiworkbench.ErrNotFound):
		status, code = http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, websearch.ErrConflict), errors.Is(err, credential.ErrConflict):
		status, code = http.StatusConflict, "CONFLICT"
	case errors.Is(err, websearch.ErrQuota):
		status, code = http.StatusTooManyRequests, "SEARCH_LIMIT"
	case errors.Is(err, errModelRouteForbidden):
		status, code = http.StatusForbidden, "FORBIDDEN"
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(status, gin.H{"code": code})
}

func (m modelRoutePricingModule) searchSettings(c *gin.Context, action, permission string) {
	ctx := c.Request.Context()
	id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	actor, org := id.UserID, id.EffectiveOrganizationID
	write := c.Request.Method == http.MethodPost
	platform := permission == authz.PermissionListingKitPlatformAdm
	if !ok || org == credential.PlatformOwner || org != id.TenantID ||
		!credential.ValidID(org) || !credential.ValidID(actor) || !credential.ValidID(id.EffectiveMemberID) ||
		!time.Now().Before(id.TokenExpiresAt) || !authz.AllowedOrganization(ctx, m.authorizer, actor, org, id.Roles, permission) {
		writeSearchError(c, errModelRouteForbidden)
		return
	}
	if platform {
		if _, _, err := sharedModelIdentity(ctx, m.authorizer, true, write); err != nil || !m.services.PlatformChannels.allows(id) {
			writeSearchError(c, errModelRouteForbidden)
			return
		}
	}
	if permission == authz.PermissionWorkbenchChannelsManage && !authz.AllowedOrganization(ctx, m.authorizer, actor, org, id.Roles, authz.PermissionWorkbenchCredentialsManage) {
		writeSearchError(c, errModelRouteForbidden)
		return
	}
	service := m.services.Search
	if service == nil || !platform && !slices.Contains(m.services.SearchOrganizations, org) {
		writeSearchError(c, websearch.ErrUnavailable)
		return
	}
	if c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
		writeSearchError(c, websearch.ErrInvalid)
		return
	}
	key := c.Param("key")
	if write {
		key = c.GetHeader("Idempotency-Key")
	}
	if (write || key != "") && !d.ValidFact(key, 128) {
		writeSearchError(c, websearch.ErrInvalid)
		return
	}
	var value any
	var err error
	owner := org
	if platform {
		owner = credential.PlatformOwner
	}
	out := gin.H{"organizationId": org, "actorId": actor, "memberId": id.EffectiveMemberID, "requestKey": key}
	switch action {
	case "list":
		value, err = service.View(ctx, org)
	case "select":
		var input struct {
			websearch.Selection
			ExpectedMemberID string `json:"expectedMemberId"`
		}
		if !platformDecode(c, &input, func() string { return input.ExpectedMemberID }, id) {
			return
		}
		value, err = service.Select(ctx, org, actor, key, input.Selection)
	case "select-receipt":
		value, err = service.Repo.SelectionReceipt(ctx, org, actor, key)
	case "platform":
		out["canWrite"] = authz.AllowedOrganization(ctx, m.authorizer, actor, org, id.Roles, authz.PermissionListingKitAdminWrite)
		out["organizations"] = m.services.SearchOrganizations
		value, err = service.Repo.Publication(ctx)
	case "publish":
		var input struct {
			websearch.Publication
			ExpectedMemberID string `json:"expectedMemberId"`
		}
		if !platformDecode(c, &input, func() string { return input.ExpectedMemberID }, id) {
			return
		}
		if input.Publication.Validate() != nil {
			writeSearchError(c, websearch.ErrInvalid)
			return
		}
		known := m.services.SearchOrganizations
		for _, allowed := range input.Organizations {
			if !slices.Contains(known, allowed) {
				writeSearchError(c, websearch.ErrInvalid)
				return
			}
		}
		if input.Published {
			state, readErr := service.Credential(ctx, owner)
			if readErr != nil || !state.Enabled || !state.KeyPresent {
				writeSearchError(c, websearch.ErrConflict)
				return
			}
		}
		value, err = service.Repo.SavePublication(ctx, actor, key, input.Publication)
	case "publish-receipt":
		value, err = service.Repo.PublicationReceipt(ctx, actor, key)
	case "platform-key", "owned-key", "platform-key-receipt", "owned-key-receipt":
		var state credential.Metadata
		if write {
			var input struct {
				Revision         int64  `json:"revision"`
				APIKey           string `json:"apiKey,omitempty"`
				Enabled          bool   `json:"enabled"`
				ExpectedMemberID string `json:"expectedMemberId"`
			}
			if !platformDecode(c, &input, func() string { return input.ExpectedMemberID }, id) {
				return
			}
			state, err = service.SaveCredential(ctx, owner, actor, key, input.Revision, input.APIKey, input.Enabled)
		} else if key != "" {
			state, err = service.CredentialReceipt(ctx, owner, actor, key)
		} else {
			state, err = service.Credential(ctx, owner)
		}
		var updatedAt *time.Time
		if err == nil && state.Revision > 0 {
			at, readErr := service.Writer.SearchCredentialVersionTime(ctx, owner, state.Revision)
			if readErr != nil {
				// A write may already have committed. The client can recover
				// the original receipt after the store becomes available.
				err = readErr
			} else {
				updatedAt = &at
			}
		}
		value = gin.H{"revision": state.Revision, "enabled": state.Enabled, "keyPresent": state.KeyPresent, "updatedAt": updatedAt}
	}
	if err != nil {
		writeSearchError(c, err)
		return
	}
	out["result"] = value
	workbenchReply(c, 200, out)
}

func (a *aiWorkbenchApplication) publicSearch(c *gin.Context, ctx context.Context, scope aiworkbench.Scope) {
	if a.search == nil {
		writeSearchError(c, websearch.ErrUnavailable)
		return
	}
	permission := authz.PermissionWorkbenchChatRead
	if c.Request.Method == http.MethodPost {
		permission = authz.PermissionWorkbenchChatUse
	}
	conversation := c.Param("conversation_id")
	initial, err := a.scopeIdentity(ctx, scope, permission)
	if err != nil || c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
		writeSearchError(c, errModelRouteForbidden)
		return
	}
	identity, _ := authidentity.AuthenticatedIdentityFromContext(initial)
	original := websearch.Scope{Organization: scope.OrganizationID, Actor: scope.ActorID, Member: identity.EffectiveMemberID, Conversation: conversation}
	authorize := func(ctx context.Context) error {
		fresh, err := a.scopeIdentity(ctx, scope, permission)
		if err != nil {
			return errModelRouteForbidden
		}
		live, _ := authidentity.AuthenticatedIdentityFromContext(fresh)
		if live.EffectiveMemberID != original.Member || a.store.ConversationMember(fresh, scope, conversation, original.Member) != nil {
			return errModelRouteForbidden
		}
		row, err := a.store.Get(fresh, scope, conversation)
		if err != nil || (permission == authz.PermissionWorkbenchChatUse && row.Archived) || (row.BusinessScope == nil || row.BusinessScope.Kind != "general") {
			return websearch.ErrNotFound
		}
		if _, allowed := a.authorizedConversation(fresh, scope, row); !allowed {
			return websearch.ErrNotFound
		}
		return nil
	}
	if err := authorize(initial); err != nil {
		writeSearchError(c, err)
		return
	}
	out := gin.H{"organizationId": original.Organization, "actorId": original.Actor, "memberId": original.Member, "conversationId": conversation}
	if c.Request.Method == http.MethodGet {
		if key := c.Param("key"); key != "" {
			if !acquisitionHTTPUUID(key) {
				writeSearchError(c, websearch.ErrInvalid)
				return
			}
			out["result"], err = a.search.Repo.Find(initial, original, key, "")
		} else {
			var option websearch.Option
			_, option, err = a.search.Current(initial, original.Organization)
			if errors.Is(err, websearch.ErrNotFound) {
				err = nil
			}
			if err == nil {
				out["option"] = option
				out["result"], err = a.search.Repo.Recent(initial, original)
			}
		}
	} else {
		var input struct {
			Query            string `json:"query"`
			Preview          string `json:"preview"`
			ExpectedMemberID string `json:"expectedMemberId"`
		}
		key, keyErr := workbenchKey(c)
		if keyErr != nil || workbenchJSONLimit(c, &input, 4096) != nil {
			writeSearchError(c, websearch.ErrInvalid)
			return
		}
		if input.ExpectedMemberID != original.Member {
			writeSearchError(c, websearch.ErrConflict)
			return
		}
		out["result"], err = a.search.Submit(initial, original, key, input.Query, input.Preview, authorize)
	}
	if err == nil {
		err = authorize(initial)
		if err != nil && c.Request.Method == http.MethodPost {
			err = websearch.ErrOutcomeUnknown
		}
	}
	if err != nil {
		writeSearchError(c, err)
		return
	}
	workbenchReply(c, 200, out)
}
