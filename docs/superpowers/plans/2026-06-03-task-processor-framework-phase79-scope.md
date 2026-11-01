> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

## Task Processor Framework Phase 79 Scope

### Recommended Next Direction

Continue looking for framework-level hotspots around shared readiness or submit-flow contracts, not local helper decomposition.

### Candidate Directions

1. audit whether `submit readiness` and `freshness readiness` still duplicate shared contract wiring besides summary shaping
2. inspect direct-submit vs temporal-submit readiness gating for duplicated cross-flow orchestration
3. only fall back to local file decomposition if a real shared seam is no longer available

### Desired Outcome

At the end of the next step:

- either another real cross-flow/shared-contract hotspot is identified
- or this readiness seam is treated as the right stopping point and discovery moves elsewhere in ListingKit
