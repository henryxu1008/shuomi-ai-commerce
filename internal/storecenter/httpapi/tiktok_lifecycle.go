package httpapi

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"task-processor/internal/httproute"
	"task-processor/internal/storecenter"
)

const TikTokLifecyclePath = "/api/v1/platform/tiktokshop/lifecycle"

type tikTokLifecycleService interface {
	TikTokLifecycleEnabled() bool
	ReceiveTikTokLifecycle(context.Context, []byte, string) error
}

func (h *Handler) tikTokLifecycleRoute() *httproute.Descriptor {
	service, ok := h.officialConnections.(tikTokLifecycleService)
	if !ok || !service.TikTokLifecycleEnabled() {
		return nil
	}
	return &httproute.Descriptor{Method: http.MethodPost, Path: TikTokLifecyclePath, Module: ModuleName, AuthPolicy: httproute.AuthPolicyPublic, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyNone, RequestTimeout: 5 * time.Second, Handler: httproute.WithRequestBodyReadTimeout(5*time.Second, func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		contentType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if err != nil || contentType != "application/json" || len(c.Request.Header.Values("Authorization")) != 1 {
			c.Status(http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, 65537))
		if err != nil || len(body) == 0 || len(body) > 65536 {
			c.Status(http.StatusBadRequest)
			return
		}
		err = service.ReceiveTikTokLifecycle(c.Request.Context(), body, c.GetHeader("Authorization"))
		if errors.Is(err, storecenter.ErrOfficialAuthorizationRejected) {
			c.Status(http.StatusUnauthorized)
			return
		}
		if err != nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		c.Status(http.StatusOK)
	})}
}
