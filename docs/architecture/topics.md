# 领域合同阅读导航

返回[文档首页](../README.md) · [架构入口与完整索引](README.md) · [退休边界导航](retirement-index.md)

本页按问题所在的领域分类，帮助找到原始合同，不复制合同正文或维护另一份批准状态。
分类包含当前合同、候选设计及仍被引用的阶段上下文；先读原文顶部的范围、准入与精确替代说明。
`IMPLEMENTATION_READY`、代码交付、实例启用与用户验收分别确认，当前进度查执行 Issue / PR。

## 先选领域

| 你在找什么 | 合同入口 | 操作 / 实现说明 |
| --- | --- | --- |
| [总体边界与工程准入](#foundation) | [领域归属与依赖](project-boundaries.md) | [源码地图](../development/repository-structure.md#current-entrypoint-map)、[开发说明](../development/README.md) |
| [身份、企业与授权](#identity) | [身份与租户边界](auth-and-tenancy.md) | [Account Compose](../../deployments/docker/account-compose/README.md)、[身份工程交接](../engineering/README.md#identity) |
| [商业、资源与存储](#commercial) | [统一基础与预付资源](unified-base-prepaid-resources-v1.md) | [资源与试用操作](../operations/README.md#ai-work)、[商业历史记录](../engineering/README.md#historical) |
| [来源采集与浏览器插件](#acquisition) | [1688 采集合同](1688-acquisition-figma-v1.md) | [采集与商品操作](../operations/README.md#products)、[采集插件](../../extensions/product-capture/README.md) |
| [商品、草稿与供应市场](#product) | [Product 领域合同](product-domain-contracts.md) | [商品与草稿操作](../operations/README.md#products) |
| [店铺、平台刊登与履约](#store) | [Store Center 接入](store-center-current-application-v1.md) | [店铺与履约操作](../operations/README.md#stores)、[平台工程交接](../engineering/README.md#commerce) |
| [Agent、Tool 与模型调用](#agent) | [Commerce Tool 基础](commerce-tool-foundation.md) | [AI 工作与企业配置](../operations/README.md#ai-work) |
| [图像、资产安全与外部 effect](#image) | [商品图片智能体](product-image-agent-v1.md) | [图片智能体操作](../operations/product-image-agent-v1.md) |
| [会话、项目、任务、知识与经营投影](#workspace) | [项目中心](ai-workbench-project-center-v1.md) | [项目中心操作](../operations/project-center-v1.md)、[工作区工程交接](../engineering/README.md#workspace) |
| [数据、工具与生态服务](#services) | [Data Services](data-services-v1.md) | [服务操作](../operations/README.md#stores)、[服务工程交接](../engineering/README.md#services) |
| [有界试用与阶段上下文](#context) | [限定本地试用](issue-36-local-trial-runtime.md) | [专项操作限制](../operations/README.md#limited-scope)、[历史归档](../archive/README.md) |

<a id="foundation"></a>

## 总体边界与工程准入

先确认 owner、依赖方向与组合入口，再读取具体领域合同。

- [App Assembly Boundaries](app-assembly-boundaries.md)
- [Architecture Review Checklist](architecture-review-checklist.md)
- [External Client Boundary Inventory](external-client-boundary-inventory.md)
- [HTTP API Assembly Boundaries](httpapi-assembly-boundaries.md)
- [Module Target Mapping](module-target-mapping.md)
- [开源替代方案评估](open-source-replacement-evaluation.md)
- [Platform Boundary Strategy](platform-boundary-strategy.md)
- [Project Boundary Rules](project-boundaries.md)
- [Project Target Architecture](project-target-architecture.md)
- [Phase 2 runtime convergence inventory](runtime-consumer-register.md)
- [Temporal Boundaries](temporal-boundaries.md)

<a id="identity"></a>

## 身份、企业与授权

产品登录决定 → 身份与租户边界 → 对应成员、角色或账号合同。

- [Account Audit 完整历史筛选 V1](account-audit-filters-v1.md)
- [Account Audit 四项真实汇总 V1](account-audit-summary-v1.md)
- [Account Audit 只读用量来源接线 V1](account-audit-usage-readers-v1.md)
- [Account facts and invitations v1](account-facts-invitations-v1.md)
- [Auth and Tenancy Context](auth-and-tenancy.md)
- [Auth.js instance cookie isolation — #706](auth-instance-cookie-isolation.md)
- [Zitadel Authentication Runtime Extraction Design](authentication-runtime.md)
- [Enterprise custom roles v1 — Issue #598](enterprise-custom-roles-v1.md)
- [Member directory query v1](member-directory-query-v1.md)
- [B1：受保护成员单业务角色的原生条件写入](protected-member-role-native-cas.md)
- [Source Account SA1 contract](source-account-sa1.md)
- [Source Account SA-2 BFF contract](source-account-sa2-bff.md)
- [个人与企业主体认证：腾讯企业认证与阿里云个人认证](subject-verification-tencent-esign-design.md)

<a id="commercial"></a>

## 商业、资源与存储

统一基础与预付资源为主入口；旧 PAY / 订阅记录按其原范围和后续替代阅读。

- [微信支付与支付宝企业钱包充值架构设计](alipay-wallet-topup-design.md)
- [店铺与 AI 预付资源正式价格 V1](commercial-retail-pricing-v1.md)
- [Commercial Wallet / Billing Owner Contract](commercial-wallet-billing-contract.md)
- [企业存储容量 V1](enterprise-storage-capacity-v1.md)
- [Local 一次性 AI 试用来源 v1](local-ai-trial-v1.md)
- [Member resources and official store connection v1](member-resource-allocation-v1.md)
- [本地 MinIO 企业容量身份准入 V1](minio-capacity-admission-v1.md)
- [PAY-041 usage ledger reconciliation and PAY-042 handoff](pay-041-usage-ledger.md)
- [PAY-042 ListingKit generation usage cutover](pay-042-listingkit-generation-usage-cutover.md)
- [Referral settlement period and promotion conduct](referral-settlement-and-conduct.md)
- [Referral Withdrawal Personal KYC Contract](referral-withdrawal-personal-kyc-contract.md)
- [Self-Service Subscription Purchase Contract](self-service-subscription-purchase-contract.md)
- [Immutable settlement reversal locking](settlement-reversal-locking.md)
- [统一基础方案与预付资源 V1](unified-base-prepaid-resources-v1.md)

<a id="acquisition"></a>

## 来源采集与浏览器插件

先确认采集渠道和来源范围，再找插件、本地或匿名服务端路径。

- [1688 在线采集、规则与待适配接入](1688-acquisition-figma-v1.md)
- [1688 批量导入（本地执行器驱动既有浏览器采集链路）设计](1688-batch-local-agent.md)
- [Issue #399 1688 `window.context` 定位规则设计（v1 提案）](1688-context-locator.md)
- [服务端匿名浏览器采集（1688 public browser acquisition）架构设计](2026-09-26-1688-server-public-browser-acquisition-design.md)
- [设计增补：匿名采集速率治理与出口代理（#514）](2026-09-27-1688-anonymous-rate-and-egress-proxy-addendum.md)
- [1688 acquisition channel pricing — design basis for #592](2026-10-03-acquisition-channel-pricing.md)
- [Amazon 新品和二手来源报价：设计草案](amazon-condition-quotes-draft.md)
- [Amazon Crawler Commercialization Checklist](amazon-crawler-commercialization-checklist.md)
- [Amazon Crawler Runtime Flow](amazon-crawler-runtime-flow.md)
- [Amazon 插件：可选采集同族其他变体](plugin-capture-amazon-variants-v1.md)
- [商品采集插件：可恢复本地草稿规划与设计 v1](plugin-capture-durable-drafts-v1.md)
- [Product capture entry and original-tab return](plugin-capture-entry-flow-v1.md)
- [插件采集增量架构：规格图片与 Amazon 接收保存 V1](plugin-capture-ingestion-v1.md)
- [插件上架资料补齐：高清相册、参数、正文/详情图与重量 V1](plugin-capture-listing-materials-v1.md)
- [Product capture: source-page primary entry](plugin-capture-page-entry-v1.md)
- [Product capture: original-tab return across worker recreation](plugin-capture-session-return-v1.md)
- [插件资料补齐与来源报价只读展示 V1](plugin-capture-source-quotes-v1.md)
- [#30 / #307 完整来源业务闭环设计](source-business-closure-v1.md)

<a id="product"></a>

## 商品、草稿与供应市场

Product owner → 商品/草稿生命周期 → 对应供应市场或优选申请合同。

- [待补全草稿保存历史与比较 v1](completion-draft-history-v1.md)
- [待补全草稿：恢复历史内容为新版本](completion-draft-restore-v1.md)
- [CompletionDraft 结构化候选 v1](completion-draft-structured-v1.md)
- [CompletionDraft → SHEIN Target 带入预览与确认](completion-shein-import-v1.md)
- [自有商品单件转入与独立待补全入口 v1](own-product-draft-entry-v1.md)
- [自有商品版本更新与独立编辑页 v1](own-product-edit-v1.md)
- [自有商品独立录入页与多变体 v1](own-product-entry-v1.md)
- [自有商品已有草稿与继续编辑 v1](own-product-existing-drafts-v1.md)
- [自有商品历史读取、旧版详情和对比 v1](own-product-history-v1.md)
- [自有商品无店铺依赖的内容优化与 Apply](own-product-optimization-v1.md)
- [自有商品描述富文本 v1](own-product-rich-description-v1.md)
- [自有商品来源图片的本地隔离试用](own-product-source-media-trial-v1.md)
- [阶段三产品域目标架构设计](product-domain-contracts.md)
- [官方 Simple Editor 与描述图片独立上传](rich-description-simple-editor-images-v1.md)
- [多商品优选申请 v1](selected-application-multi-product-v1.md)
- [优选申请总览只读汇总 v1](selected-application-overview-v1.md)
- [优选申请资格读取：正常不可申请结果](supply-market-choice-eligibility.md)
- [供应市场 v1 — 明确发布、优选申请与 SDS 定制成品](supply-market-v1.md)
- [Supply 未绑定店铺待补全草稿](supply-unbound-completion-drafts.md)

<a id="store"></a>

## 店铺、平台刊登与履约

Store owner → 平台合同 → 发布、观察或履约的具体入口；官方证据保留原核对日期。

- [ERPNext 与 SHEIN 双履约适配验证](erpnext-shein-fulfillment-fit-v1.md)
- [Listing Preview Boundaries](listing-preview-boundaries.md)
- [我的供应链：SHEIN 美国站完整交付设计](my-supply-chain-shein-v1.md)
- [订单、采购、发货与物流首版设计草案](order-procurement-fulfillment-v1.md)
- [own675 Store 核心合同对齐方案](own675-store-core-alignment-v1.md)
- [SHEIN 美国站：审核、首次上架与状态回读最小闭环](shein-publication-closure-v1.md)
- [SHEIN Submit Readiness Action Policy](shein-submit-readiness-action-policy.md)
- [独立 SHEIN worker：身份与持久化边界设计](shein-worker-identity-persistence-boundary.md)
- [店铺浏览器接口实施交接](store-browser-workbench-implementation.md)
- [Store Center 接入 current application v1](store-center-current-application-v1.md)
- [Store Center：我的店铺完整 Figma 路径 v1](store-center-figma-v1.md)
- [店铺中心：平台商品与订单物流只读首版](store-center-platform-observations-v1.md)
- [在当前 Store 安装既有订单观察模块](store-observations-current-store-install.md)
- [Store observations internal preparation runtime](store-observations-preparation-runtime.md)
- [TEMU 官方接口与站点证据](temu-official-contract-evidence.md)
- [TEMU 三种模式与全部官方站点接入设计](temu-platform-v1.md)
- [TikTok 发送前未执行的终止与显式重试：冻结设计](tiktok-pre-dispatch-no-send-v1.md)
- [TikTok Shop 美国跨境：官方合同证据](tiktok-shop-us-contract-evidence.md)
- [TikTok Shop 美国跨境：店铺授权、商品草稿与受控刊登](tiktok-shop-us-v1.md)
- [Walmart Feed：明确拒绝的修复重试与有界续读](walmart-feed-recovery-v1.md)
- [Walmart 美国站官方合同证据与边界](walmart-us-contract-evidence.md)
- [Walmart 美国站：多商家授权、商品草稿与受控刊登](walmart-us-v1.md)

<a id="agent"></a>

## Agent、Tool 与模型调用

能力/Tool 边界 → 执行合同 → Skill、企业配置及具体消费者。

- [智能体定制需求与人工进度 V1](agent-customization-v1.md)
- [Agent + Knowledge Context V1](agent-knowledge-context-v1.md)
- [AI Capability & Agent Platform 设计](ai-capability-platform.md)
- [Provider-Neutral AI Contracts Design](ai-provider-contracts.md)
- [B0 Canonical Product Inspection Tool 设计](commerce-tool-canonical-inspection.md)
- [Phase 2A 商业工具（Commerce Tool）基础设计](commerce-tool-foundation.md)
- [商品内容标题试用与现有 Chat 共存的最小运行接线](content-agent-chat-coexistence-v1.md)
- [企业智能体权限与审批模式 V1](enterprise-agent-permission-modes-v1.md)
- [模型与费率管理 V1](model-route-pricing-management-v1.md)
- [官方技能直接处理已有商品](official-skill-product-entry.md)
- [官方标题与图片技能 V1](official-skills-and-image-method-v1.md)
- [Organization Agent Configuration V1 — Slice D](organization-agent-configuration-v1.md)
- [平台上架资料智能体 v1](platform-listing-materials-agent-v1.md)
- [POD设计智能体 v1：从设计意图生成图案](pod-design-agent-v1.md)
- [指定企业私有智能体交付 V1](private-agent-delivery-v1.md)
- [Product Agent Google Interactions 标题路由增量合同](product-agent-google-interactions-v1.md)
- [Product Agent 最小运行合同与接入依赖](product-agent-runtime-contract.md)
- [Product Agent 标题文本供应商可替换合同（候选）](product-agent-text-provider-neutral-v1.md)
- [商品内容完善与本地化智能体 v1](product-content-localization-v1.md)
- [Product selection comparison Agent v1](product-selection-comparison-agent-v1.md)
- [Skill V1：三层归属与现有商品内容 Agent 接入](skill-capability-v1.md)

<a id="image"></a>

## 图像、资产安全与外部 effect

图像消费者 → 预算与授权 → effect / UNKNOWN / 恢复；各旧协议的精确替代仍按页首说明。

- [百炼确定未生成与四笔预留对账增量](bailian-image-no-generation-reconciliation-v1.md)
- [百炼商品图片接线 V1](bailian-product-image-wiring-v1.md)
- [通用图片模板与技能入口带入](generic-image-template-entry.md)
- [GRSAI VIP 图片可选绑定 V1](grsai-vip-image-binding-v1.md)
- [Image Agent Budget Authorization Design](image-agent-budget-authorization.md)
- [Image Agent Candidate Release Gate Design](image-agent-candidate-release-gate.md)
- [Image Agent external-effect finalization design](image-agent-effect-finalization.md)
- [Image Agent V3 Effect Policy Design](image-agent-effect-policy.md)
- [Image Agent External Effect Recovery Workflow Design](image-agent-effect-recovery.md)
- [Image Agent Effect Terminalization Design](image-agent-effect-terminalization.md)
- [Image Agent Recovery and Compatibility Design](image-agent-recovery.md)
- [Image Agent Workflow Design](image-agent-workflow.md)
- [Image API event-read permission correction](image-api-event-read-permissions.md)
- [Detail image literal copy assembly v3](image-detail-visible-copy-v3.md)
- [Image Download Safety Boundary Design](image-download-safety.md)
- [Single-product image plan optional route V1](image-plan-optional-route-v1.md)
- [图片技能视觉 brief V2](image-skill-visual-brief-v2.md)
- [Image target language, localization and confirmed copy](image-text-language-confirmation-v1.md)
- [商品图片智能体 V1 — 完整图片能力设计](product-image-agent-v1.md)
- [VIP original output recovery and paid-result presentation](vip-original-output-recovery.md)

<a id="workspace"></a>

## 会话、项目、任务、知识与经营投影

先确认原 owner，再读当前页面投影或关联消费合同。

- [AI Workbench Chat + BusinessTask V1 — Slice E](ai-workbench-chat-business-task-v1.md)
- [硕米 Chat 会话管理独立运行接线](ai-workbench-conversation-runtime.md)
- [AI Workbench Project Center V1](ai-workbench-project-center-v1.md)
- [AI工作台个人报告 v1](ai-workbench-reports-v1.md)
- [Chat 通用任务创建与首批内容执行](chat-task-object-goal-design-brief.md)
- [通知中心 V1：站内通知与业务状态投影](notification-center-v1.md)
- [官方知识库 V2：产品内依据，不使用 GitHub 地址](official-knowledge-provenance-v2.md)
- [Official Knowledge V1 — Product Gate and design proposal](official-knowledge-v1.md)
- [运营驾驶舱四页 Figma 与只读投影补充](operations-cockpit-figma-read-v1.md)
- [运营驾驶舱首版：人工经营事实与有据可查的经营投影](operations-cockpit-v1.md)
- [Project Center overview read V1](project-center-overview-read-v1.md)
- [Report Center：原 SHEIN 离线包诊断正常接线候选](report-center-shein-offline-package-source-v1.md)
- [Report Center：当前 SHEIN 已保存资料来源候选](report-center-shein-target-source-v1.md)
- [SHEIN 离线诊断输入、freshness 与组织范围读取设计](shein-offline-validation-input-binding.md)
- [Shop operations diagnosis Agent v1](shop-operations-diagnosis-agent-v1.md)
- [任务中心：多来源、多能力的原生运行投影](task-center-native-run-projection-v1.md)

<a id="services"></a>

## 数据、工具与生态服务

各服务 owner 与授权合同 → 已交付接线；启用和验收另看交接记录。

- [平台浏览器内核发行目录 V1](browser-kernel-release-catalog-v1.md)
- [Data Services native authorization — #619 / PR #631](data-services-native-authorization.md)
- [数据市场与 API 管理首版](data-services-v1.md)
- [Ecoservices qualification-only runtime](ecoservices-qualification-runtime.md)
- [生态服务 V1：第三方入驻、服务交付与在线收款结算](ecosystem-services-v1.md)
- [本机 TLS 撤销维护边界（#965）](local-tls-revocation-maintenance.md)
- [Platform customer support through Chatwoot official APIs](platform-support-chatwoot.md)
- [工具定制完整 Figma 表单与私有参考资料](tool-customization-figma-v1.md)
- [工具市场 v1 — 企业启用与人工定制需求](tool-market-v1.md)

<a id="context"></a>

## 有界试用与阶段上下文

这些文件有各自限定范围；旧阶段排序和试用记录不充当当前派工队列。

- [#36 已完成本地资料的店铺归属投影](issue-36-completed-work-store-scope.md)
- [#36 可保留的本机隔离试用运行边界](issue-36-local-trial-runtime.md)
- [Next Technical Priorities](next-steps.md)
- [指定企业平台草稿质检：隔离离线试用装配](private-draft-offline-trial.md)
