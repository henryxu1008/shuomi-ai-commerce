> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 75 Scope

### Recommended Next Direction

After closing the current SHEIN sale-attribute freshness sub-line, the recommended next direction is:

1. discover a different real hotspot in `shein_submit_freshness.go` or another high-signal ListingKit production file

### Why This Direction Now

Because the current sub-line has already reached a practical stopping point:

- aggregate owner is thin
- resolution home is near-routing-only
- invalid-state / repair routing has its own home
- message shaping has its own home
- behavior coverage and boundary coverage are both in place

That means the next useful framework phase should begin only after a fresh hotspot-discovery pass.

### What To Look For

The next candidate should still have the same properties that justified this line:

- a broader home directly mixing multiple distinct responsibilities
- weak or missing ownership guardrails
- clear maintenance or change-risk payoff from decomposition

### What Not To Do

- do not continue splitting the current sale-attribute freshness sub-line without new evidence
- do not create phases only to preserve numbering momentum
- do not extract tiny “base-check” helpers only for symmetry

### Desired Outcome

At the end of the next discovery step:

- either a new high-signal hotspot is identified and ranked
- or this broader SHEIN freshness area should be treated as sufficiently decomposed for now
