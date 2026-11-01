> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 65 Scope

### Candidate Hotspots

After `Phase 64`, the next plausible follow-up slices are:

1. `ListingKit failed-vs-provisional retry action-key mutation ownership`
2. `ListingKit non-retry regular action-key mutation ownership`

### Recommended Next Slice

Recommend prioritizing `ListingKit failed-vs-provisional retry action-key mutation ownership`.

Anchor it in:

- [generation_action_filters_retry_oriented_mutation.go](/D:/code/task-processor/internal/listingkit/generation_action_filters_retry_oriented_mutation.go:1)

Specifically around:

- `retry_failed_generation`
- `inspect_failed_renderer_tasks`
- `upgrade_fallback_assets`
- `retry_provisional_slots`
- `retry_section_generation`

### Why This Slice First

Because `Phase 64` already isolated the broader retry-oriented family into its own local home. Inside that home, the next clearest residual split is between:

- failed retry semantics
- provisional/section retry semantics

That is a tighter and more coherent next cut than immediately switching attention back to the remaining non-retry families.

### Out Of Scope

This next slice should still avoid:

- reopening `Phase 62` through `Phase 64` aggregate routing work
- reopening preview-capability mutation ownership
- reopening clone layering
- redesigning `buildAssetGenerationActionTarget(...)`
- introducing a generic mutation rule framework
- HTTP/bootstrap/runtime changes

### Desired Outcome

At the end of `Phase 65`:

- failed-vs-provisional retry mutation ownership is clearer
- retry-oriented home remains stable
- outward action-target behavior remains unchanged
- guardrails lock the narrowed retry split
