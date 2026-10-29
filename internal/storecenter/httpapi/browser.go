package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	"task-processor/internal/storecenter"
)

type BrowserOperationService interface {
	Admit(context.Context, string, string, string, storecenter.BrowserCommand) (storecenter.BrowserReceipt, error)
	Receipt(context.Context, string, string) (storecenter.BrowserReceipt, error)
	Report(context.Context, string, string, storecenter.BrowserReport) (storecenter.BrowserReceipt, error)
	Audit(context.Context, string, string, storecenter.BrowserAuditQuery) (storecenter.BrowserAuditPage, error)
}

func (h *Handler) SetBrowserOperations(service BrowserOperationService) error {
	if isNilInterface(service) || h.browser != nil {
		return storecenter.ErrDependencyUnavailable
	}
	h.browser = service
	return nil
}
func (h *Handler) browserRoutes() []httproute.Descriptor {
	makeRoute := func(method, path, permission string, policy httproute.OrganizationAccessPolicy, fn gin.HandlerFunc) httproute.Descriptor {
		return httproute.Descriptor{Method: method, Path: "/api/v1/workbench/stores/:store_id" + path, Module: ModuleName, Permission: permission, AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: policy, RejectUnreadRequestBody: method == http.MethodGet, RequestTimeout: 15 * time.Second, Handler: httproute.WithRequestBodyReadTimeout(15*time.Second, fn)}
	}
	return []httproute.Descriptor{
		makeRoute(http.MethodPost, "/browser-operations", authz.PermissionWorkbenchStoreBrowserUse, httproute.OrganizationAccessPolicyLiveWrite, h.browserAdmit),
		makeRoute(http.MethodGet, "/browser-operations/:operation_id", "", httproute.OrganizationAccessPolicyNone, h.browserReceipt),
		makeRoute(http.MethodPost, "/browser-operations/:operation_id/reports", "", httproute.OrganizationAccessPolicyNone, h.browserReport),
		makeRoute(http.MethodGet, "/audit", authz.PermissionWorkbenchStoreRead, httproute.OrganizationAccessPolicyLiveWrite, h.browserAudit),
	}
}
func browserDecode(c *gin.Context, value any) error {
	if c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
		return storecenter.ErrBrowserInvalid
	}
	raw, err := readBoundedBody(c.Request.Body)
	if err != nil {
		return storecenter.ErrBrowserInvalid
	}
	if err = uniqueJSON(json.NewDecoder(bytes.NewReader(raw))); err != nil {
		return storecenter.ErrBrowserInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(value); err != nil {
		return storecenter.ErrBrowserInvalid
	}
	if decoder.Decode(new(any)) != io.EOF {
		return storecenter.ErrBrowserInvalid
	}
	return nil
}
func browserFailure(c *gin.Context, err error) {
	switch {
	case errors.Is(err, storecenter.ErrBrowserInvalid):
		writeInvalid(c, "request", "invalid")
	case errors.Is(err, storecenter.ErrBrowserConflict):
		writeError(c, 409, "OPERATION_CONFLICT", "Operation conflicts with its original request", nil)
	case errors.Is(err, storecenter.ErrBrowserDenied):
		writeError(c, 403, "PERMISSION_DENIED", "Permission denied", nil)
	case errors.Is(err, storecenter.ErrNotFound):
		writeError(c, 404, "STORE_OR_OPERATION_NOT_FOUND", "Store or operation unavailable", nil)
	case errors.Is(err, storecenter.ErrInvalidTransition):
		writeError(c, 403, "PERMISSION_DENIED", "Store unavailable for this operation", nil)
	default:
		writeError(c, 503, "AUDIT_UNAVAILABLE", "Store operation unavailable; query the original result before further action", nil)
	}
}
func (h *Handler) browserAdmit(c *gin.Context) {
	id, ok := authoritativeIdentity(c)
	if !ok {
		return
	}
	key, err := requiredCanonicalUUIDHeader(c.Request, "Idempotency-Key")
	if err != nil {
		writeInvalid(c, "Idempotency-Key", "invalid")
		return
	}
	var command storecenter.BrowserCommand
	if err = browserDecode(c, &command); err != nil {
		browserFailure(c, err)
		return
	}
	receipt, err := h.browser.Admit(c.Request.Context(), id.EffectiveOrganizationID, c.Param("store_id"), key, command)
	if err != nil {
		browserFailure(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	status := http.StatusOK
	if receipt.ExecutionAuthorized {
		status = http.StatusCreated
	}
	c.JSON(status, receipt)
}
func browserReceiptRead(c *gin.Context) bool {
	if _, ok := authidentity.AuthenticatedIdentityFromContext(c.Request.Context()); !ok {
		writeError(c, 401, "AUTHENTICATION_REQUIRED", "Authentication required", nil)
		return false
	}
	if c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
		writeInvalid(c, "query", "invalid")
		return false
	}
	if c.Request.Body != nil {
		raw, err := io.ReadAll(io.LimitReader(c.Request.Body, 1))
		if err != nil || len(raw) > 0 {
			writeInvalid(c, "body", "invalid")
			return false
		}
	}
	return true
}
func (h *Handler) browserReceipt(c *gin.Context) {
	if !browserReceiptRead(c) {
		return
	}
	receipt, err := h.browser.Receipt(c.Request.Context(), c.Param("store_id"), c.Param("operation_id"))
	if err != nil {
		browserFailure(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(200, receipt)
}
func (h *Handler) browserReport(c *gin.Context) {
	var report storecenter.BrowserReport
	if err := browserDecode(c, &report); err != nil {
		browserFailure(c, err)
		return
	}
	receipt, err := h.browser.Report(c.Request.Context(), c.Param("store_id"), c.Param("operation_id"), report)
	if err != nil {
		browserFailure(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(200, receipt)
}
func (h *Handler) browserAudit(c *gin.Context) {
	id, ok := authoritativeIdentity(c)
	if !ok {
		return
	}
	values, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil || c.Request.URL.ForceQuery {
		writeInvalid(c, "query", "invalid")
		return
	}
	allowed := map[string]bool{"limit": true, "cursor": true, "actor": true, "action": true, "outcome": true, "from": true, "to": true}
	for key, v := range values {
		if !allowed[key] || len(v) != 1 {
			writeInvalid(c, "query", "invalid")
			return
		}
	}
	q := storecenter.BrowserAuditQuery{Limit: 20, Cursor: values.Get("cursor"), Actor: values.Get("actor"), Action: values.Get("action"), Outcome: values.Get("outcome"), From: values.Get("from"), To: values.Get("to")}
	if values.Has("limit") {
		q.Limit, err = strconv.Atoi(values.Get("limit"))
		if err != nil || q.Limit < 1 || q.Limit > 100 || strconv.Itoa(q.Limit) != values.Get("limit") {
			writeInvalid(c, "limit", "invalid")
			return
		}
	}
	result, err := h.browser.Audit(c.Request.Context(), id.EffectiveOrganizationID, c.Param("store_id"), q)
	if err != nil {
		browserFailure(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(200, result)
}
