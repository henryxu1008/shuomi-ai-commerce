package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	fulfillmentmodule "task-processor/internal/app/httpapi/fulfillment"
	podapp "task-processor/internal/app/pod"
	dataservicesruntime "task-processor/internal/app/runtime/dataservices"
	fulfillmentruntime "task-processor/internal/app/runtime/fulfillment"
	listingoperation "task-processor/internal/app/runtime/listingoperation"
	podruntime "task-processor/internal/app/runtime/pod"
	observationruntime "task-processor/internal/app/runtime/storeobservations"
	scheduledruntime "task-processor/internal/app/runtime/workbenchschedules"
	"task-processor/internal/integration/sds"
	"task-processor/internal/product/collection"
	"time"

	"github.com/sirupsen/logrus"
	"go.temporal.io/sdk/client"
	"gorm.io/gorm"

	aistore "task-processor/internal/aicapability/store"
	"task-processor/internal/app/httpapi"
	appruntime "task-processor/internal/app/runtime"
	"task-processor/internal/app/runtime/currentapplication"
	coreconfig "task-processor/internal/core/config"
	"task-processor/internal/imageagent"
	platformdatabase "task-processor/internal/platform/database"
)

func main() {
	if err := execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execute() error {
	manifestPath := flag.String("config", "", "absolute path to the private current-application JSON manifest")
	inspectModules := flag.Bool("inspect-modules", false, "read-only native module schema and runtime privilege inspection; never starts workers")
	shutdownFile := flag.String("shutdown-file", "", "optional absolute path to a private graceful-shutdown trigger")
	flag.Parse()
	if *manifestPath == "" {
		return fmt.Errorf("-config is required")
	}
	cfg, err := currentapplication.LoadConfig(*manifestPath)
	if err != nil {
		return err
	}
	if *inspectModules {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return json.NewEncoder(os.Stdout).Encode(currentapplication.InspectModules(ctx, cfg, nil))
	}
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *shutdownFile != "" {
		var cancel context.CancelFunc
		ctx, cancel, err = currentapplication.ContextWithShutdownFile(ctx, *shutdownFile)
		if err != nil {
			return err
		}
		defer cancel()
	}
	return currentapplication.Run(ctx, cfg, logger, currentapplication.Dependencies{
		OpenChannelCredentialWriter: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenChannelSecretReader: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingReadOnlyContext(ctx, databaseConfig(cfg))
		},
		OpenModelRouteWriter: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenDataServices: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		DialDataServicesWorkflow: func(ctx context.Context, address, namespace string) (client.Client, func() error, error) {
			c, err := client.DialContext(ctx, client.Options{HostPort: address, Namespace: namespace})
			if err != nil {
				return nil, nil, err
			}
			return c, func() error { c.Close(); return nil }, nil
		},
		NewMarketStorage: prepareSupplyMarket,
		PreparePOD: func(ctx context.Context, p *currentapplication.PODConfig) (sds.CredentialSource, error) {
			return currentapplication.PreparePODCredentials(ctx, p.CredentialFile, &http.Client{Timeout: 15 * time.Second})
		},
		IdentityPreflight: currentapplication.VerifyIdentityProvider,
		OpenReportCenter: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenToolMarket: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		NewKnowledge:   prepareKnowledge,
		NewEcoservices: prepareEcoservices,
		OpenEcoservices: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenKnowledge: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenStoreCenter: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenLocalTrial: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},

		OpenSourceAccount: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},

		OpenCommercialOwner: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenMoneyOwner: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenProductAcquisition: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenImageAgent: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenImageSetWorker: openImageSetWorker,
		DialImageSetWorkflow: func(ctx context.Context, address, namespace string) (client.Client, func() error, error) {
			current, err := client.DialContext(ctx, client.Options{HostPort: address, Namespace: namespace})
			if err != nil {
				return nil, nil, err
			}
			return current, func() error { current.Close(); return nil }, nil
		},
		OpenApprovedAssets: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		DialProductWorkflow:           dialCurrentWorkflow,
		DialStoreObservationsWorkflow: dialCurrentWorkflow,
		OpenAccountAuditUsage: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingReadOnlyContext(ctx, databaseConfig(cfg))
		},
		OpenProductAgent: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenMarketResearch: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenProjectCenter: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenAIWorkbench: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenBrowserKernel: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenNotificationCenter: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenAgentCustomization: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		DialImageAgentWorkflow: func(ctx context.Context, address, namespace string) (imageagent.WorkflowClient, func() error, error) {
			return appruntime.DialOrganizationImageAgentTemporalWorkflowClient(ctx, address, namespace)
		},
		OpenReferrals: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		OpenMembership: func(ctx context.Context, cfg currentapplication.DatabaseConfig) (*gorm.DB, error) {
			return platformdatabase.OpenExistingWritableContext(ctx, databaseConfig(cfg))
		},
		NewApplicationWithFeatures: func(ctx context.Context, source *gorm.DB, features currentapplication.ApplicationFeatures, cfg *coreconfig.Config, logger *logrus.Logger) (*http.Server, error) {
			options := make([]httpapi.CurrentApplicationOption, 0, 6)
			if features.SharedModels != nil {
				pricing := httpapi.ModelRoutePricingDependencies{Shared: features.SharedModels, Prices: features.OfficialModelPrices, Credentials: features.ChannelCredentialManagement}
				if features.AutomaticFX != nil {
					pricing.AutomaticFX = features.AutomaticFX
				}
				options = append(options, httpapi.WithModelRoutePricing(pricing))
			}
			if features.AgentCustomizationDB != nil {
				options = append(options, httpapi.WithAgentCustomization(features.AgentCustomizationDB))
			}
			if features.MarketResearchDB != nil {
				options = append(options, httpapi.WithMarketResearch(features.MarketResearchDB))
				if features.MarketResearchSupplierReview {
					options = append(options, httpapi.WithMarketResearchSupplierReview())
				}
			}
			if f := features.ResearchManualOriginal; f != nil {
				options = append(options, httpapi.WithResearchManualOriginal(f.Binding))
			}
			if f := features.ResearchFixture; f != nil {
				options = append(options, httpapi.WithResearchFixture(httpapi.ResearchFixtureBinding{
					Scope: collection.Scope{OrganizationID: f.OrganizationID, ActorID: f.ActorID, MemberID: f.MemberID}, Project: f.ProjectID,
					PaginationEnabled: f.PaginationEnabled,
				}))
			}
			if trial := features.SHEINUSTrial; trial != nil {
				option, err := httpapi.WithSHEINUSTrial(httpapi.SHEINUSTrialInputs{
					Approval: trial.Approval, Reconciler: trial.Reconciler, Access: trial.Access, Budget: trial.Budget,
					ReadKey: trial.ReadKey, ReadUsage: trial.ReadUsage,
				})
				if err != nil {
					return nil, err
				}
				options = append(options, option)
			}
			if features.ProjectCenterDB != nil {
				options = append(options, httpapi.WithProjectCenter(features.ProjectCenterDB))
			}
			if features.ReportCenterDB != nil {
				options = append(options, httpapi.WithReportCenter(features.ReportCenterDB))
			}
			if features.SheinOfflineRecords {
				options = append(options, httpapi.WithSheinOfflineRecords(features.ApprovedAssetDB))
			}
			if features.SheinTargetReportSources {
				options = append(options, httpapi.WithSheinTargetReportSources())
			}
			if features.Ecoservices != nil {
				e := features.Ecoservices
				options = append(options, httpapi.WithEcoservices(httpapi.EcoservicesDependencies{Capacity: e.Capacity, NonPaymentOnly: e.NonPaymentOnly, DB: e.DB, Objects: e.Objects, Channel: e.Channel, Protection: e.Protection, MerchantProtection: e.MerchantProtection}))
			}
			if features.NotificationCenterDB != nil {
				options = append(options, httpapi.WithNotificationCenter(features.NotificationCenterDB))
			}
			if features.BrowserKernel != nil {
				options = append(options, httpapi.WithBrowserKernelCatalog(features.BrowserKernelDB, features.BrowserKernel.ArtifactOrigins))
			}
			if features.Knowledge != nil {
				options = append(options, httpapi.WithKnowledge(features.Knowledge))
			}
			if features.ProductAgent != nil && !features.ProductAgent.Enabled {
				options = append(options, httpapi.WithAgentConfiguration(features.ProductAgentDB))
			}
			if features.ProductAgent != nil && features.ProductAgent.Enabled {
				p := features.ProductAgent
				ledger := aistore.NewGormInvocationRecorder(features.ProductAgentDB)
				var operator *httpapi.TitlePilotOperator
				if t := features.TitlePilot; t != nil {
					operator = &httpapi.TitlePilotOperator{Host: t.Address(), BatchID: t.BatchID, OrganizationID: t.OrganizationID,
						ActorID: t.ActorID, MemberID: t.MemberID, SamplesDigest: t.SamplesDigest, ProfileDigest: t.ProfileDigest,
						CredentialRevision: t.CredentialRevision, PriceVersion: t.PriceVersion, FXVersion: t.FXVersion,
						ExecutionApprovalID: t.ExecutionApprovalID, NotBefore: t.NotBefore, NotAfter: t.NotAfter, Register: features.RegisterTitlePilotOperator,
						VerifyIsolation: features.VerifyTitlePilotIsolation}
				}
				options = append(options, httpapi.WithProductAgent(httpapi.ProductAgentDependencies{TitlePilotOperator: operator, ChannelCredentials: features.ChannelCredentials, CurrentTextPolicies: features.CurrentTextPolicies, RunDB: features.ProductAgentDB, ReviewDB: features.ProductReviewDB, AssetDB: features.ProductAgentAssetDB, Ledger: ledger, TextPolicies: p.TextPolicies, ContentTextPolicies: p.ContentTextPolicies, MaterialsTextPolicies: p.MaterialsTextPolicies, SelectionTextPolicies: p.SelectionTextPolicies, ResearchTextPolicies: p.ResearchTextPolicies, DiagnosisTextPolicies: p.DiagnosisTextPolicies, ImageAnalysisPolicies: p.ImageAnalysisPolicies, Enabled: true, ContentTitleOnly: p.ContentTitleOnly, ContentOnly: p.ContentOnly, AllowedOrganizationIDs: p.AllowedOrganizationIDs, Limits: p.Limits()}))
			}
			if features.AIWorkbench != nil && features.AIWorkbench.Enabled {
				chat := httpapi.AIWorkbenchDependencies{DB: features.AIWorkbenchDB, PlanningTextPolicies: features.AIWorkbench.PlanningTextPolicies, StoreRouteTextPolicies: features.AIWorkbench.StoreRouteTextPolicies, ConversationOnly: features.AIWorkbench.ConversationOnly, ContentTasks: features.AIWorkbench.ContentTasks, AllowedOrganizationIDs: features.AIWorkbench.AllowedOrganizationIDs}
				if features.AIWorkbenchScheduleClient != nil {
					chat.Schedules = &scheduledruntime.Backend{Client: features.AIWorkbenchScheduleClient}
					chat.ScheduleStop = features.AIWorkbenchScheduleStop
				}
				options = append(options, httpapi.WithAIWorkbench(chat))
			}
			if features.RuntimeContext != nil {
				options = append(options, httpapi.WithRuntimeContext(features.RuntimeContext))
			}
			if features.CommercialOwnerDB != nil {
				options = append(options, httpapi.WithCommercialOwnerDatabase(features.CommercialOwnerDB))
			}
			if features.StoreCenterDB != nil {
				options = append(options, httpapi.WithStoreCenter(features.StoreCenterDB))
				if features.OfficialStoreApplications != nil {
					options = append(options, httpapi.WithStoreOfficialApplications(features.OfficialStoreApplications))
				}
			}
			if features.Fulfillment != nil {
				options = append(options, httpapi.WithFulfillment(fulfillmentmodule.Dependencies{Configuration: features.Fulfillment, Starter: fulfillmentruntime.Starter{Client: features.FulfillmentWorkflow}, NewWorker: fulfillmentruntime.WorkerFactory(features.FulfillmentWorkflow, features.FulfillmentLifecycle), Lifecycle: features.FulfillmentLifecycle}))
			}
			if features.OperationsCockpit {
				options = append(options, httpapi.WithOperationsCockpit())
			}
			if features.StoreObservationsWorkflow != nil {
				options = append(options, httpapi.WithStoreObservations(httpapi.StoreObservationsDependencies{PreparationOnly: features.StoreObservationsPreparationOnly, Starter: observationruntime.Starter{Client: features.StoreObservationsWorkflow}, Lifecycle: features.StoreObservationsLifecycle, NewWorker: observationruntime.WorkerFactory(features.StoreObservationsWorkflow, features.StoreObservationsLifecycle)}))
			}
			if features.LocalTrialDB != nil {
				options = append(options, httpapi.WithIssue36Trial(features.LocalTrialDB))
			}
			if features.MoneyOwnerDB != nil {
				options = append(options, httpapi.WithMoneyOwnerDatabase(features.MoneyOwnerDB))
			}
			if features.ProductAcquisitionDB != nil {
				options = append(options, httpapi.WithProductAcquisition(features.ProductAcquisitionDB))
				options = append(options, httpapi.WithBrowserCapture())
			}
			if features.ToolMarketDB != nil && features.ToolMarket != nil {
				options = append(options, httpapi.WithToolMarket(httpapi.ToolMarketDependencies{DB: features.ToolMarketDB, Package: features.ToolMarket.PackageConfig()}))
			}
			if features.DataServices != nil {
				d := features.DataServices
				dataDependencies := httpapi.DataServicesDependencies{ExecutionEnabled: d.ExecutionEnabled, ProductDB: features.DataServicesDB, TrustedProxyCIDRs: d.TrustedProxyCIDRs}
				if d.ExecutionEnabled {
					dataDependencies.Provider, dataDependencies.Worker = features.DataServicesProvider, features.DataServicesWorker
					dataDependencies.Starter = dataservicesruntime.TemporalStarter{Client: features.DataServicesWorkflow}
					dataDependencies.NewWorker = func(runner dataservicesruntime.JobRunner) (dataservicesruntime.Worker, error) {
						return dataservicesruntime.NewWorker(features.DataServicesWorkflow, runner)
					}
				}
				options = append(options, httpapi.WithDataServices(dataDependencies))
			}
			if features.ProductCollections {
				if features.ProductAcquisitionDB != nil {
					options = append(options, httpapi.WithAmazonBrowserCapture())
				}
				options = append(options, httpapi.WithProductCollections())
			}
			if features.SourceMaterials {
				options = append(options, httpapi.WithSourceMaterials(features.ApprovedAssetDB))
			}
			if features.AcquisitionPlans {
				options = append(options, httpapi.WithAcquisitionPlans())
			}
			if features.SupplyCompletionDrafts {
				options = append(options, httpapi.WithListingCompletionDrafts())
			}
			if features.SourceMediaStorage != nil {
				options = append(options, httpapi.WithCollectionSourceMedia(features.SourceMediaStorage))
			}
			if t := features.PrivateDraftTrial; t != nil {
				options = append(options, httpapi.WithPrivateDraftTrial(httpapi.PrivateDraftTrialDependencies{Scope: collection.Scope{OrganizationID: t.OrganizationID, ActorID: t.ActorID, MemberID: t.MemberID}, StoreID: t.StoreID}))
			}
			if t := features.ControlledLocalDraft; t != nil {
				options = append(options, httpapi.WithControlledLocalDraft(httpapi.ControlledLocalDraftDependencies{Policy: t.Policy(), AssetDB: features.ApprovedAssetDB}))
			}
			if features.ListingWorkflow != nil {
				options = append(options, httpapi.WithListing(httpapi.ListingDependencies{AssetDB: features.ApprovedAssetDB, Starter: listingoperation.TemporalOperationStarter{Client: features.ListingWorkflow}, NewWorker: listingoperation.WorkerFactory(features.ListingWorkflow), Worker: features.ListingWorker}))
			}
			if features.MarketStorage != nil {
				options = append(options, httpapi.WithSupplyMarket(httpapi.SupplyMarketDependencies{Storage: features.MarketStorage}))
			}
			if features.POD != nil {
				p := features.POD
				options = append(options, httpapi.WithPOD(httpapi.PODDependencies{AssetDB: features.PODAssetDB, Credentials: features.PODCredentials, HTTP: &http.Client{Timeout: 15 * time.Second}, OSSHosts: p.OSSHosts, Starter: podruntime.TemporalStarter{Client: features.PODWorkflow}, NewWorker: func(p *podapp.Processor) (podapp.Worker, error) { return podruntime.NewWorker(features.PODWorkflow, p) }, Worker: features.PODWorker}))
			}
			if features.ImageAgentDB != nil {
				if features.FullImageSet {
					options = append(options, httpapi.WithFullImageSet(features.ImageAgentDB, features.ImageAgentWorkflow, httpapi.FullImageSetDependencies{ChannelCredentials: features.ChannelCredentials, CurrentImagePolicies: features.CurrentImagePolicies, AssetDB: features.ImageSetAssetDB, WorkerDB: features.ImageSetWorkerDB, WorkerConfig: features.ImageSetWorkerConfig, Client: features.ImageSetTemporal, Worker: features.ImageSetWorker}))
				} else {
					options = append(options, httpapi.WithAcquisitionImageAgent(features.ImageAgentDB, features.ImageAgentWorkflow))
				}
			}
			if features.AccountAuditImageDB != nil || features.AccountAuditProductDB != nil {
				options = append(options, httpapi.WithAccountAuditUsageSources(features.AccountAuditImageDB, features.AccountAuditProductDB))
			}
			if features.ReferralDB != nil {
				options = append(options, httpapi.WithReferrals(features.ReferralDB))
			}
			if features.MembershipDB != nil {
				if features.Membership == nil {
					return nil, fmt.Errorf("membership configuration unavailable")
				}
				options = append(options, httpapi.WithMembership(httpapi.MembershipDependencies{ReceiptDB: features.MembershipDB, ProviderOrigin: features.Membership.ProviderOrigin, ReadToken: features.Membership.ReadToken, WriteToken: features.Membership.WriteToken, InvitationMail: features.Membership.InvitationMail}))
			}
			server, buildErr := httpapi.NewCurrentApplicationWithOptions(ctx, source, cfg, logger, options...)
			return server, buildErr
		},
		NewApplication: func(ctx context.Context, source *gorm.DB, cfg *coreconfig.Config, logger *logrus.Logger) (*http.Server, error) {
			return httpapi.NewCurrentApplication(ctx, source, cfg, logger)
		},
		NewReferralsApplication: func(ctx context.Context, source, referrals *gorm.DB, cfg *coreconfig.Config, logger *logrus.Logger) (*http.Server, error) {
			return httpapi.NewCurrentApplicationWithOptions(ctx, source, cfg, logger, httpapi.WithReferrals(referrals))
		},
		CloseDatabase: platformdatabase.Close,
	})
}

func databaseConfig(cfg currentapplication.DatabaseConfig) *platformdatabase.Config {
	return &platformdatabase.Config{
		Host: cfg.Host, Port: cfg.Port, User: cfg.User, Password: cfg.Password, Database: cfg.Database,
		MaxConnections: cfg.MaxConnections, MaxIdleConnections: cfg.MaxConnections, ConnectionMaxLifetime: time.Hour,
	}
}

func dialCurrentWorkflow(ctx context.Context, address, namespace string) (client.Client, func() error, error) {
	current, err := client.DialContext(ctx, client.Options{HostPort: address, Namespace: namespace})
	if err != nil {
		return nil, nil, err
	}
	return current, func() error { current.Close(); return nil }, nil
}
