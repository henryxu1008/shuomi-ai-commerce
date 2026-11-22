package httpapi

import (
	"context"
	"github.com/sirupsen/logrus"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/app/localtrial"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/core/config"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	kernelmodule "github.com/henryxu1008/shuomi-ai-commerce/internal/kernel/module"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/workbenchcontext"

	"gorm.io/gorm"
)

type issue36TrialModule struct{ routes []httproute.Descriptor }

func (issue36TrialModule) Name() string                { return "issue36-local-trial" }
func (issue36TrialModule) Enabled(*config.Config) bool { return true }
func (m issue36TrialModule) Register(reg *kernelmodule.Registry) error {
	reg.AddRoutes(m.routes...)
	return nil
}

// buildIssue36TrialModule borrows one explicitly admitted current Product,
// ApprovedAsset, Store Center and Listing PostgreSQL boundary. It only
// assembles existing owner routes; schema installation and sample preparation
// belong to a separate one-shot local installer.
func buildIssue36TrialModule(ctx context.Context, db *gorm.DB, resolver *workbenchcontext.Resolver, authorizer *authz.ListingKitAuthorizer, logs func(string) *logrus.Entry) (kernelmodule.Module, error) {
	if err := localtrial.VerifyRuntimePermissions(ctx, db, "issue36_trial_runtime", "store_center"); err != nil {
		return nil, err
	}
	reviewRoutes, err := buildLocalTrialReviewRoutes(db, resolver, authorizer)
	if err != nil {
		return nil, err
	}
	listingRoutes, _, err := buildLocalTrialListingRoutes(db, authorizer, logs)
	if err != nil {
		return nil, err
	}
	routes := make([]httproute.Descriptor, 0, len(reviewRoutes)+len(listingRoutes))
	routes = append(routes, reviewRoutes...)
	routes = append(routes, listingRoutes...)
	return issue36TrialModule{routes: routes}, nil
}
