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

var retiredNumericListingSupportFiles = []string{
	"internal/app/worker/job_handler.go",
	"internal/core/metrics/shein_metrics.go",
	"internal/core/metrics/task_metrics.go",
	"internal/listingadmin/category.go",
	"internal/listingadmin/category_model.go",
	"internal/listingadmin/category_model_test.go",
	"internal/listingadmin/category_query.go",
	"internal/listingadmin/category_query_test.go",
	"internal/listingadmin/category_repository.go",
	"internal/listingadmin/dto_value_helpers.go",
	"internal/listingadmin/filter_rule.go",
	"internal/listingadmin/filter_rule_model.go",
	"internal/listingadmin/filter_rule_model_test.go",
	"internal/listingadmin/filter_rule_query.go",
	"internal/listingadmin/filter_rule_query_test.go",
	"internal/listingadmin/filter_rule_repository.go",
	"internal/listingadmin/gorm_filter_rule_api.go",
	"internal/listingadmin/gorm_operation_strategy_api.go",
	"internal/listingadmin/gorm_product_data_api.go",
	"internal/listingadmin/gorm_product_data_api_test.go",
	"internal/listingadmin/gorm_product_import_mapping_api.go",
	"internal/listingadmin/gorm_product_import_mapping_api_test.go",
	"internal/listingadmin/gorm_raw_json_data_api.go",
	"internal/listingadmin/import_task_owner.go",
	"internal/listingadmin/import_task_owner_test.go",
	"internal/listingadmin/numeric_helpers.go",
	"internal/listingadmin/operation_strategy.go",
	"internal/listingadmin/operation_strategy_model.go",
	"internal/listingadmin/operation_strategy_model_test.go",
	"internal/listingadmin/operation_strategy_query.go",
	"internal/listingadmin/operation_strategy_query_test.go",
	"internal/listingadmin/operation_strategy_repository.go",
	"internal/listingadmin/owner_scoped_batch_test.go",
	"internal/listingadmin/owner_test_helpers_test.go",
	"internal/listingadmin/pricing_rule.go",
	"internal/listingadmin/pricing_rule_model.go",
	"internal/listingadmin/pricing_rule_model_test.go",
	"internal/listingadmin/pricing_rule_query.go",
	"internal/listingadmin/pricing_rule_query_test.go",
	"internal/listingadmin/pricing_rule_repository.go",
	"internal/listingadmin/product_data.go",
	"internal/listingadmin/product_data_model.go",
	"internal/listingadmin/product_data_model_test.go",
	"internal/listingadmin/product_data_query.go",
	"internal/listingadmin/product_data_query_test.go",
	"internal/listingadmin/product_data_repository.go",
	"internal/listingadmin/product_import_mapping.go",
	"internal/listingadmin/product_import_mapping_model.go",
	"internal/listingadmin/product_import_mapping_model_test.go",
	"internal/listingadmin/product_import_mapping_query.go",
	"internal/listingadmin/product_import_mapping_query_test.go",
	"internal/listingadmin/product_import_mapping_repository.go",
	"internal/listingadmin/profit_rule.go",
	"internal/listingadmin/profit_rule_model.go",
	"internal/listingadmin/profit_rule_model_test.go",
	"internal/listingadmin/profit_rule_query.go",
	"internal/listingadmin/profit_rule_query_test.go",
	"internal/listingadmin/profit_rule_repository.go",
	"internal/listingadmin/raw_json_data.go",
	"internal/listingadmin/raw_json_data_query_test.go",
	"internal/listingadmin/raw_json_data_repository.go",
	"internal/listingadmin/rule_status.go",
	"internal/marketplace/shein/workspace/category_search.go",
	"internal/marketplace/shein/workspace/category_search_test.go",
	"internal/marketplace/shein/workspace/display_title_policy.go",
	"internal/marketplace/shein/workspace/display_title_policy_test.go",
	"internal/marketplace/shein/workspace/editor_context_builder.go",
	"internal/marketplace/shein/workspace/editor_context_builder_test.go",
	"internal/marketplace/shein/workspace/editor_context_model.go",
	"internal/marketplace/shein/workspace/editor_dirty_hints.go",
	"internal/marketplace/shein/workspace/editor_effects.go",
	"internal/marketplace/shein/workspace/editor_effects_test.go",
	"internal/marketplace/shein/workspace/editor_model.go",
	"internal/marketplace/shein/workspace/editor_progress.go",
	"internal/marketplace/shein/workspace/editor_progress_test.go",
	"internal/marketplace/shein/workspace/editor_recommendations.go",
	"internal/marketplace/shein/workspace/editor_recommendations_test.go",
	"internal/marketplace/shein/workspace/editor_revision_from_context.go",
	"internal/marketplace/shein/workspace/editor_revision_from_context_test.go",
	"internal/marketplace/shein/workspace/editor_revision_minimal.go",
	"internal/marketplace/shein/workspace/editor_revision_minimal_test.go",
	"internal/marketplace/shein/workspace/editor_revision_model.go",
	"internal/marketplace/shein/workspace/editor_revision_skeleton.go",
	"internal/marketplace/shein/workspace/history_detail.go",
	"internal/marketplace/shein/workspace/history_detail_test.go",
	"internal/marketplace/shein/workspace/inspection.go",
	"internal/marketplace/shein/workspace/inspection_test.go",
	"internal/marketplace/shein/workspace/overview.go",
	"internal/marketplace/shein/workspace/overview_test.go",
	"internal/marketplace/shein/workspace/preview_card.go",
	"internal/marketplace/shein/workspace/preview_card_test.go",
	"internal/marketplace/shein/workspace/preview_projection.go",
	"internal/marketplace/shein/workspace/preview_projection_test.go",
	"internal/marketplace/shein/workspace/preview_support.go",
	"internal/marketplace/shein/workspace/preview_support_test.go",
	"internal/marketplace/shein/workspace/readiness_guidance.go",
	"internal/marketplace/shein/workspace/readiness_guidance_test.go",
	"internal/marketplace/shein/workspace/repair.go",
	"internal/marketplace/shein/workspace/repair_center_from_readiness.go",
	"internal/marketplace/shein/workspace/repair_center_from_readiness_test.go",
	"internal/marketplace/shein/workspace/repair_revision.go",
	"internal/marketplace/shein/workspace/repair_revision_test.go",
	"internal/marketplace/shein/workspace/repair_test.go",
	"internal/marketplace/shein/workspace/review_state.go",
	"internal/marketplace/shein/workspace/review_state_test.go",
	"internal/marketplace/shein/workspace/revision_apply_changes.go",
	"internal/marketplace/shein/workspace/revision_apply_patch.go",
	"internal/marketplace/shein/workspace/revision_apply_patch_test.go",
	"internal/marketplace/shein/workspace/revision_diff.go",
	"internal/marketplace/shein/workspace/revision_diff_model.go",
	"internal/marketplace/shein/workspace/revision_diff_test.go",
	"internal/marketplace/shein/workspace/revision_field_validation.go",
	"internal/marketplace/shein/workspace/revision_field_validation_test.go",
	"internal/marketplace/shein/workspace/revision_history_compare.go",
	"internal/marketplace/shein/workspace/revision_history_compare_test.go",
	"internal/marketplace/shein/workspace/revision_history_detail_builder.go",
	"internal/marketplace/shein/workspace/revision_history_restore_detail.go",
	"internal/marketplace/shein/workspace/revision_history_restore_detail_test.go",
	"internal/marketplace/shein/workspace/revision_restore_draft.go",
	"internal/marketplace/shein/workspace/revision_restore_draft_test.go",
	"internal/marketplace/shein/workspace/revision_restore_presentation.go",
	"internal/marketplace/shein/workspace/revision_restore_preview.go",
	"internal/marketplace/shein/workspace/revision_restore_preview_builder.go",
	"internal/marketplace/shein/workspace/revision_restore_preview_test.go",
	"internal/marketplace/shein/workspace/revision_restore_request.go",
	"internal/marketplace/shein/workspace/revision_success.go",
	"internal/marketplace/shein/workspace/revision_success_model.go",
	"internal/marketplace/shein/workspace/revision_success_test.go",
	"internal/marketplace/shein/workspace/revision_validation.go",
	"internal/marketplace/shein/workspace/revision_validation_test.go",
	"internal/marketplace/shein/workspace/source_product.go",
	"internal/marketplace/shein/workspace/source_product_test.go",
	"internal/marketplace/shein/workspace/store_resolution.go",
	"internal/marketplace/shein/workspace/store_resolution_test.go",
	"internal/marketplace/shein/workspace/submit_freshness.go",
	"internal/marketplace/shein/workspace/submit_freshness_readiness.go",
	"internal/marketplace/shein/workspace/submit_freshness_readiness_test.go",
	"internal/marketplace/shein/workspace/submit_freshness_test.go",
	"internal/marketplace/shein/workspace/task_queue.go",
	"internal/marketplace/shein/workspace/task_queue_test.go",
	"internal/marketplace/shein/workspace/task_taxonomy.go",
	"internal/marketplace/shein/workspace/task_taxonomy_test.go",
	"internal/model/task.go",
	"internal/model/task_retry.go",
	"internal/model/task_retry_test.go",
	"internal/model/task_status.go",
	"internal/model/task_status_test.go",
	"internal/model/task_test.go",
	"internal/shein/sherr/handled_errors.go",
	"internal/taskrpcapi/build.go",
	"internal/taskrpcapi/build_test.go",
	"internal/taskrpcapi/handler.go",
	"internal/taskrpcapi/http_module.go",
	"internal/taskrpcapi/http_module_test.go",
	"internal/taskrpcapi/types.go",
	"internal/workspace/shein/compatibility_guard_test.go",
	"internal/workspace/shein/doc.go",
	"internal/workspace/shein/editor_context_builder.go",
	"internal/workspace/shein/editor_context_builder_test.go",
	"internal/workspace/shein/editor_context_model.go",
	"internal/workspace/shein/editor_dirty_hints.go",
	"internal/workspace/shein/editor_effects.go",
	"internal/workspace/shein/editor_model.go",
	"internal/workspace/shein/editor_progress.go",
	"internal/workspace/shein/editor_recommendations.go",
	"internal/workspace/shein/editor_revision_from_context.go",
	"internal/workspace/shein/editor_revision_minimal.go",
	"internal/workspace/shein/editor_revision_minimal_test.go",
	"internal/workspace/shein/editor_revision_model.go",
	"internal/workspace/shein/editor_revision_skeleton.go",
	"internal/workspace/shein/history_detail.go",
	"internal/workspace/shein/inspection.go",
	"internal/workspace/shein/inspection_payloads.go",
	"internal/workspace/shein/inspection_test.go",
	"internal/workspace/shein/overview.go",
	"internal/workspace/shein/overview_test.go",
	"internal/workspace/shein/readiness.go",
	"internal/workspace/shein/readiness_guidance.go",
	"internal/workspace/shein/readiness_guidance_test.go",
	"internal/workspace/shein/readiness_test.go",
	"internal/workspace/shein/repair.go",
	"internal/workspace/shein/repair_test.go",
	"internal/workspace/shein/revision_apply_changes.go",
	"internal/workspace/shein/revision_diff.go",
	"internal/workspace/shein/revision_diff_model.go",
	"internal/workspace/shein/revision_field_validation.go",
	"internal/workspace/shein/revision_field_validation_test.go",
	"internal/workspace/shein/revision_history_compare.go",
	"internal/workspace/shein/revision_history_detail_builder.go",
	"internal/workspace/shein/revision_history_restore_detail.go",
	"internal/workspace/shein/revision_restore_draft.go",
	"internal/workspace/shein/revision_restore_presentation.go",
	"internal/workspace/shein/revision_restore_preview.go",
	"internal/workspace/shein/revision_restore_preview_builder.go",
	"internal/workspace/shein/revision_restore_request.go",
	"internal/workspace/shein/revision_success.go",
	"internal/workspace/shein/revision_success_model.go",
	"internal/workspace/shein/revision_success_test.go",
	"internal/workspace/shein/revision_validation.go",
	"internal/workspace/shein/state.go",
}

