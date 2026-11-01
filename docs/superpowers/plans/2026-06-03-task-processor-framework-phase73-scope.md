> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 73 Scope

### Target

Continue the SHEIN freshness framework line by narrowing:

- [evaluateSheinSaleAttributeFreshnessResolution(...)](/D:/code/task-processor/internal/listingkit/shein_submit_sale_attribute_freshness_evaluation.go:30)

### Why This Slice

After `Phase 72`, the invalid-state and repair logic has its own local home.

The remaining mixed ownership is now concentrated around:

- base issue detection
- outward issue aggregation
- final message shaping

That is a cleaner, smaller, and more coherent next cut than jumping elsewhere inside the file.

### Desired Outcome

At the end of `Phase 73`:

- resolution home should move closer to `routing-only` or `near-routing-only`
- message shaping or issue aggregation should have a distinct local owner
- boundary coverage should lock the new split
