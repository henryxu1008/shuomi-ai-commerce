> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

# Phase 83 checkpoint

## Theme

`ListingKit shared submission projection contract`

## What changed

- confirmed that `repair center / workspace overview / final review` remain preview-local composition and should not be abstracted further without new consumers
- introduced shared submission projection seam in `internal/listingkit/shein_submission_projection.go`
- centralized the common package-to-outward-status contract for:
  - workflow status
  - latest submission status
  - latest submission error
  - remote submission status
  - remote checked-at
  - remote record id
- rewired the existing adapters in `internal/listingkit/submission_state_support.go` to delegate to the shared projection seam

## Why this is framework-level

- the seam supports multiple outward consumers:
  - task result projection
  - task list projection
- it removes split ownership between `applySheinSubmissionStatusFields(...)` and `applySheinSubmissionRemoteSummary(...)`
- it reduces drift risk between common status fields and task-list remote summary fields

## Guardrails

- behavior coverage:
  - `internal/listingkit/shein_submission_projection_test.go`
- boundary coverage:
  - `internal/listingkit/phase83_shein_submission_projection_boundary_test.go`

## Validation

- `go test ./internal/listingkit -run "TestBuildSheinSubmissionProjection.*|TestSheinSubmissionProjectionBoundary|TestDeriveSheinWorkflowStatus.*|TestApplySheinSubmissionRemoteSummaryFallsBackToPublishRecord|TestBuildTaskListItemUsesLatestSubmissionOutcomeInsteadOfPhaseEvent" -count=1`
- `go test ./internal/listingkit -run "TestSheinSubmissionProjectionBoundary|TestBuildSheinSubmissionProjection.*|TestDeriveSheinWorkflowStatus.*|TestApplySheinSubmissionRemoteSummaryFallsBackToPublishRecord|TestBuildTaskListItemUsesLatestSubmissionOutcomeInsteadOfPhaseEvent|TestSheinSubmitReadinessProjectionBoundary|TestBuildRevisionSuccessFollowUpDataBuildsSharedPayload|TestBuildSheinPreviewPayloadPopulatesSemanticFields" -count=1`
- `go test ./internal/listingkit/httpapi ./internal/listingkit/temporal -count=1`
