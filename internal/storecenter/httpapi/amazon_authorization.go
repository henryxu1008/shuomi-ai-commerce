package httpapi

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

const (
	amazonFlowCookieName = "__Secure-shuomi-amazon-flow"
	amazonFlowCookiePath = "/workbench/stores/amazon"
	amazonBeginPath      = "/api/v1/workbench/stores/:store_id/amazon-authorization/begin"
	amazonLoginPath      = "/api/v1/workbench/store-amazon-authorization/login"
	amazonRedirectPath   = "/api/v1/workbench/store-amazon-authorization/redirect"
)

type AmazonAuthorizationService interface {
	Begin(context.Context, storecenter.AmazonBrowserScope, string, int64) (storecenter.AmazonConnectionStart, error)
	Continue(context.Context, storecenter.AmazonBrowserScope, storecenter.AmazonLoginCallback) (string, error)
	Complete(context.Context, storecenter.AmazonBrowserScope, storecenter.AmazonRedirectCallback) (storecenter.AmazonConnectionResult, error)
}

type amazonAuthorizationHTTP struct {
	service      AmazonAuthorizationService
	sessionProof func(context.Context) [32]byte
}

func (h *Handler) SetAmazonAuthorizations(service AmazonAuthorizationService, sessionProof func(context.Context) [32]byte) error {
	if isNilInterface(service) || sessionProof == nil || h.amazonAuthorization != nil {
		return errors.New("Amazon Store authorization unavailable or already supplied")
	}
	h.amazonAuthorization = &amazonAuthorizationHTTP{service: service, sessionProof: sessionProof}
	return nil
}

func (h *Handler) amazonAuthorizationRoutes() []httproute.Descriptor {
	if h.amazonAuthorization == nil {
		return nil
	}
	return []httproute.Descriptor{
		amazonAuthorizationRoute(amazonBeginPath, h.beginAmazonAuthorization),
		amazonAuthorizationRoute(amazonLoginPath, h.continueAmazonAuthorization),
		amazonAuthorizationRoute(amazonRedirectPath, h.completeAmazonAuthorization),
	}
}

func amazonAuthorizationRoute(path string, handle func(*gin.Context)) httproute.Descriptor {
	descriptor := route(http.MethodPost, path, authz.PermissionWorkbenchStoreUpdate, httproute.OrganizationAccessPolicyLiveWrite, handle)
	// Code exchange and seller validation use multiple bounded provider reads.
	descriptor.RequestTimeout = 35 * time.Second
	descriptor.RejectUnreadRequestBody = true
	return descriptor
}

func AmazonAuthorizationPaths() [3]string {
	return [3]string{amazonBeginPath, amazonLoginPath, amazonRedirectPath}
}

// IsAmazonAuthorizationPath lets the current application validate explicitly
// configured descriptors without making Amazon routes mandatory by default.
func IsAmazonAuthorizationPath(path string) bool {
	return path == amazonBeginPath || path == amazonLoginPath || path == amazonRedirectPath
}

