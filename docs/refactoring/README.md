# 当前重构规则

返回[文档首页](../README.md)。按领域查找具体退出矩阵，使用[退休边界导航](../architecture/retirement-index.md)；查当前领域合同，使用[合同导航](../architecture/topics.md)。

本目录只保留 [Legacy Hard-Cut Policy](legacy-hard-cut-policy.md) 与 [Legacy Register](legacy-register.md) 的当前规则和登记。
产品遵守 [PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08](../product/greenfield-no-legacy-migration.md)，执行排序与进度属于 [#137](https://github.com/henryxu1008/shuomi-ai-commerce/issues/137)及当前执行 Issue / PR。

## 当前合同与工程入口

- [Project Boundaries](../architecture/project-boundaries.md)、[Module Target Mapping](../architecture/module-target-mapping.md)。
- [当前 Runtime Consumer Register](../architecture/runtime-consumer-register.md)：有效 consumer register 仍按实际imports检查；其中dated计数与历史ceilings不当作当前规模。
- [SHEIN动作分类规则](../architecture/shein-submit-readiness-action-policy.md)、[代码健康规则与决定](../engineering/code-health-decisions.md)。
- [源码入口](../development/repository-structure.md#current-entrypoint-map)、[依赖与包边界检查](../development/README.md#dependency-and-package-checks)。生成报告写入既有.local产物目录，不覆盖文档或历史证据。

## Current active direction

- Use the current product-source flow and its approved owner; do not route new work through legacy ListingKit.
- Extract current-owner behavior and RETIRE legacy ListingKit ownership; the old root is a shrinking extraction/RETIRE surface.
- Prefer small, testable current-owner changes over broad rewrites. Verify behavior before and after each current-owner change; normal new-system schema evolution does not authorize legacy-data migration.

## Current refactoring documents

当前只维护 [Hard-Cut Policy](legacy-hard-cut-policy.md) 和 [Legacy Register](legacy-register.md)。它们约束现存源码的正确归属，不产生历史迁移或兼容义务；后续开发范围由当前产品决定和执行 Issue 给出。

<a id="historical-records"></a>

旧阶段报告和计划不在当前工作树维护，也不是当前实施或验收依据。需要追溯时使用固定 Git 版本，不恢复旧排期、旧命令或真实环境操作权限。
