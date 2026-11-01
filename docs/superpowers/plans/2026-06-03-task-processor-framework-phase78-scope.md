> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 78 Scope

### Target

Run a closure-style audit for the current SHEIN attribute freshness ownership line.

### Why This Scope

After `Phase 77`, the sub-line already has:

- a thin aggregate owner
- a near-routing-only evaluation home
- a dedicated issue-state collection home
- a dedicated message-shape home

The remaining logic may no longer justify another real refactor phase.

### Desired Outcome

At the end of `Phase 78`:

- either the attribute freshness sub-line is declared practically complete
- or a concrete remaining hotspot is identified with enough signal to justify one more phase
