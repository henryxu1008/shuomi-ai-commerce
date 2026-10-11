package httpapi

import (
	"context"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	storeapp "task-processor/internal/app/storecenter"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	memberprovider "task-processor/internal/integration/zitadel/membership"
	o "task-processor/internal/marketplace/shein/observations"
	"task-processor/internal/organization/membership"
	"task-processor/internal/storecenter"
)

type storeResponsibleReader struct{ directory membership.Directory }

func (r storeResponsibleReader) ValidateResponsible(ctx context.Context, org, id string) error {
	if r.directory == nil {
		return storecenter.ErrDependencyUnavailable
	}
	member, err := r.directory.Read(ctx, org, id)
	if err != nil || member.OrganizationID != org || member.ID != id || member.State != "active" {
		return storecenter.ErrNotFound
	}
	return nil
}

type storeCatalogSignals struct {
	records      *storecenter.MemberScopedStoreRepository
	connections  *storeapp.OfficialConnections
	applications *storeapp.OfficialApplicationRegistry
	observations *o.Service
}

func catalogMetric(metric o.Metric) storecenter.CatalogMetric {
	return storecenter.CatalogMetric{Value: metric.Value, Unknown: metric.Unknown, Complete: metric.Complete, Reason: metric.Reason}
}
func (s storeCatalogSignals) ReadCatalogSignals(ctx context.Context, org string, ids []string, thresholds map[string]*int64) (map[string]storecenter.CatalogSignals, error) {
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.EffectiveOrganizationID != org {
		return nil, storecenter.ErrNotFound
	}
	scope := o.Scope{OrganizationID: org, ActorID: identity.UserID, MemberID: identity.EffectiveMemberID}
	values := make([]storecenter.CatalogSignals, len(ids))
	active := make([]bool, len(ids))
	group, reads := errgroup.WithContext(ctx)
	group.SetLimit(16)
	choices := map[string]storecenter.OfficialApplicationChoice{}
	for _, choice := range s.applications.Applications() {
		choices[choice.AppID+":"+choice.Revision] = choice
	}
	for i, id := range ids {
		group.Go(func() error {
			record, err := s.records.Get(reads, org, id)
			if err != nil {
				return err
			}
			active[i] = record.RecordStatus() == storecenter.RecordStatusActive
			view, err := s.connections.Read(reads, org, id)
			signal := storecenter.CatalogSignals{Connection: storecenter.ConnectionStatusUnavailable, Reason: "observations_unavailable", Products: storecenter.CatalogMetric{Reason: "observations_unavailable"}, Orders: storecenter.CatalogMetric{Reason: "observations_unavailable"}, Inventory: storecenter.CatalogMetric{Reason: "observations_unavailable"}}
			if err == nil {
				signal.Connection = view.Status
				if choice, exists := choices[view.AppID+":"+view.AppRevision]; exists {
					signal.ApplicationType = choice.Type
				}
			}
			values[i] = signal
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	eligible := []string{}
	for i, id := range ids {
		if active[i] {
			eligible = append(eligible, id)
		}
	}
	products, orders := map[string]o.StoreSummary{}, map[string]o.StoreSummary{}
	if s.observations != nil && len(eligible) > 0 {
		// Each owner keeps its own permission/source validation. Failure stays local
		// to the corresponding metric; Store readers never gain product/order read.
		products, _ = s.observations.StoreSummaries(ctx, scope, o.Products, eligible, thresholds)
		orders, _ = s.observations.StoreSummaries(ctx, scope, o.Orders, eligible, thresholds)
	}
	out := map[string]storecenter.CatalogSignals{}
	for i, id := range ids {
		signal := values[i]
		if product, exists := products[id]; exists {
			signal.Products = catalogMetric(product.Metric)
			signal.Inventory = catalogMetric(product.Inventory)
			signal.ProductSync = product.SyncStatus
			signal.ObservedAt = product.ObservedAt
			signal.Reason = product.Reason
		}
		if order, exists := orders[id]; exists {
			signal.Orders = catalogMetric(order.Metric)
			signal.OrderSync = order.SyncStatus
			if order.ObservedAt != nil && (signal.ObservedAt == nil || order.ObservedAt.After(*signal.ObservedAt)) {
				signal.ObservedAt = order.ObservedAt
			}
		}
		product, pok := products[id]
		order, ook := orders[id]
		signal.SyncHealthy = pok && ook && product.Complete && order.Complete && product.SyncStatus == "completed" && order.SyncStatus == "completed"
		if thresholds[id] == nil {
			signal.Inventory = storecenter.CatalogMetric{Reason: "threshold_not_set"}
		}
		out[id] = signal
	}
	return out, nil
}
func buildStoreCatalog(db *gorm.DB, permissions *authz.ListingKitAuthorizer, applications *storeapp.OfficialApplicationRegistry, observations *o.Service, members *MembershipDependencies, cfg *config.Config) (*storecenter.Catalog, *storecenter.MetadataRepository, error) {
	access := currentStoreMemberAuthorizer{authorizer: permissions}
	records, err := storecenter.NewMemberScopedStoreRepository(db, access)
	if err != nil {
		return nil, nil, err
	}
	var connections *storeapp.OfficialConnections
	if applications == nil {
		connections, err = storeapp.NewUnconfiguredOfficialConnections(records)
	} else {
		connections, err = storeapp.NewOfficialConnections(records, applications)
	}
	if err != nil {
		return nil, nil, err
	}
	var directory membership.Directory
	if members != nil {
		directory, err = memberprovider.NewClient(members.ProviderOrigin, members.ReadToken, cfg.ListingKit.Zitadel.ProjectID, nil)
		if err != nil {
			return nil, nil, err
		}
	}
	metadata, err := storecenter.NewMetadataRepository(db, access, storeResponsibleReader{directory: directory})
	if err != nil {
		return nil, nil, err
	}
	catalog, err := storecenter.NewCatalog(db, access, storeCatalogSignals{records: records, connections: connections, applications: applications, observations: observations})
	return catalog, metadata, err
}