var retiredNumericListingSupportDeclarations = map[string][]string{
	"internal/app/worker": strings.Fields(`
NewTaskJobHandler TaskCompletionNotifier TaskJobHandler
`),
	"internal/listingadmin": strings.Fields(`
AutoMigrateCategoryRepository AutoMigrateFilterRuleRepository AutoMigrateOperationStrategyRepository
AutoMigratePricingRuleRepository AutoMigrateProductDataRepository AutoMigrateProductImportMappingRepository
AutoMigrateProfitRuleRepository Category CategoryQuery CategoryRepository CommonResult ErrCategoryHasChildren
ErrCategoryNotFound ErrFilterRuleNotFound ErrOperationStrategyNotFound ErrPricingRuleNotFound
ErrProductDataNotFound ErrProductImportMappingNotFound ErrProductImportMappingOwnerRequired
ErrProfitRuleNotFound ErrRawJSONDataNotFound FilterRule FilterRuleAPI FilterRulePage FilterRuleQuery
FilterRuleRepository FilterRuleReqDTO FilterRuleRespDTO FilterRuleToRespDTO GormCategoryRepository
GormFilterRuleRepository GormOperationStrategyRepository GormPricingRuleRepository GormProductDataRepository
GormProductImportMappingRepository GormProfitRuleRepository GormRawJSONDataRepository ImageDownloader ImageInfo
IsRuleStatusEnabled NewGormCategoryRepository NewGormFilterRuleAPI NewGormFilterRuleRepository
NewGormOperationStrategyAPI NewGormOperationStrategyRepository NewGormPricingRuleRepository
NewGormProductDataAPI NewGormProductDataRepository NewGormProductImportMappingAPI
NewGormProductImportMappingRepository NewGormProfitRuleRepository NewGormRawJSONDataRepository
NewGormRawJsonDataAPI NewProductDataBatchSaveReqDTO NewProductDataItemDTO OperationStrategy
OperationStrategyAPI OperationStrategyDTO OperationStrategyPage OperationStrategyQuery
OperationStrategyRepository OperationStrategyToDTO PricingRule PricingRulePage PricingRuleQuery
PricingRuleRepository ProductAttributesItemDTO ProductData ProductDataAPI ProductDataBatchSaveReqDTO
ProductDataBatchUpdateAttributesReqDTO ProductDataDTO ProductDataFromAttributesBatchReq
ProductDataFromBatchSaveReq ProductDataItemDTO ProductDataListByStorePageReqDTO ProductDataPage
ProductDataQuery ProductDataRepository ProductDataRespDTO ProductDataToDTO ProductImportMapping
ProductImportMappingAPI ProductImportMappingCheckReqDTO ProductImportMappingCreateReqDTO
ProductImportMappingFromCreateReq ProductImportMappingGetByPlatformProductIdAndStoreReqDTO
ProductImportMappingGetBySkuReqDTO ProductImportMappingGetByTaskAndSkuReqDTO ProductImportMappingGetReqDTO
ProductImportMappingPage ProductImportMappingQuery ProductImportMappingRepository ProductImportMappingRespDTO
ProductImportMappingToRespDTO ProductVariantConfirmationReqDTO ProfitRule ProfitRulePage ProfitRuleQuery
ProfitRuleRepository RawJSONData RawJSONDataRepository RawJSONDataToRespDTO RawJsonDataAPI
RawJsonDataCreateReqDTO RawJsonDataReqDTO RawJsonDataRespDTO ResolveStoreOwnerUserID RuleStatusDisabled
RuleStatusEnabled ShelfStatusDeleted ShelfStatusOffShelf ShelfStatusOnShelf ShelfStatusPending
ShelfStatusRejected ShelfStatusReviewing activityProfitRatePtr applyCategoryAuditFields applyCategoryDefaults
applyCategoryQuery applyFilterRuleAuditFields applyFilterRuleDefaults applyFilterRuleQuery
applyOperationStrategyAuditFields applyOperationStrategyQuery applyPricingRuleAuditFields
applyPricingRuleDefaults applyPricingRuleQuery applyProductDataAuditFields applyProductDataQuery
applyProductImportMappingAuditFields applyProductImportMappingDefaults applyProductImportMappingQuery
applyProfitRuleAuditFields applyProfitRuleDefaults applyProfitRuleQuery boolToInt16 defaultActivityPartakeType
ensureOperationStrategyActivityColumns ensureTextColumn findCategoryRows findFilterRuleRows
findOperationStrategyRows findPricingRuleRows findProductDataRows findProductImportMappingRows
findProfitRuleRows flexibleStringToFloat64 flexibleTimeValue floatPtrIfPositive floatValue gormFilterRuleAPI
gormOperationStrategyAPI gormProductDataAPI gormProductImportMappingAPI gormRawJsonDataAPI int64Ptr
int64PtrIfPositive int64Value intPtrIfPositive intValue listingCategory listingCategoryFromCategory
listingFilterRule listingFilterRuleFromFilterRule listingOperationStrategy
listingOperationStrategyFromOperationStrategy listingPricingRule listingPricingRuleFromPricingRule
listingProductData listingProductDataFromProductData listingProductImportMapping
listingProductImportMappingFromProductImportMapping listingProfitRule listingProfitRuleFromProfitRule
listingRawJSONData nullableFloat64 operationStrategyFlexibleString operationStrategyFloat64Value
operationStrategyIntValue parseOptionalMultiplier ptrFloat64 ptrInt ptrString rawJSONBytes rawJSONOrString
resolveStoreOwnerUserID resolveStoreOwnerUserIDForUpdate
`),
	"internal/listingruntime": strings.Fields(`
FilterRule ProductImportMappingUpsert ProfitRule StorePauseStatusDetail StoreService
`),
	"internal/marketplace/shein/workspace": strings.Fields(`
ActionItem ActionQueueAttributes ActionQueueClassification ActionQueueDescriptors ActionQueueFinalReview
ActionQueueManualReview ActionQueueMedia ActionQueuePayloadRebuild ActionQueuePricing ActionQueueSourceReview
ActionQueueStoreAuth ActionQueueSubmitReady ActionQueueVariant ApplyAttributeResolutionPatch
ApplyCategoryResolutionPatch ApplySKCRevisionPatches ApplySKURevisionPatches ApplySaleAttributeResolutionPatch
AttachSubmissionEventStoreResolution AttributeResolutionPatch BuildAppliedChangesPreview BuildAttributeEffects
BuildAttributePayload BuildAttributeRecommendationMeta BuildAttributeResolutionPatch BuildAttributeSuggestions
BuildCategoryEffects BuildCategoryPayload BuildCategoryRecommendationMeta BuildCategoryResolutionPatch
BuildCurrentHistoryComparePreview BuildEditorContext BuildEditorDirtyHints BuildEditorProgress
BuildEditorRevisionSkeleton BuildEditorSKCPatches BuildFinalReviewImages BuildFinalReviewSKU
BuildFinalReviewSKUs BuildFreshnessAttributeCheck BuildFreshnessAuthFailureCheck BuildFreshnessAuthSuccessCheck
BuildFreshnessCategoryCheck BuildFreshnessSaleAttributeCheck BuildHistoryComparePreview BuildHistoryDetail
BuildHistoryNavigation BuildHistoryRestoreContext BuildHistoryRestoreDetailData BuildHistoryRestoreMessages
BuildHistoryRestoreNextActions BuildHistoryRestoreOverview BuildHistoryRestorePresentationData
BuildHistoryRestoreSafety BuildImageUploadPreflight BuildInspection BuildMinimalRevisionSkeleton
BuildPreviewCardStatus BuildPreviewCardSummary BuildPreviewReviewSummary BuildReadinessGuidanceSpec
BuildReadinessPatchPayload BuildReadinessReason BuildRepairApplyQueue BuildRepairCenter
BuildRepairCenterFromReadiness BuildRepairCenterStatus BuildRepairCenterSummary BuildRepairPlan
BuildRepairReason BuildRepairRevisionInput BuildRepairRevisionSeed BuildRepairSession BuildRepairStateInput
BuildRepairValidationPreview BuildResolutionCacheSummary BuildRestoreDraftFromSkeleton
BuildRestorePreviewPayload BuildRestoreRequestSeed BuildRevisionDiffBetweenRevisions BuildRevisionDiffPreview
BuildRevisionDiffPreviewFromInput BuildRevisionInputFromEditorContext BuildSaleAttributeEffects
BuildSaleAttributePayload BuildSaleAttributeResolutionPatch BuildSaleCandidateSuggestions
BuildSaleRecommendationMeta BuildSourceProductSummary BuildStatusOverview BuildStoreResolutionSummary
BuildSubmissionStoreResolution BuildSubmitChecklist BuildSubmitStateInput BuildSuccessFollowUpChecklist
BuildSuccessFollowUpOverview BuildSuccessMessages BuildSuccessNextActions BuildSuccessPayload
BuildSuccessPresentation BuildSuccessRecommendedView BuildSuccessStatusSummary
BuildSuccessSuggestedFollowUpRevision BuildSuccessSummaryCard BuildTaskActionQueue BuildTaskWorkQueue
BuildValidationPayload BuildWorkspaceOverview CategoryResolutionPatch CategorySearchCandidate
ChecklistGroupItem ChecklistItemCount CloneAttributeResolutionPatch CloneCategoryResolutionPatch
CloneEditorRevisionSkeleton CloneReadinessItems CloneReadinessReason CloneRepairPatchPayload
CloneRepairValidationPreview CloneRevisionDiffPreview CloneRevisionInput CloneSKCRevisionPatches
CloneSaleAttributeResolutionPatch CookieUnavailableReviewNotes DisplayTitleInput EditorAttributeContext
EditorAttributeSuggestion EditorBasicsContext EditorCategoryContext EditorContext EditorDirtyHintSection
EditorDirtyHints EditorEffect EditorProgress EditorProgressSection EditorRecommendationMeta
EditorRevisionSkeleton EditorSaleAttributeContext EditorSaleCandidateSuggestion EvaluateAttributeFreshness
EvaluateCategoryFreshness EvaluateSaleAttributeFreshness EvaluateSaleAttributeFreshnessWithCustomValidation
FacetDescriptor FieldError FilterOutCookieUnavailableReviewNotes FinalReviewImage FinalReviewSKU FindKeys
FindLabels FreshnessAttributeKey FreshnessAuthKey FreshnessCategoryKey FreshnessSaleAttributeKey
HasCookieUnavailableReviewNotes HistoryComparePreview HistoryCompareRecord HistoryDetail HistoryNavigation
HistoryRestoreCompareInput HistoryRestoreContext HistoryRestoreDetailData HistoryRestoreMessages
HistoryRestoreOverview HistoryRestorePresentationData HistoryRestoreRecommendedView HistoryRestoreRecordInput
HistoryRestoreSafety HistoryRestoreStateInput HistoryRestoreTimeline ImageUploadCacheHit ImageUploadClassifier
ImageUploadPreflight InspectionReviewReasons IsCookieUnavailableText IsEmptyRevisionInput
IsSaleAttributeResolved JoinReadinessLabels NewFieldError PreviewCardNeedsReview PruneAttributeResolutionPatch
PruneCategoryResolutionPatch PruneRevisionInput PruneSaleAttributeResolutionPatch ReadinessGuidanceSpec
ReadinessHintSpec ReadinessReason ReadinessReasonSpec RebuildRestorePreviewPayload RepairApplyQueue
RepairApplyQueueItem RepairArtifacts RepairCenter RepairCenterAction RepairCenterActionCount
RepairCenterDirectApplyCount RepairCenterFromReadinessOptions RepairCenterPrimaryPlanStatus RepairCenterSection
RepairCenterSeedAction RepairCenterSessionStatus RepairCenterStats RepairCompletionSnapshot RepairHint
RepairHintAccessors RepairPatchPayload RepairPlan RepairPlanStep RepairResumeState RepairRevisionSeed
RepairRunbookStep RepairSession RepairSessionActionInfo RepairStateInput RepairValidationPreview
ResolutionCacheSummary ResolveDisplayTitle ResolveHistoryCompareTarget RestorePreviewCoreData
RestorePreviewPayload RestoreRequestSeed RevisionDiffPreview RevisionFieldChange RevisionInput SKCRevisionPatch
SKURevisionPatch SaleAttributeResolutionPatch SceneApplySuccess SceneRestoreSuccess SearchCategoryCandidates
SessionInput SourceProductSummary StatusOverview StoreResolutionSummary StripCookieUnavailableReviewNotes
SubmitChecklist SubmitChecklistGroupForKey SubmitStateInput SuccessCoreData SuccessFollowUpChecklist
SuccessFollowUpOverview SuccessInteractionPresentation SuccessMessages SuccessModeApply SuccessModeRestore
SuccessPayload SuccessRecommendedView SuccessStatusSummary SuccessSummaryCard ToActionItems
ValidateRevisionInput ValidationPayload WorkQueueDescriptors WorkQueueDraft WorkQueueGeneration
WorkQueueGenerationFailed WorkQueuePublished WorkQueueRepair WorkQueueReview WorkQueueSubmitFailed
WorkQueueSubmitReady WorkflowStatusDescriptors WorkflowStatusDraftSaved WorkflowStatusPendingConfirmation
WorkflowStatusPublishFailed WorkflowStatusPublished WorkflowStatusReadyToSubmit WorkspaceOverview
WorkspaceRepairState WorkspaceSessionEntry WorkspaceSubmitState actionQueueDescriptors actionQueueForKey
addPreviewImageInfo addPreviewProductImages appendAppliedSaleAttributeReviewChanges appendAttributesChange
appendCompareAttributesChange appendCompareImageChange appendCompareIntChange appendCompareIntPtrChange
appendCompareIntSliceChange appendCompareResolvedAttributesChange appendCompareSalePatchChange
appendCompareStringChange appendImageChange appendImageInfoURLs appendIntChange appendIntPtrChange
appendIntSliceChange appendResolvedAttributesChange appendSalePatchChange appendStringChange
applyPrimaryActionFromItems attributeConfidence attributeFreshnessIssueState attributeFreshnessTemplateContext
attributeReason attributeSuggestionConfidence attributeSuggestionReason buildApplyHighlights
buildApplyPrimaryAction buildApplySubtitle buildAttributeActions buildAttributeFreshnessOutcome
buildAttributeFreshnessTemplateContext buildAttributeInspectionSection buildAttributePayload
buildAttributePayloadMap buildAttributeProgressSection buildBasicProgressSection buildCategoryActions
buildCategoryInspectionSection buildCategoryPayload buildCategoryPayloadMap buildCategoryProgressSection
buildEditorSKUPatches buildFreshnessSaleTemplateOptions buildHistoryRestoreRecommendedView
buildImageUploadPreflightSummary buildNextActions buildNextActionsFromReadiness buildPendingAttributeCandidates
buildPendingAttributes buildRepairAffectedSections buildRepairCompletionRule buildRepairCompletionSnapshot
buildRepairPlanCopy buildRepairPlanStatus buildRepairResumeState buildRepairSessionCopy
buildRepairSessionStatus buildRepairSkippedSteps buildRepairStepHighlights buildRepairStepReason
buildResolvedAttributeTemplateCandidates buildResolvedAttributeTemplateDriftDetails buildSKCPatchSuggestions
buildSKUPatchSuggestions buildSaleAttributeActions buildSaleAttributeFreshnessResolutionOutcome
buildSaleAttributeFreshnessTemplateContext buildSaleAttributeInspectionSection buildSaleAttributePayload
buildSaleAttributePayloadMap buildSaleProgressSection buildStatusHighlights buildSuccessFollowUpHeadline
categoryConfidence categoryMatchScore categoryReason categorySearchMatch checklistItemAsReadinessItem
cloneAttributePatch cloneBoolPointer cloneCategoryPatch cloneFacetDescriptors cloneImageSet cloneIntPointer
cloneMap cloneNonNegativeIntPointer clonePendingAttributeCandidates clonePositiveIntPointer
cloneRepairArtifacts cloneRepairReason cloneRepairSKCPatches cloneRepairSKUPatches cloneRequestSKCs
cloneRequestSKUs cloneResolvedSaleAttributeMap cloneRevisionDiffPreview cloneSKCPatches cloneSKUPatches
cloneSalePatch cloneSourceDimensionsForInspection cloneStringPointer cloneSuccessFollowUpChecklist
cloneSuccessFollowUpOverview cloneSuccessMessages cloneSuccessRecommendedView cloneSuccessStatusSummary
cloneSuccessSummaryCard cloneTemplateOptionsForInspection collectAttributeDirtyFields collectBasicDirtyFields
collectCategoryDirtyFields collectInvalidSaleAttributes collectProductImageURLs collectRepairCandidates
collectSaleDirtyFields compareRepairActions conditionalOtherAttribute containsNormalized
dedupeCustomAttributeRelations defaultInspectionReviewReason defaultSuccessEditorReason
defaultSuccessInspectionReason defaultSuccessLabel defaultSuccessSubmitReason defaultSuccessTitle
defaultSuccessWarningTitle dependencyIsActive diffResolvedAttributes enrichAttributeResolutionCacheInfo
enrichCategoryResolutionCacheInfo enrichPricingResolutionCacheInfo enrichSaleAttributeResolutionCacheInfo
enrichSizeAttributeResolutionCacheInfo ensureImageDraft equalIntSlices equalPlatformAttributes
equalResolvedAttributes evaluateAttributeFreshnessIssueState evaluateResolvedAttributeFreshness
evaluateSaleAttributeFreshnessInvalidState evaluateSaleAttributeFreshnessResolution
filterFreshnessDisplayAttributes filterFreshnessSaleScopeAttributes filterNonEmptyStrings findPackageSKC
findPackageSKU findReadinessItem findRequestSKC findRequestSKU findSKCPackageBySupplierCode firstAction
firstNonEmptyString firstNonEmptyValue firstSummaryLine flattenAttributeTemplatesByID
formatFreshnessAttributeName formatResolvedAttributeDiffItem formatResolvedSaleAttributeDiffItem
freshnessCustomAttributeValueIDs hasHistoryRestoreWarning hasImageContent hasResolvedSaleAttributeDraft
indexNormalized intPointer isEmptyAttributePatch isEmptyCategoryPatch isEmptySalePatch isFinalReviewSwatchRole
isTemplateRequired maxCategorySearchResults maxInt mergeFinalReviewImage messageDescription messageTitle
messageWarnings normalizeFinalReviewAttributeName normalizeRepairSection normalizeReviewReasons normalizeText
orderFreshnessSaleScopeAttributes parseCascadeValueIDs parseMoney previewImageDetailRole progressSection
pruneSKCRevisionPatches pruneSKURevisionPatches readinessSummary recommendedViewValue repairActionCandidate
repairActionID repairExecutionMode repairFreshnessSaleAttributes repairPriorityRank repairSectionLabel
repairSectionRank repairStatusRank resolveFinalReviewImageRole resolvedAttributeStillLegal
saleAttributeCategoryReviewReason saleAttributeFreshnessInvalidState saleAttributeFreshnessTemplateContext
saleAttributeRecommendCategoryReview saleCandidateConfidence saleCandidateReason saleConfidence saleReason
stringPointerOrNil stringsContainsNormalized stringsTrim summaryHighlights summaryStatus summarySubheadline
toSubmitStateInput tryRepairFreshnessSaleAttribute workQueueDescriptors workflowStatusDescriptors
workspaceHeadlineFromReadiness workspacePrimaryView
`),
	"internal/model": strings.Fields(`
ApplyRetryFailure ParseTaskStatus ResolveTaskMaxRetries RetryDecision Task TaskStatus TaskStatusCancelled
TaskStatusCrawlFailed TaskStatusCrawled TaskStatusDraft TaskStatusPaused TaskStatusPending
TaskStatusPendingRetry TaskStatusProcessing TaskStatusPublished TaskStatusQueued TaskStatusRepublishing
TaskStatusResumed TaskStatusResuming TaskStatusTerminated ValidateTaskStatusTransition
ValidateTaskStatusTransitionCode defaultTaskMaxRetries retryPriorityPenalty taskStatusNames
taskStatusTransitions
`),
	"internal/shein/sherr": strings.Fields(`
AsTaskHandledError IsTaskHandledError NewTaskHandledError TaskHandledError
`),
}

