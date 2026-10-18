package tests

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestListingKitTaskBackendSourcesAreRetired(t *testing.T) {
	rootFiles, err := filepath.Glob(filepath.Join("..", "internal", "listingkit", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rootFiles) != 0 {
		t.Errorf("mixed ListingKit root must be absent, found %d Go files", len(rootFiles))
	}
	rootProcessorFiles, err := filepath.Glob(filepath.Join("..", "internal", "temu", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rootProcessorFiles) != 0 {
		t.Errorf("retired TEMU task processor root remains: %v", rootProcessorFiles)
	}
	for _, relative := range []string{
		"internal/listingkit/api", "internal/listingkit/httpapi", "internal/listingkit/core",
		"internal/listingkit/temporal", "internal/listingkit/workflow", "internal/listingkit/workspace",
		"internal/listingkit/sheinadapter",
		"cmd/listing-scheduler", "cmd/shein-listing", "cmd/temu-listing",
		"cmd/listing-control-plane", "cmd/shein-import-platform-recovery",
		"internal/app/runtime/listingcontrol", "internal/app/runtime/sheinplatformrecovery", "internal/listingcontrol",
		"internal/app/task", "internal/app/taskstatus", "internal/processor", "internal/app/crawler/distributed",
		"internal/amazon/processor.go", "internal/amazon/task_status.go", "internal/amazon/task_status_test.go",
		"internal/crawler/fetcher/distributed_fetcher.go", "internal/crawler/fetcher/builder_test.go",
		"internal/temu/sync/inventory_sync_factory.go",
		"internal/shein/inventory", "internal/temu/sync", "internal/temu/bulkrelist",
		"internal/temu/category", "internal/temu/template",
		"internal/temu/image/init_handler.go", "internal/temu/image/validator.go",
		"internal/temu/image/upload_worker.go", "internal/temu/image/main_validator.go",
		"internal/temu/image/sku_validator.go", "internal/temu/image/parallel_validator.go",
		"internal/temu/image/single_validator.go",
		"internal/temu/product/save_handler.go", "internal/temu/product/submit_handler.go",
		"internal/temu/product/commit_create_handler.go", "internal/temu/product/commit_detail_handler.go",
		"internal/temu/product/save_publish_result_handler.go", "internal/temu/product/publish_result_steps.go",
		"internal/temu/product/publish_result_input.go", "internal/temu/product/publish_result_input_test.go",
		"internal/temu/product/request_models.go", "internal/temu/product/result_output.go",
		"internal/temu/product/submit_utils.go", "internal/temu/product/submit_error_analyzer.go",
		"internal/temu/product/submit_fixer.go", "internal/temu/product/submit_validator.go",
		"internal/temu/product/check_daily_limit_handler.go",
		"internal/temu/product/exists_check_handler.go", "internal/temu/product/out_goods_sn_check_handler.go",
		"internal/temu/product/raw_json_data_handler.go", "internal/temu/product/cache_handler.go",
		"internal/temu/product/build_spu_handler.go", "internal/temu/product/spu_builder.go",
		"internal/temu/product/spu_validator.go", "internal/temu/product/price_query_handler.go",
		"internal/temu/sku/variant_json_data_handler.go", "internal/temu/sku/parallel_variant_handler.go",
		"internal/temu/sku/cache_variants_handler.go", "internal/temu/sku/variant_data_processor.go",
		"internal/temu/sku/variant_filter_handler.go", "internal/temu/sku/ai_mapping_handler.go",
		"internal/temu/sku/builder.go", "internal/temu/sku/item_builder.go", "internal/temu/sku/skc_builder.go",
		"internal/temu/sku/parallel_builder.go", "internal/temu/sku/build_input.go", "internal/temu/sku/price_calculator.go",
		"internal/temu/filter/rule_handler.go", "internal/temu/filter/rule_manager.go", "internal/temu/filter/rule_stats_provider.go",
		"internal/temu/product/filter_checker.go", "internal/temu/rules/rule_validator.go", "internal/temu/rules/rule_validator_test.go",
		"internal/temu/handlerbase/fulfillment_checker.go", "internal/temu/handlerbase/filter_types.go",
		"internal/temu/store", "internal/temu/handlerbase/init_handler.go",
		"internal/shein/pricing/auto_pricing.go", "internal/shein/pricing/pricing_evaluator.go",
		"internal/shein/pricing/pricing_calculator.go", "internal/shein/pricing/pricing_builder.go",
		"internal/shein/pricing/time_helper.go", "internal/shein/pricing/auto_pricing_repo_test.go",
		"internal/shein/pricing/time_helper_test.go",
		"internal/shein/activity/registration.go", "internal/shein/activity/mixed.go",
		"internal/shein/activity/profit.go", "internal/shein/activity/product_data_helper.go",
		"internal/shein/activity/config.go", "internal/shein/activity/config_test.go",
		"internal/shein/activity/repository_test.go",
		"internal/temu/pricing/auto_pricing_service.go", "internal/temu/pricing/pricing_decision_service.go",
		"internal/temu/pricing/pricing_rules.go", "internal/temu/pricing/pricing_data_service.go",
		"internal/temu/pricing/store_config_service.go", "internal/temu/pricing/runtime_adapter.go",
		"internal/temu/pricing/pricing_rules_test.go", "internal/temu/pricing/pricing_data_service_test.go",
		"internal/temu/pricing/store_config_service_test.go",
		"internal/shein/productsync/product_sync.go", "internal/shein/productsync/product_sync_fetcher.go",
		"internal/shein/productsync/product_sync_enricher.go", "internal/shein/productsync/product_snapshot.go",
		"internal/shein/productsync/product_snapshot_repository.go",
		"internal/shein/productsync/product_sync_repository_test.go", "internal/shein/productsync/product_sync_save_test.go",
		"internal/shein/productsync/product_snapshot_test.go", "internal/shein/productsync/product_snapshot_repository_test.go",
		"internal/listingadmin/inventory_record.go", "internal/listingadmin/inventory_record_repository.go",
		"internal/listingadmin/gorm_inventory_record_api.go", "internal/listingadmin/inventory_record_query_test.go",
		"internal/listingadmin/platform_recovery.go", "internal/listingadmin/platform_recovery_test.go",
		"internal/listingadmin/import_task.go", "internal/listingadmin/import_task_model.go",
		"internal/listingadmin/import_task_query.go", "internal/listingadmin/import_task_repository.go",
		"internal/listingadmin/dispatch_event_repository.go", "internal/listingadmin/paused_task_recovery.go",
		"internal/listingadmin/scheduled_task_config.go", "internal/listingadmin/scheduled_task_config_model.go",
		"internal/listingadmin/scheduled_task_config_query.go", "internal/listingadmin/scheduled_task_config_repository.go",
		"internal/listingadmin/store_statistics.go", "internal/listingadmin/runtime_prerequisites.go",
		"internal/listingadmin/import_task_dispatch_test.go", "internal/listingadmin/import_task_model_test.go",
		"internal/listingadmin/import_task_query_test.go", "internal/listingadmin/dispatch_event_repository_test.go",
		"internal/listingadmin/paused_task_recovery_test.go", "internal/listingadmin/scheduled_task_config_repository_test.go",
		"internal/listingadmin/store_statistics_test.go",
		"internal/app/runtime/listing", "internal/app/runtime/listingscheduler",
		"internal/app/bootstrap", "internal/app/consumer", "internal/app/runner", "internal/app/scheduler",
		"internal/listingruntime/local", "internal/listingkit/sheinsync", "internal/listingkit/store",
		"internal/platforms", "internal/shein/scheduler", "internal/temu/scheduler",
		"internal/platformtask", "internal/scheduler", "internal/shein/pipeline",
		"internal/platformbase/base_factory.go", "internal/platformbase/display.go", "internal/platformbase/task_type.go",
		"internal/app/httpapi/bootstrap.go", "internal/app/httpapi/image_agent_asset_catalog.go",
		"internal/pkg/httpapicmd", "cmd/product-listing-api", "scripts/listingkit-shein-pod-image-index-backfill",
	} {
		t.Run(relative, func(t *testing.T) {
			path := filepath.Join("..", filepath.FromSlash(relative))
			err := filepath.WalkDir(path, func(name string, entry os.DirEntry, err error) error {
				if os.IsNotExist(err) {
					return nil
				}
				if err != nil {
					return err
				}
				if !entry.IsDir() && strings.HasSuffix(name, ".go") {
					t.Errorf("retired Task backend source remains: %s", name)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	for relative, retired := range map[string][]string{
		"internal/listingruntime/types.go": {
			"type ImportTask struct", "type ScheduledTaskConfig struct", "type TaskStatusUpdate struct",
		},
		"internal/listingadmin/schema_migrate.go": {"ImportTask", "importTask"},
		"internal/listingadmin/management_api_types.go": {
			"type InventoryRecordCreateReqDTO struct", "type InventoryRecordRespDTO struct", "type InventoryRecordAPI interface",
		},
		"internal/temu/pricing/interfaces.go": {
			"type StoreConfigProvider interface", "type ProductDataProvider interface", "type ProductFetcher interface",
			"type DecisionMaker interface", "type PricingProcessor interface",
		},
		"internal/app/runtime/listingkitschemamigrate/runtime_test.go": {
			"func openRuntimeSchemaTestDB(", "\"listing_product_import_task\"",
		},
	} {
		content, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		for _, token := range retired {
			if strings.Contains(string(content), token) {
				t.Errorf("%s must not recreate retired numeric task contract %q", relative, token)
			}
		}
	}
	for relative, required := range map[string][]string{
		"internal/temu/api/api.go": {
			"InventoryAPI", "inventory.API", "inventory.RelistRequest", "inventory.RelistResponse",
			"product.BulkRelistOptions", "product.BulkRelistSummary",
			"product.SaveRequest", "product.SaveResponse", "product.SaveResult", "product.SubmitRequest", "product.SubmitResponse",
			"product.CreateCommitRequest", "product.CreateCommitResponse", "query.CommitDetailRequest", "query.CommitDetailResponse",
			"NewProductAPI", "NewSubmitAPI", "product.NewAPI",
			"query.SkuSnCheckRequest", "query.SkuSnCheckResponse", "query.OutSkuSnItem", "query.OutGoodsSnCheckResult",
			"product.PriceQueryRequest", "product.PriceQueryResponse", "product.MaxRetailPriceQueryItem", "product.MaxRetailPriceResultItem",
			"CategoryAPI", "category.API", "NewCategoryAPI", "category.NewAPI",
			"QueryAPIInterface", "query.QueryAPI", "NewQueryAPI", "query.NewAPI",
			"query.CostTemplateRequest", "query.CostTemplateResponse",
			"ImageAPI", "image.API", "NewImageAPI", "image.NewAPI",
			"image.UploadSignature", "image.UploadResult", "image.ValidationResult",
		},
		"internal/temu/api/category/category.go": {
			"type API struct", "func NewAPI(", "func (a *API) Recommend(", "func (a *API) GetDisclaimer(",
			"type RecommendRequest struct", "type RecommendResponse struct", "type RecommendResult struct", "type Category struct",
			"type DisclaimRequest struct", "type DisclaimResponse struct", "type DisclaimResult struct", "type DisclaimerDTO struct",
		},
		"internal/temu/api/product/api.go": {
			"func (a *API) Save(", "func (a *API) Submit(", "func (a *API) CreateCommit(",
			"func (a *API) QueryMaxRetailPrice(",
		},
		"internal/temu/api/query/query.go": {
			"func (a *API) QueryCommitDetail(", "func (a *API) CheckSkuSn(", "func (a *API) QuerySpec(",
			"func (a *API) QueryTemplate(", "func (a *API) QueryTemplateAdvanced(", "func (a *API) QueryCostTemplate(",
		},
		"internal/temu/api/query/query_types.go": {
			"type CommitDetailRequest struct", "type CommitDetailResponse struct", "type CommitDetailResult struct", "type CommitDetailCategoryTree struct",
			"type SkuSnCheckRequest struct", "type SkuSnCheckResponse struct", "type OutSkuSnItem struct",
			"type OutGoodsSnCheckResult struct", "type OutSkuSnFailItem struct",
			"type TemplateQueryRequest struct", "type TemplateQueryResponse struct", "type TemplateInfo struct",
			"type CostTemplateRequest struct", "type CostTemplateResponse struct", "type CostTemplateResult struct", "type CostTemplateItem struct",
		},
		"internal/temu/api/template/types.go": {
			"type TemplateQueryRequest struct", "type TemplateQueryResponse struct", "type TemplateQueryResult struct",
			"type TemplateInfo struct", "type UserInputParentSpec struct",
		},
		"internal/temu/context/temu_context.go": {
			"type TemuTaskContext struct", "type PublishState struct", "PriceQueryResponse", "TemuProduct", "StoreInfo",
			"RuntimeState", "ProductState", "TemplateState", "AssetState", "VariantState", "RuleState", "AISkuMapping",
			"SetAPIClients", "AttachRuntime", "SetAmazonProduct", "SetVariants", "SetTemuProduct",
			"type TemplateState struct", "TemplateInfo", "UserInputParentSpecList", "InputMaxSpecNum", "SingleSpecValueNum", "QueryAPI",
			"PaddedImages", "PaddedImageSizes", "make(map[string][]byte)", "make(map[string][2]int)",
		},
		"internal/temu/image/config.go":              {"type imageConfig struct", "var defaultImageConfig", "func isCDNImage(", "func needsUpload("},
		"internal/temu/image/processor.go":           {"type ImageProcessor struct", "func NewImageProcessor("},
		"internal/temu/image/parallel_processor.go":  {"type ParallelImageProcessor struct", "func (pip *ParallelImageProcessor) ApplyImageResults("},
		"internal/temu/image/carousel_builder.go":    {"type ImageCarouselBuilder struct"},
		"internal/temu/image/dimension_builder.go":   {"type ImageDimensionBuilder struct", "PaddedImages"},
		"internal/temu/image/dimension_models.go":    {"type DimensionInfo = handlerbase.DimensionInfo"},
		"internal/temu/image/dimension_annotator.go": {"type ImageDimensionAnnotator struct", "func NewImageDimensionAnnotatorWithOpenAI("},
		"internal/temu/image/dimension_drawer.go":    {"type DimensionDrawer struct"},
		"internal/temu/image/drawing_utils.go":       {"type DrawingUtils struct"},
		"internal/temu/image/padding_processor.go":   {"type ImagePaddingProcessor struct", "func (p *ImagePaddingProcessor) PadImageToAspectRatio("},
		"internal/temu/image/vision_detector.go":     {"type VisionDetector struct", "openaiClient.ChatCompleter"},
		"internal/temu/image/upload_processor.go":    {"type ImageUploadProcessor struct", "func (h *ImageUploadProcessor) UploadSingleImage(", "uploadResult, err :=", "PaddedImages", "PaddedImageSizes"},
		"internal/temu/image/upload_utils.go":        {"type ImageUploadUtils struct", "func (iuu *ImageUploadUtils) padImagesIfNeeded("},
		"internal/temu/image/upload_helpers.go":      {"type ImageUploadHelpers struct"},
		"internal/temu/image/upload_service.go":      {"type TemuAPIClient interface", "func getUploadSignature(", "func uploadImageWithSignature("},
		"internal/temu/api/image/image.go": {
			"type API struct", "type ValidationResult struct", "type UploadResult struct", "type UploadSignature struct",
			"func (a *API) GetUploadSignature(", "func (a *API) UploadWithSignature(", "func (a *API) Upload(",
		},
		"internal/temu/handlerbase/types.go": {"type ImageRequirement struct"},
		"internal/marketplace/temu/goods/images.go": {
			"func ValidImageObservation(", "func resolveImages(", "inventory asset.ApprovedAssetInventory",
		},
		"internal/temu/context/sku_runtime_input.go":         {"type SKUBuildRuntime struct", "func BuildSKUBuildRuntime(", "StoreInfo"},
		"internal/temu/handlerbase/base_handler.go":          {"type BaseTemuHandler struct", "func NewBaseTemuHandler(", "func (h *BaseTemuHandler) GetTemuContext("},
		"internal/temu/handlerbase/temu_handler.go":          {"type TemuHandler interface"},
		"internal/temu/handlerbase/sku_builder_interface.go": {"type SkuBuilder interface", "type SpecHandler interface"},
		"internal/temu/product/price_handler.go": {
			"type PriceHandler struct", "func NewPriceHandler(", "func (ph *PriceHandler) CalculateVariantPriceWithRuntime(",
			"func (ph *PriceHandler) GetDefaultStockWithRuntime(", "func (ph *PriceHandler) GetPriceMultiplierWithRuntime(",
		},
		"internal/temu/sku/spec_handler.go": {
			"type SkuSpecHandler struct", "func NewSkuSpecHandler(", "func (sh *SkuSpecHandler) ValidateSpecs(",
			"func (sh *SkuSpecHandler) IsSizeSpec(", "func (sh *SkuSpecHandler) convertUserInputSpecsToGoodsSpecProperties(", "func (sh *SkuSpecHandler) CreateDefaultSpec(",
		},
		"internal/temu/sku/convert.go":             {"func convertSpecInfos(", "ParentSpecName: s.ParentSpecName", "ParentID:       s.ParentID"},
		"internal/temu/spec/resolver_service.go":   {"type SpecResolverService struct", "func (s *SpecResolverService) HasTemporaryIDs("},
		"internal/temu/property/mapper_adapter.go": {"type PropertyMapper struct", "func (m *PropertyMapper) BuildGoodsProperties("},
		"internal/temu/ai/content_rewriter.go":     {"type AIContentRewriter struct", "func NewAIContentRewriterHandler("},
		"internal/shein/product/build/spu.go":      {"type BuildSpuHandler struct", "func NewBuildSpuHandler("},
		"internal/temu/api/client/client.go": {
			"func NewAPIClient(", "func (c *APIClient) initializeMallID(",
			"func (c *APIClient) GetMallID(", "func (c *APIClient) SetMallID(", ".UpdateStoreId(",
		},
		"internal/temu/api/client/interfaces.go":     {"type StoreRuntime interface", "GetStoreAPI("},
		"internal/temu/api/client/cookie_manager.go": {"type CookieManager struct"},
		"internal/temu/api/product/goods_types.go": {
			"type Product struct", "type GoodsBasicInfo struct", "type ServicePromise struct", "type ExtraInfo struct",
			"type CategoryTree struct", "type Disclaimer struct",
		},
		"internal/shared/ptr/pointer.go":            {"func BoolPtr("},
		"internal/listingadmin/gorm_store_api.go":   {"func NewGormStoreAPI(", "func (a gormStoreAPI) GetStore(", "func (a gormStoreAPI) UpdateStoreId(", "func (a gormStoreAPI) UpdateStoreStatus("},
		"internal/listingadmin/store_repository.go": {"type GormStoreRepository struct", "func (r *GormStoreRepository) GetStore("},
		"internal/storecenter/gorm_repository.go":   {"type GormStoreRepository struct"},
		"internal/shein/store/store_info.go":        {"type StoreInfoHandler struct", "func NewStoreInfoHandler(", "func GetStoreCacheStats("},
		"internal/shein/store/store_id.go":          {"type StoreIDHandler struct", "func NewStoreIDHandler(", "func GetInstanceID("},
		"internal/amazon/pipeline/store_info.go":    {"type StoreInfoHandler struct", "func NewStoreInfoHandler("},
		"internal/pkg/appenv/apputil.go":            {"func GetInstanceID("},
		"internal/marketplace/temu/publishing/submission_validation_policy.go": {
			"type submissionSpec struct", "func validateSubmissionSpecs(", "func validateSubmissionPackaging(",
		},
		"internal/marketplace/temu/publishing/spu_validation_policy.go": {"func validateSPUStructure("},
		"internal/marketplace/temu/goods/builder.go": {
			"func Build(", "snapshot.SelectableLeaf", "func checkLocal(",
			"p.ServicePromise.ShipmentLimitDay", "value.ID == p.ServicePromise.CostTemplateID",
		},
		"internal/marketplace/temu/goods/supplier_category.go": {"func checkSupplierCategory(", "SelectableLeaf()"},
		"internal/marketplace/temu/rules/local_variations.go":  {"func CheckLocalVariations("},
		"internal/marketplace/temu/publishing/sku_value_policy.go": {
			"func deduplicateSKUSpecs(", "func hasUnresolvedSKUSpecs(", "func normalizeSingleItemPackaging(",
		},
		"internal/marketplace/temu/publishing/sku_sn_validation_policy.go": {
			"func buildUniqueSKUSNs(", "func validateSKUSNFailures(", "type skuSNFailure struct",
		},
		"internal/pkg/skugen/skugen.go": {"func Generate(", "StrategyASINOnly"},
		"internal/crawler/fetcher/contracts.go": {
			"type ProductReader interface", "type ProductCache interface", "type ProductFetcherStats interface",
			"FetchVariants(", "CacheVariants(",
		},
		"internal/marketplace/sourceproduct/product_fetcher.go": {
			"func (f *ProductFetcher) FetchVariants(", "func (f *ProductFetcher) CacheVariants(", "VariantFetchRequest(",
		},
		"internal/marketplace/sourceproduct/cache_manager.go": {"func (c *CacheManager) CacheVariants("},
		"internal/crawler/fetcher/remote_fetcher.go": {
			"func (f *RemoteAPIProductFetcher) FetchVariants(", "func (f *RemoteAPIProductFetcher) CacheVariants(",
		},
		"internal/temu/sku/variant_processor.go": {
			"type SkuVariantProcessor struct", "func NewSkuVariantProcessor(", "NewSkuMappingProcessor(",
			"func (vp *SkuVariantProcessor) GenerateAISkuMapping(", "func (vp *SkuVariantProcessor) validateAIMappingSpecs(",
			"func (vp *SkuVariantProcessor) prepareAIMappingForBuild(", "func (vp *SkuVariantProcessor) resolveAIMappingSpecIDs(",
			"func (vp *SkuVariantProcessor) ensureAIMappingMatchesVariants(", "func (vp *SkuVariantProcessor) normalizeAIMappingForBuild(",
			"func (vp *SkuVariantProcessor) unifyAIMappingSpecDimensions(",
		},
		"internal/temu/sku/ai_mapping_handler_input.go": {
			"type AIBatchInput struct", "func buildAIBatchInput(",
			"func (input *AIBatchInput) TotalBatches(", "func (input *AIBatchInput) BatchVariants(",
		},
		"internal/temu/sku/ai_mapping_batch_processor.go": {"buildAIBatchInput(", ".TotalBatches(", ".BatchVariants("},
		"internal/temu/sku/ai_mapping_input.go":           {"type AIMappingInput struct", "func BuildAIMappingInput("},
		"internal/marketplace/productpolicy/rule_checker.go": {
			"type RuleChecker struct", "func NewRuleChecker(", "CheckPriceRange(", "CheckInventory(", "CheckRating(",
			"CheckReviewCount(", "CheckDeliveryTime(", "CheckFulfillmentType(", "CheckAllRules(",
		},
		"internal/marketplace/productpolicy/filter_rule.go":  {"type FilterRule struct"},
		"internal/marketplace/productpolicy/price_helper.go": {"func GetProductPrice(", "func GetInventory("},
		"internal/marketplace/productpolicy/fulfillment.go":  {"func IsFBAFulfillment(", "func IsAMZSeller("},
		"internal/listingadmin/rule_status.go": {
			"RuleStatusEnabled int16 = 0", "RuleStatusDisabled int16 = 1", "func IsRuleStatusEnabled(",
		},
		"internal/listingadmin/gorm_filter_rule_api.go":   {"func NewGormFilterRuleAPI(", ".ResolveFilterRules("},
		"internal/listingadmin/filter_rule_repository.go": {"func (r *GormFilterRuleRepository) ResolveFilterRules("},
		"internal/shein/validation/rule_checker.go":       {"productpolicy.RuleChecker", "productpolicy.NewRuleChecker("},
		"internal/shein/validation/filter_rule.go":        {"NewFilterRuleChecker(", ".ToFilterRule(", ".CheckPriceRange(", ".CheckInventory("},
		"internal/shein/validation/reapply_filter.go":     {"NewFilterRuleChecker(", ".ToFilterRule(", ".CheckPriceRange(", ".CheckInventory("},
		"internal/temu/sku/ai_mapping_utils_helper.go": {
			"func (vp *SkuVariantProcessor) getProductDimensions(", "func (vp *SkuVariantProcessor) getItemWeight(",
			"func (vp *SkuVariantProcessor) getProductDetailField(", "func (vp *SkuVariantProcessor) marshalWithoutHTMLEscape(",
			"func (vp *SkuVariantProcessor) buildAIVariant(",
		},
		"internal/temu/sku/ai_mapping_single_processor.go": {"vp.getProductDimensions(", "vp.getItemWeight(", "vp.buildAIVariant("},
		"internal/shared/strx/cleaner.go":                  {"func CleanProductTitle("},
		"internal/temu/rules/text_processor.go":            {"strx.CleanProductTitle(", "type TextProcessor struct", "func (tp *TextProcessor) ProcessProductTitle("},
		"internal/shein/productdata/variant_data.go": {
			"type VariantJsonDataHandler struct", "func NewVariantJsonDataHandler(", "func GetVariantByAsinFromVariants(",
		},
		"internal/shein/productdata/fetch_and_cache_variants.go": {
			"type FetchAndCacheVariantsHandler struct", "h.reader.FetchVariants(", "h.cache.CacheVariants(",
		},
		"internal/shein/publish/mapping_helper.go": {"productdata.GetVariantByAsinFromVariants("},
		"internal/listingadmin/management_api_types.go": {
			"type ProductImportMappingAPI interface", "CheckProductExists(",
			"type FilterRuleAPI interface", "type FilterRuleReqDTO struct", "type FilterRuleRespDTO struct", "ToFilterRule(",
			"type StoreAPI interface", "type StoreRespDTO struct", "type StoreIdUpdateReqDTO struct", "type StoreStatusUpdateReqDTO struct",
			"type ProfitRuleAPI interface", "type ProfitRuleReqDTO struct", "type ProfitRuleRespDTO struct",
		},
		"internal/shein/publish/exists_check.go": {"type ProductExistsCheckHandler struct", "func NewProductExistsCheckHandler("},
		"internal/temu/api/inventory/inventory.go": {
			"type RelistRequest struct", "type RelistResponse struct", "type SearchRequest struct", "type SearchResponse struct",
			"func (a *API) Relist(", "func (a *API) SearchOffline(",
		},
		"internal/temu/api/product/request_types.go": {
			"type BulkRelistOptions struct", "type SkipConditions struct", "type BulkRelistSummary struct", "type CategoryStats struct",
			"type SaveRequest struct", "type SaveResponse struct", "type SaveResult struct", "type SubmitRequest struct", "type SubmitResponse struct",
			"type CreateCommitRequest struct", "type CreateCommitResponse struct",
			"type PriceQueryRequest struct", "type PriceQueryResponse struct", "type MaxRetailPriceQueryItem struct", "type MaxRetailPriceResultItem struct",
		},
		"internal/shein/activity/price_calculator.go": {"func calculateActivityPrice(", "func amazonCostBySKU("},
		"internal/shein/activity/registration_config.go": {
			"type activityRuleInputs struct", "func minimumPromotionSKUDiscountRate(", "func calculateActivityStock(",
			"func buildActivityConfigsFromProvidedProducts(", "func ensureLimitedPromotionDropRatesGreater(", "func promotionConfigListForActivityType(",
		},
		"internal/shein/activity/promotion_create_products.go": {"func buildCalculateRequestForPromotionProducts(", "func promotionSKUPriceInputs("},
		"internal/shein/activity/time_limited.go":              {"func buildCreateActivityRequest(", "func isTimeLimitedDiscountValid(", "func validatePriceRisk("},
		"internal/shein/activity/validation_utils.go":          {"func ValidateDropRate(", "func CalculateDropRateFromDiscount("},
		"internal/shein/activity/errors.go":                    {"ErrProductPriceRisk"},
		"internal/shein/pricing/price_manager.go":              {"type PriceManager struct", "func NewPriceManager(", "type CostPriceRequest struct"},
		"internal/temu/pricing/interfaces.go": {
			"type PriceCalculator interface",
			"GetDefaultPricingRules(originCostPrice float64, rules *[]listingadmin.PricingRuleRespDTO) *listingadmin.PricingRuleRespDTO",
			"CalculateMinAcceptablePrice(originCostPrice float64, rule *listingadmin.PricingRuleRespDTO) float64",
		},
	} {
		content, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		for _, token := range required {
			if !strings.Contains(string(content), token) {
				t.Errorf("%s must preserve retained platform/shared contract %q", relative, token)
			}
		}
	}
	productRoot := filepath.Join("..", "internal", "temu", "product")
	if err := filepath.WalkDir(productRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, retired := range []string{
			"ProductSaveHandler", "ProductSubmitHandler", "CommitCreateHandler", "CommitDetailHandler", "SavePublishResultHandler", "CheckDailyLimitHandler",
			"ProductRequestInput", "CommitDetailOutput", "SaveProductOutput", "SubmitProductOutput", "SavePublishResultInput",
			"ProductSubmitUtils", "ProductSubmitErrorAnalyzer", "ProductSubmitFixer", "ProductSubmitValidator",
			"ProductExistsCheckHandler", "OutGoodsSnCheckHandler", "RawJsonDataHandlerV2", "CacheProductHandler",
		} {
			if strings.Contains(string(content), retired) {
				t.Errorf("%s must not recreate retired TEMU Task orchestration %q", path, retired)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	retiredTEMUSymbols := map[string]bool{
		"VariantJsonDataHandler": true, "NewVariantJsonDataHandler": true,
		"ParallelVariantHandler": true, "NewParallelVariantHandler": true,
		"CacheVariantsHandler": true, "NewCacheVariantsHandler": true,
		"VariantProcessor": true, "NewVariantProcessor": true, "extractAsinListFromMap": true,
		"VariantFilterHandler": true, "NewVariantFilterHandler": true,
		"AISkuMappingHandler": true, "NewAISkuMappingHandler": true,
		"FilterRuleHandler": true, "NewFilterRuleHandler": true,
		"FilterRuleManager": true, "NewFilterRuleManager": true,
		"FilterRuleStatsProvider": true, "NewFilterRuleStatsProvider": true,
		"ProductFilterChecker": true, "NewProductFilterChecker": true,
		"RuleValidator": true, "NewRuleValidator": true,
		"FulfillmentChecker": true, "NewFulfillmentChecker": true, "buildFulfillmentProductValue": true,
		"FilterCheckResult": true, "AIHandlerInput": true, "buildAIHandlerInput": true,
	}
	retiredProductSPUSymbols := map[string]bool{
		"BuildSpuHandler": true, "NewBuildSpuHandler": true,
		"SpuBuilder": true, "NewSpuBuilder": true,
		"SpuValidator": true, "NewSpuValidator": true,
		"PriceQueryHandler": true, "NewPriceQueryHandler": true,
	}
	for _, pkg := range []string{"sku", "filter", "product", "rules", "handlerbase", "image"} {
		if err := filepath.WalkDir(filepath.Join("..", "internal", "temu", pkg), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				return err
			}
			if pkg == "sku" {
				for _, violation := range retiredSKUConstructionViolations(file) {
					t.Errorf("%s: %s", path, violation)
				}
			}
			if pkg == "image" {
				for _, violation := range retiredImageOrchestrationViolations(file) {
					t.Errorf("%s: %s", path, violation)
				}
			}
			ast.Inspect(file, func(node ast.Node) bool {
				ident, ok := node.(*ast.Ident)
				if !ok {
					return true
				}
				retiredInit := pkg == "handlerbase" && (ident.Name == "InitDataHandler" || ident.Name == "NewInitDataHandler")
				retiredSPU := pkg == "product" && retiredProductSPUSymbols[ident.Name]
				if retiredTEMUSymbols[ident.Name] || retiredInit || retiredSPU {
					t.Errorf("%s must not recreate retired TEMU Task orchestration %q", path, ident.Name)
				}
				return true
			})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	contextPath := filepath.Join("..", "internal", "temu", "context", "temu_context.go")
	contextFile, err := parser.ParseFile(token.NewFileSet(), contextPath, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	retiredFields := map[string]bool{
		"SaveResult": true, "SavedToDraft": true, "CommitDetail": true, "SubmitResponse": true, "ProductData": true,
		"DailyQuotaReserved": true, "DailyQuotaDate": true, "DailyQuotaIncrement": true,
	}
	retiredMethods := map[string]bool{
		"SetPublishProductData": true, "SetSaveResponse": true, "SetCommitDetailResponse": true, "SetSubmitResponse": true,
		"SetSavedToDraft": true, "SetDailyQuotaReservation": true, "ClearDailyQuotaReservation": true, "ApplySaveResult": true,
	}
	retiredAssetFields := map[string]bool{
		"CurrentSkuContext": true, "RequiresImageUpload": true, "TotalImageCount": true,
	}
	ast.Inspect(contextFile, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.TypeSpec:
			if node.Name.Name != "PublishState" && node.Name.Name != "TemuTaskContext" && node.Name.Name != "AssetState" {
				return false
			}
			if state, ok := node.Type.(*ast.StructType); ok {
				for _, field := range state.Fields.List {
					for _, name := range field.Names {
						if retiredFields[name.Name] {
							t.Errorf("TEMU %s must not retain exclusive publish field %s", node.Name, name)
						}
						if (node.Name.Name == "AssetState" || node.Name.Name == "TemuTaskContext") && retiredAssetFields[name.Name] {
							t.Errorf("TEMU %s must not retain exclusive image field %s", node.Name, name)
						}
					}
				}
			}
		case *ast.FuncDecl:
			if node.Recv != nil && retiredMethods[node.Name.Name] {
				t.Errorf("TEMU context must not retain exclusive publish method %s", node.Name)
			}
		}
		return true
	})
	for _, policyName := range []string{"submission_validation_policy.go", "sku_value_policy.go"} {
		policyPath := filepath.Join("..", "internal", "marketplace", "temu", "publishing", policyName)
		policyFile, err := parser.ParseFile(token.NewFileSet(), policyPath, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range policyFile.Imports {
			if imported.Path.Value != "\"fmt\"" && imported.Path.Value != "\"strings\"" {
				t.Errorf("submission validation must remain a value-only rule, found import %s", imported.Path.Value)
			}
		}
		for _, declaration := range policyFile.Decls {
			if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.IsExported() {
				t.Errorf("submission extraction must not add exported contract %s", function.Name)
			}
			if group, ok := declaration.(*ast.GenDecl); ok {
				for _, spec := range group.Specs {
					if typ, ok := spec.(*ast.TypeSpec); ok && typ.Name.IsExported() {
						t.Errorf("submission extraction must not add exported type %s", typ.Name)
					}
				}
			}
		}
	}
	for _, platform := range []string{"shein", "temu"} {
		root := filepath.Join("..", "internal", platform, "pricing")
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			content, err := os.ReadFile(filepath.Join(root, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			for _, retired := range []string{
				"type AutoPricingService ", "NewAutoPricingService", "autoPricingServiceImpl",
				"type PricingDecisionService ", "type PricingRuntime ", "NewPricingRuntime",
				"type PricingDataService ", "type StoreConfigService ", "type PricingRequestBuilder ", "type TimeHelper ",
				"type StoreConfigProvider interface", "type ProductDataProvider interface", "type ProductFetcher interface",
				"type DecisionMaker interface", "type PricingProcessor interface",
			} {
				if strings.Contains(string(content), retired) {
					t.Errorf("%s must not recreate retired automatic pricing %q", filepath.Join(root, entry.Name()), retired)
				}
			}
		}
	}
	activityRoot := filepath.Join("..", "internal", "shein", "activity")
	activityFiles, err := os.ReadDir(activityRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range activityFiles {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(activityRoot, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, retired := range []string{
			"ActivityRegistrationService", "activityRegistrationServiceImpl", "PromotionRegistrationBridge",
			"PromotionRegistrationSession", "promotionRegistrationSession", "PromotionRegistrationResult", "ProductDataHelper",
			"TimeLimitedDiscountConfig", "ErrInvalidActivityName", "ErrInvalidActivityTime", "ErrInvalidTimeZone",
			"ErrNoAvailableProducts", "ErrInsufficientStock", "ErrActivityCreationFailed", "ErrActivityAlreadyExists",
			"buildCalculateRequestWithPriceMode", "GenerateActivityName", "defaultActivityStartLeadTime",
			"\"context\"", "task-processor/internal/listingadmin", "task-processor/internal/core/logger",
			"task-processor/internal/infra/clients/management", "marketing.MarketingAPI", "marketing.NewClient(",
			".SaveConfig(", ".UpdateConfigState(", ".CreateActivity(", ".QueryPromotionGoods(", ".CalculateSupplyPrice(",
		} {
			if strings.Contains(string(content), retired) {
				t.Errorf("%s must not recreate retired SHEIN activity orchestration %q", entry.Name(), retired)
			}
		}
	}
	attributePath := filepath.Join("..", "internal", "shein", "productsync", "product_sync_types.go")
	attributes, err := os.ReadFile(attributePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, retained := range []string{"type EnrichedSkuInfo struct", "type EnrichedSkcInfo struct"} {
		if !strings.Contains(string(attributes), retained) {
			t.Errorf("shared SHEIN attribute contract must remain: %s", retained)
		}
	}
	entries, err := os.ReadDir(filepath.Dir(attributePath))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(filepath.Dir(attributePath), entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, retired := range []string{"type ProductSyncService interface", "func NewProductSyncService(", "type ProductSnapshot struct", "func sheinProductDataFromSnapshot("} {
			if strings.Contains(string(content), retired) {
				t.Errorf("%s must not recreate retired product synchronization %q", entry.Name(), retired)
			}
		}
	}
}

func TestCurrentCommandsDoNotCompileLegacyBusinessBackends(t *testing.T) {
	for _, relative := range []string{
		"internal/crawler/fetcher/contracts.go", "internal/crawler/fetcher/remote_fetcher.go",
		"internal/platformbase/fetcher_mode.go", "internal/platformbase/fetcher_builder.go",
		"internal/amazon/api/client.go", "internal/amazon/pipeline/manager.go",
		"internal/listingruntime/types.go", "internal/listingadmin/import_task_owner.go",
		"internal/listingadmin/import_task_owner_test.go", "internal/listingadmin/schema_migrate.go",
		"internal/listingadmin/product_data_repository.go", "internal/listingadmin/product_import_mapping_repository.go",
		"internal/core/config/type_platform.go",
		"internal/temu/api/api.go", "internal/temu/api/client/interfaces.go",
		"internal/temu/api/client/client.go", "internal/temu/api/client/cookie_manager.go",
		"internal/temu/api/client/auth_pause_handler.go", "internal/temu/api/inventory/inventory.go",
		"internal/temu/api/product/request_types.go",
		"internal/temu/api/product/api.go", "internal/temu/api/query/query.go", "internal/temu/api/query/query_types.go",
		"internal/temu/context/temu_context.go",
		"internal/marketplace/temu/publishing/submission_validation_policy.go",
		"internal/marketplace/temu/publishing/submission_validation_policy_test.go",
		"internal/marketplace/temu/publishing/sku_sn_validation_policy.go",
		"internal/marketplace/temu/publishing/sku_sn_validation_policy_test.go",
		"internal/shein/productsync/product_sync_types.go", "internal/shein/productsync/marketing_client_boundary_test.go",
		"internal/shein/api/marketing/client.go", "internal/shein/api/marketing/client_test.go",
		"internal/shein/api/marketing/interface.go", "internal/shein/api/marketing/activity_types.go",
		"internal/shein/api/marketing/price_types.go", "internal/shein/api/marketing/promotion_types.go",
		"internal/shein/api/marketing/workflow_example.go",
		"internal/shein/activity/price_calculator_test.go", "internal/shein/activity/activity_stock_validation_test.go",
		"internal/shein/activity/registration_direct_products_test.go", "internal/shein/activity/validation_utils_test.go",
		"internal/shein/pricing/price_manager.go", "internal/shein/pricing/pricing_repo.go",
		"internal/shein/pricing/calculator.go", "internal/shein/pricing/calculator_test.go",
		"internal/shein/pricing/cost_profit_calculator.go", "internal/shein/pricing/cost_profit_calculator_test.go",
		"internal/shein/product/product_repo.go",
		"internal/temu/pricing/interfaces.go", "internal/temu/pricing/pricing_rule_calculator.go",
		"internal/temu/pricing/pricing_rule_calculator_test.go",
		"internal/marketplace/temu/publishing/pricing_policy.go", "internal/marketplace/temu/publishing/pricing_policy_test.go",
	} {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(relative))); err != nil {
			t.Errorf("retained platform/shared contract %s must remain: %v", relative, err)
		}
	}
	for _, relative := range []string{".github/workflows/ci.yml", ".github/workflows/commercial-readiness.yml"} {
		content, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), "go test -race ./internal/listingadmin -count=1") {
			t.Errorf("%s must run the retained listingadmin tests with the race detector", relative)
		}
		for _, retired := range []string{
			"TestConcurrentClaimForDispatchOnlyOneWorkerWins",
			"TestConcurrentRollbackDispatchOnlyOriginalQueuedClaimIsRestoredOnce",
			"TestConcurrentRecoveryOnlyUpdatesStillEligibleRowsOnce",
		} {
			if strings.Contains(string(content), retired) {
				t.Errorf("%s must not select retired task test %q", relative, retired)
			}
		}
	}
	for _, command := range []string{"current-application", "shein-login-worker", "image-agent-temporal-worker", "source-account-registry-schema-init", "1688-local-agent", "1688-batch-import", "1688-public-browser-collector"} {
		t.Run(command, func(t *testing.T) {
			cmd := exec.Command("go", "list", "-deps", "./cmd/"+command)
			cmd.Dir = ".."
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go list current command: %v\n%s", err, output)
			}
			for _, dependency := range strings.Fields(string(output)) {
				for _, retired := range []string{"task-processor/internal/shein/inventory", "task-processor/internal/temu/sync", "task-processor/internal/temu/bulkrelist", "task-processor/internal/temu/store", "task-processor/internal/app/bootstrap", "task-processor/internal/app/consumer", "task-processor/internal/app/runner", "task-processor/internal/app/scheduler", "task-processor/internal/app/runtime/listingcontrol", "task-processor/internal/app/runtime/sheinplatformrecovery", "task-processor/internal/listingcontrol", "task-processor/internal/app/task", "task-processor/internal/app/taskstatus", "task-processor/internal/processor", "task-processor/internal/app/crawler/distributed", "task-processor/internal/listingruntime/local", "task-processor/internal/platforms", "task-processor/internal/listingkit", "task-processor/internal/tenantbridge", "task-processor/internal/compatibility", "task-processor/internal/sourceaccount", "task-processor/internal/integration/persistence/sourceaccount/ownershipmigration"} {
					if dependency == retired || strings.HasPrefix(dependency, retired+"/") {
						t.Errorf("%s still compiles retired business dependency %s", command, dependency)
					}
				}
			}
		})
	}
}

func TestStandaloneSheinWorkerDoesNotCompileCurrentHTTPApplication(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "./cmd/shein-login-worker")
	cmd.Dir = ".."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list standalone SHEIN worker: %v\n%s", err, output)
	}
	for _, dependency := range strings.Fields(string(output)) {
		if dependency == "task-processor/internal/sourceaccount" || strings.HasPrefix(dependency, "task-processor/internal/sourceaccount/") {
			t.Errorf("standalone SHEIN worker still compiles retired source-account persistence %s", dependency)
		}
		if dependency == "task-processor/internal/app/httpapi" {
			t.Fatal("standalone SHEIN worker must use its own command runtime, not the current HTTP application assembly")
		}
		if dependency == "task-processor/internal/tenantbridge" || strings.HasPrefix(dependency, "task-processor/internal/tenantbridge/") {
			t.Errorf("standalone SHEIN worker still compiles legacy numeric tenant transport %s", dependency)
		}
	}
	for _, name := range []string{"shein_login_worker.go", "shein_login_worker_test.go"} {
		if _, err := os.Stat(filepath.Join("..", "internal", "app", "httpapi", name)); !os.IsNotExist(err) {
			t.Fatalf("shared HTTP worker source %s must be retired, without a forwarding wrapper", name)
		}
	}
	for _, name := range []string{"api.go", "api_test.go", "http_module.go", "http_module_test.go", "tenant_context.go", "tenant_context_test.go"} {
		if _, err := os.Stat(filepath.Join("..", "internal", "sheinlogin", name)); !os.IsNotExist(err) {
			t.Errorf("unused SHEIN login HTTP source %s must be retired; the standalone worker retains its service and health endpoint", name)
		}
	}
}

func TestLegacyReleaseIdentityPreflightIsRetired(t *testing.T) {
	for _, relative := range []string{
		"cmd/listingkit-identity-preflight", "internal/app/runtime/listingkitidentitypreflight",
		"internal/listingkit/identitypreflight", "internal/listingkit/ownerreconcile",
		"internal/listingkit/userdirectory", "internal/tenantbridge",
		"deployments/docker/Dockerfile.listingkit-identity-preflight",
		"deployments/kubernetes/listingkit-workbench/jobs/listingkit-identity-preflight-job.yaml",
		"scripts/listingkit-identity-preflight-job.sh", "scripts/tests/listingkit-identity-preflight-job-test.sh",
		"scripts/listingkit-clean-legacy-identity-secret.sh", "scripts/tests/listingkit-clean-legacy-identity-secret-test.sh",
	} {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Errorf("retired identity-preflight artifact remains: %s (stat: %v)", relative, err)
		}
	}
	for _, relative := range []string{
		"deployments/docker/Dockerfile.product-listing-api",
		"deployments/kubernetes/listingkit-workbench/release-authority/release-policy.yaml",
		"deployments/kubernetes/listingkit-workbench/release-authority/listingkit-api-release-role.yaml",
		"deployments/kubernetes/listingkit-workbench/release-authority/listingkit-release-gate-runners.yaml",
	} {
		content, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "listingkit-identity-preflight") {
			t.Errorf("%s still packages or authorizes retired identity preflight", relative)
		}
	}
}

func TestListingAdminLegacyHTTPProjectionIsRetired(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "internal", "listingadmin", "*handler*.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("retired ListingAdmin HTTP projection remains: %v", files)
	}
	for _, name := range []string{"request_context.go", "list_query_scope.go"} {
		if _, err := os.Stat(filepath.Join("..", "internal", "listingadmin", name)); !os.IsNotExist(err) {
			t.Fatalf("retired ListingAdmin HTTP helper %s must be absent", name)
		}
	}
}

func TestCurrentBuildDoesNotPackageRetiredTaskEntrypoints(t *testing.T) {
	for _, relative := range []string{"Makefile", "deployments/docker/Dockerfile.product-listing-api", "scripts/test-fast.ps1"} {
		contents, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		for _, retired := range []string{"./cmd/product-listing-api ", "./cmd/product-listing-api\"", "cmd/product-listing-api/", "./internal/listingkit ", "./internal/listingkit\"", "listingkit-shein-pod-image-index-backfill"} {
			if strings.Contains(string(contents), retired) {
				t.Errorf("%s still builds or packages retired entrypoint %q", relative, retired)
			}
		}
	}
}

func TestRetiredLocalLaunchersRejectBeforeOperations(t *testing.T) {
	for _, name := range []string{"start-listingkit-local-api.ps1", "start-listingkit-local-dev.ps1"} {
		content, err := os.ReadFile(filepath.Join("..", "scripts", name))
		if err != nil {
			t.Fatal(err)
		}
		source := string(content)
		retired := strings.Index(source, "throw \"Legacy ListingKit")
		firstFunction := strings.Index(source, "function ")
		if retired < 0 || firstFunction < 0 || retired > firstFunction {
			t.Errorf("%s must reject the retired entry before process, config or connection operations", name)
		}
	}
	if _, err := os.Stat(filepath.Join("..", "scripts", "start-listingkit-api-local-replay.ps1")); !os.IsNotExist(err) {
		t.Fatal("dedicated retired API replay entry must be absent")
	}
}

func TestRetiredListingCLIsHaveNoBuildOrDeploymentEntrypoints(t *testing.T) {
	for _, relative := range []string{
		"deployments/docker/Dockerfile.listing",
		"deployments/kubernetes/shein-listing/base/deployment.yaml",
		"deployments/kubernetes/temu-listing/base/deployment.yaml",
		"deployments/kubernetes/shein-listing/overlays/prod-auto-shard-statefulset/statefulset.yaml",
		"deployments/kubernetes/shein-listing/overlays/prod-auto-shard-statefulset/listing-scheduler.yaml",
		"deployments/kubernetes/shein-listing/templates/single-store-deployment.yaml.tpl",
		"deployments/kubernetes/shein-listing/stores.single-pod.json",
		"scripts/build-push-listing-scheduler.ps1", "scripts/build-push-deploy-shein-listing.ps1",
		"scripts/deploy-single-store-listing.ps1", "scripts/sync-single-store-listings.ps1",
		"scripts/rollout-shein-shard-statefulset.ps1",
		"deployments/docker/Dockerfile.listing-control-plane",
		"scripts/build-push-deploy-listing-control-plane.ps1", "scripts/shein-import-platform-recovery.ps1",
		"deployments/kubernetes/shein-listing/overlays/prod-auto-shard-statefulset/listing-control-plane.yaml",
		"deployments/kubernetes/shein-listing/overlays/prod-auto-shard-statefulset/ownership-controller.yaml",
		"deployments/kubernetes/shein-listing/overlays/prod-auto-shard-statefulset/kustomization.yaml",
		"deployments/kubernetes/shein-listing/overlays/prod-store-auto-shard/configmap-heavy.yaml",
		"deployments/kubernetes/shein-listing/overlays/prod-store-auto-shard/configmap-lite.yaml",
		"deployments/kubernetes/shein-listing/overlays/prod-store-auto-shard/kustomization.yaml",
	} {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Errorf("retired CLI operation entrypoint remains: %s (stat: %v)", relative, err)
		}
	}
	for _, relative := range []string{"Makefile", "scripts/test-fast.ps1", ".github/workflows/ci.yml", ".github/workflows/commercial-readiness.yml"} {
		content, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		for _, command := range []string{"listing-scheduler", "shein-listing", "temu-listing", "listing-control-plane", "shein-import-platform-recovery"} {
			if strings.Contains(string(content), "./cmd/"+command) {
				t.Errorf("%s still invokes retired CLI %s", relative, command)
			}
		}
		for _, retired := range []string{"./internal/app/runtime/listingcontrol", "./internal/app/runtime/sheinplatformrecovery", "./internal/listingcontrol", "listing-control-plane.yaml"} {
			if strings.Contains(string(content), retired) {
				t.Errorf("%s still references retired runtime or workload %s", relative, retired)
			}
		}
	}
	content, err := os.ReadFile(filepath.Join("..", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"current-application", "shein-login-worker"} {
		if !strings.Contains(string(content), "go build ./cmd/"+command) {
			t.Errorf("CI must compile retained command %s", command)
		}
	}
	for _, relative := range []string{
		"internal/core/config/type_rabbitmq.go", "internal/core/config/loader_rabbitmq.go",
		"internal/core/config/config.go", "internal/core/config/validator_rabbitmq.go",
	} {
		content, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		for _, retired := range []string{"AutoShard", "rabbitmq.autoShard", "RABBITMQ_AUTO_SHARD_"} {
			if strings.Contains(string(content), retired) {
				t.Errorf("%s still declares or loads retired auto-shard configuration %s", relative, retired)
			}
		}
	}
}

func TestImageRetirementGuardPreservesSharedNames(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		wantRetired  bool
	}{
		{name: "retained platform selector", source: `package image; import api "task-processor/internal/temu/api/image"; var _ api.UploadResult`},
		{name: "retained shared requirement", source: `package image; import handlerbase "task-processor/internal/temu/handlerbase"; type CurrentRequirement = handlerbase.ImageRequirement`},
		{name: "local upload result and field", source: `package image; type Metadata struct { uploadResult int }; func retained() { uploadResult := 1; _ = uploadResult; type ImageValidator struct{}; _ = ImageValidator{} }`},
		{name: "receiver method with shared name", source: `package image; func (*ImageUploadProcessor) getImageRequirement() {}`},
		{name: "retired validator", source: `package image; type SingleImageValidator struct{}`, wantRetired: true},
		{name: "retired result DTO", source: `package image; type uploadResult struct{}`, wantRetired: true},
		{name: "retired requirement alias", source: `package image; type ImageRequirement = CurrentRequirement`, wantRetired: true},
		{name: "retired constructor", source: `package image; func NewImageUploadWorker() {}`, wantRetired: true},
		{name: "retired constructor binding", source: `package image; var NewImageInitHandler = factory`, wantRetired: true},
		{name: "retired config helper", source: `package image; func getImageRequirement() {}`, wantRetired: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "image.go", tt.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			violations := retiredImageOrchestrationViolations(file)
			if (len(violations) != 0) != tt.wantRetired {
				t.Fatalf("retirement violations = %v, wantRetired=%v", violations, tt.wantRetired)
			}
		})
	}
}

func retiredImageOrchestrationViolations(file *ast.File) []string {
	retired := map[string]bool{
		"ImageInitHandler": true, "NewImageInitHandler": true,
		"ImageValidator": true, "NewImageValidator": true,
		"ImageUploadWorker": true, "NewImageUploadWorker": true,
		"MainImageValidator": true, "NewMainImageValidator": true,
		"SkuImageValidator": true, "NewSkuImageValidator": true,
		"ParallelImageValidator": true, "NewParallelImageValidator": true,
		"SingleImageValidator": true, "NewSingleImageValidator": true,
		"uploadResult": true, "uploadSkuResult": true,
		"getImageRequirement": true, "ImageRequirement": true,
	}
	var violations []string
	checkName := func(name string) {
		if retired[name] {
			violations = append(violations, "retired image declaration "+name)
		}
	}
	// Package declarations recreate an owner; selectors, fields and local
	// names (including UploadSingleImage's uploadResult) do not.
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			if declaration.Recv == nil {
				checkName(declaration.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					checkName(spec.Name.Name)
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						checkName(name.Name)
					}
				}
			}
		}
	}
	return violations
}

func TestSKURetirementGuardPreservesSharedNames(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		wantRetired  bool
	}{
		{name: "retained selector", source: `package sku; import handlerbase "task-processor/internal/temu/handlerbase"; var _ handlerbase.SkuBuilder`},
		{name: "retained alias with current name", source: `package sku; import handlerbase "task-processor/internal/temu/handlerbase"; type CurrentBuilder = handlerbase.SkuBuilder`},
		{name: "local names and fields", source: `package sku; type Metadata struct { SpecValue string }; func retained(SkuBuilder int) { SpecValue := SkuBuilder; _ = SpecValue; type SKUProcessInput struct{}; _ = SKUProcessInput{}; type SkuVariantProcessor struct { itemBuilder int }; _ = SkuVariantProcessor{} }`},
		{name: "concrete type", source: `package sku; type SkuBuilder struct{}`, wantRetired: true},
		{name: "retired alias", source: `package sku; type SkuBuilder = CurrentBuilder`, wantRetired: true},
		{name: "constructor", source: `package sku; func NewSkuBuilder() {}`, wantRetired: true},
		{name: "constructor binding", source: `package sku; var NewSkuBuilder = factory`, wantRetired: true},
		{name: "retired method", source: `package sku; func (*SkuVariantProcessor) BuildVariantSkcs() {}`, wantRetired: true},
		{name: "retired field", source: `package sku; type SkuVariantProcessor struct { itemBuilder any }`, wantRetired: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "sku.go", tt.source, 0)
			if err != nil {
				t.Fatal(err)
			}
			violations := retiredSKUConstructionViolations(file)
			if (len(violations) != 0) != tt.wantRetired {
				t.Fatalf("retirement violations = %v, wantRetired=%v", violations, tt.wantRetired)
			}
		})
	}
}

