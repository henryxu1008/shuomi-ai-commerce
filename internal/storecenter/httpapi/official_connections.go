package httpapi

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"task-processor/internal/storecenter"
)

func (h *Handler) ReadOfficialConnection(c *gin.Context) {
	identity, id, ok := h.itemIdentity(c)
	if !ok {
		return
	}
	view, err := h.officialConnections.Read(c.Request.Context(), identity.EffectiveOrganizationID, id)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, view)
}
func (h *Handler) BeginOfficialConnection(c *gin.Context) {
	command, ok := h.connectionCommand(c, false)
	if !ok {
		return
	}
	values, field, err := parseStringObject(c.Request.Body, map[string]bool{"appId": true, "identityPartition": false})
	if err != nil || len(values["appId"]) < 1 || len(values["appId"]) > 128 {
		writeInvalid(c, field, "invalid")
		return
	}
	command.ApplicationID = values["appId"]
	command.IdentityPartition = storecenter.OfficialIdentityPartition(values["identityPartition"])
	result, err := h.officialConnections.Begin(c.Request.Context(), command)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.JSON(http.StatusOK, result)
}

func (h *Handler) InstallOfficialToken(c *gin.Context) {
	identity, id, ok := h.itemIdentity(c)
	if !ok {
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	if len(c.Request.Header.Values("If-Match")) != 0 || len(c.Request.Header.Values("Idempotency-Key")) != 0 {
		writeInvalid(c, "headers", "not_allowed")
		return
	}
	values, field, err := parseStringObject(c.Request.Body, map[string]bool{"attemptId": true, "accessToken": true})
	if err != nil {
		writeInvalid(c, field, "invalid")
		return
	}
	attempt, err := canonicalUUID(values["attemptId"])
	if err != nil {
		writeInvalid(c, "attemptId", "invalid")
		return
	}
	if len(values["accessToken"]) < 1 || len(values["accessToken"]) > 4096 {
		writeInvalid(c, "accessToken", "invalid")
		return
	}
	view, err := h.officialConnections.InstallToken(c.Request.Context(), storecenter.InstallOfficialToken{OrganizationID: identity.EffectiveOrganizationID, StoreID: id, AttemptID: attempt, AccessToken: values["accessToken"]})
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}
func (h *Handler) DisconnectOfficialConnection(c *gin.Context) {
	command, ok := h.connectionCommand(c, true)
	if !ok {
		return
	}
	result, err := h.officialConnections.Disconnect(c.Request.Context(), command)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, result)
}
func (h *Handler) connectionCommand(c *gin.Context, noBody bool) (storecenter.OfficialConnectionCommand, bool) {
	identity, id, ok := h.itemIdentity(c)
	if !ok {
		return storecenter.OfficialConnectionCommand{}, false
	}
	version, err := requiredIfMatch(c.Request)
	if err != nil {
		writeInvalid(c, "If-Match", "invalid")
		return storecenter.OfficialConnectionCommand{}, false
	}
	key, err := requiredCanonicalUUIDHeader(c.Request, "Idempotency-Key")
	if err != nil {
		writeInvalid(c, "Idempotency-Key", "invalid")
		return storecenter.OfficialConnectionCommand{}, false
	}
	if noBody {
		if err := requireNoBody(c.Request.Body); err != nil {
			writeInvalid(c, "body", "not_allowed")
			return storecenter.OfficialConnectionCommand{}, false
		}
	}
	return storecenter.OfficialConnectionCommand{OrganizationID: identity.EffectiveOrganizationID, StoreID: id, AttemptID: key, ExpectedStoreVersion: version}, true
}
func (h *Handler) OfficialApplications(c *gin.Context) {
	identity, id, ok := h.itemIdentity(c)
	if !ok {
		return
	}
	choices, err := h.officialConnections.Applications(c.Request.Context(), identity.EffectiveOrganizationID, id)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"applications": choices})
}
func (h *Handler) CompleteOfficialConnection(c *gin.Context) {
	identity, id, ok := h.itemIdentity(c)
	if !ok {
		return
	}
	if len(c.Request.Header.Values("If-Match")) != 0 || len(c.Request.Header.Values("Idempotency-Key")) != 0 {
		writeInvalid(c, "headers", "not_allowed")
		return
	}
	values, field, err := parseStringObject(c.Request.Body, map[string]bool{"attemptId": true, "appId": true, "state": false, "tempToken": false, "authCode": false, "sellerId": false, "type": false, "merchantConfirmation": false})
	if err != nil {
		writeInvalid(c, field, "invalid")
		return
	}
	attempt, err := canonicalUUID(values["attemptId"])
	if err != nil {
		writeInvalid(c, "attemptId", "invalid")
		return
	}
	_, tokenPresent := values["tempToken"]
	_, codePresent := values["authCode"]
	confirmPresent := values["merchantConfirmation"] != ""
	if (!confirmPresent && tokenPresent == codePresent) || confirmPresent && (tokenPresent || codePresent || values["state"] != "" || values["sellerId"] != "" || values["type"] != "") || !confirmPresent && values["state"] == "" || len(values["merchantConfirmation"]) > 200 || len(values["sellerId"]) > 200 || len(values["type"]) > 16 || len(values["appId"]) > 200 || len(values["state"]) > 128 || len(values["tempToken"]) > 4096 || len(values["authCode"]) > 2048 {
		writeInvalid(c, "body", "invalid")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	result, err := h.officialConnections.Complete(c.Request.Context(), storecenter.CompleteOfficialConnection{OrganizationID: identity.EffectiveOrganizationID, StoreID: id, AttemptID: attempt, AppID: values["appId"], State: values["state"], TempToken: values["tempToken"], AuthCode: values["authCode"], SellerID: values["sellerId"], CallbackType: values["type"], MerchantConfirmation: values["merchantConfirmation"]})
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ResumeOfficialConnectionQuery(c *gin.Context) {
	identity, id, ok := h.itemIdentity(c)
	if !ok {
		return
	}
	if len(c.Request.Header.Values("If-Match")) != 0 || len(c.Request.Header.Values("Idempotency-Key")) != 0 {
		writeInvalid(c, "headers", "not_allowed")
		return
	}
	values, field, err := parseStringObject(c.Request.Body, map[string]bool{"attemptId": true})
	if err != nil {
		writeInvalid(c, field, "invalid")
		return
	}
	attempt, err := canonicalUUID(values["attemptId"])
	if err != nil {
		writeInvalid(c, "attemptId", "invalid")
		return
	}
	result, err := h.officialConnections.ResumeQuery(c.Request.Context(), identity.EffectiveOrganizationID, id, attempt)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, result)
}

func (h *Handler) RefreshOfficialConnection(c *gin.Context) {
	identity, id, ok := h.itemIdentity(c)
	if !ok {
		return
	}
	if len(c.Request.Header.Values("If-Match")) != 0 || len(c.Request.Header.Values("Idempotency-Key")) != 0 {
		writeInvalid(c, "headers", "not_allowed")
		return
	}
	values, field, e := parseStringObject(c.Request.Body, map[string]bool{"operationId": true, "generation": true})
	if e != nil {
		writeInvalid(c, field, "invalid")
		return
	}
	op, e := canonicalUUID(values["operationId"])
	if e != nil {
		writeInvalid(c, "operationId", "invalid")
		return
	}
	generation, e := strconv.ParseInt(values["generation"], 10, 64)
	if e != nil || generation < 1 || strconv.FormatInt(generation, 10) != values["generation"] {
		writeInvalid(c, "generation", "invalid")
		return
	}
	service, ok := h.officialConnections.(interface {
		RefreshWalmart(context.Context, string, string, string, int64) (storecenter.OfficialConnectionView, error)
	})
	if !ok {
		writeStoreError(c, storecenter.ErrOfficialConnectionUnavailable)
		return
	}
	result, e := service.RefreshWalmart(c.Request.Context(), identity.EffectiveOrganizationID, id, op, generation)
	if e != nil {
		writeStoreError(c, e)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, result)
}
