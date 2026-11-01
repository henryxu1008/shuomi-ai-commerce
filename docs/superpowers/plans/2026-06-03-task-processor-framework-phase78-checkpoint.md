> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 78 Checkpoint

### Status

`Phase 78` deliberately corrected back toward framework-oriented work.

Instead of continuing local helper decomposition, this round identified and extracted a real shared seam across multiple readiness builders:

- [buildSheinSubmitReadinessWithPodForAction(...)](/D:/code/task-processor/internal/listingkit/shein_submit_readiness.go:24)
- [buildSheinSubmitFreshnessReadiness(...)](/D:/code/task-processor/internal/listingkit/shein_submit_freshness.go:164)

### Why This Is A Framework Slice

This cut is framework-level because it affects a shared contract across flows:

- direct submit and temporal submit both ultimately rely on submit readiness summaries via [firstSubmitReadinessMessage(...)](/D:/code/task-processor/internal/listingkit/service_submit_events.go:141)
- standard submit readiness and freshness readiness both used the same underlying `BuildSubmitReadiness(...)` primitive
- both builders previously owned local `summary post-shaping` policy inline

That made `summary shaping ownership` a real cross-flow, multi-consumer seam rather than a single-function cleanup.

### What Changed

- [shein_submit_readiness_summary.go](/D:/code/task-processor/internal/listingkit/shein_submit_readiness_summary.go:1)
  - new shared seam:
    - `shapeSheinSubmitReadinessSummary(...)`
    - `sheinSubmitReadinessSummaryShape`
- [shein_submit_readiness.go](/D:/code/task-processor/internal/listingkit/shein_submit_readiness.go:260)
  - standard submit readiness now delegates summary post-shaping to the shared seam
- [shein_submit_freshness.go](/D:/code/task-processor/internal/listingkit/shein_submit_freshness.go:184)
  - freshness readiness now delegates summary post-shaping to the same shared seam
- [shein_submit_readiness_summary_test.go](/D:/code/task-processor/internal/listingkit/shein_submit_readiness_summary_test.go:1)
  - new direct behavior fixtures for shared summary shaping policy
- [phase78_submit_readiness_summary_boundary_test.go](/D:/code/task-processor/internal/listingkit/phase78_submit_readiness_summary_boundary_test.go:1)
  - new boundary guardrail for the shared seam

### Resulting Shape

Current ownership is now:

- builders still own their own checklists and wording inputs
- shared seam owns summary post-shape policy:
  - prepend-first-blocker behavior
  - blocking label append
  - warning label append
  - final summary de-duplication

### Why This Matters

This improves a real framework boundary:

- there is now one explicit home for readiness summary post-processing
- cross-flow submit blocking semantics are more consistent
- future readiness builders can reuse the same policy seam without cloning summary logic inline

### Verification Summary

Fresh verification passed:

```powershell
go test ./internal/listingkit -run "TestShapeSheinSubmitReadinessSummary.*|TestSheinSubmitReadinessSummaryBoundary|TestTaskTemporalSubmissionAdapterValidateReadinessBlocksOnFreshnessGate|TestEvaluateSheinAttributeFreshness.*|TestEvaluateSheinSaleAttributeFreshness.*" -count=1
go test ./internal/listingkit/httpapi ./internal/listingkit/temporal -count=1
```
