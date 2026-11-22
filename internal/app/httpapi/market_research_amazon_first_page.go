package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	dataservicesapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/dataservices"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/dataservice"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/acquisition/zenrows"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	acq "github.com/henryxu1008/shuomi-ai-commerce/internal/product/dataacquisition"
	rh "github.com/henryxu1008/shuomi-ai-commerce/internal/product/research/httpapi"
)

const amazonFirstPageModule = "market-research-amazon-first-page"
const amazonFirstPagePath = rh.Base + "/:project_id/amazon-first-page/:approval_id"
const amazonFirstPageHTTPTimeout = 195 * time.Second

type amazonFirstPageInputs struct {
	approval acq.AmazonFirstPageApproval
	send     acq.AmazonFirstPageSender
}

func WithAmazonFirstPageTrial(approval acq.AmazonFirstPageApproval, readKey func(context.Context) (string, error)) (CurrentApplicationOption, error) {
	if approval.Validate() != nil || readKey == nil {
		return nil, acq.ErrInvalid
	}
	return func(o *currentApplicationOptions) {
		o.amazonFirstPage = &amazonFirstPageInputs{approval: approval, send: zenrows.NewAmazonFirstPageFetch(readKey)}
		o.amazonFirstPageConfigs++
	}, nil
}

type marketResearchAmazonFirstPage struct {
	trial *dataservicesapp.AmazonFirstPageTrial
	bind  func(context.Context, string) (context.Context, collection.Scope, error)
}

func amazonFirstPageRoutes(a *marketResearchAmazonFirstPage) []httproute.Descriptor {
	descriptor := func(method, permission string, timeout time.Duration) httproute.Descriptor {
		return httproute.Descriptor{
			Method: method, Path: amazonFirstPagePath, Module: amazonFirstPageModule, Permission: permission,
			AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite,
			RequestTimeout: timeout, RejectUnreadRequestBody: true,
			Handler: func(c *gin.Context) { a.serve(c) },
		}
	}
	return []httproute.Descriptor{
		descriptor(http.MethodGet, dataservice.PermissionMarket, 10*time.Second),
		descriptor(http.MethodPost, "workbench.project.manage", amazonFirstPageHTTPTimeout),
	}
}

func validateAmazonFirstPageDescriptor(d httproute.Descriptor) error {
	for _, expected := range amazonFirstPageRoutes(nil) {
		if d.Method == expected.Method && d.Path == expected.Path && d.Module == expected.Module &&
			d.Permission == expected.Permission && d.AuthPolicy == expected.AuthPolicy &&
			d.OrganizationAccessPolicy == expected.OrganizationAccessPolicy && d.OrganizationTargetResolver == nil &&
			d.RequestTimeout == expected.RequestTimeout && d.RejectUnreadRequestBody && d.Handler != nil {
			return nil
		}
	}
	return errors.New("Amazon first-page trial loses its admitted authorization boundary")
}

func isAmazonFirstPageHTTPPath(path string) bool {
	tail, ok := strings.CutPrefix(path, rh.Base+"/")
	if !ok {
		return false
	}
	project, tail, ok := strings.Cut(tail, "/amazon-first-page/")
	return ok && collection.ValidID(project) && collection.ValidID(tail)
}

func (a *marketResearchAmazonFirstPage) serve(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	project, id := c.Param("project_id"), c.Param("approval_id")
	r := c.Request
	if !collection.ValidID(project) || !collection.ValidID(id) || r.URL.RawQuery != "" || r.URL.ForceQuery ||
		r.ContentLength != 0 || len(r.TransferEncoding) != 0 || r.Header.Get("Content-Encoding") != "" || len(r.Header.Values("Idempotency-Key")) != 0 {
		httproute.RejectUnreadRequestBody(c)
		writeAmazonFirstPageError(c, acq.ErrInvalid)
		return
	}
	ctx, scope, err := a.bind(r.Context(), c.GetHeader("Authorization"))
	if err != nil || scope.Validate() != nil {
		writeAmazonFirstPageError(c, acq.ErrForbidden)
		return
	}
	p := acq.Principal{Scope: scope}
	var out acq.AmazonFirstPageReceipt
	if r.Method == http.MethodGet {
		out, err = a.trial.Read(ctx, p, project, id)
	} else {
		out, err = a.trial.Submit(ctx, p, project, id)
	}
	if err != nil {
		writeAmazonFirstPageError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func writeAmazonFirstPageError(c *gin.Context, err error) {
	status, code := http.StatusServiceUnavailable, "AMAZON_TRIAL_UNAVAILABLE"
	switch {
	case errors.Is(err, acq.ErrInvalid):
		status, code = http.StatusBadRequest, "INVALID_REQUEST"
	case errors.Is(err, acq.ErrForbidden):
		status, code = http.StatusForbidden, "FORBIDDEN"
	case errors.Is(err, acq.ErrNotFound):
		status, code = http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, acq.ErrConflict):
		status, code = http.StatusConflict, "AMAZON_TRIAL_CONFLICT"
	case errors.Is(err, acq.ErrUnknown):
		status, code = http.StatusConflict, "OUTCOME_UNKNOWN"
	}
	c.JSON(status, gin.H{"code": code})
}
