> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 70 Scope

### Candidate Next Directions

After `Phase 69`, the next plausible directions are:

1. `ListingKit framework closure summary and integration readiness review`
2. `Start a new hotspot line elsewhere in ListingKit`

### Recommended Next Direction

Recommend prioritizing `ListingKit framework closure summary and integration readiness review`.

### Why This Direction First

Because the `Phase 69` completion audit did **not** find another same-class hotspot on the current action-target mutation line that clearly justifies more decomposition.

That means the highest-value next move is no longer another refactor slice. It is to:

- summarize what the line accomplished
- note the remaining non-blocking asymmetries
- review merge/integration readiness
- decide whether to close the line or deliberately switch to a different hotspot

### Out Of Scope

This next direction should avoid:

- speculative decomposition done only for symmetry
- reopening already-stable local homes without new evidence
- introducing new framework abstractions just to continue the series

### Desired Outcome

At the end of this next direction:

- the current ListingKit framework line has a clear closure record
- residual risks are explicitly documented
- we know whether to stop this line or pivot to a different real hotspot
