> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 67 Scope

### Candidate Hotspots

After `Phase 66`, the next plausible follow-up slices are:

1. `ListingKit non-retry regular action-key mutation ownership`
2. `ListingKit framework completion audit and residual hotspot review`

### Recommended Next Slice

Recommend prioritizing `ListingKit non-retry regular action-key mutation ownership`.

Anchor it in:

- [generation_action_filters_regular_mutation.go](/D:/code/task-processor/internal/listingkit/generation_action_filters_regular_mutation.go:1)

Specifically around:

- `generate_missing_assets`
- `review_missing_slots`
- `review_ready_assets`
- `continue_publish_review`
- `defer_section_review`
- `approve_section_review`

### Why This Slice First

Because `Phase 63` through `Phase 66` have already reduced the retry-oriented side to a much cleaner shape. The most obvious residual mixed owner now is the non-retry family that still remains in the broader regular-action-key home.

This gives one more bounded, high-signal ownership cut before it makes sense to stop and do a broader completion audit.

### Out Of Scope

This next slice should still avoid:

- reopening `Phase 64` through `Phase 66` retry-oriented work
- reopening preview-capability mutation ownership
- reopening clone layering
- redesigning `buildAssetGenerationActionTarget(...)`
- introducing a generic mutation rule framework
- HTTP/bootstrap/runtime changes

### Desired Outcome

At the end of `Phase 67`:

- non-retry regular action-key mutation ownership is clearer
- retry-oriented homes remain stable
- outward action-target behavior remains unchanged
- guardrails lock the narrowed non-retry split
