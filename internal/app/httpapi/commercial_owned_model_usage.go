package httpapi

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"task-processor/internal/aicapability"
	"task-processor/internal/authidentity"
)

const ownedUsagePageSize = 30

var errOwnedUsageForbidden = errors.New("owned model usage access denied")
var errOwnedUsageCursor = errors.New("invalid owned model usage cursor")

type ownedModelUsageItem struct {
	InvocationID string                 `json:"invocation_id"`
	StartedAt    time.Time              `json:"started_at"`
	Purpose      aicapability.Operation `json:"purpose"`
	Model        string                 `json:"model"`
	Status       string                 `json:"status"`
	InputTokens  *int                   `json:"input_tokens,omitempty"`
	OutputTokens *int                   `json:"output_tokens,omitempty"`
}

type ownedModelUsagePage struct {
	OrganizationID string                `json:"organization_id"`
	Items          []ownedModelUsageItem `json:"items"`
	NextCursor     *string               `json:"next_cursor"`
}

type ownedModelUsageReader interface {
	ListOwnedModelUsage(context.Context, string, string, string, time.Time, string, int) ([]aicapability.InvocationRecord, error)
}

func (m commercialResourcesModule) readOwnedModelUsage(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	if !readCommercialEmptyBody(c) {
		return
	}
	i, ok := authidentity.AuthenticatedIdentityFromContext(c.Request.Context())
	if !ok || !authidentity.IsBoundedIdentifier(i.UserID) {
		writeWorkbenchProtocolError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication is required")
		return
	}
	if !authidentity.IsBoundedIdentifier(i.EffectiveOrganizationID) || i.TenantID != i.EffectiveOrganizationID {
		writeWorkbenchProtocolError(c, http.StatusForbidden, "ORGANIZATION_ACCESS_DENIED", "Organization access is denied")
		return
	}
	query, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil || c.Request.URL.ForceQuery ||
		(len(query) > 0 && (len(query) != 1 || len(query["cursor"]) != 1 || query.Get("cursor") == "")) {
		writeWorkbenchProtocolError(c, http.StatusBadRequest, "INVALID_REQUEST", "Usage cursor is invalid")
		return
	}
	if m.objects == nil || m.objects.ownedModels == nil {
		writeWorkbenchProtocolError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "Model usage is unavailable")
		return
	}
	page, err := m.objects.ownedModels(c, i.EffectiveOrganizationID, query.Get("cursor"))
	switch {
	case errors.Is(err, errOwnedUsageForbidden):
		writeWorkbenchProtocolError(c, http.StatusForbidden, "PERMISSION_DENIED", "Model usage access is denied")
	case errors.Is(err, errOwnedUsageCursor):
		writeWorkbenchProtocolError(c, http.StatusBadRequest, "INVALID_REQUEST", "Usage cursor is invalid")
	case err != nil || page.OrganizationID != i.EffectiveOrganizationID:
		writeWorkbenchProtocolError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "Model usage is unavailable")
	default:
		c.JSON(http.StatusOK, page)
	}
}

func (a *aiWorkbenchApplication) ownedModelUsage(c *gin.Context, org, cursor string) (ownedModelUsagePage, error) {
	ctx, i, ok := a.bindUsageModel(c, org)
	if !ok {
		return ownedModelUsagePage{}, errOwnedUsageForbidden
	}
	var at time.Time
	var id string
	if cursor != "" {
		if len(cursor) > 240 {
			return ownedModelUsagePage{}, errOwnedUsageCursor
		}
		raw, err := base64.RawURLEncoding.Strict().DecodeString(cursor)
		if err != nil {
			return ownedModelUsagePage{}, errOwnedUsageCursor
		}
		value, identifier, found := strings.Cut(string(raw), "\n")
		at, err = time.Parse(time.RFC3339Nano, value)
		if !found || err != nil || at.IsZero() || !authidentity.IsBoundedIdentifier(identifier) {
			return ownedModelUsagePage{}, errOwnedUsageCursor
		}
		id = identifier
	}
	reader, ok := a.agent.config.Ledger.(ownedModelUsageReader)
	if !ok {
		return ownedModelUsagePage{}, errors.New("owned model usage reader unavailable")
	}
	facts, err := reader.ListOwnedModelUsage(ctx, org, i.UserID, i.EffectiveMemberID, at, id, ownedUsagePageSize+1)
	if err != nil {
		return ownedModelUsagePage{}, err
	}
	page := ownedModelUsagePage{OrganizationID: org, Items: make([]ownedModelUsageItem, 0, min(len(facts), ownedUsagePageSize))}
	for idx, fact := range facts {
		if !usageInvocationOwnedBy(fact, i) || fact.BillingMode != aicapability.ModelUsageOnly {
			return ownedModelUsagePage{}, errors.New("owned model usage scope differs")
		}
		if idx == ownedUsagePageSize {
			last := facts[idx-1]
			next := base64.RawURLEncoding.EncodeToString([]byte(last.StartedAt.UTC().Format(time.RFC3339Nano) + "\n" + last.InvocationID))
			page.NextCursor = &next
			break
		}
		item := ownedModelUsageItem{InvocationID: fact.InvocationID, StartedAt: fact.StartedAt.UTC(), Purpose: fact.Operation, Model: fact.ModelID, Status: "unknown"}
		if fact.UsageKnown {
			item.Status, item.InputTokens, item.OutputTokens = "observed", &fact.PromptTokens, &fact.CompletionTokens
		} else if fact.NotSent() {
			item.Status = "not_sent"
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}
