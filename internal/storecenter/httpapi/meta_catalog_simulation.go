package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	meta "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/metacatalog"
)

const MetaStorePath = "/api/v1/workbench/stores/:store_id/connection/meta-catalog"

// MetaCatalogSimulationRoutes are mounted only by the isolated fixture.
func MetaCatalogSimulationRoutes(h *Handler, service *storecenter.MetaCatalogSimulation) []httproute.Descriptor {
	if h == nil || service == nil {
		return nil
	}
	suffixes := []string{"", "/begin", "/complete", "/choices", "/bind", "/disconnect"}
	routes := make([]httproute.Descriptor, 0, len(suffixes))
	for _, suffix := range suffixes {
		method, permission := http.MethodPost, authz.PermissionWorkbenchStoreUpdate
		if suffix == "" {
			method, permission = http.MethodGet, authz.PermissionWorkbenchStoreRead
		}
		routes = append(routes, route(method, MetaStorePath+suffix, permission, httproute.OrganizationAccessPolicyLiveWrite, func(c *gin.Context) {
			handleMetaConnection(c, h, service, suffix)
		}))
	}
	return routes
}

type metaConnectionInput struct {
	Version    int64  `json:"version"`
	AttemptID  string `json:"attemptId"`
	State      string `json:"state"`
	Code       string `json:"code"`
	CatalogID  string `json:"catalogId"`
	WhatsAppID string `json:"whatsAppId"`
}

func (v metaConnectionInput) valid(action string) bool {
	if action == "/begin" || action == "/disconnect" {
		return v.Version > 0 && v == (metaConnectionInput{Version: v.Version})
	}
	if _, err := canonicalUUID(v.AttemptID); err != nil {
		return false
	}
	allowed := metaConnectionInput{AttemptID: v.AttemptID}
	switch action {
	case "/complete":
		if len(v.State) != 43 || len(v.Code) < 1 || len(v.Code) > 1024 {
			return false
		}
		allowed.State, allowed.Code = v.State, v.Code
	case "/choices":
	case "/bind":
		if !meta.ValidID(v.CatalogID) || v.WhatsAppID != "" && !meta.ValidID(v.WhatsAppID) {
			return false
		}
		allowed.CatalogID, allowed.WhatsAppID = v.CatalogID, v.WhatsAppID
	default:
		return false
	}
	return v == allowed
}

func handleMetaConnection(c *gin.Context, h *Handler, service *storecenter.MetaCatalogSimulation, action string) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
	identity, id, ok := h.itemIdentity(c)
	if !ok {
		return
	}
	if c.Request.URL.RawQuery != "" {
		writeInvalid(c, "query", "invalid")
		return
	}
	var input metaConnectionInput
	if action != "" {
		reader := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, requestBodyMaxBytes))
		reader.DisallowUnknownFields()
		if reader.Decode(&input) != nil || reader.Decode(new(any)) != io.EOF || !input.valid(action) {
			writeInvalid(c, "body", "invalid")
			return
		}
	}
	ctx, org := c.Request.Context(), identity.EffectiveOrganizationID
	var result any = gin.H{"ok": true}
	var err error
	switch action {
	case "":
		result, err = service.Read(ctx, org, id)
	case "/begin":
		result, err = service.Begin(ctx, org, id, input.Version)
	case "/complete":
		err = service.Complete(ctx, org, id, input.AttemptID, input.State, input.Code)
	case "/choices":
		result, err = service.Choices(ctx, org, id, input.AttemptID)
	case "/bind":
		result, err = service.Bind(ctx, org, id, input.AttemptID, input.CatalogID, input.WhatsAppID)
	case "/disconnect":
		err = service.Disconnect(ctx, org, id, input.Version)
	}
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
