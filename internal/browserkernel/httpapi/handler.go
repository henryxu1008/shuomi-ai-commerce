package kernelhttp

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	k "task-processor/internal/browserkernel"
	"task-processor/internal/httproute"
)

const Module = "browser-kernel-catalog"
const ReadBase = "/api/v1/browser/kernel-releases"
const WriteBase = "/api/v1/platform/browser/kernel-releases"

type Handler struct{ Service *k.Service }

func Routes(h *Handler) []httproute.Descriptor {
	var out []httproute.Descriptor
	for _, route := range []struct {
		method, path, action string
		write                bool
	}{
		{"GET", ReadBase, "list", false}, {"GET", ReadBase + "/:releaseId/artifact", "artifact", false},
		{"PUT", WriteBase + "/:releaseId", "publish", true}, {"POST", WriteBase + "/:releaseId/withdraw", "withdraw", true}, {"PUT", WriteBase + "/:releaseId/recommendation", "recommend", true},
	} {
		action := route.action
		d := httproute.Descriptor{Method: route.method, Path: route.path, Module: Module, AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyNone, RequestTimeout: 15 * time.Second, RejectUnreadRequestBody: route.method == "GET" || action == "withdraw", Handler: httproute.WithRequestBodyReadTimeout(5*time.Second, func(c *gin.Context) { h.handle(c, action) })}
		if route.write {
			d.AuthPolicy = httproute.AuthPolicyCurrentIdentityWithVerifiedRoles
			d.Permission = authz.PermissionListingKitPlatformAdm
		}
		out = append(out, d)
	}
	return out
}
func ValidateDescriptor(d httproute.Descriptor) error {
	for _, v := range Routes(nil) {
		if v.Method == d.Method && v.Path == d.Path {
			if d.Module == v.Module && d.AuthPolicy == v.AuthPolicy && d.OrganizationAccessPolicy == v.OrganizationAccessPolicy && d.Permission == v.Permission && d.RequestTimeout == v.RequestTimeout && d.RejectUnreadRequestBody == v.RejectUnreadRequestBody && d.OrganizationTargetResolver == nil && d.Handler != nil {
				return nil
			}
			break
		}
	}
	return k.ErrInvalid
}
func write(c *gin.Context, v any, e error) {
	c.Header("Cache-Control", "no-store")
	if e == nil {
		c.JSON(200, v)
		return
	}
	status, code := 503, "KERNEL_CATALOG_UNAVAILABLE"
	switch {
	case errors.Is(e, k.ErrInvalid):
		status, code = 400, "INVALID_REQUEST"
	case errors.Is(e, k.ErrNotFound):
		status, code = 404, "KERNEL_RELEASE_NOT_FOUND"
	case errors.Is(e, k.ErrConflict):
		status, code = 409, "KERNEL_RELEASE_CONFLICT"
	case errors.Is(e, k.ErrWithdrawn):
		status, code = 409, "KERNEL_RELEASE_WITHDRAWN"
	}
	c.JSON(status, gin.H{"code": code})
}
func emptyBody(c *gin.Context) bool {
	if c.Request.Body == nil {
		return true
	}
	b, e := io.ReadAll(io.LimitReader(c.Request.Body, 1))
	return e == nil && len(b) == 0
}
func readJSON(c *gin.Context, v any, required ...string) error {
	media, params, e := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if e != nil || media != "application/json" || len(params) > 1 || (params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8")) || c.GetHeader("Content-Encoding") != "" || c.Request.Body == nil {
		return k.ErrInvalid
	}
	raw, e := io.ReadAll(io.LimitReader(c.Request.Body, (16<<10)+1))
	if e != nil || httproute.DecodeJSON(raw, v, 16<<10, true) != nil {
		return k.ErrInvalid
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || len(fields) != len(required) {
		return k.ErrInvalid
	}
	for _, name := range required {
		value, ok := fields[name]
		if !ok || string(value) == "null" {
			return k.ErrInvalid
		}
	}
	return nil
}
func parseQuery(c *gin.Context, list bool) (k.Query, error) {
	q := k.Query{}
	if len(c.Request.URL.RawQuery) > 512 {
		return q, k.ErrInvalid
	}
	values, e := url.ParseQuery(c.Request.URL.RawQuery)
	if e != nil {
		return q, k.ErrInvalid
	}
	for name, v := range values {
		if len(v) != 1 || !(name == "os" || name == "arch" || name == "clientVersion" || list && (name == "cursor" || name == "limit")) {
			return q, k.ErrInvalid
		}
	}
	q.OS, q.Arch, q.ClientVersion = values.Get("os"), values.Get("arch"), values.Get("clientVersion")
	q.Cursor = values.Get("cursor")
	if v, ok := values["limit"]; ok {
		q.Limit, e = strconv.Atoi(v[0])
		if e != nil || q.Limit < 1 || q.Limit > 100 || strconv.Itoa(q.Limit) != v[0] {
			return q, k.ErrInvalid
		}
	}
	return q, nil
}
func (h *Handler) handle(c *gin.Context, action string) {
	c.Header("Cache-Control", "no-store")
	identity, ok := authidentity.AuthenticatedIdentityFromContext(c.Request.Context())
	if !ok || !authidentity.IsBoundedIdentifier(identity.UserID) || identity.TokenExpiresAt.IsZero() || !time.Now().Before(identity.TokenExpiresAt) {
		c.JSON(401, gin.H{"code": "UNAUTHORIZED"})
		return
	}
	if h == nil || h.Service == nil {
		write(c, nil, k.ErrUnavailable)
		return
	}
	ctx := c.Request.Context()
	if action == "list" || action == "artifact" {
		if !emptyBody(c) {
			write(c, nil, k.ErrInvalid)
			return
		}
		q, e := parseQuery(c, action == "list")
		if e != nil {
			write(c, nil, e)
			return
		}
		if action == "list" {
			v, e := h.Service.List(ctx, q)
			write(c, v, e)
		} else {
			v, e := h.Service.Artifact(ctx, c.Param("releaseId"), q)
			write(c, v, e)
		}
		return
	}
	if c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
		write(c, nil, k.ErrInvalid)
		return
	}
	id := c.Param("releaseId")
	switch action {
	case "publish":
		var p k.Publish
		if e := readJSON(c, &p, "version", "os", "arch", "minClientVersion", "maxClientVersionExclusive", "archiveFormat", "size", "sha256", "downloadUrl"); e != nil {
			write(c, nil, e)
			return
		}
		v, e := h.Service.Publish(ctx, id, p, identity.UserID)
		write(c, v, e)
	case "withdraw":
		if !emptyBody(c) {
			write(c, nil, k.ErrInvalid)
			return
		}
		v, e := h.Service.Withdraw(ctx, id, identity.UserID)
		write(c, v, e)
	case "recommend":
		var p struct {
			Recommended bool `json:"recommended"`
		}
		if e := readJSON(c, &p, "recommended"); e != nil {
			write(c, nil, e)
			return
		}
		v, e := h.Service.Recommend(ctx, id, p.Recommended, identity.UserID)
		write(c, v, e)
	default:
		write(c, nil, k.ErrInvalid)
	}
}
