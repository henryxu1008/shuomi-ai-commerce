package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"mime"
	"net/http"
	"net/url"
	strict "sigs.k8s.io/json"
	"strconv"
	"strings"
	"task-processor/internal/httproute"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
	"time"
)

const Base = "/api/v1/workbench/research/projects"
const ModuleName = "market-research"
const SupplierReviewPath = Base + "/:project_id/candidates/:candidate/supplier-review"

type Handler struct {
	Service *research.Service
	Bind    func(context.Context, string) (context.Context, collection.Scope, error)
}
type route struct{ method, path string }

func Routes(h *Handler) []httproute.Descriptor {
	return researchRoutes(h, h != nil && h.Service != nil && h.Service.SupplierReviewsEnabled())
}

func researchRoutes(h *Handler, supplier bool) []httproute.Descriptor {
	out := []httproute.Descriptor{}
	specs := []route{{"GET", Base + "/:project_id"}, {"POST", Base + "/:project_id/commands"}, {"GET", Base + "/:project_id/commands/:key"}}
	if supplier {
		specs = append(specs, route{"GET", SupplierReviewPath})
	}
	for _, spec := range specs {
		permission := "workbench.project.read"
		if spec.method == "POST" {
			permission = "workbench.project.manage"
		}
		out = append(out, httproute.Descriptor{Method: spec.method, Path: spec.path, Module: ModuleName, Permission: permission, AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 10 * time.Second, RejectUnreadRequestBody: spec.method == "GET", Handler: httproute.WithRequestBodyReadTimeout(3*time.Second, func(c *gin.Context) {
			if h == nil || h.Service == nil || h.Bind == nil {
				failure(c, research.ErrUnavailable)
				return
			}
			h.serve(c)
		})})
	}
	return out
}
func ValidateDescriptor(d httproute.Descriptor) error {
	for _, e := range researchRoutes(nil, true) {
		if d.Method == e.Method && d.Path == e.Path && d.Module == e.Module && d.Permission == e.Permission && d.AuthPolicy == e.AuthPolicy && d.OrganizationAccessPolicy == e.OrganizationAccessPolicy && d.OrganizationTargetResolver == nil && d.RequestTimeout == e.RequestTimeout && d.RejectUnreadRequestBody == e.RejectUnreadRequestBody && d.Handler != nil {
			return nil
		}
	}
	return research.ErrForbidden
}
func failure(c *gin.Context, e error) {
	status, code := 503, "DEPENDENCY_UNAVAILABLE"
	for _, pair := range []struct {
		e      error
		status int
	}{{research.ErrUnsupported, 400}, {research.ErrInvalid, 400}, {research.ErrForbidden, 403}, {research.ErrNotFound, 404}, {research.ErrConflict, 409}, {research.ErrArchived, 409}, {research.ErrUnknown, 503}} {
		if errors.Is(e, pair.e) {
			status, code = pair.status, pair.e.Error()
			break
		}
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(status, gin.H{"code": code})
}
func (h *Handler) serve(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	ctx, scope, err := h.Bind(ctx, c.GetHeader("Authorization"))
	if err != nil {
		failure(c, research.ErrForbidden)
		return
	}
	project := c.Param("project_id")
	if !collection.ValidID(project) || c.GetHeader("Content-Encoding") != "" {
		failure(c, research.ErrInvalid)
		return
	}
	var value any
	if c.Request.Method == http.MethodPost {
		keys := c.Request.Header.Values("Idempotency-Key")
		if len(keys) != 1 || !collection.ValidID(keys[0]) || c.Request.URL.RawQuery != "" || c.Request.ContentLength > 64<<10 {
			failure(c, research.ErrInvalid)
			return
		}
		media, _, e := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if e != nil || media != "application/json" {
			failure(c, research.ErrInvalid)
			return
		}
		raw, e := io.ReadAll(io.LimitReader(c.Request.Body, (64<<10)+1))
		if e != nil || len(raw) > 64<<10 {
			failure(c, research.ErrInvalid)
			return
		}
		var command research.Command
		violations, e := strict.UnmarshalStrict(raw, &command, strict.DisallowDuplicateFields, strict.DisallowUnknownFields)
		if e != nil || len(violations) > 0 || research.Validate(command) != nil {
			failure(c, research.ErrInvalid)
			return
		}
		value, err = h.Service.Execute(ctx, scope, project, keys[0], command)
	} else {
		if c.Request.ContentLength != 0 {
			httproute.RejectUnreadRequestBody(c)
			failure(c, research.ErrInvalid)
			return
		}
		if candidate := c.Param("candidate"); candidate != "" {
			value, err = h.supplierRead(ctx, scope, project, candidate, c.Request.URL.RawQuery)
		} else if key := c.Param("key"); key != "" {
			if c.Request.URL.RawQuery != "" {
				failure(c, research.ErrInvalid)
				return
			}
			value, err = h.Service.Lookup(ctx, scope, project, key)
		} else {
			query, e := url.ParseQuery(c.Request.URL.RawQuery)
			if e != nil || strings.Contains(c.Request.URL.RawQuery, ";") || len(query) > 1 || len(query["after"]) > 1 {
				failure(c, research.ErrInvalid)
				return
			}
			for key := range query {
				if key != "after" {
					failure(c, research.ErrInvalid)
					return
				}
			}
			value, err = h.Service.List(ctx, scope, project, query.Get("after"))
		}
	}
	if err != nil {
		failure(c, err)
		return
	}
	raw, err := json.Marshal(value)
	if err != nil || len(raw) > 2<<20 {
		failure(c, research.ErrUnavailable)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Data(200, "application/json", raw)
}

func (h *Handler) supplierRead(ctx context.Context, scope collection.Scope, project, candidate, raw string) (any, error) {
	q, err := url.ParseQuery(raw)
	if err != nil || len(raw) > 1024 {
		return nil, research.ErrInvalid
	}
	if len(q) == 0 && raw == "" {
		return h.Service.SupplierReviews(ctx, scope, project, candidate, 0)
	}
	if len(q) == 1 && len(q["before"]) == 1 {
		before, err := strconv.ParseInt(q.Get("before"), 10, 64)
		if err != nil || before <= 0 || strconv.FormatInt(before, 10) != q.Get("before") {
			return nil, research.ErrInvalid
		}
		return h.Service.SupplierReviews(ctx, scope, project, candidate, before)
	}
	if len(q) != 4 || len(q["itemId"]) != 1 || len(q["revision"]) != 1 || len(q["publicationId"]) != 1 || len(q["version"]) != 1 {
		return nil, research.ErrInvalid
	}
	revision, revErr := strconv.ParseInt(q.Get("revision"), 10, 64)
	version, verErr := strconv.ParseUint(q.Get("version"), 10, 63)
	if revErr != nil || verErr != nil || strconv.FormatInt(revision, 10) != q.Get("revision") || strconv.FormatUint(version, 10) != q.Get("version") {
		return nil, research.ErrInvalid
	}
	in := research.Selection{ItemID: q.Get("itemId"), Revision: revision, PublicationID: q.Get("publicationId"), Version: version}
	return h.Service.SupplierPreview(ctx, scope, project, candidate, in)
}
