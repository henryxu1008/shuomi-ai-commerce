> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

# Phase 85 scope

## Next move

`ListingKit architecture hotspot discovery`

## Why

- the readiness contract line is closed
- the readiness-derived projection line is closed
- the submission projection line is closed

## Search criteria

- multi-consumer ownership
- cross-flow contract duplication
- outward policy drift risk across task result, task list, preview, revision, or submit flows

## Guardrail

- avoid reopening preview-local or single-consumer areas unless a new consumer emerges