func retiredSKUConstructionViolations(file *ast.File) []string {
	retiredSKUSymbols := map[string]bool{
		"SkuBuilder": true, "NewSkuBuilder": true, "SkuItemBuilder": true, "NewSkuItemBuilder": true,
		"SkuSkcBuilder": true, "NewSkuSkcBuilder": true, "SkuParallelBuilder": true, "NewSkuParallelBuilder": true,
		"SKUVariantBuildInput": true, "SKUProcessInput": true, "buildSKUVariantBuildInputWithRuntime": true,
		"buildSKUVariantBuildInput": true, "buildSKUProcessInput": true,
		"SkuPriceCalculator": true, "NewSkuPriceCalculator": true,
		"skuPackagingInfo": true, "skuExpressInfo": true, "skuPricingInfo": true, "SpecDimension": true, "SpecValue": true,
	}
	retiredSKUMethods := map[string]map[string]bool{
		"SkuVariantProcessor": {
			"BuildVariantSkcs": true, "resolveVariantAIMapping": true, "resolveAndBuildVariantSkcs": true,
			"assignBuiltVariantSkcs": true, "BuildSkcsFromAIMapping": true, "prepareAndBuildVariantSkcs": true,
			"buildSkcsFromPreparedAIMapping": true, "resolveSkcBuildSpecs": true, "buildMultipleSkcs": true, "buildSingleSkc": true,
			"CreateDefaultSkc": true, "resolveDefaultBuildProduct": true, "resolveAndBuildDefaultSkc": true,
			"prepareAndBuildDefaultSkc": true, "buildDefaultSkcFromPreparedMapping": true, "selectDefaultAIMapping": true,
			"buildDefaultSkuFromAIMapping": true, "resolveDefaultAIMapping": true, "prepareDefaultAIMappingForBuild": true,
		},
		"SkuSpecHandler": {
			"collectNonColorSpecsForColor": true, "generateNonColorSpecCombinations": true, "generateCartesianProduct": true,
			"createSpecCombinationKey": true, "createNonColorSpecKey": true, "createSpecCombinationKeyFromSpecs": true, "isColorSpec": true,
		},
	}
	var violations []string
	checkName := func(name string) {
		if retiredSKUSymbols[name] {
			violations = append(violations, "retired SKU declaration "+name)
		}
	}
	// Inspect package declarations only: retained selectors and local names do
	// not recreate the retired concrete package owner.
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			if declaration.Recv == nil {
				checkName(declaration.Name.Name)
				continue
			}
			if len(declaration.Recv.List) != 1 {
				continue
			}
			receiver := declaration.Recv.List[0].Type
			if pointer, ok := receiver.(*ast.StarExpr); ok {
				receiver = pointer.X
			}
			if owner, ok := receiver.(*ast.Ident); ok && retiredSKUMethods[owner.Name][declaration.Name.Name] {
				violations = append(violations, "retired SKU method "+owner.Name+"."+declaration.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				switch spec := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						checkName(name.Name)
					}
				case *ast.TypeSpec:
					checkName(spec.Name.Name)
					if spec.Name.Name != "SkuVariantProcessor" {
						continue
					}
					if state, ok := spec.Type.(*ast.StructType); ok {
						for _, field := range state.Fields.List {
							for _, name := range field.Names {
								if name.Name == "skcBuilder" || name.Name == "itemBuilder" {
									violations = append(violations, "exclusive SKU construction field "+name.Name)
								}
							}
						}
					}
				}
			}
		}
	}
	return violations
}
