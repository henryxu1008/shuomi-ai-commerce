> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

# Phase 86 scope

## Candidate hotspot

`ListingKit asset-generation projection closure audit`

## Why this is the next step

- `buildAssetGenerationOverview(...)` was already a shared owner
- phase 85 closed the larger bundle contract around result / preview / export outward projections
- before opening another refactor line, we should verify whether the remaining generation-queue/review wiring is already at a practical closure point

## Decision rule

- only continue if we find another real multi-consumer outward contract nearby
- if remaining logic is review-local or queue-local composition, stop slicing this line
