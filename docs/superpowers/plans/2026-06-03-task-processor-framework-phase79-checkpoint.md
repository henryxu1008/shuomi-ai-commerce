> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 79 Checkpoint

### Status

`Phase 79` continued along a framework-oriented direction by extracting a real shared submit-flow contract:

- direct submit readiness gating
- temporal submit readiness gating

### Why This Is A Framework Slice

This cut is framework-level because it unifies a cross-flow contract used by multiple consumers:

- [task_direct_submission_service.go](/D:/code/task-processor/internal/listingkit/task_direct_submission_service.go:46)
- [task_temporal_submission_adapter.go](/D:/code/task-processor/internal/listingkit/task_temporal_submission_adapter.go:87)

Before this round, both flows locally owned the same policy:

- block on base submit readiness
- block on freshness readiness
- convert readiness into `ErrSubmitBlocked`
- resolve outward error text through [firstSubmitReadinessMessage(...)](/D:/code/task-processor/internal/listingkit/service_submit_events.go:141)

That duplication was a stronger framework hotspot than any remaining local helper cleanup.

### What Changed

- [shein_submit_readiness_gate.go](/D:/code/task-processor/internal/listingkit/shein_submit_readiness_gate.go:1)
  - new shared seam:
    - `validateSheinSubmitReadinessGates(...)`
- [task_direct_submission_service.go](/D:/code/task-processor/internal/listingkit/task_direct_submission_service.go:53)
  - direct submit now delegates gating to the shared seam
- [task_temporal_submission_adapter.go](/D:/code/task-processor/internal/listingkit/task_temporal_submission_adapter.go:109)
  - temporal submit now delegates gating to the same shared seam
- [shein_submit_readiness_gate_test.go](/D:/code/task-processor/internal/listingkit/shein_submit_readiness_gate_test.go:1)
  - direct behavior fixtures for base and freshness gate blocking
- [phase79_submit_readiness_gate_boundary_test.go](/D:/code/task-processor/internal/listingkit/phase79_submit_readiness_gate_boundary_test.go:1)
  - boundary coverage for the shared gate contract

### Resulting Shape

Ownership is now:

- each flow still owns its own surrounding orchestration
- shared gate seam owns:
  - base-readiness blocking rule
  - freshness-readiness blocking rule
  - `ErrSubmitBlocked` conversion
  - outward message resolution via `firstSubmitReadinessMessage(...)`

### Why This Matters

This reduces real framework risk:

- direct and temporal submission flows can no longer drift independently on readiness-blocking semantics
- future readiness gate changes now have one explicit contract home
- the shared gate seam composes cleanly with the shared readiness-summary seam from the previous round

### Verification Summary

Fresh verification passed:

```powershell
go test ./internal/listingkit -run "TestValidateSheinSubmitReadinessGates.*|TestSheinSubmitReadinessGateBoundary|TestTaskDirectSubmissionServiceSubmitSheinTaskDirectStopsOnReadinessFailure|TestTaskTemporalSubmissionAdapterValidateReadinessBlocksOnFreshnessGate|TestShapeSheinSubmitReadinessSummary.*|TestSheinSubmitReadinessSummaryBoundary" -count=1
go test ./internal/listingkit/httpapi ./internal/listingkit/temporal -count=1
```
