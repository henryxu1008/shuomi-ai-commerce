package httpapi

import (
	"context"
	"errors"
	"gorm.io/gorm"
	k "task-processor/internal/browserkernel"
	kh "task-processor/internal/browserkernel/httpapi"
	"task-processor/internal/core/config"
	kp "task-processor/internal/integration/persistence/browserkernel"
	kernelmodule "task-processor/internal/kernel/module"
)

// WithBrowserKernelCatalog opts in to a pre-installed, independently owned
// metadata pool. Neither this constructor nor startup publishes any releases.
func WithBrowserKernelCatalog(db *gorm.DB, origins []string) CurrentApplicationOption {
	return func(o *currentApplicationOptions) {
		o.browserKernels++
		o.browserKernelDB = db
		o.browserKernelOrigins = append([]string(nil), origins...)
	}
}

type browserKernelModule struct{ handler *kh.Handler }

func (browserKernelModule) Name() string                { return kh.Module }
func (browserKernelModule) Enabled(*config.Config) bool { return true }
func (m browserKernelModule) Register(reg *kernelmodule.Registry) error {
	reg.AddRoutes(kh.Routes(m.handler)...)
	return nil
}
func buildBrowserKernelModule(ctx context.Context, source *gorm.DB, o currentApplicationOptions) (kernelmodule.Module, error) {
	db := o.browserKernelDB
	if db == nil || o.browserKernels != 1 {
		return nil, errors.New("browser kernel requires one owner pool")
	}
	others := []*gorm.DB{source, o.commercialOwnerDB, o.moneyOwnerDB, o.referralDB, o.productAcquisitionDB, o.imageAgentDB, o.storeCenterDB, o.localTrialDB, o.agentConfigurationDB, o.accountAuditImageDB, o.accountAuditProductDB, o.notificationDB, o.agentCustomizationDB, o.reportCenterDB, o.projectCenterDB, o.sourceMaterialDB}
	if o.ecoservices != nil {
		others = append(others, o.ecoservices.DB)
	}
	if o.supplyChain != nil {
		others = append(others, o.supplyChain.AssetDB)
	}
	if o.toolMarket != nil {
		others = append(others, o.toolMarket.DB)
	}
	if o.pod != nil {
		others = append(others, o.pod.AssetDB)
	}
	if o.membership != nil {
		others = append(others, o.membership.ReceiptDB)
	}
	if o.productAgent != nil {
		others = append(others, o.productAgent.RunDB, o.productAgent.ReviewDB, o.productAgent.AssetDB)
	}
	if o.fullImages != nil {
		others = append(others, o.fullImages.AssetDB, o.fullImages.WorkerDB)
	}
	if o.aiWorkbench != nil {
		others = append(others, o.aiWorkbench.DB)
	}
	if o.dataServices != nil {
		others = append(others, o.dataServices.ProductDB)
	}
	for _, other := range others {
		if db == other {
			return nil, errors.New("browser kernel cannot share an owner pool")
		}
	}
	if e := kp.VerifySchema(ctx, db); e != nil {
		return nil, e
	}
	repo, e := kp.New(db)
	if e != nil {
		return nil, e
	}
	service, e := k.New(repo, o.browserKernelOrigins)
	if e != nil {
		return nil, e
	}
	return browserKernelModule{&kh.Handler{Service: service}}, nil
}
