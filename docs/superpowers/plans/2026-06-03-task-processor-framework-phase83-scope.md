> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

# Phase 83 scope

## Candidate hotspot

`ListingKit readiness-derived workspace projection contract`

## Why this is the next candidate

- `buildSheinPreviewPayload(...)` still owns the final assembly from:
  - readiness projection
  - repair center
  - repair state
  - workspace overview
- that assembly is currently only consumed by preview, so it is not automatically a framework seam
- however, it is adjacent to the new shared projection layer and is the next place where cross-flow reuse may emerge

## Decision rule

- only continue if we find a real multi-consumer or cross-flow contract
- if the remaining assembly is preview-local, stop slicing and keep it local
