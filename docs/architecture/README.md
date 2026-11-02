# 当前系统架构合同

返回[文档首页](../README.md)。先读 `project-boundaries.md` 和 `project-target-architecture.md`，再按业务领域选择合同；源码入口见[Repository Structure](../development/repository-structure.md#current-entrypoint-map)。

产品基线为 [PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08](../product/greenfield-no-legacy-migration.md)。页面与交互见[产品 Authority](../product/final-ui-ia-authority.md)；领域事实和运行边界由下列合同负责。合同中的设计准入不代替源码完成、实例启用或真实用户验收。

<a id="foundation"></a>

## 系统边界与应用装配

- `app-assembly-boundaries.md` — [App Assembly Boundaries](app-assembly-boundaries.md)
- `architecture-review-checklist.md` — [Architecture Review Checklist](architecture-review-checklist.md)
- `httpapi-assembly-boundaries.md` — [HTTP API Assembly Boundaries](httpapi-assembly-boundaries.md)
- `project-boundaries.md` — [Project Boundary Rules](project-boundaries.md)
- `project-target-architecture.md` — [Project Target Architecture](project-target-architecture.md)
- `temporal-boundaries.md` — [Temporal Boundaries](temporal-boundaries.md)

<a id="identity"></a>

## 身份、企业与授权

- `account-audit-filters-v1.md` — [Account Audit 完整历史筛选 V1](account-audit-filters-v1.md)
- `account-audit-summary-v1.md` — [Account Audit 四项真实汇总 V1](account-audit-summary-v1.md)
- `account-audit-usage-readers-v1.md` — [Account Audit 只读用量来源接线 V1](account-audit-usage-readers-v1.md)
- `account-facts-invitations-v1.md` — [Account facts and invitations v1](account-facts-invitations-v1.md)
- `auth-and-tenancy.md` — [Authentication and Organization Scope](auth-and-tenancy.md)
- `auth-instance-cookie-isolation.md` — [Auth.js instance cookie isolation — #706](auth-instance-cookie-isolation.md)
- `authentication-runtime.md` — [Zitadel Authentication Runtime Extraction Design](authentication-runtime.md)
- `browser-kernel-release-catalog-v1.md` — [平台浏览器内核发行目录 V1](browser-kernel-release-catalog-v1.md)
- `enterprise-custom-roles-v1.md` — [Enterprise custom roles v1 — Issue #598](enterprise-custom-roles-v1.md)
- `local-tls-revocation-maintenance.md` — [本机 TLS 撤销维护边界（#965）](local-tls-revocation-maintenance.md)
- `member-directory-query-v1.md` — [Member directory query v1](member-directory-query-v1.md)
- `protected-member-role-native-cas.md` — [B1：受保护成员单业务角色的原生条件写入](protected-member-role-native-cas.md)
- `source-account-sa1.md` — [Source Account SA1 contract](source-account-sa1.md)
- `source-account-sa2-bff.md` — [Source Account SA-2 BFF contract](source-account-sa2-bff.md)

<a id="commercial"></a>

## 资金、资源与计费

- `alipay-wallet-topup-design.md` — [微信支付与支付宝企业钱包充值架构设计](alipay-wallet-topup-design.md)
- `commercial-retail-pricing-v1.md` — [店铺与 AI 预付资源正式价格 V1](commercial-retail-pricing-v1.md)
- `enterprise-storage-capacity-v1.md` — [企业存储容量 V1](enterprise-storage-capacity-v1.md)
- `local-ai-trial-v1.md` — [Local 一次性 AI 试用来源 v1](local-ai-trial-v1.md)
- `member-resource-allocation-v1.md` — [Member resources and official store connection v1](member-resource-allocation-v1.md)
- `minio-capacity-admission-v1.md` — [本地 MinIO 企业容量身份准入 V1](minio-capacity-admission-v1.md)
- `model-route-pricing-management-v1.md` — [模型与费率管理 V1](model-route-pricing-management-v1.md)
- `referral-settlement-and-conduct.md` — [Referral settlement period and promotion conduct](referral-settlement-and-conduct.md)
- `referral-withdrawal-personal-kyc-contract.md` — [Referral Withdrawal Personal KYC Contract](referral-withdrawal-personal-kyc-contract.md)
- `settlement-reversal-locking.md` — [Immutable settlement reversal locking](settlement-reversal-locking.md)
- `subject-verification-tencent-esign-design.md` — [个人与企业主体认证：腾讯企业认证与阿里云个人认证](subject-verification-tencent-esign-design.md)
- `unified-base-prepaid-resources-v1.md` — [统一基础方案与预付资源 V1](unified-base-prepaid-resources-v1.md)

<a id="acquisition"></a>

## 来源采集与浏览器插件

- `1688-acquisition-figma-v1.md` — [1688 在线采集、规则与待适配接入](1688-acquisition-figma-v1.md)
- `1688-batch-local-agent.md` — [1688 批量导入合同](1688-batch-local-agent.md)
- `1688-context-locator.md` — [1688 window.context 定位合同](1688-context-locator.md)
- `2026-09-26-1688-server-public-browser-acquisition-design.md` — [服务端匿名浏览器采集（1688 public browser acquisition）架构设计](2026-09-26-1688-server-public-browser-acquisition-design.md)
- `2026-09-27-1688-anonymous-rate-and-egress-proxy-addendum.md` — [设计增补：匿名采集速率治理与出口代理（#514）](2026-09-27-1688-anonymous-rate-and-egress-proxy-addendum.md)
- `2026-10-03-acquisition-channel-pricing.md` — [1688 acquisition channel pricing — design basis for #592](2026-10-03-acquisition-channel-pricing.md)
- `plugin-capture-amazon-variants-v1.md` — [Amazon 插件：可选采集同族其他变体](plugin-capture-amazon-variants-v1.md)
- `plugin-capture-durable-drafts-v1.md` — [商品采集插件：可恢复本地草稿规划与设计 v1](plugin-capture-durable-drafts-v1.md)
- `plugin-capture-entry-flow-v1.md` — [Product capture entry and original-tab return](plugin-capture-entry-flow-v1.md)
- `plugin-capture-ingestion-v1.md` — [插件采集增量架构：规格图片与 Amazon 接收保存 V1](plugin-capture-ingestion-v1.md)
- `plugin-capture-listing-materials-v1.md` — [插件上架资料补齐：高清相册、参数、正文/详情图与重量 V1](plugin-capture-listing-materials-v1.md)
- `plugin-capture-page-entry-v1.md` — [Product capture: source-page primary entry](plugin-capture-page-entry-v1.md)
- `plugin-capture-session-return-v1.md` — [Product capture: original-tab return across worker recreation](plugin-capture-session-return-v1.md)
- `plugin-capture-source-quotes-v1.md` — [插件资料补齐与来源报价只读展示 V1](plugin-capture-source-quotes-v1.md)
- `source-business-closure-v1.md` — [#30 / #307 完整来源业务闭环设计](source-business-closure-v1.md)

<a id="product"></a>

## 商品、资产与草稿

- `completion-draft-history-v1.md` — [待补全草稿保存历史与比较 v1](completion-draft-history-v1.md)
- `completion-draft-restore-v1.md` — [待补全草稿：恢复历史内容为新版本](completion-draft-restore-v1.md)
- `completion-draft-structured-v1.md` — [CompletionDraft 结构化候选 v1](completion-draft-structured-v1.md)
- `own-product-draft-entry-v1.md` — [自有商品单件转入与独立待补全入口 v1](own-product-draft-entry-v1.md)
- `own-product-edit-v1.md` — [自有商品版本更新与独立编辑页 v1](own-product-edit-v1.md)
- `own-product-entry-v1.md` — [自有商品独立录入页与多变体 v1](own-product-entry-v1.md)
- `own-product-existing-drafts-v1.md` — [自有商品已有草稿与继续编辑 v1](own-product-existing-drafts-v1.md)
- `own-product-history-v1.md` — [自有商品历史读取、旧版详情和对比 v1](own-product-history-v1.md)
- `own-product-optimization-v1.md` — [自有商品无店铺依赖的内容优化与 Apply](own-product-optimization-v1.md)
- `own-product-rich-description-v1.md` — [自有商品描述富文本 v1](own-product-rich-description-v1.md)
- `own-product-source-media-trial-v1.md` — [自有商品来源图片的本地隔离试用](own-product-source-media-trial-v1.md)
- `product-content-localization-v1.md` — [商品内容完善与本地化智能体 v1](product-content-localization-v1.md)
- `product-domain-contracts.md` — [Product Domain Contracts](product-domain-contracts.md)
- `rich-description-simple-editor-images-v1.md` — [官方 Simple Editor 与描述图片独立上传](rich-description-simple-editor-images-v1.md)
- `selected-application-multi-product-v1.md` — [多商品优选申请 v1](selected-application-multi-product-v1.md)
- `selected-application-overview-v1.md` — [优选申请总览只读汇总 v1](selected-application-overview-v1.md)
- `supply-unbound-completion-drafts.md` — [Supply 未绑定店铺待补全草稿](supply-unbound-completion-drafts.md)

<a id="store"></a>

## 店铺、平台与履约

- `amazon-condition-quotes-draft.md` — [Amazon 新品和二手来源报价：设计草案](amazon-condition-quotes-draft.md)
- `completion-shein-import-v1.md` — [CompletionDraft → SHEIN Target 带入预览与确认](completion-shein-import-v1.md)
- `erpnext-shein-fulfillment-fit-v1.md` — [ERPNext 与 SHEIN 双履约适配验证](erpnext-shein-fulfillment-fit-v1.md)
- `my-supply-chain-shein-v1.md` — [我的供应链：SHEIN 美国站完整交付设计](my-supply-chain-shein-v1.md)
- `order-procurement-fulfillment-v1.md` — [订单、采购、发货与物流首版设计草案](order-procurement-fulfillment-v1.md)
- `own675-store-core-alignment-v1.md` — [own675 Store 核心合同对齐方案](own675-store-core-alignment-v1.md)
- `report-center-shein-offline-package-source-v1.md` — [Report Center：原 SHEIN 离线包诊断正常接线候选](report-center-shein-offline-package-source-v1.md)
- `report-center-shein-target-source-v1.md` — [Report Center：当前 SHEIN 已保存资料来源候选](report-center-shein-target-source-v1.md)
- `shein-native-validator-contract-v1.md` — [SHEIN 原生资料公共诊断与材料 Tool v2](shein-native-validator-contract-v1.md)
- `shein-offline-validation-input-binding.md` — [SHEIN 离线诊断输入、freshness 与组织范围读取设计](shein-offline-validation-input-binding.md)
- `shein-publication-closure-v1.md` — [SHEIN 美国站：审核、首次上架与状态回读最小闭环](shein-publication-closure-v1.md)
- `shein-submit-readiness-action-policy.md` — [SHEIN Submit Readiness Action Policy](shein-submit-readiness-action-policy.md)
- `store-browser-workbench-implementation.md` — [店铺浏览器接口实施交接](store-browser-workbench-implementation.md)
- `store-center-current-application-v1.md` — [Store Center 接入 current application v1](store-center-current-application-v1.md)
- `store-center-figma-v1.md` — [Store Center：我的店铺完整 Figma 路径 v1](store-center-figma-v1.md)
- `store-center-platform-observations-v1.md` — [店铺中心：平台商品与订单物流只读首版](store-center-platform-observations-v1.md)
- `store-observations-current-store-install.md` — [在当前 Store 安装既有订单观察模块](store-observations-current-store-install.md)
- `store-observations-preparation-runtime.md` — [Store observations internal preparation runtime](store-observations-preparation-runtime.md)
- `supply-market-choice-eligibility.md` — [优选申请资格读取：正常不可申请结果](supply-market-choice-eligibility.md)
- `supply-market-v1.md` — [供应市场 v1 — 明确发布、优选申请与 SDS 定制成品](supply-market-v1.md)
- `temu-official-contract-evidence.md` — [TEMU 官方接口与站点证据](temu-official-contract-evidence.md)
- `temu-platform-v1.md` — [TEMU 三种模式与全部官方站点接入设计](temu-platform-v1.md)
- `tiktok-pre-dispatch-no-send-v1.md` — [TikTok 发送前未执行的终止与显式重试：冻结设计](tiktok-pre-dispatch-no-send-v1.md)
- `tiktok-shop-us-contract-evidence.md` — [TikTok Shop 美国跨境：官方合同证据](tiktok-shop-us-contract-evidence.md)
- `tiktok-shop-us-v1.md` — [TikTok Shop 美国跨境：店铺授权、商品草稿与受控刊登](tiktok-shop-us-v1.md)
- `walmart-feed-recovery-v1.md` — [Walmart Feed：明确拒绝的修复重试与有界续读](walmart-feed-recovery-v1.md)
- `walmart-us-contract-evidence.md` — [Walmart 美国站官方合同证据与边界](walmart-us-contract-evidence.md)
- `walmart-us-v1.md` — [Walmart 美国站：多商家授权、商品草稿与受控刊登](walmart-us-v1.md)

<a id="agent"></a>

## Agent、Tool、Skill 与知识

- `agent-customization-v1.md` — [智能体定制需求与人工进度 V1](agent-customization-v1.md)
- `agent-knowledge-context-v1.md` — [Agent + Knowledge Context V1](agent-knowledge-context-v1.md)
- `ai-capability-platform.md` — [AI Capability & Agent Platform 设计](ai-capability-platform.md)
- `ai-provider-contracts.md` — [Provider-Neutral AI Contracts Design](ai-provider-contracts.md)
- `commerce-tool-canonical-inspection.md` — [B0 Canonical Product Inspection Tool 设计](commerce-tool-canonical-inspection.md)
- `commerce-tool-foundation.md` — [Phase 2A 商业工具（Commerce Tool）基础设计](commerce-tool-foundation.md)
- `enterprise-agent-permission-modes-v1.md` — [企业智能体权限与审批模式 V1](enterprise-agent-permission-modes-v1.md)
- `official-knowledge-provenance-v2.md` — [官方知识库 V2：产品内依据，不使用 GitHub 地址](official-knowledge-provenance-v2.md)
- `official-knowledge-v1.md` — [Official Knowledge V1 — Product Gate and design proposal](official-knowledge-v1.md)
- `official-skill-product-entry.md` — [官方技能直接处理已有商品](official-skill-product-entry.md)
- `organization-agent-configuration-v1.md` — [Organization Agent Configuration V1 — Slice D](organization-agent-configuration-v1.md)
- `platform-listing-materials-agent-v1.md` — [平台上架资料智能体 v1](platform-listing-materials-agent-v1.md)
- `pod-design-agent-v1.md` — [POD设计智能体 v1：从设计意图生成图案](pod-design-agent-v1.md)
- `private-agent-delivery-v1.md` — [指定企业私有智能体交付 V1](private-agent-delivery-v1.md)
- `product-agent-google-interactions-v1.md` — [Product Agent Google Interactions 标题路由增量合同](product-agent-google-interactions-v1.md)
- `product-agent-runtime-contract.md` — [Product Agent 最小运行合同与接入依赖](product-agent-runtime-contract.md)
- `product-agent-text-provider-neutral-v1.md` — [Product Agent 标题文本供应商可替换合同（候选）](product-agent-text-provider-neutral-v1.md)
- `product-selection-comparison-agent-v1.md` — [Product selection comparison Agent v1](product-selection-comparison-agent-v1.md)
- `shop-operations-diagnosis-agent-v1.md` — [Shop operations diagnosis Agent v1](shop-operations-diagnosis-agent-v1.md)
- `skill-capability-v1.md` — [Skill V1：三层归属与现有商品内容 Agent 接入](skill-capability-v1.md)
- `tool-customization-figma-v1.md` — [工具定制完整 Figma 表单与私有参考资料](tool-customization-figma-v1.md)
- `tool-market-v1.md` — [工具市场 v1 — 企业启用与人工定制需求](tool-market-v1.md)

<a id="image"></a>

## 图片执行、预算与恢复

- `bailian-product-image-wiring-v1.md` — [百炼商品图片接线 V1](bailian-product-image-wiring-v1.md)
- `generic-image-template-entry.md` — [通用图片模板与技能入口带入](generic-image-template-entry.md)
- `grsai-vip-image-binding-v1.md` — [GRSAI VIP 图片可选绑定 V1](grsai-vip-image-binding-v1.md)
- `image-agent-budget-authorization.md` — [Image Agent Budget Authorization Design](image-agent-budget-authorization.md)
- `image-agent-candidate-release-gate.md` — [Image Agent Candidate Release Gate Design](image-agent-candidate-release-gate.md)
- `image-agent-effect-finalization.md` — [Image Agent external-effect finalization design](image-agent-effect-finalization.md)
- `image-agent-effect-policy.md` — [Image Agent V3 Effect Policy Design](image-agent-effect-policy.md)
- `image-agent-effect-recovery.md` — [Image Agent External Effect Recovery Workflow Design](image-agent-effect-recovery.md)
- `image-agent-effect-terminalization.md` — [Image Agent Effect Terminalization Design](image-agent-effect-terminalization.md)
- `image-agent-recovery.md` — [Image Agent Recovery and Compatibility Design](image-agent-recovery.md)
- `image-agent-workflow.md` — [Image Agent Workflow](image-agent-workflow.md)
- `image-api-event-read-permissions.md` — [Image API event-read permission correction](image-api-event-read-permissions.md)
- `image-download-safety.md` — [Image Download Safety Boundary Design](image-download-safety.md)
- `image-plan-optional-route-v1.md` — [Single-product image plan optional route V1](image-plan-optional-route-v1.md)
- `image-text-language-confirmation-v1.md` — [Image target language, localization and confirmed copy](image-text-language-confirmation-v1.md)
- `product-image-agent-v1.md` — [商品图片智能体 V1 — 完整图片能力设计](product-image-agent-v1.md)
- `source-white-background.md` — [来源商品白底图与资源结算合同](source-white-background.md)
- `vip-original-output-recovery.md` — [VIP original output recovery and paid-result presentation](vip-original-output-recovery.md)

<a id="workspace"></a>

## 工作台、项目与任务投影

- `ai-workbench-chat-business-task-v1.md` — [AI Workbench Chat + BusinessTask V1 — Slice E](ai-workbench-chat-business-task-v1.md)
- `ai-workbench-conversation-runtime.md` — [硕米 Chat 会话管理独立运行接线](ai-workbench-conversation-runtime.md)
- `ai-workbench-project-center-v1.md` — [AI Workbench Project Center V1](ai-workbench-project-center-v1.md)
- `ai-workbench-reports-v1.md` — [AI工作台个人报告 v1](ai-workbench-reports-v1.md)
- `chat-task-object-goal-design-brief.md` — [Chat 通用任务创建与首批内容执行](chat-task-object-goal-design-brief.md)
- `content-agent-chat-coexistence-v1.md` — [商品内容标题试用与现有 Chat 共存的最小运行接线](content-agent-chat-coexistence-v1.md)
- `issue-36-completed-work-store-scope.md` — [#36 已完成本地资料的店铺归属投影](issue-36-completed-work-store-scope.md)
- `issue-36-local-trial-runtime.md` — [#36 可保留的本机隔离试用运行边界](issue-36-local-trial-runtime.md)
- `notification-center-v1.md` — [通知中心 V1：站内通知与业务状态投影](notification-center-v1.md)
- `operations-cockpit-figma-read-v1.md` — [运营驾驶舱四页 Figma 与只读投影补充](operations-cockpit-figma-read-v1.md)
- `operations-cockpit-v1.md` — [运营驾驶舱首版：人工经营事实与有据可查的经营投影](operations-cockpit-v1.md)
- `private-draft-offline-trial.md` — [指定企业平台草稿质检：隔离离线试用装配](private-draft-offline-trial.md)
- `project-center-overview-read-v1.md` — [Project Center overview read V1](project-center-overview-read-v1.md)
- `task-center-native-run-projection-v1.md` — [任务中心：多来源、多能力的原生运行投影](task-center-native-run-projection-v1.md)

<a id="services"></a>

## 数据与生态服务

- `data-services-native-authorization.md` — [Data Services native authorization — #619 / PR #631](data-services-native-authorization.md)
- `data-services-v1.md` — [数据市场与 API 管理首版](data-services-v1.md)
- `ecoservices-qualification-runtime.md` — [Ecoservices qualification-only runtime](ecoservices-qualification-runtime.md)
- `ecosystem-services-v1.md` — [生态服务 V1：第三方入驻、服务交付与在线收款结算](ecosystem-services-v1.md)
- `platform-support-chatwoot.md` — [Platform customer support through Chatwoot official APIs](platform-support-chatwoot.md)

Every architecture document is explicitly indexed above. 新增、合并或删除合同时同步本索引；当前规则写入已有 owner 的合同，不追加退休记录、历史索引或逐轮评审日志。