func (h *Handler) beginAmazonAuthorization(c *gin.Context) {
	amazonAuthorizationHeaders(c)
	identity, storeID, ok := h.itemIdentity(c)
	if !ok {
		return
	}
	scope, ok := h.amazonScope(c, identity.EffectiveOrganizationID)
	if !ok {
		return
	}
	version, err := requiredIfMatch(c.Request)
	if err != nil {
		writeInvalid(c, "If-Match", "invalid")
		return
	}
	if err := requireNoBody(c.Request.Body); err != nil {
		writeInvalid(c, "body", "not_allowed")
		return
	}
	result, err := h.amazonAuthorization.service.Begin(c.Request.Context(), scope, storeID, version)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	lifetime := time.Until(result.ExpiresAt)
	if !canonicalAmazonFlowCookie(result.FlowCookie) || lifetime < time.Second || lifetime > 9*time.Minute {
		writeStoreError(c, storecenter.ErrOfficialConnectionUnavailable)
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name: amazonFlowCookieName, Value: result.FlowCookie, Path: amazonFlowCookiePath,
		Expires: result.ExpiresAt.UTC(), MaxAge: int(lifetime / time.Second),
		Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	c.JSON(http.StatusOK, result)
}

func (h *Handler) continueAmazonAuthorization(c *gin.Context) {
	scope, flow, data, ok := h.amazonCallback(c, map[string]bool{"callbackUri": true, "amazonState": true, "sellerId": true, "version": false})
	if !ok {
		return
	}
	target, err := h.amazonAuthorization.service.Continue(c.Request.Context(), scope, storecenter.AmazonLoginCallback{
		FlowCookie: flow, AmazonCallbackURI: data["callbackUri"], AmazonState: data["amazonState"], SellerID: data["sellerId"], Version: data["version"],
	})
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"authorizationUrl": target})
}

func (h *Handler) completeAmazonAuthorization(c *gin.Context) {
	scope, flow, data, ok := h.amazonCallback(c, map[string]bool{"state": true, "sellerId": true, "code": true})
	if !ok {
		return
	}
	result, err := h.amazonAuthorization.service.Complete(c.Request.Context(), scope, storecenter.AmazonRedirectCallback{
		FlowCookie: flow, State: data["state"], SellerID: data["sellerId"], Code: data["code"],
	})
	if err != nil {
		writeStoreError(c, err)
		return
	}
	// Keep the short-lived flow cookie for an exact successful replay if the
	// browser lost this response. It expires along with the stored attempt.
	c.JSON(http.StatusOK, result)
}

func (h *Handler) amazonCallback(c *gin.Context, fields map[string]bool) (storecenter.AmazonBrowserScope, string, map[string]string, bool) {
	amazonAuthorizationHeaders(c)
	empty := storecenter.AmazonBrowserScope{}
	identity, ok := authoritativeIdentity(c)
	if !ok {
		return empty, "", nil, false
	}
	scope, ok := h.amazonScope(c, identity.EffectiveOrganizationID)
	if !ok {
		return empty, "", nil, false
	}
	if err := requireNoQuery(c.Request.URL.RawQuery); err != nil {
		writeInvalid(c, "query", "not_allowed")
		return empty, "", nil, false
	}
	cookies := c.Request.CookiesNamed(amazonFlowCookieName)
	if len(cookies) != 1 || !canonicalAmazonFlowCookie(cookies[0].Value) {
		writeInvalid(c, "authorization", "invalid")
		return empty, "", nil, false
	}
	if c.ContentType() != "application/json" {
		writeInvalid(c, "content-type", "invalid")
		return empty, "", nil, false
	}
	data, _, err := parseStringObject(c.Request.Body, fields)
	if err != nil {
		// Field names are untrusted too. Never echo arbitrary callback keys.
		writeInvalid(c, "body", "invalid")
		return empty, "", nil, false
	}
	return scope, cookies[0].Value, data, true
}

func (h *Handler) amazonScope(c *gin.Context, organization string) (storecenter.AmazonBrowserScope, bool) {
	fingerprint := h.amazonAuthorization.sessionProof(c.Request.Context())
	if fingerprint == [32]byte{} {
		writeError(c, http.StatusUnauthorized, "AUTHENTICATION_REQUIRED", "Authentication is required", nil)
		return storecenter.AmazonBrowserScope{}, false
	}
	return storecenter.AmazonBrowserScope{OrganizationID: organization, SessionDigest: fingerprint}, true
}

func canonicalAmazonFlowCookie(value string) bool {
	if len(value) != base64.RawURLEncoding.EncodedLen(32) {
		return false
	}
	bytes, err := base64.RawURLEncoding.Strict().DecodeString(value)
	return err == nil && len(bytes) == 32
}

func amazonAuthorizationHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
}
