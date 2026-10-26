package httpapi

import (
	"time"

	"gorm.io/gorm"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	"task-processor/internal/product/review"
	"task-processor/internal/workbenchcontext"
)

// buildLocalTrialReviewRoutes binds the current Review owner to an explicitly
// admitted local trial database. It has no candidate generation capability.
func buildLocalTrialReviewRoutes(db *gorm.DB, resolver *workbenchcontext.Resolver, auth *authz.ListingKitAuthorizer) ([]httproute.Descriptor, error) {
	core, err := buildProductReviewCore(db, resolver, auth)
	if err != nil {
		return nil, err
	}
	service, err := review.NewCandidateService(core.reader, core.sourceReader, core.store, auth)
	if err != nil {
		return nil, err
	}
	binder := productReviewCapabilityBinder{now: time.Now}
	routes := productReviewRoutes(service, binder.Bind)
	for i := range routes {
		routes[i].RequestTimeout = review.Timeout + 2*time.Second
	}
	return routes, nil
}
