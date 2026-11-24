package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

const (
	ShopifyStartedInstallationPath    = "/api/v1/platform/shopify/installation"
	ShopifyStartedInstallationTimeout = 20 * time.Second
)

type shopifyStartedInstallationService interface {
	StartedInstallationEnabled() bool
	BeginStartedInstallation(context.Context, string, string, string, string) (string, error)
	StageStartedInstallation(context.Context, string, string, string, string) (string, error)
	InspectStartedInstallation(context.Context, storecenter.ShopifyPairingKey, string) (storecenter.ShopifyInstallSelection, error)
	ConfirmStartedInstallation(context.Context, storecenter.OfficialConnectionCommand, string) (storecenter.ShopifyBinding, error)
	ExpireStartedInstallationProofs(context.Context) error
}

func (h *Handler) shopifyStartedInstallationRoute() *httproute.Descriptor {
	service, ok := h.shopifyConnections.(shopifyStartedInstallationService)
	if !ok || !service.StartedInstallationEnabled() {
		return nil
	}
	return &httproute.Descriptor{
		Method: http.MethodPost, Path: ShopifyStartedInstallationPath, Module: ModuleName,
		AuthPolicy: httproute.AuthPolicyPublic, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyNone,
		RequestTimeout: ShopifyStartedInstallationTimeout,
		Handler: httproute.WithRequestBodyReadTimeout(5*time.Second, func(c *gin.Context) {
			c.Header("Cache-Control", "no-store")
			c.Header("Referrer-Policy", "no-referrer")
			fields, field, err := parseStringObject(http.MaxBytesReader(c.Writer, c.Request.Body, 18<<10), map[string]bool{
				"action": true, "attemptId": true, "browser": true, "state": true, "query": true,
			})
			if err != nil || len(fields["browser"]) != 43 || len(fields["state"]) != 43 || len(fields["query"]) > 8192 || c.Request.URL.RawQuery != "" {
				writeInvalid(c, field, "invalid")
				return
			}
			attempt, err := canonicalUUID(fields["attemptId"])
			if err != nil {
				writeInvalid(c, "attemptId", "invalid")
				return
			}
			ctx := c.Request.Context()
			switch fields["action"] {
			case "launch":
				target, err := service.BeginStartedInstallation(ctx, attempt, fields["browser"], fields["state"], fields["query"])
				if err != nil {
					writeStoreError(c, err)
					return
				}
				c.JSON(http.StatusOK, gin.H{"authorizationUrl": target})
			case "stage":
				domain, err := service.StageStartedInstallation(ctx, attempt, fields["browser"], fields["state"], fields["query"])
				if err != nil {
					writeShopifyInstallationError(c, err)
					return
				}
				c.JSON(http.StatusOK, gin.H{"domain": domain})
			default:
				writeInvalid(c, "action", "invalid")
			}
		}),
	}
}

func writeShopifyInstallationError(c *gin.Context, err error) {
	if errors.Is(err, storecenter.ErrShopifyOwnerRequired) {
		c.JSON(http.StatusForbidden, gin.H{"code": "SHOPIFY_OWNER_REQUIRED"})
		return
	}
	writeStoreError(c, err)
}
