> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 69 Scope

### Candidate Hotspots

After `Phase 68`, the next plausible follow-up directions are:

1. `ListingKit framework completion audit and residual hotspot review`
2. `ListingKit last-mile micro-ownership symmetry cleanup`

### Recommended Next Slice

Recommend prioritizing `ListingKit framework completion audit and residual hotspot review`.

### Why This Slice First

Because `Phase 61` through `Phase 68` have now isolated:

- preview-capability mutation
- retry-oriented mutation
- failed retry mutation
- provisional retry mutation
- section retry mutation
- review-ready / section-review mutation
- missing-slot mutation

and reduced the broader action-key mutation homes to routing-only or near-routing-only shape.

At this point, another mechanical decomposition phase is more likely to optimize for symmetry than to remove a real hotspot. A completion audit is now higher value:

- confirm whether any true mixed owners remain
- distinguish real residual hotspots from cosmetic asymmetry
- decide whether this refactor line is effectively done

### Out Of Scope

This next slice should still avoid:

- reopening already-stable action-key mutation homes without evidence
- redesigning `buildAssetGenerationActionTarget(...)`
- introducing generic frameworks just for consistency
- HTTP/bootstrap/runtime changes

### Desired Outcome

At the end of `Phase 69`:

- we have an evidence-backed assessment of whether the ListingKit framework refactor line is effectively complete
- any remaining real hotspots are explicitly named and ranked
- if no meaningful hotspots remain, the line can move toward closure instead of more mechanical slicing
