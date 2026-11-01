> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 82 Scope

### Recommended Next Direction

After closing the shared readiness contract line, the recommended next direction is:

1. discover a different real architecture hotspot elsewhere in ListingKit

### Why This Direction Now

Because the current readiness architecture already has explicit shared seams for:

- summary shaping
- cross-flow readiness gating
- guidance resolver wiring

and further decomposition on the same line would more likely optimize symmetry than remove real shared-contract risk.

### What To Look For

The next candidate should still have the same framework-level properties:

- duplicated policy across multiple flows
- a shared helper with confused ownership
- a contract used by multiple consumers with no explicit seam
- a change-risk payoff larger than the added boundary surface

### What Not To Do

- do not continue the current readiness line without new structural evidence
- do not open another phase just because there is still some code left in the area
- do not resume local helper-only decomposition by default

### Desired Outcome

At the end of the next discovery step:

- either a new high-signal architecture hotspot is identified and ranked
- or the broader framework work should pause and the current readiness line should be treated as complete
