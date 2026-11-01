# 设计与实施计划资料

返回[文档首页](../README.md)。本目录保留历次设计、实施拆解和验证记录。目录名与计划中的 agent / sub-skill 指令不构成当前派工授权。

| 目录 | 用途 | 阅读条件 |
| --- | --- | --- |
| [specs](specs) | 产品或技术设计，包括仍被当前架构引用的合同 | 按当前 Product Authority、批准记录和明确替代关系判断；不能按日期或版本号批量判废 |
| [plans](plans) | 对特定代码基线的实施步骤、文件清单和预期测试 | 只在当前执行 Issue 明确引用且目标仍有效时使用；未勾选项不自动成为当前待办 |
| [verification](verification) | 特定环境或版本的验证记录 | 仅证明记录中的操作与基线，不能替代当前实例或产品验收 |

正式开发从 [AGENTS.md](../../AGENTS.md)、[当前架构索引](../architecture/README.md)和执行 Issue 进入。当前 UI 服从 [Figma Product Authority](../product/final-ui-ia-authority.md)，新业务服从[全新系统基线](../product/greenfield-no-legacy-migration.md)，旧代码仅按 [EXTRACT / RETIRE](../refactoring/legacy-hard-cut-policy.md) 处理。

## 52 份计划逐份核对：保留入口

