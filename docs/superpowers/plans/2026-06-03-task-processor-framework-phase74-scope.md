> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 74 Scope

### Target

Review whether the remaining base-check logic inside:

- [evaluateSheinSaleAttributeFreshnessResolution(...)](/D:/code/task-processor/internal/listingkit/shein_submit_sale_attribute_freshness_evaluation.go:28)

still justifies another refactor phase.

### Why This Scope

After `Phase 73`, the resolution layer is already close to routing-only:

- invalid-state / repair routing has its own home
- message shaping has its own home

The remaining logic is small and semantically coherent enough that another phase may be cosmetic rather than high-signal.

### Desired Outcome

At the end of `Phase 74`:

- either a real remaining hotspot is identified in this SHEIN freshness line
- or this sub-line is declared practically complete and the next work should move to another hotspot
