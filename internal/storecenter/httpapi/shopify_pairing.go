package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

const (
	ShopifyPairingPath    = "/api/v1/workbench/stores/:store_id/shopify-pairing"
	ShopifyPairingTimeout = 25 * time.Second
)

type ShopifyConnectionService interface {
	Begin(context.Context, storecenter.OfficialConnectionCommand, string, string) (storecenter.OfficialConnectionBegin, error)
	Launch(context.Context, storecenter.ShopifyPairingKey, string, string, string) (string, error)
	Stage(context.Context, storecenter.ShopifyPairingKey, string, string, string) (string, error)
	Confirm(context.Context, storecenter.ShopifyPairingKey, string) (storecenter.ShopifyBinding, error)
}

func (h *Handler) SetShopifyConnections(service ShopifyConnectionService) error {
	if isNilInterface(service) || h.shopifyConnections != nil {
		return errors.New("Shopify Store connection service unavailable or already supplied")
	}
	h.shopifyConnections = service
	return nil
}

func (h *Handler) ShopifyPairing(c *gin.Context) {
	identity, storeID, ok := h.itemIdentity(c)
	if !ok {
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	values, field, err := parseStringObject(c.Request.Body, map[string]bool{
		"action": true, "browser": true, "state": false, "attemptId": false, "query": false,
	})
	if err != nil || len(values["browser"]) != 43 || len(values["query"]) > 8192 {
		writeInvalid(c, field, "invalid")
		return
	}
	ctx := c.Request.Context()
	if values["action"] == "begin" {
		h.beginShopify(c, values)
		return
	}
	if values["action"] == "confirm_install" {
		h.confirmShopifyInstallation(c, values)
		return
	}
	attempt, err := canonicalUUID(values["attemptId"])
	if err != nil || len(c.Request.Header.Values("If-Match")) != 0 || len(c.Request.Header.Values("Idempotency-Key")) != 0 {
		writeInvalid(c, "attemptId", "invalid")
		return
	}
	key := storecenter.ShopifyPairingKey{OrganizationID: identity.EffectiveOrganizationID, StoreID: storeID, AttemptID: attempt}
	switch values["action"] {
	case "inspect_install":
		h.inspectShopifyInstallation(c, key, values)
	case "launch", "stage":
		if len(values["state"]) != 43 || values["query"] == "" {
			writeInvalid(c, "query", "invalid")
			return
		}
		if values["action"] == "launch" {
			target, err := h.shopifyConnections.Launch(ctx, key, values["browser"], values["state"], values["query"])
			if err != nil {
				writeStoreError(c, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"authorizationUrl": target})
			return
		}
		domain, err := h.shopifyConnections.Stage(ctx, key, values["browser"], values["state"], values["query"])
		if err != nil {
			writeShopifyInstallationError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"domain": domain})
	case "confirm":
		if values["state"] != "" || values["query"] != "" {
			writeInvalid(c, "body", "invalid")
			return
		}
		binding, err := h.shopifyConnections.Confirm(ctx, key, values["browser"])
		if err != nil {
			writeStoreError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"domain": binding.Domain})
	default:
		writeInvalid(c, "action", "invalid")
	}
}

func (h *Handler) beginShopify(c *gin.Context, values map[string]string) {
	if len(values["state"]) != 43 || values["attemptId"] != "" || values["query"] != "" {
		writeInvalid(c, "body", "invalid")
		return
	}
	command, ok := h.connectionCommand(c, false)
	if !ok {
		return
	}
	start, err := h.shopifyConnections.Begin(c.Request.Context(), command, values["browser"], values["state"])
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, start)
}

func (h *Handler) confirmShopifyInstallation(c *gin.Context, values map[string]string) {
	service, ok := h.shopifyConnections.(shopifyStartedInstallationService)
	if !ok || !service.StartedInstallationEnabled() {
		c.Status(http.StatusNotFound)
		return
	}
	if values["attemptId"] != "" || values["state"] != "" || values["query"] != "" {
		writeInvalid(c, "body", "invalid")
		return
	}
	command, ok := h.connectionCommand(c, false)
	if !ok {
		return
	}
	binding, err := service.ConfirmStartedInstallation(c.Request.Context(), command, values["browser"])
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"domain": binding.Domain})
}

func (h *Handler) inspectShopifyInstallation(c *gin.Context, key storecenter.ShopifyPairingKey, values map[string]string) {
	service, ok := h.shopifyConnections.(shopifyStartedInstallationService)
	if !ok || !service.StartedInstallationEnabled() {
		c.Status(http.StatusNotFound)
		return
	}
	if values["state"] != "" || values["query"] != "" {
		writeInvalid(c, "body", "invalid")
		return
	}
	view, err := service.InspectStartedInstallation(c.Request.Context(), key, values["browser"])
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}
