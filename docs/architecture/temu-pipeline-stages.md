> #301 / #921 当前适用边界：用户已确认并按[旧任务身份与数字映射合同完整闭包 §1–§7/§6.1](legacy-task-identity-mapping-closure-retirement-boundary.md)整批实施。本文旧 createProductImportMappingWithInput 等数字刊登映射实现示例仅为历史，不再是当前复用入口；本轮退出最后 listingruntime.ProductImportMapping 值合同。平台 API/DTO、当前来源/刊登/Store owner 和有效规则保持，不由这项源码退休触发平台写入或数据清理。 当前能力/采集/图像、独立 SHEIN 登录与 Store 同步及原 owner/fence/UNKNOWN 保留；原历史正文/计数/批准/勾选完整保留为后缀，workflow/inventory 不改，不操作既有实例/数据/凭据/进程，不增加试用前置。

# TEMU Pipeline Stages

## Goal

This note captures the stage language that has emerged while refactoring the
TEMU flows. The purpose is to give future refactors a stable vocabulary, so new
changes can align with the existing structure instead of introducing another
ad-hoc flow.

## SKU Main Chain

The SKU / AI response flow is converging on this order:

1. `generate`
2. `finalize`
3. `prepare`
4. `cleanup / apply`
5. `build`
6. `assign`

### 1. Generate

Responsible for producing AI mapping data.

Representative entry points:

- `GenerateAISkuMapping(...)`
- `generateAISkuMappingSingle(...)`
- `generateAISkuMappingInBatches(...)`

### 2. Finalize

Responsible for post-generation normalization that belongs to the generated
response itself, before build preparation starts.

Representative steps:

- `finalizeGeneratedAIMapping(...)`
- `finalizeMergedAIMapping(...)`
- `normalizeGeneratedAIMapping(...)`
- `normalizeMergedAIMapping(...)`

### 3. Prepare

Responsible for making AI mapping build-ready.

Representative steps:

- `prepareAIMappingForBuild(...)`
- `prepareDefaultAIMappingForBuild(...)`
- `ensureAIMappingMatchesVariants(...)`
- `normalizeAIMappingForBuild(...)`

### 4. Cleanup / Apply

Responsible for correcting mappings and applying spec / property adjustments.

Representative modules:

- `SkuMappingProcessor`
- `SpecDimensionUnifier`
- `SpecResolverService`
- `MixedAttributesProcessor`

Representative steps:

- `buildValidAsins(...)`
- `analyzeInvalidAndDuplicateAsins(...)`
- `collectFilteredMappings(...)`
- `finalizeFilteredMappings(...)`
- `countSkusNeedingDefaultSpecs(...)`
- `applyUnifiedSpecs(...)`
- `countTemporarySpecIDs(...)`
- `refreshResolvedSKUIdentifiers(...)`
- `collectDimensionCombinations(...)`
- `applyUnifiedDimensionToSKU(...)`

### 5. Build

Responsible for converting prepared mapping data into SKU / SKC results.

Representative steps:

- `buildSkcsFromPreparedAIMapping(...)`
- `resolveSkcBuildSpecs(...)`
- `buildMultipleSkcs(...)`
- `buildSingleSkc(...)`
- `buildDefaultSkcFromPreparedMapping(...)`
- `buildDefaultSkuFromAIMapping(...)`

### 6. Assign

Responsible for writing built results back to the task context / TEMU product.

Representative steps:

- `assignBuiltVariantSkcs(...)`

## Publish Result Chain

The publish-result flow is converging on a lighter structure:

1. build input
2. consume input in steps
3. execute side effects

Representative input object:

- `SavePublishResultInput`

Representative step functions:

- `logSubmitResponseWithInput(...)`
- `createProductImportMappingWithInput(...)`
- `recordDailyListingCountWithInput(...)`
- `pauseShopUntilEndOfDayWithInput(...)`

The design rule here is:

- input objects own field access, small rules, and metadata assembly
- steps own orchestration and side effects

## Object Roles

### Input objects

Input objects should own:

- field access
- task scope access
- small rules
- request assembly
- metadata assembly

Examples:

- `SavePublishResultInput`
- `AIBatchInput`

### Response objects

Response objects should own:

- reading
- controlled mutation
- aggregation
- construction

Example:

- `AISkuMappingResponse`

## Refactor Guidance

When continuing TEMU refactors, prefer these moves:

1. Move direct field reads from steps into input / response objects.
2. Keep entry functions thin and orchestration-focused.
3. Introduce stage-named helpers only when they clarify the main chain order.
4. Avoid large abstractions until the stage names and boundaries stay stable
   across multiple paths.

## Current Heuristic

If a new function fits one of these stage names:

- `resolve`
- `generate`
- `finalize`
- `prepare`
- `cleanup`
- `apply`
- `build`
- `assign`

then it likely belongs on the main TEMU pipeline and should be named and placed
accordingly.
