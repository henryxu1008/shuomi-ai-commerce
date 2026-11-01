# 重构边界与历史证据

当前执行排序、范围和进度来自 [Roadmap #137](https://github.com/henryxu1008/shuomi-ai-commerce/issues/137)与执行 Issue / PR。
产品遵守 [PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08](../product/greenfield-no-legacy-migration.md)，领域归属见[Project Boundaries](../architecture/project-boundaries.md)，源码入口见[Repository Structure](../development/repository-structure.md#current-entrypoint-map)。

## 当前边界与规则

- [Code Health Decisions](code-health-decisions.md)
- [Dependency Baseline](dependency-baseline.md)
- [Legacy Hard-Cut Policy](legacy-hard-cut-policy.md)
- [Legacy Register](legacy-register.md)
- [Module Target Mapping](module-target-mapping.md)
- [SHEIN POD Submit Readiness Policy](shein-pod-submit-readiness-policy-design.md)
- [SHEIN Submit Readiness Action Policy](shein-submit-readiness-action-policy-design.md)

Hard-Cut 只允许 `EXTRACT | RETIRE`。设计中保留的有效平台规则不等于旧 ListingKit
适配仍存在，也不授权兼容层、数据迁移或已有环境操作。依赖扫描工具是有界分析入口，
其当前参数以 `scripts/analyze-project-deps.ps1` 为准；生成输出保存在本地产物目录。

<a id="historical-records"></a>

## Current active direction

These retained structural principles are not a product delivery queue:

- Use the current product-source flow and its approved owner; do not route new work through legacy ListingKit.
- Extract current-owner behavior and RETIRE legacy ListingKit ownership; the old root is a shrinking extraction/RETIRE surface.
- Prefer small, testable current-owner changes over broad rewrites. Verify behavior before and after each current-owner change; normal new-system schema evolution does not authorize legacy-data migration.

## Current refactoring documents

`project-migration-roadmap.md` is a historical reference, not a current implementation authority.
`next-phase-plan.md` is a historical reference, not a current execution authority.

## 保留的批准、观察与验证记录

以下保留原始基线、计数、失败/取消、PASS 与 NOT_RUN。旧状态中的“当前”“下一步”
不替代现在的任务；历史兼容/迁移措辞不能恢复已撤回要求。

- [Issue 30 prepared slices — 2026-09-05](2026-09-05-issue-30-prepared-slices.md)
- [Draft prerequisite: Publication Identity Cutover](2026-09-05-publication-identity-cutover-issue-draft.md)
- [Checkpoint Validation - 2026-06-17](checkpoint-validation-2026-06-17.md)
- [Current Refactoring Status](current-refactoring-status.md)
- [Decision: Refactoring Closeout, Product Sources, Then Sales Platforms](decisions/2026-06-26-next-growth-sequence.md)
- [ListingKit Module Dependency Analysis](dependency-analysis.md)
- [ListingKit 重构工作最终总结](final-summary.md)
- [HTTPAPI Runtime Inventory](httpapi-runtime-inventory.md)
- [Legacy Store Routing Retirement Checklist](legacy-store-routing-retirement-checklist.md)
- [ListingKit Boundary Checkpoint](listingkit-boundary-checkpoint.md)
- [ListingKit 前端重构路线图](listingkit-frontend-refactoring-roadmap.md)
- [ListingKit 重构进展审查（2026-06-24）](listingkit-refactoring-progress-2026-06-24.md)
- [Platform-aware Asset Refactoring Implementation Plan](next-phase-plan.md)
- [第一阶段重构验收测试报告](phase1-acceptance-report.md)
- [第一阶段重构完成总结](phase1-summary.md)
- [方案 B 系统性重构 - 最终总结报告](phase2-final-summary.md)
- [第二阶段重构进展报告](phase2-progress-report.md)
- [Phase 2 runtime convergence inventory](phase2-runtime-inventory.md)
- [方案 B 系统性重构 - 执行总结](phase2-systematic-refactoring-summary.md)
- [Product Sourcing Inventory](product-sourcing-inventory.md)
- [Project Migration Roadmap](project-migration-roadmap.md)
- [Task Processor Project-wide Refactoring Plan](project-wide-refactoring-plan.md)
- [ListingKit Service Slimming Checkpoint](service-slimming-checkpoint.md)
- [Studio 迁移执行尝试报告](studio-migration-attempt-report.md)
- [ListingKit Submission Inventory](submission-inventory.md)
- [TDD 驱动的全局变量消除重构](tdd-global-state-elimination.md)
- [配套证据/配置附件](test-baseline.txt)
- [2026-07-09 Documentation Cleanup Validation](validation/2026-07-09-doc-cleanup-validation.md)
- [ListingKit Backend Baseline Validation — 2026-07-13](validation/2026-07-13-listingkit-backend-baseline.md)
- [Week 1 重构进展报告 - Core层抽取（部分完成）](week1-progress-report.md)
- [Week 2 最终总结 - Studio 服务迁移分析完成](week2-final-summary.md)
- [Week 2 重构进展报告 - Service层分析](week2-progress-report.md)
- [Workflow 模块整理决策记录](workflow-reorganization-decision.md)

旧 API/Workflow/Studio 移动计划、Preview/Payload 可行性和重复执行路线图已清理；完整原件见[固定历史目录](https://github.com/henryxu1008/shuomi-ai-commerce/tree/c0210ff34f44c371b5a1993aed89f439cfbdc1ff/docs/refactoring)。真实失败、取消、基线测试输出和受保护快照继续留在上表。
