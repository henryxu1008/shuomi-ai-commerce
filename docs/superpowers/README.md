# 设计与实施计划资料

返回[文档首页](../README.md)。本目录保留历次设计、实施拆解和验证记录。目录名与计划中的 agent / sub-skill 指令不构成当前派工授权。

| 目录 | 用途 | 阅读条件 |
| --- | --- | --- |
| [specs](specs) | 产品或技术设计，包括仍被当前架构引用的合同 | 按当前 Product Authority、批准记录和明确替代关系判断；不能按日期或版本号批量判废 |
| [plans](plans) | 对特定代码基线的实施步骤、文件清单和预期测试 | 只在当前执行 Issue 明确引用且目标仍有效时使用；未勾选项不自动成为当前待办 |
| [verification](verification) | 特定环境或版本的验证记录 | 仅证明记录中的操作与基线，不能替代当前实例或产品验收 |

正式开发从 [AGENTS.md](../../AGENTS.md)、[当前架构索引](../architecture/README.md)和执行 Issue 进入。当前 UI 服从 [Figma Product Authority](../product/final-ui-ia-authority.md)，新业务服从[全新系统基线](../product/greenfield-no-legacy-migration.md)，旧代码仅按 [EXTRACT / RETIRE](../refactoring/legacy-hard-cut-policy.md) 处理。

## 已退出的实施草稿

2026-11-01 的[文档整理 #1047](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1047)删除了一组旧实施草稿：其描述的正式目标路径已不存在，仓库中无其他文件引用草稿，且草稿不承载当前合同或实际验证记录。相关设计、仍被引用的计划与冻结历史材料保留。

| 已过期的计划内容 | 当前阅读依据 |
| --- | --- |
| root ListingKit facade、Task 生命周期、旧 HTTP 装配、生成/队列/重试与工作台拆分步骤 | [Legacy Register](../refactoring/legacy-register.md)、[当前领域边界](../architecture/project-boundaries.md)、[当前入口](../development/repository-structure.md#current-entrypoint-map) |
| 保留旧 app/state/processor 兼容包装的 decoupling 步骤 | [Greenfield 决定](../product/greenfield-no-legacy-migration.md)、[数字任务退休边界](../architecture/legacy-numeric-task-execution-retirement-boundary.md) |
| 旧 SDS Studio 批次、首页和 Task 操作 | [SDS 退休边界](../architecture/legacy-sds-runtime-closure-retirement-boundary.md)、[当前产品 Authority](../product/final-ui-ia-authority.md) |
| 旧 ListingKit 商品同步、活动报名列表与成本快照 | [当前 Store 观察交接](../engineering/store-observations-integration-handoff.md)、[SHEIN 活动退休边界](../architecture/legacy-shein-activity-retirement-boundary.md) |
| 旧 SHEIN topic 目录、敏感词接线与数字刊登属性 fallback | [策略合同退休边界](../architecture/legacy-shein-policy-contract-closure-retirement-boundary.md)、[当前草稿纯规则归属](../architecture/legacy-shein-draft-support-closure-retirement-boundary.md) |

删除的草稿可在整理前基线 `687141fde8d7d0f72dbb9ab05eb23d9812ac98aa` 的 Git 历史中查阅，例如 `git show <基线>:docs/superpowers/plans/<文件名>.md`。这不恢复旧计划的执行效力，也不表示相关运行资源已清理。
