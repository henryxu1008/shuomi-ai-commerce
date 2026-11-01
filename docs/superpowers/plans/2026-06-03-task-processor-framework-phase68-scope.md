> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 68 Scope

### Candidate Hotspots

After `Phase 67`, the next plausible follow-up slices are:

1. `ListingKit missing-slot action-key mutation ownership`
2. `ListingKit framework completion audit and residual hotspot review`

### Recommended Next Slice

Recommend prioritizing `ListingKit missing-slot action-key mutation ownership`.

Anchor it in:

- [generation_action_filters_regular_mutation.go](/D:/code/task-processor/internal/listingkit/generation_action_filters_regular_mutation.go:1)

Specifically around:

- `generate_missing_assets`
- `review_missing_slots`

### Why This Slice First

Because `Phase 67` already isolated the review-ready / section-review family away from the broader regular-action-key home. The last obvious action-key mutation family still left in that home is the missing-slot pair.

This is the cleanest remaining bounded slice before a broader completion audit becomes more valuable than another refactor phase.

### Out Of Scope

This next slice should still avoid:

- reopening `Phase 64` through `Phase 67` retry/review-ready work
- reopening preview-capability mutation ownership
- reopening clone layering
- redesigning `buildAssetGenerationActionTarget(...)`
- introducing a generic mutation rule framework
- HTTP/bootstrap/runtime changes

### Desired Outcome

At the end of `Phase 68`:

- missing-slot mutation ownership is clearer
- retry-oriented and review-ready homes remain stable
- outward action-target behavior remains unchanged
- guardrails lock the narrowed missing-slot split
