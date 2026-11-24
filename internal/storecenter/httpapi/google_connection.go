package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	market "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
)

type GoogleConnectionService interface {
	Read(context.Context, string, string) (storecenter.GoogleConnectionView, error)
	Begin(context.Context, string, string, int64) (storecenter.OfficialConnectionBegin, error)
	Complete(context.Context, string, string, string, string, string) error
	Accounts(context.Context, string, string, string, string) (market.AccountPage, error)
	Sources(context.Context, string, string, string, string, string) (market.SourcePage, error)
	Bind(context.Context, string, string, string, storecenter.GoogleAccountSelection) (storecenter.GoogleConnectionView, error)
	Disconnect(context.Context, string, string, int64) error
}

const GoogleStorePath = "/api/v1/workbench/stores/:store_id/connection/google"

var googleStoreMethods = map[string]string{
	"": http.MethodGet, "/begin": http.MethodPost, "/complete": http.MethodPost,
	"/accounts": http.MethodPost, "/sources": http.MethodPost,
	"/bind": http.MethodPost, "/disconnect": http.MethodPost,
}

func IsGoogleStoreDescriptor(method, path, permission string) bool {
	for suffix, expected := range googleStoreMethods {
		wantPermission := authz.PermissionWorkbenchStoreUpdate
		if expected == http.MethodGet {
			wantPermission = authz.PermissionWorkbenchStoreRead
		}
		if path == GoogleStorePath+suffix && method == expected && permission == wantPermission {
			return true
		}
	}
	return false
}

func (h *Handler) SetGoogleConnections(service GoogleConnectionService) error {
	if isNilInterface(service) || h.google != nil {
		return errors.New("Google Merchant connection service unavailable or already supplied")
	}
	h.google = service
	return nil
}

// GoogleStoreRoutes describes the bounded Google Merchant routes. Current
// application admission uses the same definitions without a serving handler.
func GoogleStoreRoutes(h *Handler) []httproute.Descriptor {
	result := make([]httproute.Descriptor, 0, len(googleStoreMethods))
	for suffix, method := range googleStoreMethods {
		permission := authz.PermissionWorkbenchStoreUpdate
		if method == http.MethodGet {
			permission = authz.PermissionWorkbenchStoreRead
		}
		result = append(result, route(method, GoogleStorePath+suffix, permission, httproute.OrganizationAccessPolicyLiveWrite, func(c *gin.Context) {
			h.googleConnection(c, suffix)
		}))
	}
	return result
}

type googleConnectionInput struct {
	Version                  int64  `json:"version"`
	AttemptID                string `json:"attemptId"`
	State                    string `json:"state"`
	Code                     string `json:"code"`
	AccountID                string `json:"accountId"`
	DataSourceID             string `json:"dataSourceId"`
	ContentLanguage          string `json:"contentLanguage"`
	FeedLabel                string `json:"feedLabel"`
	Page                     string `json:"page"`
	DedicatedSourceConfirmed bool   `json:"dedicatedSourceConfirmed"`
}

func (h *Handler) googleConnection(c *gin.Context, action string) {
	identity, storeID, ok := h.itemIdentity(c)
	if !ok {
		return
	}
	if c.Request.URL.RawQuery != "" {
		writeInvalid(c, "query", "invalid")
		return
	}
	ctx, org := c.Request.Context(), identity.EffectiveOrganizationID
	var input googleConnectionInput
	if action != "" {
		reader := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, requestBodyMaxBytes))
		reader.DisallowUnknownFields()
		if reader.Decode(&input) != nil || reader.Decode(new(any)) != io.EOF || !input.valid(action) {
			writeInvalid(c, "body", "invalid")
			return
		}
	}
	var response any = gin.H{"ok": true}
	var err error
	switch action {
	case "":
		response, err = h.google.Read(ctx, org, storeID)
	case "/begin":
		response, err = h.google.Begin(ctx, org, storeID, input.Version)
	case "/complete":
		err = h.google.Complete(ctx, org, storeID, input.AttemptID, input.State, input.Code)
	case "/accounts":
		response, err = h.google.Accounts(ctx, org, storeID, input.AttemptID, input.Page)
	case "/sources":
		response, err = h.google.Sources(ctx, org, storeID, input.AttemptID, input.AccountID, input.Page)
	case "/bind":
		response, err = h.google.Bind(ctx, org, storeID, input.AttemptID, storecenter.GoogleAccountSelection{
			AccountID: input.AccountID, DataSourceID: input.DataSourceID, ContentLanguage: input.ContentLanguage,
			FeedLabel: input.FeedLabel, DedicatedSourceConfirmed: input.DedicatedSourceConfirmed,
		})
	case "/disconnect":
		err = h.google.Disconnect(ctx, org, storeID, input.Version)
	}
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.JSON(http.StatusOK, response)
}

func (v googleConnectionInput) valid(action string) bool {
	if action == "/begin" || action == "/disconnect" {
		return v.Version > 0 && v == (googleConnectionInput{Version: v.Version})
	}
	if _, err := canonicalUUID(v.AttemptID); err != nil {
		return false
	}
	allowed := googleConnectionInput{AttemptID: v.AttemptID}
	switch action {
	case "/complete":
		if len(v.State) != 43 || v.Code == "" || len(v.Code) > 8192 {
			return false
		}
		allowed.State, allowed.Code = v.State, v.Code
	case "/accounts":
		allowed.Page = v.Page
	case "/sources":
		allowed.AccountID, allowed.Page = v.AccountID, v.Page
		if !market.ValidResourceID(v.AccountID) {
			return false
		}
	case "/bind":
		if !market.ValidResourceID(v.AccountID) || !market.ValidResourceID(v.DataSourceID) || !v.DedicatedSourceConfirmed {
			return false
		}
		allowed.AccountID, allowed.DataSourceID = v.AccountID, v.DataSourceID
		allowed.ContentLanguage, allowed.FeedLabel, allowed.DedicatedSourceConfirmed = v.ContentLanguage, v.FeedLabel, true
	default:
		return false
	}
	return len(v.Page) <= 4096 && v == allowed
}
