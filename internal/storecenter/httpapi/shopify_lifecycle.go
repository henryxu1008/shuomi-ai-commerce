package httpapi

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

const (
	ShopifyUninstalledPath   = "/api/v1/platform/shopify/app-uninstalled"
	ShopifyScopesUpdatedPath = "/api/v1/platform/shopify/app-scopes-update"
	ShopifyLifecycleTimeout  = 4 * time.Second
	shopifyLifecycleMaxBody  = 64 << 10
)

type shopifyLifecycleService interface {
	LifecycleNotificationsEnabled() bool
	ReceiveShopifyLifecycle(context.Context, string, string, string, []byte) error
	ReconcileShopifyLifecycle(context.Context) error
}

func (m routeModule) ReconcileShopifyLifecycle(ctx context.Context) error {
	service, ok := m.handler.shopifyConnections.(shopifyLifecycleService)
	if !ok {
		return storecenter.ErrOfficialConnectionUnavailable
	}
	return service.ReconcileShopifyLifecycle(ctx)
}

func (h *Handler) shopifyLifecycleRoutes() []httproute.Descriptor {
	service, ok := h.shopifyConnections.(shopifyLifecycleService)
	if !ok || !service.LifecycleNotificationsEnabled() {
		return nil
	}
	return []httproute.Descriptor{
		shopifyNoticeRoute(ShopifyUninstalledPath, "app/uninstalled", service.ReceiveShopifyLifecycle),
		shopifyNoticeRoute(ShopifyScopesUpdatedPath, "app/scopes_update", service.ReceiveShopifyLifecycle),
	}
}

type shopifyNoticeReceiver func(context.Context, string, string, string, []byte) error

func shopifyNoticeRoute(path, topic string, receive shopifyNoticeReceiver) httproute.Descriptor {
	return httproute.Descriptor{Method: http.MethodPost, Path: path, Module: ModuleName, AuthPolicy: httproute.AuthPolicyPublic,
		OrganizationAccessPolicy: httproute.OrganizationAccessPolicyNone, RequestTimeout: ShopifyLifecycleTimeout,
		Handler: httproute.WithRequestBodyReadTimeout(ShopifyLifecycleTimeout, func(c *gin.Context) {
			c.Header("Cache-Control", "no-store")
			request := c.Request
			mimeType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
			eventID := request.Header.Values("X-Shopify-Event-Id")
			signatures := request.Header.Values("X-Shopify-Hmac-Sha256")
			topics := request.Header.Values("X-Shopify-Topic")
			if err != nil || mimeType != "application/json" || request.URL.RawQuery != "" || len(topics) != 1 || topics[0] != topic ||
				len(eventID) != 1 || len(eventID[0]) > 128 || eventID[0] == "" || request.Body == nil || request.ContentLength > shopifyLifecycleMaxBody {
				c.Status(http.StatusBadRequest)
				return
			}
			if len(signatures) != 1 || len(signatures[0]) != 44 {
				c.Status(http.StatusUnauthorized)
				return
			}
			body, err := io.ReadAll(io.LimitReader(request.Body, shopifyLifecycleMaxBody+1))
			if err != nil || len(body) == 0 || len(body) > shopifyLifecycleMaxBody {
				c.Status(http.StatusBadRequest)
				return
			}
			err = receive(request.Context(), topic, eventID[0], signatures[0], body)
			switch {
			case errors.Is(err, storecenter.ErrOfficialAuthorizationRejected):
				c.Status(http.StatusUnauthorized)
			case errors.Is(err, storecenter.ErrVersionConflict), errors.Is(err, storecenter.ErrNotFound):
				c.Status(http.StatusBadRequest)
			case err != nil:
				c.Status(http.StatusServiceUnavailable)
			default:
				c.Status(http.StatusOK)
			}
		}),
	}
}
