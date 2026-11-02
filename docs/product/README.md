# 产品文档

返回[文档首页](../README.md)。本目录记录硕米智能引擎的产品语义、产品投影、业务边界和历史产品证据。

## 当前产品权威

产品 UI、信息架构、导航、页面命名和用户交互语义，以 [硕米最终 UI / IA Authority](./final-ui-ia-authority.md) 记录的已批准 Figma 当前可见、非归档终稿为准。

ListingKit 是需要退休的旧产品投影与混合架构，不是硕米的长期子产品、默认工作台或永久执行引擎。旧 ListingKit 文档继续保留为历史需求、历史验收或旧路径维护证据，但不能据此恢复旧 Workspace、Task-first 产品模型、永久 facade、fallback 或双事实源。

领域事实、安全和副作用边界不由 Figma 定义。商品、资产、Listing、Marketplace、Store、身份、组织、资金、资源、幂等、恢复等规则继续由当前领域合同和架构文档决定。

## 建议阅读顺序

1. [最终 UI / IA Authority](./final-ui-ia-authority.md) — 用户看到什么、在哪里操作、如何命名和交互。
2. [AI Commerce Agent Platform 产品战略](./ai-commerce-agent-platform-strategy.md) — 长期产品方向与 Agent / Tool 边界；技术能力图不覆盖 Figma。
3. [全新系统产品基线](./greenfield-no-legacy-migration.md) — 新业务按全新安装、空业务数据和当前模型推进，不恢复旧迁移/兼容前置。
4. [领域合同导航](../architecture/topics.md) — 领域 owner、权限、事务、幂等、资金和外部副作用。
5. [Legacy Register](../refactoring/legacy-register.md) — 旧能力的 EXTRACT / RETIRE 去向。
6. [Repository Structure](../development/repository-structure.md#current-entrypoint-map) — 当前真实入口、命令和代码落点。
7. GitHub Roadmap #137、具体执行 Issue / PR — 当前排序、范围、准确 HEAD、验证证据和操作权限。

官方登录产品决定：[Phase1 官方 ZITADEL Login V2 / Auth.js](login-phase1-official-zitadel.md)。旧账号入口、Console首期设计和V7计划已退出当前工作树；后续开发遵守[全新系统开发基线](greenfield-no-legacy-migration.md)。

## 当前执行规则

- 新功能先确定在 Figma 中的产品位置，再消费当前领域能力；不能由旧包名、旧路由或旧 Task 模型反向创造产品页面。
- ListingKit 中仍有效的行为按职责抽取到当前 Product / Listing / Marketplace / Integration / App 等 owner，调用方切换后退休旧实现。
- ProductSnapshot、ApprovedAsset、Platform Draft、Listing 等是稳定领域事实，不意味着需要独立顶层 Product Center / Listing Center。
- 设计目标、代码存在、运行接线、部署和产品验收必须分别取证；旧文档中的“当前”“下一阶段”“已完成”只对其注明的历史基线有效。
- 未开放能力应真实表达 unavailable，不用旧路径兜底、原型示例或模拟成功替代。
- Agent / Tool 不直接访问数据库绕过领域规则，也不建立第二套权限、商品事实或业务状态机。

## 商业约束与旧运行记录

以下文档保留尚未被明确替代的商业、安全和恢复责任。旧订阅、Token 与影子计量源码已按 [#988 完整边界](../architecture/legacy-subscription-metering-closure-retirement-boundary.md)退休；与[统一基础方案及预付资源](../architecture/unified-base-prepaid-resources-v1.md)直接冲突的旧套餐、配额和接线不再是当前入口：

- [付费试点产品目录与用量政策](./listingkit-paid-pilot-product-catalog.md) — 保留原政策和当前合同的衔接，不能据此恢复 `paid_pilot` 订阅或旧配额准入，也不能借源码退休取消有效授权、收费或发布约束。
- [付费商业试点上线执行计划](./listingkit-paid-pilot-execution-plan.md) — 保留外部准入、收费、Go/No-Go、安全、防重、UNKNOWN 与回滚责任；旧 PAY 队列和 Task 接线不是当前开发或内部试用前置。
- [错误恢复 SOP](./ops/listingkit-error-recovery-sop.md) — 保留故障、防重、恢复和升级责任；旧 Task/Workspace 操作仅限有事实证据的历史运行现场，不证明实例仍存活或已停用。当前操作从[运行交接](../operations/README.md)进入。

这些文档必须先读页首的后续合同和窄范围替代说明；原政策、批准、失败与待核实事实保留，不能由标题或旧 Active 标签推断当前源码、部署或产品验收。

## 已归档的旧产品投影

旧 ListingKit 产品目标、总览、页面操作、路线图、下一阶段排期和候选 backlog
已退出当前树，完整原文见[清理前固定版本](https://github.com/henryxu1008/shuomi-ai-commerce/tree/c0210ff34f44c371b5a1993aed89f439cfbdc1ff/docs/product)。
其中共有的安全、幂等、UNKNOWN 与外部副作用底线继续由当前领域合同承接，
清理旧投影不取消这些约束。

[错误恢复历史](listingkit-error-recovery.md)、下方商业门禁及真实运行记录仍保留。
旧 Task 验收空模板已按 #844 取消为当前门禁，历史模板仅在固定 Git 版本追溯。

## 专项产品文档

Product Sourcing、Account、Commercial、Marketplace、Agent 等专项文档只有在其自身仍被当前 Authority / Issue 引用且未被后续决定替代时，才作为对应范围的依据。文档标题或旧 Active 标签本身不构成当前执行授权。

- Agent 职责：[业务智能体分工](business-agent-responsibilities.md)。
- 商品与草稿：[未绑定店铺草稿](supply-unbound-completion-drafts.md)、[属性与 SKU 候选](completion-draft-structured-v1.md)、[SHEIN 带入预览](completion-shein-import-v1.md)、[多商品优选申请](selected-application-multi-product.md)。
- 采集决定：[服务端匿名浏览器范围](pd-1688-server-public-browser-2026-09-26.md)、[1688 Figma 与后续阶段决定](pd-1688-acquisition-figma-2026-10-11.md)、[插件字段设计](plugin-capture-listing-fields-v1.md)。插件字段设计本身仍标为未达到实现准入，不覆盖后续已准入的局部架构。
- 有界试用与界面：[Local 非现金 AI 试用](local-ai-trial-v1.md)、[工作台主题统一](workbench-theme-unification.md)。产品决定不等于实例操作或真实调用已获准。
- 来源合同与交接：[Product Sourcing Handoff](product-sourcing-handoff.md)、[MVP 当前合同及历史收口](product-sourcing-mvp-plan.md)。

AI Control Plane 与 Agent Runtime 的技术设计见：

- [AI Capability / Agent Platform](../architecture/ai-capability-platform.md)

## 面向读者

- 产品和运营：先看 Figma Authority 与当前执行 Issue，不从历史 ListingKit 路线图决定产品。
- 前后端工程师：按当前 owner、Repository Structure 与具体合同修改实现。
- AI / Agent 工程：判断需求应使用确定性能力、AI capability、Commerce Tool 还是 Agent Runtime。
- QA / 交付：按真实用户路径和当前验收条件验证，不用历史报告替代当前证据。
- 商务 / 运维 / 安全：以当前商业合同、仍有效的专项商业政策、运行文档和明确环境授权为准；不要从已经标记 historical 的旧产品计划推导生产政策。

## 文档边界

本目录记录产品级语义和历史产品证据。滚动 SHA、CI、运行状态、真实环境结果和具体操作权限留在对应 Issue / PR /验收记录；不要在多个产品文档里维护互相冲突的“当前状态”。
