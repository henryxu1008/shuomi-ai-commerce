> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

# Phase 84 checkpoint

## Theme

`ListingKit submission projection closure audit`

## Audit result

The `submission projection` line is now **practically complete**.

## Evidence

- `repair center / workspace overview / final review` are still preview-local composition
- `buildSheinRepairCenter(...)` has a single production caller
- `buildSheinFinalReviewPayload(...)` has a single production caller
- the shared submission projection seam now owns the multi-consumer outward contract used by:
  - task result projection
  - task list projection
- no third production consumer is rebuilding the same `workflow status / latest submission / remote summary` mapping nearby

## Architecture conclusion

- keep preview-local assembly local
- keep `deriveSheinWorkflowStatus(...)` and submission fallback policy inside the single projection line unless another consumer appears
- do not keep slicing this area for symmetry alone

## Validation

- `go test ./internal/listingkit -run "TestBuildSheinSubmissionProjection.*|TestSheinSubmissionProjectionBoundary|TestDeriveSheinWorkflowStatus.*|TestApplySheinSubmissionRemoteSummaryFallsBackToPublishRecord|TestBuildTaskListItemUsesLatestSubmissionOutcomeInsteadOfPhaseEvent" -count=1`
- `go test ./internal/listingkit -run "TestSheinSubmissionProjectionBoundary|TestBuildSheinSubmissionProjection.*|TestDeriveSheinWorkflowStatus.*|TestApplySheinSubmissionRemoteSummaryFallsBackToPublishRecord|TestBuildTaskListItemUsesLatestSubmissionOutcomeInsteadOfPhaseEvent|TestSheinSubmitReadinessProjectionBoundary|TestBuildRevisionSuccessFollowUpDataBuildsSharedPayload|TestBuildSheinPreviewPayloadPopulatesSemanticFields" -count=1`
- `go test ./internal/listingkit/httpapi ./internal/listingkit/temporal -count=1`