[#1050](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1050)首轮逐份核对了固定的 52 份旧实施计划：34 份只描述已退休的执行面、兼容路径或旧 UI 拆分，且无完成或实际验证记录，已删除；18 份保留合同参考、已勾选的修复过程或明确的历史证据，原路径和原 Git 正文保持完整，仅增加阅读范围说明。下节记录用户继续整理后的扩展范围；逐份处置与替代依据均留在同一主要 PR。

以下保留材料供追溯，不是当前实施队列。文内的 `Expected: PASS`、已勾选步骤和未来验收计划不能替代实际执行证据；历史 `NOT_RUN` 不改为通过。当前 owner、准入和事实边界仍以链接的现行合同及明确退休决定为准。

| 保留材料 | 保留原因与阅读边界 |
| --- | --- |
| [认证职责收敛](plans/2026-06-21-listingkit-zitadel-auth-convergence.md) | 认证职责参考；旧 ListingKit 代理接线为历史，当前依据见 [Auth / Tenancy](../architecture/auth-and-tenancy.md) |
| [PAY-011 权威身份](plans/2026-07-15-pay-011-authoritative-identity.md) | 身份/租户约束及 2026-07-15 实际验证、旧 lint 警告记录 |
| [PAY-012 店铺归属](plans/2026-07-15-pay-012-store-ownership.md) | 店铺归属与外部动作前校验参考；不恢复旧 StoreAccessValidator |
| [PR #146 owner finding](plans/2026-08-13-pr-146-owner-followup.md) | 三项 owner 丢失问题的历史上下文；当前归属仍见 [Project Boundaries](../architecture/project-boundaries.md) |
| [PAY-041 账本](plans/2026-08-14-pay-041-usage-ledger.md) | 幂等与原子结算约束参考；旧订阅 schema / outbox / 迁移已退休 |
| [PAY-042 reservation 恢复](plans/2026-08-17-pay-042-reservation-recovery.md) | 中断恢复不得自动重放 provider 的约束参考；旧 Task intent 不再执行 |
| [图片计量生命周期](plans/2026-08-22-listingkit-product-image-usage-lifecycle.md) | 计量归属与固定 route 的历史约束；当前资源事实见[统一基础资源](../architecture/unified-base-prepaid-resources-v1.md) |
| [图片下载安全](plans/2026-08-21-image-download-safety.md) | SSRF、重定向、大小与 TLS 约束；当前实现为 [httpimage](../../internal/integration/httpimage/client.go) |
| [日志与源码产物边界](plans/2026-08-30-internal-target-architecture-phase1.md) | 默认日志无源码副作用及 `.local/logs` 约束；当前入口见 [Repository Structure](../development/repository-structure.md) |
| [保本 SKU 修复过程](plans/2026-07-13-listingkit-breakeven-final-price.md) | 保留已勾选的历史步骤；旧活动算法不作为当前定价合同 |
| [同币种 SKU 定价修复过程](plans/2026-07-13-shein-pricing-promotion.md) | 保留已勾选过程；不重开已退休活动产品 |
| [销售属性缓存修复过程](plans/2026-07-19-shein-sale-attribute-cache.md) | 保留原勾选与预期结果；不复活旧缓存协议 |
| [部署 Secret preflight 过程](plans/2026-08-09-listingkit-deploy-secret-preflight.md) | 保留安全修复历史；旧部署脚本已退出，生产门禁不变 |
| [SDS 子任务重试修复过程](plans/2026-08-13-listingkit-child-retry-error-reporting.md) | 保留已勾选过程；旧表和 API 不成为当前恢复入口 |
| [system-owned 例外报告范围](plans/2026-08-11-listingkit-system-owned-exceptions.md) | 报告 `648cdfab03c4` / 312 组范围的历史记载；不授权当前例外或数据处理 |
| [Store 986 历史恢复记录](plans/2026-08-12-shein-import-platform-normalization.md) | 保留实际提交/验证及现场 dry-run 未执行原因；不授权重跑 |
| [手机号会话可行性计划](plans/2026-08-25-listingkit-zitadel-phone-onboarding-feasibility.md) | 版本限定与原验收边界；未记录真实设备通过，不执行旧 runner 或恢复 fork |
| [PR #243 审核栏 finding](plans/2026-08-28-pr243-review-fixes.md) | 保留历史问题与方案上下文；当前交互服从 [UI / IA Authority](../product/final-ui-ia-authority.md) |

本批删除草稿的原文可从起始 main `0fba4f8e1602c0760a1ea977c5580887571a7599` 的 Git 历史查阅。身份、资源、平台规则和真实数据的当前边界未因文档清理改变。

## 继续整理：Framework 阶段与专项历史

[实施计划与历史索引](plans/README.md)按阶段收纳了 238 份旧 Framework 资料：94 份 checkpoint、88 份 scope、56 份方案/设计。原文和原位置均保留，“当前”“下一步”和验证结果按原基线追溯；它们不是当前执行队列，也不表示仍有效的 kernel/module 或领域合同已退休。18 份无引用、无当前生产目标和执行结果的旧 Framework 拆分/探索草稿已清理，阶段检查点与范围记录继续可查。

此外，旧调度事件后台和 SDS OSS 恢复两份纯实施草稿已删除；它们的旧运行 owner 已明确退休，不恢复旧状态机或执行批次恢复。以下 8 份专项材料保留原文，仅补阅读说明：

| 保留材料 | 保留依据 |
| --- | --- |
| [HTTP module builder 6A](plans/2026-05-20-httpapi-module-builder-phase6a.md) | 已勾选的抽取与验证过程；当前装配仍看现行 HTTP 边界 |
| [HTTP module builder 6B](plans/2026-05-20-httpapi-module-builder-phase6b.md) | 已勾选的 ProductImage builder 过程；不恢复旧 owner |
| [HTTP 结构 Phase 1](plans/2026-05-20-httpapi-structure-phase1.md) | HTTP 职责参考；旧集中模块拆分绑定原基线 |
| [旧 Ownership Controller](plans/2026-06-23-shein-listing-ownership-controller.md) | 原文的生产热修复、专店 976 与旧 rollout 状态记录；controller 操作已退休 |
| [PAY-013 资产隔离](plans/2026-07-16-pay-013-listingkit-asset-isolation.md) | 私有资产、租户隔离、补偿及重复删除约束与已勾选过程 |
| [Product / SHEIN Draft 边界](plans/2026-08-05-standard-product-shein-draft-boundary.md) | 领域边界及已勾选验证；原 JSON 兼容描述按原合同追溯 |
| [owner 写入不变量](plans/2026-08-18-listingkit-owner-write-invariant.md) | canonical owner 约束参考；旧 backfill/部署步骤不授予当前操作权限 |
| [ImageAgent 入口与 style 授权](plans/2026-08-29-image-agent-workspace-entry-and-style-authorization.md) | 当前授权约束参考；旧 Task Workspace 入口不决定当前 IA |

PR #1051 当前累计核对 318 份原始文档：删除 54 份过期草稿，保留 264 份合同参考和历史记录；索引只是阅读导航，不新增产品/架构权威或验收前置。原始内容可从起始 main `0fba4f8e1602c0760a1ea977c5580887571a7599` 的 Git 历史追溯。

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
