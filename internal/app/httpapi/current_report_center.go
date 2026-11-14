package httpapi

import (
	"context"
	"errors"
	"gorm.io/gorm"
	listingapp "task-processor/internal/app/listing"
	reportapp "task-processor/internal/app/reportcenter"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/listing/record"
	"task-processor/internal/reportcenter"
	reporthttp "task-processor/internal/reportcenter/httpapi"
)

func WithReportCenter(db *gorm.DB) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.reportCenters++; o.reportCenterDB = db }
}

func WithSheinTargetReportSources() CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.sheinTargetReportSources++ }
}

func validateReportCenterPool(o currentApplicationOptions, source *gorm.DB) error {
	if o.offlineRecords > 0 && (o.offlineRecords != 1 || o.offlineRecordAssets == nil || o.productCollections != 1 || o.productAcquisitionDB == nil || o.storeCenters != 1 || o.storeCenterDB == nil || o.reportCenters != 1 || o.localTrials > 0 || o.privateDraftTrials > 0 || o.offlineRecordAssets == source || o.offlineRecordAssets == o.productAcquisitionDB || o.offlineRecordAssets == o.storeCenterDB || o.offlineRecordAssets == o.reportCenterDB || o.listing != nil && o.listing.AssetDB != o.offlineRecordAssets || o.sourceMaterials > 0 && o.sourceMaterialDB != o.offlineRecordAssets) {
		return errors.New("offline SHEIN records require normal independent owners")
	}
	if o.sheinTargetReportSources > 0 && (o.sheinTargetReportSources != 1 || o.reportCenters != 1 || o.listing == nil || o.privateDraftTrial != nil) {
		return errors.New("SHEIN report sources require one Report owner and normal full Supply")
	}
	if o.reportCenters == 0 {
		return nil
	}
	if o.reportCenters != 1 || o.reportCenterDB == nil {
		return errors.New("report center requires one existing owner pool")
	}
	others := []*gorm.DB{source, o.commercialOwnerDB, o.moneyOwnerDB, o.referralDB, o.productAcquisitionDB, o.imageAgentDB, o.accountAuditImageDB, o.accountAuditProductDB, o.storeCenterDB, o.localTrialDB, o.agentConfigurationDB, o.notificationDB, o.projectCenterDB, o.agentCustomizationDB, o.offlineRecordAssets}
	if o.toolMarket != nil {
		others = append(others, o.toolMarket.DB)
	}
	if o.ecoservices != nil {
		others = append(others, o.ecoservices.DB)
	}
	if o.listing != nil {
		others = append(others, o.listing.AssetDB)
	}
	if o.pod != nil {
		others = append(others, o.pod.AssetDB)
	}
	if o.productAgent != nil {
		others = append(others, o.productAgent.RunDB, o.productAgent.ReviewDB, o.productAgent.AssetDB)
	}
	if o.aiWorkbench != nil {
		others = append(others, o.aiWorkbench.DB)
	}
	if o.membership != nil {
		others = append(others, o.membership.ReceiptDB)
	}
	for _, db := range others {
		if db == o.reportCenterDB {
			return errors.New("report center requires an independent owner pool")
		}
	}
	return nil
}

type reportCenterModule struct {
	routes  []httproute.Descriptor
	service *reportcenter.Service
}

func (reportCenterModule) Name() string                  { return reporthttp.ModuleName }
func (reportCenterModule) Enabled(c *config.Config) bool { return c != nil && c.Workbench.Enabled }
func (m reportCenterModule) Register(r *kernelmodule.Registry) error {
	r.AddRoutes(m.routes...)
	return nil
}

// These are the already mounted current-request readers. Missing owners disable
// only new capture; reading saved reports never depends on an Agent or model.
func buildReportCenter(ctx context.Context, db *gorm.DB, deps routeAuthDependencies, policy *authz.ListingKitAuthorizer, product *productAgentApplication, supply *listingapp.Application, offline ...record.Reader) (kernelmodule.Module, error) {
	var reviews reportapp.ReviewReader
	if product != nil {
		reviews = product.reviews
	}
	var targets reportapp.TargetRecordReader
	if supply != nil {
		if supply.SourceMaterialsOnly || supply.Authorization == nil || supply.Sources == nil || supply.Records == nil || supply.ValidateDraftRead != nil {
			return nil, errors.New("report capture requires the normal full Supply reader")
		}
		targets = supply
	}
	if len(offline) > 1 {
		return nil, errors.New("duplicate offline source")
	}
	var records record.Reader
	if len(offline) == 1 {
		records = offline[0]
	}
	// Offline Record IDs and native TargetRecord IDs remain distinct sources.
	h, err := reportapp.New(ctx, reportapp.Dependencies{DB: db, Resolver: deps.organizationResolver, Policy: policy, Reviews: reviews, TargetRecords: targets, Records: records})
	if err != nil {
		return nil, err
	}
	// Review's existing live authorizer consumes this capability, not a fixed
	// execution scope. Report authorization keeps the values while refreshing id.
	bind := h.Bind
	h.Bind = func(ctx context.Context, header string) (context.Context, error) {
		bound, err := (productReviewCapabilityBinder{}).Bind(ctx, header)
		if err != nil {
			return nil, err
		}
		return bind(bound, header)
	}
	routes, err := reporthttp.BuildRoutes(h)
	if err != nil {
		return nil, err
	}
	return reportCenterModule{routes: routes, service: h.Service}, nil
}
