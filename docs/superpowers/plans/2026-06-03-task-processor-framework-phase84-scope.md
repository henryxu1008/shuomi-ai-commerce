> **历史 Framework 记录（#1050）。** 本文记录 2026 年 5–6 月当时的方案、范围建议或检查点；“当前”“下一步”和验证结果只对应原记录的基线。当前工作从[架构索引](../../architecture/README.md)、[源码入口](../../development/repository-structure.md)和执行 Issue 进入；仍有效的 kernel/module 与领域规则继续由当前批准 owner 承载，不据此恢复旧 ListingKit / Task-first 路径。原机器绝对链接、命令、历史结果和正文完整保留，阶段资料见[历史索引](README.md#framework-history)。

# Phase 84 scope

## Candidate hotspot

`ListingKit submission projection closure audit`

## Why this is the next step

- we now have two adjacent architecture-level seams:
  - readiness-derived outward projection
  - submission status / remote summary projection
- before opening another refactor line, we should verify whether submission projection is already at a practical closure point

## Decision rule

- if the remaining logic in `submission_state_support.go` is mostly package-local derivation and fallback policy, stop slicing
- only continue if we find another real multi-consumer outward contract nearby
