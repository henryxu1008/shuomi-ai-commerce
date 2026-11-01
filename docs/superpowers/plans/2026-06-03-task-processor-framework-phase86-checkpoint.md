> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

# Phase 86 checkpoint

## Theme

`ListingKit asset-generation projection closure audit`

## Audit result

The `asset-generation projection` line is now **practically complete**.

## Evidence

- the shared bundle seam in `internal/listingkit/asset_generation_projection.go` now owns the multi-consumer outward contract used by:
  - result decoration
  - preview payload decoration
  - export payload decoration
- `buildAssetGenerationOverview(...)` was already a shared owner before phase 85
- `buildGenerationReviewSession(...)` still computes `Overview`, but it does so from a locally cloned `reviewQueue`
- `decorateListingKitResultReview(...)` also recomputes `Overview`, but only after applying review-local queue mutations
- these review/session paths are not rebuilding the same `tasks + summary + queue + overview` bundle contract as result / preview / export

## Architecture conclusion

- keep the review/session layer local to review-state composition
- keep the shared outward projection seam focused on result / preview / export bundle ownership
- do not keep slicing this area for symmetry alone

## Validation

- `go test ./internal/listingkit -run "TestBuildAssetGenerationProjectionBuildsSharedBundle|TestAssetGenerationProjectionBoundary|TestBuildGenerationWorkQueue.*" -count=1`
- `go test ./internal/listingkit -run "TestBuildAssetGenerationProjectionBuildsSharedBundle|TestAssetGenerationProjectionBoundary|TestBuildListingKitExportPopulatesSemanticFields|TestBuildSheinPreviewPayloadPopulatesSemanticFields|TestBuildRevisionSuccessFollowUpDataBuildsSharedPayload|TestBuildGenerationWorkQueue.*" -count=1`
- `go test ./internal/listingkit/httpapi ./internal/listingkit/temporal -count=1`
