> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 71 Scope

### Recommended Next Direction

After closing the current ListingKit action-target mutation line, the recommended next direction is:

1. `Discover a different real hotspot elsewhere in ListingKit`

### Why This Direction Now

Because the current action-target mutation line has already reached a practical stopping point:

- broader homes are routing-only or near-routing-only
- local homes are semantically coherent
- boundary coverage is broad
- further decomposition would mostly optimize symmetry, not reduce real complexity

That means the next useful framework phase should begin only after a fresh hotspot-discovery pass elsewhere in the codebase.

### What To Look For

The next candidate should have the same properties that justified the previous line:

- a broader home still directly mixing multiple distinct responsibilities
- weak or missing ownership guardrails
- repeated business semantics living in one place with poor locality
- clear maintenance or change-risk payoff from decomposition

### What Not To Do

- do not continue splitting the current action-target mutation line without new evidence
- do not create phases just to preserve numbering momentum
- do not introduce generic abstractions only for symmetry

### Desired Outcome

At the end of the next discovery step:

- either a new high-signal hotspot is identified and ranked
- or the framework work should pause and the current line should be treated as complete
