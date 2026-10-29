package httpapi

import (
	"context"
	"net/http"
	"strings"
	zitadelruntime "task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	billing "task-processor/internal/commercial/billing"
	"time"

	"github.com/sirupsen/logrus"
)

type exactServiceAuthorizationReader interface {
	ReadExactServiceProjectAuthorization(context.Context, string, string, string, string) (zitadelruntime.ExactServiceProjectAuthorization, error)
}

type financialRecoveryAuthorizer struct {
	reader       exactServiceAuthorizationReader
	serviceToken string
	projectID    string
	authorizer   *authz.ListingKitAuthorizer
}

func (adapter financialRecoveryAuthorizer) ReauthorizeCommercialPurchase(ctx context.Context, organizationID, actorID string) (billing.CommercialPurchaseAuthorization, error) {
	organizationID = strings.TrimSpace(organizationID)
	actorID = strings.TrimSpace(actorID)
	if adapter.reader == nil || adapter.authorizer == nil || strings.TrimSpace(adapter.serviceToken) == "" || strings.TrimSpace(adapter.projectID) == "" || organizationID == "" || actorID == "" {
		return billing.CommercialPurchaseAuthorization{}, billing.ErrFeatureUnavailable
	}
	result, err := adapter.reader.ReadExactServiceProjectAuthorization(ctx, adapter.serviceToken, actorID, adapter.projectID, organizationID)
	if err != nil {
		return billing.CommercialPurchaseAuthorization{}, err
	}
	allowed := result.Found && result.State == "STATE_ACTIVE" && authz.AllowedOrganization(ctx, adapter.authorizer, actorID, organizationID, result.Roles, authz.PermissionWorkbenchCommercialPurchase)
	return billing.CommercialPurchaseAuthorization{OrganizationID: organizationID, ActorID: actorID, Roles: append([]string(nil), result.Roles...), Allowed: allowed, ObservedAt: time.Now().UTC()}, nil
}

func startCommercialRecoveryLoop(parent context.Context, server *http.Server, reconcile func(context.Context) error, interval time.Duration, name string, logger *logrus.Logger) {
	if parent == nil || server == nil || reconcile == nil || interval <= 0 {
		return
	}
	ctx, cancel := context.WithCancel(parent)
	server.RegisterOnShutdown(cancel)
	go func() {
		defer cancel()
		run := func() {
			runCtx, runCancel := context.WithTimeout(ctx, 20*time.Second)
			defer runCancel()
			if err := reconcile(runCtx); err != nil && logger != nil {
				logger.WithField("recovery", name).Warn("commercial recovery sweep incomplete")
			}
		}
		run()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				run()
			case <-ctx.Done():
				return
			}
		}
	}()
}