func TestNumericListingSupportClosureSourcesAreRetired(t *testing.T) {
	for _, relative := range retiredNumericListingSupportFiles {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Errorf("retired numeric listing support source remains: %s (stat: %v)", relative, err)
		}
	}
	for _, relative := range []string{"internal/taskrpcapi", "internal/workspace/shein", "internal/core/metrics"} {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Errorf("retired numeric listing support package remains: %s (stat: %v)", relative, err)
		}
	}
	sources := trackedProductionTextSources(t, []string{"."}, func(path string) bool {
		return strings.HasSuffix(path, ".go")
	})
	violations, err := phase3BannedImportDeclarationViolations(sources, []string{
		"task-processor/internal/taskrpcapi",
		"task-processor/internal/workspace/shein",
		"task-processor/internal/core/metrics",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, violation := range violations {
		t.Errorf("retired numeric listing support import remains: %s", violation)
	}
	banned := make(map[string]map[string]bool)
	for relative, names := range retiredNumericListingSupportDeclarations {
		banned[relative] = make(map[string]bool)
		for _, name := range names {
			banned[relative][name] = true
		}
	}
	for _, source := range sources {
		file, err := parser.ParseFile(token.NewFileSet(), source.path, source.text, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range file.Imports {
			imported, err := decodeGoImportPath(spec.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			// Only the old root hook is retired; imageagent remains current.
			if imported == "task-processor/internal/app/worker" {
				t.Errorf("%s imports retired root Task worker hook", source.path)
			}
		}
		if strings.HasSuffix(source.path, "_test.go") || strings.HasPrefix(source.path, "tests/") {
			continue
		}
		for _, route := range []string{
			"/api/v1/management/tasks/health",
			"/api/v1/management/tasks/:task_id/status",
			"/api/v1/management/tasks/queue-stats",
		} {
			if strings.Contains(source.text, route) {
				t.Errorf("%s recreates retired Task RPC route %s", source.path, route)
			}
		}
		names := banned[filepath.ToSlash(filepath.Dir(source.path))]
		checkName := func(name string) {
			if names[name] {
				t.Errorf("%s recreates retired numeric listing support declaration %s", source.path, name)
			}
		}
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
	}
}

var retiredTEMUListingPackages = []string{
	"internal/temu/ai", "internal/temu/context", "internal/temu/filter",
	"internal/temu/handlerbase", "internal/temu/pricing", "internal/temu/product",
	"internal/temu/property", "internal/temu/rules", "internal/temu/sku", "internal/temu/spec",
	"internal/pipeline", "internal/pipeline/handlers", "internal/pkg/skugen",
	"internal/marketplace/temu/publishing",
}

var retiredAmazonListingPackages = []string{
	"internal/amazon/attribute", "internal/amazon/image", "internal/amazon/listing",
	"internal/amazon/llm", "internal/amazon/model", "internal/amazon/pipeline", "internal/amazon/schema",
	"internal/amazonlisting", "internal/marketplace/amazon", "internal/state",
	"internal/httpbootstrap", "internal/shared/timex",
}

func TestAmazonListingClosureSourcesAreRetired(t *testing.T) {
	paths := append([]string{}, retiredAmazonListingPackages...)
	paths = append(paths, "hack/debug/test-amazon-listing", "deployments/kubernetes/amazon-listing")
	for _, relative := range paths {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Errorf("retired Amazon listing source remains: %s (stat: %v)", relative, err)
		}
	}
	var banned []string
	for _, relative := range retiredAmazonListingPackages {
		banned = append(banned, "task-processor/"+relative)
	}
	sources := trackedProductionTextSources(t, []string{"."}, func(path string) bool {
		return strings.HasSuffix(path, ".go")
	})
	violations, err := phase3BannedImportDeclarationViolations(sources, banned)
	if err != nil {
		t.Fatal(err)
	}
	for _, violation := range violations {
		t.Errorf("retired Amazon listing import remains: %s", violation)
	}
}

func TestTemuListingClosureSourcesAreRetired(t *testing.T) {
	paths := append([]string{}, retiredTEMUListingPackages...)
	paths = append(paths,
		"internal/listingadmin/gorm_profit_rule_api.go", "internal/listingadmin/gorm_pricing_rule_api.go",
		"prompts/temu/attribute_mapping.yaml", "prompts/temu/content_rewriter.yaml", "prompts/temu/sku_variant_mapping.yaml",
		"data/prohibited_items_temu.json", "data/sensitive_words_temu.json",
	)
	for _, relative := range paths {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Errorf("retired TEMU listing source remains: %s (stat: %v)", relative, err)
		}
	}
	for relative, retired := range map[string][]string{
		"internal/listingadmin": {"ProfitRuleAPI", "ProfitRuleReqDTO", "ProfitRuleRespDTO", "PricingRuleAPI", "PricingRuleReqDTO", "PricingRuleRespDTO", "NewGormProfitRuleAPI", "gormProfitRuleAPI", "ProfitRuleToRespDTO", "NewGormPricingRuleAPI", "gormPricingRuleAPI", "PricingRuleToRespDTO", "DailyListingCountGetReqDTO", "DailyListingCountSetReqDTO", "DailyListingCountRespDTO", "TryConsumeDailyQuotaReqDTO", "TryConsumeDailyQuotaRespDTO", "RollbackDailyQuotaReqDTO", "DailyListingCountAPI"},
		"internal/core/config":  {"SensitiveWordsConfig", "DefaultSensitiveWordsConfig", "GetWordsForLanguage", "AddStaticWords", "AddDynamicWords"},
	} {
		root := filepath.Join("..", filepath.FromSlash(relative))
		index, err := loadGoFileIndex(root, "")
		if err != nil {
			t.Fatal(err)
		}
		for path := range index.files {
			if filepath.Dir(path) != root || strings.HasSuffix(path, "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, declaration := range file.Decls {
				var names []*ast.Ident
				switch declaration := declaration.(type) {
				case *ast.FuncDecl:
					names = append(names, declaration.Name)
				case *ast.GenDecl:
					for _, spec := range declaration.Specs {
						switch spec := spec.(type) {
						case *ast.TypeSpec:
							names = append(names, spec.Name)
						case *ast.ValueSpec:
							names = append(names, spec.Names...)
						}
					}
				}
				for _, name := range names {
					for _, symbol := range retired {
						if name.Name == symbol {
							t.Errorf("%s recreates retired TEMU listing declaration %s", path, symbol)
						}
					}
				}
			}
		}
	}
}

func TestListingKitTaskBackendSourcesAreRetired(t *testing.T) {
	for _, relative := range []string{"internal/temu/image", "internal/temu/handlerbase/types.go", "internal/temu/api/image", "prompts/temu/vision_detector.yaml", "prompts/shein/attribute_selector.yaml", "prompts/shein/attribute_value_fallback.yaml", "prompts/shein/translation.yaml"} {
		if _, err := os.Stat(filepath.Join("..", filepath.FromSlash(relative))); !os.IsNotExist(err) {
			t.Errorf("retired image artifact remains: %s (stat: %v)", relative, err)
		}
	}
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
		"internal/shein/addresscopy/helpers.go",
		"internal/shein/addresscopy/helpers_test.go",
		"internal/shein/addresscopy/models.go",
		"internal/shein/addresscopy/service.go",
		"internal/shein/attribute.go",
		"internal/shein/error_exports.go",
		"internal/shein/product.go",
		"internal/shein/sensitive_words_filter.go",
		"internal/shein/task_reason.go",
		"internal/shein/types.go",
		"internal/shein/category/ai_selector.go",
		"internal/shein/category/category_repo.go",
		"internal/shein/category/tree.go",
		"internal/shein/content/words_processor.go",
		"internal/shein/context/attribute.go",
		"internal/shein/context/context.go",
		"internal/shein/context/context_test.go",
		"internal/shein/context/sale_attribute_selection_state.go",
		"internal/shein/languageconfig/languages.go",
		"internal/shein/languageconfig/languages_test.go",
		"internal/shein/managedclient/api_client_test.go",
		"internal/shein/managedclient/bridge.go",
		"internal/shein/managedclient/manager.go",
		"internal/shein/managedclient/runtime_adapters.go",
		"internal/shein/mapping/builder.go",
		"internal/shein/mapping/builder_owner_test.go",
		"internal/shein/mapping/handler.go",
		"internal/shein/mapping/service.go",
		"internal/shein/mapping/strategies.go",
		"internal/shein/mapping/strategies_test.go",
		"internal/shein/mapping/types.go",
		"internal/shein/product/attribute/apparel_size_resolver.go",
		"internal/shein/product/attribute/custom_processor.go",
		"internal/shein/product/attribute/custom_processor_test.go",
		"internal/shein/product/attribute/fill.go",
		"internal/shein/product/attribute/importance.go",
		"internal/shein/product/attribute/importance_test.go",
		"internal/shein/product/attribute/input.go",
		"internal/shein/product/attribute/mapper.go",
		"internal/shein/product/attribute/mapper_runtime_input.go",
		"internal/shein/product/attribute/mapper_test.go",
		"internal/shein/product/attribute/matcher.go",
		"internal/shein/product/attribute/matcher_test.go",
		"internal/shein/product/attribute/platform_value_domain.go",
		"internal/shein/product/attribute/platform_value_fallback.go",
		"internal/shein/product/attribute/platform_value_fallback_llm.go",
		"internal/shein/product/attribute/platform_value_fallback_llm_test.go",
		"internal/shein/product/attribute/platform_value_resolver.go",
		"internal/shein/product/attribute/prompt.go",
		"internal/shein/product/attribute/selector.go",
		"internal/shein/product/attribute/size_chart_resolver.go",
		"internal/shein/product/attribute/size_rules.go",
		"internal/shein/product/attribute/size_system.go",
		"internal/shein/product/attribute/structured_size_resolver.go",
		"internal/shein/product/attribute/task_context_sale_attribute_selection_test.go",
		"internal/shein/product/attribute/template.go",
		"internal/shein/product/attribute/types.go",
		"internal/shein/product/attribute/utils.go",
		"internal/shein/product/attribute/validate_repair.go",
		"internal/shein/product/attribute/validate_repair_test.go",
		"internal/shein/product/attribute/validator.go",
		"internal/shein/product/attribute/sale/batch_processor.go",
		"internal/shein/product/attribute/sale/batch_processor_test.go",
		"internal/shein/product/attribute/sale/cache_consistency.go",
		"internal/shein/product/attribute/sale/cache_consistency_test.go",
		"internal/shein/product/attribute/sale/comparison.go",
		"internal/shein/product/attribute/sale/context.go",
		"internal/shein/product/attribute/sale/context_test.go",
		"internal/shein/product/attribute/sale/debug.go",
		"internal/shein/product/attribute/sale/file_saver.go",
		"internal/shein/product/attribute/sale/filter.go",
		"internal/shein/product/attribute/sale/gpt.go",
		"internal/shein/product/attribute/sale/gpt_test.go",
		"internal/shein/product/attribute/sale/handler.go",
		"internal/shein/product/attribute/sale/json_parser.go",
		"internal/shein/product/attribute/sale/json_parser_test.go",
		"internal/shein/product/attribute/sale/metadata.go",
		"internal/shein/product/attribute/sale/metadata_test.go",
		"internal/shein/product/attribute/sale/preparation.go",
		"internal/shein/product/attribute/sale/product_data.go",
		"internal/shein/product/attribute/sale/prompt_generator.go",
		"internal/shein/product/attribute/sale/request.go",
		"internal/shein/product/attribute/sale/request_test.go",
		"internal/shein/product/attribute/sale/sale_attribute_input.go",
		"internal/shein/product/attribute/sale/sale_attribute_output.go",
		"internal/shein/product/attribute/sale/single_processor.go",
		"internal/shein/product/attribute/sale/single_processor_test.go",
		"internal/shein/product/attribute/sale/smart_filter.go",
		"internal/shein/product/attribute/sale/smart_filter_input.go",
		"internal/shein/product/attribute/sale/smart_filter_test.go",
		"internal/shein/product/attribute/sale/utils.go",
		"internal/shein/product/attribute/sale/utils_test.go",
		"internal/shein/product/attribute/sale/validation.go",
		"internal/shein/product/attribute/sale/validation_test.go",
		"internal/shein/product/attribute/sale/value_filter.go",
		"internal/shein/product/attribute/sale/value_filter_test.go",
		"internal/shein/product/attribute_repo.go",
		"internal/shein/product/init_data.go",
		"internal/shein/product/init_data_test.go",
		"internal/shein/product/inventory_repo.go",
		"internal/shein/product/product_manager.go",
		"internal/shein/product/product_repo.go",
		"internal/shein/product/result_merger.go",
		"internal/shein/product/shelf_quota.go",
		"internal/shein/product/shelf_quota_test.go",
		"internal/shein/product/spu_limit.go",
		"internal/shein/product/spu_record.go",
		"internal/shein/product/spu_record_test.go",
		"internal/shein/product/build/attribute_builder.go",
		"internal/shein/product/build/attribute_builder_test.go",
		"internal/shein/product/build/attribute_classifier.go",
		"internal/shein/product/build/build_attribute.go",
		"internal/shein/product/build/build_input.go",
		"internal/shein/product/build/skc_list.go",
		"internal/shein/product/build/spu.go",
		"internal/shein/product/image/processor.go",
		"internal/shein/product/image/validation.go",
		"internal/shein/product/size/chart.go",
		"internal/shein/product/size/chart_test.go",
		"internal/shein/product/size/normalizer.go",
		"internal/shein/product/size/normalizer_test.go",
		"internal/shein/product/skc/attribute_strategy.go",
		"internal/shein/product/skc/builder.go",
		"internal/shein/product/skc/image.go",
		"internal/shein/product/skc/selection_strategy.go",
		"internal/shein/product/skc/selection_strategy_test.go",
		"internal/shein/product/skc/skc_build_input.go",
		"internal/shein/product/skc/skc_build_output.go",
		"internal/shein/product/skc/translation.go",
		"internal/shein/product/skc/translation_test.go",
		"internal/shein/product/skc/validation.go",
		"internal/shein/product/skc/validation_test.go",
		"internal/shein/product/skc/variant.go",
		"internal/shein/product/skc/variant_runtime_test.go",
		"internal/shein/product/sku/builder.go",
		"internal/shein/product/sku/creator.go",
		"internal/shein/product/sku/generator.go",
		"internal/shein/product/sku/image_fixer.go",
		"internal/shein/product/sku/image_processor.go",
		"internal/shein/product/sku/sku_runtime_input.go",
		"internal/shein/product/sku/strategy.go",
		"internal/shein/product/sku/strategy_test.go",
		"internal/shein/product/sku/utils.go",
		"internal/shein/product/sku/utils_test.go",
		"internal/shein/product/variant/composite_matcher.go",
		"internal/shein/product/variant/exact_matcher.go",
		"internal/shein/product/variant/exact_matcher_test.go",
		"internal/shein/product/variant/fuzzy_matcher.go",
		"internal/shein/product/variant/match_helpers.go",
		"internal/shein/product/variant/matcher.go",
		"internal/shein/product/variant/matcher_helpers.go",
		"internal/shein/productdata/fetch_and_cache.go",
		"internal/shein/productdata/fetch_and_cache_variants.go",
		"internal/shein/productdata/raw_json.go",
		"internal/shein/productdata/raw_json_test.go",
		"internal/shein/productdata/variant_data.go",
		"internal/shein/productdata/variant_limit.go",
		"internal/shein/productdata/variant_limit_test.go",
		"internal/shein/publish/checker.go",
		"internal/shein/publish/error_handler.go",
		"internal/shein/publish/exists_check.go",
		"internal/shein/publish/handler.go",
		"internal/shein/publish/handler_test.go",
		"internal/shein/publish/mapping_helper.go",
		"internal/shein/publish/publish_input.go",
		"internal/shein/publish/publish_input_test.go",
		"internal/shein/publish/repository_helpers.go",
		"internal/shein/publish/repository_helpers_test.go",
		"internal/shein/publish/result.go",
		"internal/shein/publish/result_test.go",
		"internal/shein/publish/sale_attribute_policy.go",
		"internal/shein/publish/saver.go",
		"internal/shein/publish/validator.go",
		"internal/shein/publish/validator_attributes.go",
		"internal/shein/publish/validator_duplicate_test.go",
		"internal/shein/publish/validator_test.go",
		"internal/shein/publish/variant_success.go",
		"internal/shein/siteconfig/sites.go",
		"internal/shein/siteconfig/sites_test.go",
		"internal/shein/store/region.go",
		"internal/shein/store/region_test.go",
		"internal/shein/store/site.go",
		"internal/shein/store/site_test.go",
		"internal/shein/store/store_id.go",
		"internal/shein/store/store_id_test.go",
		"internal/shein/store/store_info.go",
		"internal/shein/store/store_info_test.go",
		"internal/shein/store/supplier.go",
		"internal/shein/store/warehouse.go",
		"internal/shein/store/warehouse_repo.go",
		"internal/shein/translate/translate.go",
		"internal/shein/translate/translate_repo.go",
		"internal/shein/translate/translate_test.go",
		"internal/shein/validation/build_attribute.go",
		"internal/shein/validation/build_attribute_test.go",
		"internal/shein/validation/daily_limit.go",
		"internal/shein/validation/daily_limit_test.go",
		"internal/shein/validation/filter_rule.go",
		"internal/shein/validation/filter_rule_test.go",
		"internal/shein/validation/get_rule.go",
		"internal/shein/validation/get_rule_test.go",
		"internal/shein/validation/quantity.go",
		"internal/shein/validation/quantity_test.go",
		"internal/shein/validation/reapply_filter.go",
		"internal/shein/validation/rule_checker.go",
		"internal/sheinbridge/saleattribute/handler.go",
		"internal/sheinbridge/saleattribute/handler_test.go",
		"internal/shein/product",
		"internal/shein/productdata",
		"internal/shein/publish",
		"internal/shein/mapping",
		"internal/shein/validation",
		"internal/shein/translate",
		"internal/shein/addresscopy",
		"internal/shein/managedclient",
		"internal/shein/store",
		"internal/shein/context",
		"internal/shein/languageconfig",
		"internal/shein/siteconfig",
		"internal/sheinbridge/saleattribute",
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
		"internal/temu/product/name_validator.go", "internal/temu/product/name_optimizer.go",
		"internal/temu/product/description_validator.go", "internal/temu/product/description_validator_rules.go",
		"internal/temu/product/description_validator_enhance.go", "internal/temu/product/description_validator_score.go",
		"internal/temu/product/brand_clear_handler.go", "internal/temu/product/name_validator_test.go",
		"internal/temu/product/brand_clear_handler_test.go",
		"internal/marketplace/temu/publishing/name_validation_policy.go", "internal/marketplace/temu/publishing/name_validation_policy_test.go",
		"internal/marketplace/temu/publishing/description_sanitization_policy.go", "internal/marketplace/temu/publishing/description_sanitization_policy_test.go",
		"internal/marketplace/temu/publishing/brand_text_policy.go", "internal/marketplace/temu/publishing/brand_text_policy_test.go",
		"internal/marketplace/temu/publishing/name_format_policy_test.go",
		"internal/temu/rules/text_check_handler.go", "internal/temu/rules/text_processor.go",
		"internal/temu/rules/bullet_points_validator.go", "internal/temu/rules/bullet_points_optimizer.go",
		"internal/temu/rules/bullet_points_validator_test.go",
		"internal/temu/property/mapper_adapter.go", "internal/temu/property/orchestrator.go",
		"internal/temu/property/pipeline.go", "internal/temu/property/mapping_stage.go",
		"internal/temu/property/validation_stage.go", "internal/temu/property/fixing_stage.go",
		"internal/temu/property/finalization_stage.go", "internal/temu/property/stage.go",
		"internal/temu/property/context.go", "internal/temu/property/cache.go",
		"internal/temu/property/feature_detector.go",
		"internal/marketplace/temu/publishing/submission_name_policy.go",
		"internal/marketplace/temu/publishing/submission_name_policy_test.go",
		"internal/marketplace/temu/publishing/name_format_policy.go",
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
			"QueryAPI          = query.API", "QueryAPIInterface = query.QueryAPI",
			"NewQueryAPI           = query.NewAPI",
			"TextCheckRequest               = query.TextCheckRequest", "TextCheckResponse              = query.TextCheckResponse",
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
			"type QueryAPI interface {\n\tCheckText(request *TextCheckRequest) (*TextCheckResponse, error)",
			"func (a *API) CheckText(request *TextCheckRequest) (*TextCheckResponse, error)",
			"a.postQuery(\"/mms/marigold/query/commit/check_text\", request, a.defaultHeaders(), &result)",
			"func (a *API) postQuery(url string, body any, headers map[string]string, result any) error {",
			"\"method\": \"POST\"", "return a.client.SendTEMURequest(req, result)",
		},
		"internal/temu/api/query/query_types.go": {
			"type CommitDetailRequest struct", "type CommitDetailResponse struct", "type CommitDetailResult struct", "type CommitDetailCategoryTree struct",
			"type SkuSnCheckRequest struct", "type SkuSnCheckResponse struct", "type OutSkuSnItem struct",
			"type OutGoodsSnCheckResult struct", "type OutSkuSnFailItem struct",
			"type TemplateQueryRequest struct", "type TemplateQueryResponse struct", "type TemplateInfo struct",
			"type CostTemplateRequest struct", "type CostTemplateResponse struct", "type CostTemplateResult struct", "type CostTemplateItem struct",
			"type TextCheckRequest struct {\n\tContent string `json:\"content\"`\n\tType    int    `json:\"type\"`\n}",
			"type TextCheckResponse struct {\n\tSuccess   bool `json:\"success\"`\n\tErrorCode int  `json:\"error_code\"`\n\tResult    struct {\n\t\tSuccess bool `json:\"success\"`\n\t} `json:\"result\"`\n}",
		},
		"internal/temu/api/template/types.go": {
			"type TemplateQueryRequest struct", "type TemplateQueryResponse struct", "type TemplateQueryResult struct",
			"type TemplateInfo struct", "type UserInputParentSpec struct",
		},
		"internal/marketplace/temu/goods/images.go": {
			"func ValidImageObservation(", "func resolveImages(", "inventory asset.ApprovedAssetInventory",
		},
		"internal/temu/api/client/client.go": {
			"func NewAPIClient(", "func (c *APIClient) initializeMallID(",
			"func (c *APIClient) GetMallID(", "func (c *APIClient) SetMallID(", ".UpdateStoreId(",
		},
		"internal/temu/api/client/interfaces.go":     {"type StoreRuntime interface", "GetStoreAPI(", "type ClientAPI interface {", "SendTEMURequest(request map[string]any, result any) error"},
		"internal/temu/api/client/cookie_manager.go": {"type CookieManager struct"},
		"internal/temu/api/product/goods_types.go": {
			"type Product struct", "type GoodsBasicInfo struct", "type ServicePromise struct", "type ExtraInfo struct",
			"type CategoryTree struct", "type Disclaimer struct",
		},
		"internal/shared/ptr/pointer.go":            {"func BoolPtr("},
		"internal/listingadmin/gorm_store_api.go":   {"func NewGormStoreAPI(", "func (a gormStoreAPI) GetStore(", "func (a gormStoreAPI) UpdateStoreId(", "func (a gormStoreAPI) UpdateStoreStatus("},
		"internal/listingadmin/store_repository.go": {"type GormStoreRepository struct", "func (r *GormStoreRepository) GetStore("},
		"internal/storecenter/gorm_repository.go":   {"type GormStoreRepository struct"},
		"internal/pkg/appenv/apputil.go":            {"func GetInstanceID("},
		"internal/marketplace/temu/goods/builder.go": {
			"func Build(", "snapshot.SelectableLeaf", "func checkLocal(",
			"p.ServicePromise.ShipmentLimitDay", "value.ID == p.ServicePromise.CostTemplateID",
		},
		"internal/marketplace/temu/goods/supplier_category.go": {"func checkSupplierCategory(", "SelectableLeaf()"},
		"internal/marketplace/temu/rules/local_variations.go":  {"func CheckLocalVariations("},
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
		"internal/marketplace/productpolicy/rule_checker.go": {
			"type RuleChecker struct", "func NewRuleChecker(", "CheckPriceRange(", "CheckInventory(", "CheckRating(",
			"CheckReviewCount(", "CheckDeliveryTime(", "CheckFulfillmentType(", "CheckAllRules(",
		},
		"internal/marketplace/productpolicy/filter_rule.go":      {"type FilterRule struct"},
		"internal/marketplace/productpolicy/price_helper.go":     {"func GetProductPrice(", "func GetInventory("},
		"internal/marketplace/productpolicy/fulfillment.go":      {"func IsFBAFulfillment(", "func IsAMZSeller("},
		"internal/marketplace/temu/goods/local_properties.go":    {"func CheckLocalProperties("},
		"internal/marketplace/temu/goods/supplier_properties.go": {"func CheckSupplierProperties("},
		"internal/marketplace/temu/goods/supplier_specs.go":      {"func checkSupplierSpecifications("},
		"internal/marketplace/temu/rules/local_numbers.go":       {"func (c *Catalog) CheckLocalNumbers("},
		"internal/listingadmin/management_api_types.go": {
			"type StoreAPI interface", "type StoreRespDTO struct", "type StoreIdUpdateReqDTO struct", "type StoreStatusUpdateReqDTO struct",
		},
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
		"internal/shein/activity/promotion_create_products.go":                  {"func buildCalculateRequestForPromotionProducts(", "func promotionSKUPriceInputs("},
		"internal/shein/activity/time_limited.go":                               {"func buildCreateActivityRequest(", "func isTimeLimitedDiscountValid(", "func validatePriceRisk("},
		"internal/shein/activity/validation_utils.go":                           {"func ValidateDropRate(", "func CalculateDropRateFromDiscount("},
		"internal/shein/activity/errors.go":                                     {"ErrProductPriceRisk"},
		"internal/shein/pricing/price_manager.go":                               {"type PriceManager struct", "func NewPriceManager(", "type CostPriceRequest struct"},
		"internal/shein/models.go":                                              {"type AmazonMonitorData struct", "ASIN", "LastCheckTime", "json:\"asin\"", "json:\"last_check_time\""},
		"internal/shein/productsync/product_sync_types.go":                      {"type EnrichedSkuInfo struct", "type EnrichedSkcInfo struct", "AmazonMonitorData"},
		"internal/marketplace/shein/publishing/attribute_value_match_policy.go": {"func MatchPlatformAttributeValue("},
		"internal/marketplace/shein/publishing/site_policy.go":                  {"func DefaultSiteListByRegion("},
		"internal/shein/content/cleanup_context.go":                             {"type CleanupContext struct", "type ValidationResult struct", "type SKCValidationMessages struct", "SourceBrand", "other_language_message_map", "otherLanguageMessageMap"},
		"internal/publishing/shein/attribute_value_matcher.go":                  {"sheinmarketpub.MatchPlatformAttributeValue(", "content.SanitizeForSheinAttribute(", "\"attribute_value_legacy_matcher\""},
		"internal/publishing/shein/submit_site_sku_policy.go":                   {"sheinmarketpub.DefaultSiteListByRegion("},
		"internal/publishing/shein/sensitive_content_sanitizer.go":              {"content.CleanupContext", "ensureSanitizerContext", "authorizedbrand.FromContext"},
		"internal/shein/submitprep/sensitive_words.go":                          {"sheincontent.CleanupContext", "sheincontent.ValidationResult", "applyAuthorizedBrandFromRuntime"},
	} {
		content, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		for _, token := range required {
			if !strings.Contains(strings.ReplaceAll(string(content), "\r\n", "\n"), token) {
				t.Errorf("%s must preserve retained platform/shared contract %q", relative, token)
			}
		}
	}
	productRoot := filepath.Join("..", "internal", "temu", "product")
	if err := filepath.WalkDir(filepath.Dir(productRoot), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasPrefix(path, productRoot+string(os.PathSeparator)) {
			return nil
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
	retiredProductContentSymbols := map[string]bool{
		"ProductNameValidator": true, "NewProductNameValidator": true,
		"ProductNameOptimizer": true, "NewProductNameOptimizer": true,
		"ProductDescriptionValidator": true, "NewProductDescriptionValidator": true,
		"BrandClearHandler": true, "NewBrandClearHandler": true,
		"DescriptionValidationResult": true,
	}
	retiredRulesTextSymbols := map[string]bool{
		"TextCheckHandler": true, "NewTextCheckHandler": true,
		"TextProcessor": true, "NewTextProcessor": true,
		"BulletPointsValidator": true, "NewBulletPointsValidator": true,
		"BulletPointsOptimizer": true, "NewBulletPointsOptimizer": true,
		"BulletPointValidationResult": true, "OptimizationStrategy": true,
	}
	retiredPropertyPipelineSymbols := map[string]bool{
		"PropertyMapper": true, "NewPropertyMapper": true,
		"PropertyMappingOrchestrator": true, "NewPropertyMappingOrchestrator": true,
		"PropertyProcessingPipeline": true, "DefaultPropertyProcessingPipeline": true, "NewPropertyProcessingPipeline": true,
		"PropertyMappingStage": true, "NewPropertyMappingStage": true,
		"PropertyValidationStage": true, "NewPropertyValidationStage": true, "NewPropertyValidationStageWithEngine": true,
		"PropertyFixingStage": true, "NewPropertyFixingStage": true,
		"PropertyFinalizationStage": true, "NewPropertyFinalizationStage": true,
		"PropertyStage": true, "BasePropertyStage": true, "NewBasePropertyStage": true,
		"StageError": true, "NewStageError": true, "StageResult": true, "NewStageResult": true,
		"PropertyStageChain": true, "NewPropertyStageChain": true,
		"PropertyContext": true, "PropertyFeature": true, "ProcessingStats": true, "StageStats": true,
		"ProcessingConfig": true, "NewPropertyContext": true, "NewProcessingStats": true, "NewDefaultProcessingConfig": true,
		"PropertyCache": true, "InMemoryPropertyCache": true, "CacheItem": true,
		"NewPropertyCache": true, "NewPropertyCacheWithTTL": true, "GenerateFeatureCacheKey": true, "CacheStats": true,
		"PropertyFeatureDetector": true, "NewPropertyFeatureDetector": true,
	}
	for _, pkg := range []string{"sku", "filter", "product", "rules", "handlerbase", "property"} {
		root := filepath.Join("..", "internal", "temu", pkg)
		if err := filepath.WalkDir(filepath.Dir(root), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
				return nil
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
			if pkg == "product" || pkg == "rules" || pkg == "property" {
				for _, declaration := range file.Decls {
					var names []*ast.Ident
					switch declaration := declaration.(type) {
					case *ast.FuncDecl:
						if declaration.Recv == nil {
							names = append(names, declaration.Name)
						}
					case *ast.GenDecl:
						for _, spec := range declaration.Specs {
							switch spec := spec.(type) {
							case *ast.TypeSpec:
								names = append(names, spec.Name)
							case *ast.ValueSpec:
								names = append(names, spec.Names...)
							}
						}
					}
					for _, name := range names {
						if (pkg == "product" && retiredProductContentSymbols[name.Name]) ||
							(pkg == "rules" && retiredRulesTextSymbols[name.Name]) ||
							(pkg == "property" && retiredPropertyPipelineSymbols[name.Name]) {
							t.Errorf("%s recreates retired TEMU content declaration %s", path, name.Name)
						}
					}
				}
			}
			if pkg == "handlerbase" {
				for _, declaration := range file.Decls {
					if group, ok := declaration.(*ast.GenDecl); ok {
						for _, spec := range group.Specs {
							if typ, ok := spec.(*ast.TypeSpec); ok && typ.Name.Name == "ImageRequirement" {
								t.Errorf("%s must not recreate retired handlerbase.ImageRequirement", path)
							}
						}
					}
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
	publishingPaths, err := filepath.Glob(filepath.Join("..", "internal", "marketplace", "temu", "publishing", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	retiredPublishingContentSymbols := map[string]bool{
		"SanitizeProductName": true, "ProductNameSanitizationResult": true,
		"SanitizeProductDescription": true, "ProductDescriptionSanitizationResult": true,
		"RemoveBrandFromText": true, "NormalizeParenthesesSpacing": true,
		"NormalizeProductSubmissionName": true,
	}
	for _, path := range publishingPaths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			var names []*ast.Ident
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				if declaration.Recv == nil {
					names = append(names, declaration.Name)
				}
			case *ast.GenDecl:
				for _, spec := range declaration.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						names = append(names, spec.Name)
					case *ast.ValueSpec:
						names = append(names, spec.Names...)
					}
				}
			}
			for _, name := range names {
				if retiredPublishingContentSymbols[name.Name] {
					t.Errorf("%s recreates retired TEMU publishing content declaration %s", path, name.Name)
				}
			}
		}
	}
	strxPaths, err := filepath.Glob(filepath.Join("..", "internal", "shared", "strx", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range strxPaths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		// Only package declarations recreate the retired shared contract.
		// SHEIN receiver methods, fields, selectors and local names are unrelated.
		for _, declaration := range file.Decls {
			var names []*ast.Ident
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				if declaration.Recv == nil {
					names = append(names, declaration.Name)
				}
			case *ast.GenDecl:
				for _, spec := range declaration.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						names = append(names, spec.Name)
					case *ast.ValueSpec:
						names = append(names, spec.Names...)
					}
				}
			}
			for _, name := range names {
				if name.Name == "CleanProductTitle" || name.Name == "CleanWhitespace" {
					t.Errorf("%s recreates retired shared text declaration %s", path, name.Name)
				}
			}
		}
	}
	contextPaths, err := filepath.Glob(filepath.Join("..", "internal", "temu", "context", "*.go"))
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
		"PaddedImages": true, "PaddedImageSizes": true,
	}
	for _, contextPath := range contextPaths {
		if strings.HasSuffix(contextPath, "_test.go") {
			continue
		}
		contextFile, err := parser.ParseFile(token.NewFileSet(), contextPath, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		// Inspect package owners, including aliases in new context files, without
		// banning unrelated selectors or local names in other domains.
		for _, declaration := range contextFile.Decls {
			switch declaration := declaration.(type) {
			case *ast.GenDecl:
				for _, spec := range declaration.Specs {
					typ, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if typ.Name.Name == "AssetState" {
						t.Errorf("%s must not recreate retired context.AssetState", contextPath)
					}
					state, ok := typ.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, field := range state.Fields.List {
						if len(field.Names) == 0 {
							embedded := field.Type
							if pointer, ok := embedded.(*ast.StarExpr); ok {
								embedded = pointer.X
							}
							if name, ok := embedded.(*ast.Ident); ok && name.Name == "AssetState" {
								t.Errorf("%s must not embed retired context.AssetState", contextPath)
							}
						}
						for _, name := range field.Names {
							if (typ.Name.Name == "PublishState" || typ.Name.Name == "TemuTaskContext") && retiredFields[name.Name] {
								t.Errorf("TEMU %s must not retain exclusive publish field %s", typ.Name, name)
							}
							if retiredAssetFields[name.Name] || name.Name == "AssetState" {
								t.Errorf("TEMU %s must not retain exclusive image field %s", typ.Name, name)
							}
						}
					}
				}
			case *ast.FuncDecl:
				if declaration.Recv != nil && retiredMethods[declaration.Name.Name] {
					t.Errorf("TEMU context must not retain exclusive publish method %s", declaration.Name)
				}
			}
		}
		ast.Inspect(contextFile, func(node ast.Node) bool {
			literal, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			owner, ok := literal.Type.(*ast.Ident)
			if !ok || (owner.Name != "TemuTaskContext" && owner.Name != "AssetState") {
				return true
			}
			for _, element := range literal.Elts {
				if entry, ok := element.(*ast.KeyValueExpr); ok {
					if key, ok := entry.Key.(*ast.Ident); ok && (key.Name == "AssetState" || retiredAssetFields[key.Name]) {
						t.Errorf("%s must not initialize exclusive image field %s", contextPath, key.Name)
					}
				}
			}
			return true
		})
	}
	policyIndex, err := loadGoFileIndex(filepath.Join("..", "internal", "marketplace", "temu"), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, policyName := range []string{"submission_validation_policy.go", "sku_value_policy.go"} {
		for policyPath := range policyIndex.files {
			if policyPath != filepath.Join("..", "internal", "marketplace", "temu", "publishing", policyName) {
				continue
			}
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
	}
	for _, platform := range []string{"shein", "temu"} {
		root := filepath.Join("..", "internal", platform, "pricing")
		index, err := loadGoFileIndex(filepath.Dir(root), "")
		if err != nil {
			t.Fatal(err)
		}
		for path := range index.files {
			if filepath.Dir(path) != root || strings.HasSuffix(path, "_test.go") {
				continue
			}
			content, err := os.ReadFile(path)
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
					t.Errorf("%s must not recreate retired automatic pricing %q", path, retired)
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
		"internal/amazon/api/client.go",
		"internal/listingruntime/types.go", "internal/listingadmin/schema_migrate.go",
		"internal/core/config/type_platform.go",
		"internal/temu/api/api.go", "internal/temu/api/client/interfaces.go",
		"internal/temu/api/client/client.go", "internal/temu/api/client/cookie_manager.go",
		"internal/temu/api/client/auth_pause_handler.go", "internal/temu/api/inventory/inventory.go",
		"internal/temu/api/product/request_types.go",
		"internal/temu/api/product/api.go", "internal/temu/api/query/query.go", "internal/temu/api/query/query_types.go",
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
	for _, command := range []string{"current-application", "shein-login-worker", "image-agent-temporal-worker", "source-account-registry-schema-init", "product-listing-api-schema-migrate", "1688-local-agent", "1688-batch-import", "1688-public-browser-collector"} {
		t.Run(command, func(t *testing.T) {
			cmd := exec.Command("go", "list", "-deps", "./cmd/"+command)
			cmd.Dir = ".."
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go list current command: %v\n%s", err, output)
			}
			for _, dependency := range strings.Fields(string(output)) {
				for _, relative := range append(append([]string{}, retiredTEMUListingPackages...), retiredAmazonListingPackages...) {
					retired := "task-processor/" + relative
					if dependency == retired || strings.HasPrefix(dependency, retired+"/") {
						t.Errorf("%s still compiles retired listing dependency %s", command, dependency)
					}
				}
				for _, retired := range []string{"task-processor/internal/shein/product", "task-processor/internal/shein/productdata", "task-processor/internal/shein/publish", "task-processor/internal/shein/mapping", "task-processor/internal/shein/validation", "task-processor/internal/shein/translate", "task-processor/internal/shein/addresscopy", "task-processor/internal/shein/managedclient", "task-processor/internal/shein/store", "task-processor/internal/shein/context", "task-processor/internal/shein/languageconfig", "task-processor/internal/shein/siteconfig", "task-processor/internal/sheinbridge/saleattribute", "task-processor/internal/shein/inventory", "task-processor/internal/temu/sync", "task-processor/internal/temu/bulkrelist", "task-processor/internal/temu/store", "task-processor/internal/app/bootstrap", "task-processor/internal/app/consumer", "task-processor/internal/app/runner", "task-processor/internal/app/scheduler", "task-processor/internal/app/runtime/listingcontrol", "task-processor/internal/app/runtime/sheinplatformrecovery", "task-processor/internal/listingcontrol", "task-processor/internal/app/task", "task-processor/internal/app/taskstatus", "task-processor/internal/processor", "task-processor/internal/app/crawler/distributed", "task-processor/internal/listingruntime/local", "task-processor/internal/platforms", "task-processor/internal/listingkit", "task-processor/internal/tenantbridge", "task-processor/internal/compatibility", "task-processor/internal/sourceaccount", "task-processor/internal/integration/persistence/sourceaccount/ownershipmigration", "task-processor/internal/taskrpcapi", "task-processor/internal/workspace/shein", "task-processor/internal/core/metrics"} {
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

func TestTemuImageAPIAndVisionResourcesAreRetired(t *testing.T) {
	apiPaths, err := filepath.Glob(filepath.Join("..", "internal", "temu", "api", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	retiredNames := map[string]bool{
		"ImageAPI": true, "NewImageAPI": true, "UploadSignature": true,
		"SignatureResponse": true, "UploadResult": true, "ValidationResult": true,
	}
	for _, path := range apiPaths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		// Only package declarations recreate the retired root API contract;
		// selectors, fields, methods and local names can belong to retained APIs.
		for _, declaration := range file.Decls {
			var names []*ast.Ident
			switch declaration := declaration.(type) {
			case *ast.FuncDecl:
				if declaration.Recv == nil {
					names = append(names, declaration.Name)
				}
			case *ast.GenDecl:
				for _, spec := range declaration.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						names = append(names, spec.Name)
					case *ast.ValueSpec:
						names = append(names, spec.Names...)
					}
				}
			}
			for _, name := range names {
				if retiredNames[name.Name] {
					t.Errorf("%s recreates retired TEMU root API declaration %s", path, name.Name)
				}
			}
		}
	}
	for _, relative := range []string{"internal/prompt/keys.go", "internal/promptmgmt/catalog.go", "internal/promptmgmt/catalog_manifest.go"} {
		content, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		for _, retired := range []string{"KTemuAttributeMappingSystem", "KTemuContentRewriterSystem", "KTemuSkuVariantMappingSystem", "temu.attribute_mapping.system", "temu.content_rewriter.system", "temu.sku_variant_mapping.system", "KTemuVisionDetectorDetect", "temu.vision_detector.detect", "KSheinAttributeSelectorSystem", "KSheinAttributeValueFallbackSystem", "KSheinAttributeValueFallbackUser", "KSheinTranslationBatchOptimizeSystem", "shein.attribute_selector.system", "shein.attribute_value_fallback.system", "shein.attribute_value_fallback.user", "shein.translation.batch_optimize_system"} {
			if strings.Contains(string(content), retired) {
				t.Errorf("%s retains retired vision resource %s", relative, retired)
			}
		}
		if relative == "internal/promptmgmt/catalog_manifest.go" && strings.Contains(string(content), `"Base64Image"`) {
			t.Errorf("%s retains exclusive vision variable metadata", relative)
		}
	}
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

func TestSheinListingMixedDeclarationsAreRetired(t *testing.T) {
	for relative, retired := range map[string][]string{
		"internal/shein":          {"ImageUploadResult", "AttributeImportance", "EnrichedSkuInfo", "EnrichedSkcInfo", "InventoryInfo", "SkuInfo", "MappingInfo", "CostPriceInfo", "Task", "TaskContext", "StepHandler", "VariantFilterInfo", "PreValidResult", "SkcErrorMessage", "NewTaskContext", "AttributeValue", "AttributePriorityConfig", "AttributeValueIDManager", "ResultSaleAttribute", "VariantInfo", "Variant", "AttributeImportanceCalculator", "AttributeMetadata", "AttributeConfig", "ImportanceRules", "ResultAttribute", "SaleAttribute", "AttributeData", "AttributeImportanceResult", "GenerationRequest", "AttributeNameMapping", "ProductVariantData", "GenerateAttributeValue", "GenerateAttribute", "BuildAttributeInfo", "EnhancedGenerateAttribute", "AttributeStrategy", "CustomAttributeResult", "NewAttributeValueIDManager", "NewAttributeImportanceCalculator", "NewAttributeImportanceCalculatorWithRules", "RetryableError", "CookieLoadError", "FilteredError", "TaskHandledError", "NewRetryableError", "NewNonRetryableError", "NewFilteredError", "NewTaskHandledError", "AsTaskHandledError", "IsFilteredError", "IsTaskHandledError", "IsRetryableError", "IsAuthenticationExpiredError", "NewCookieLoadError", "IsCookieLoadError", "SheinProductResponse", "SkcInfo", "ParseTime", "SKUCreationParams", "SKCCreationParams", "SKUBuildRequest", "SensitiveWordsFilter", "NewSensitiveWordsFilterWithDefaults", "NewSensitiveWordsFilter", "initHardcodedWords", "FormatTaskReasonMessage", "FormatTaskStageMessage"},
		"internal/shein/category": {"AICategorySelectorHandler", "NewAICategorySelectorHandler", "CategoryAPI", "NewCategoryAPI", "GetCategoryTreeHandler", "NewGetCategoryTreeHandler"},
		"internal/shein/content":  {"SensitiveWordsFilter", "SensitiveWordsMode", "ModeBlock", "ModeClean", "ModeWarn", "SensitiveWordsProcessor", "NewSensitiveWordsProcessor", "NewSensitiveWordsBlockHandler", "NewSensitiveWordsCleanHandler", "NewSensitiveWordsWarnHandler"},
		"internal/shein/aicache":  {"TypeAttribute", "TypeSaleAttr", "TypeSKCTranslate", "TypeAttrValueFallback"},
	} {
		paths, err := filepath.Glob(filepath.Join("..", filepath.FromSlash(relative), "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		if len(paths) == 0 {
			t.Fatalf("retained mixed package is absent: %s", relative)
		}
		for _, path := range paths {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, declaration := range file.Decls {
				var names []*ast.Ident
				switch declaration := declaration.(type) {
				case *ast.FuncDecl:
					if declaration.Recv == nil || (relative == "internal/shein/category" && declaration.Name.Name == "GetCategoryIDListWithTree") {
						names = append(names, declaration.Name)
					}
				case *ast.GenDecl:
					for _, spec := range declaration.Specs {
						switch spec := spec.(type) {
						case *ast.TypeSpec:
							names = append(names, spec.Name)
						case *ast.ValueSpec:
							names = append(names, spec.Names...)
						}
					}
				}
				for _, name := range names {
					for _, symbol := range retired {
						if name.Name == symbol {
							t.Errorf("%s recreates retired SHEIN declaration %s", path, symbol)
						}
					}
					if relative == "internal/shein/category" && name.Name == "GetCategoryIDListWithTree" {
						t.Errorf("%s retains old category Task method", path)
					}
				}
			}
		}
	}
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", "internal", "shein", "content", "cleanup_context.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(file, func(node ast.Node) bool {
		typ, ok := node.(*ast.TypeSpec)
		if !ok || typ.Name.Name != "CleanupContext" {
			return true
		}
		shape, ok := typ.Type.(*ast.StructType)
		if !ok {
			t.Fatal("CleanupContext must be a narrow value struct")
		}
		expected := map[string]bool{"Context": true, "AuthorizedBrand": true, "SourceBrand": true, "ProductData": true}
		for _, field := range shape.Fields.List {
			if len(field.Names) != 1 || !expected[field.Names[0].Name] {
				t.Errorf("CleanupContext must not embed or add old Task state: %v", field.Names)
				continue
			}
			delete(expected, field.Names[0].Name)
		}
		if len(expected) != 0 {
			t.Errorf("missing cleanup inputs: %v", expected)
		}
		return false
	})
}
