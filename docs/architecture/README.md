# 当前系统架构合同

返回[文档首页](../README.md)。先读 `project-boundaries.md` 和 `project-target-architecture.md`，再按业务领域选择合同；源码入口见[Repository Structure](../development/repository-structure.md#current-entrypoint-map)。

产品基线为 [PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08](../product/greenfield-no-legacy-migration.md)。页面与交互见[产品 Authority](../product/final-ui-ia-authority.md)；领域事实和运行边界由下列合同负责。合同中的设计准入不代替源码完成、实例启用或真实用户验收。

## 研究与商业评估规划

- `research-supplier-review-v1.md` · [单候选人工核对已保存 1688 货源](research-supplier-review-v1.md) · #1074 独立设计候选，Admission NOT_READY。
- `research-manual-cost-preview-v1.md` · [单候选人工费用参考试算](research-manual-cost-preview-v1.md) · #1074 独立设计候选。
- `research-commercial-assessment-persistence-v1.md` — [保存经营评估与人工决定引用](research-commercial-assessment-persistence-v1.md)；#1205 D1–D3 已确认，冻结设计 IMPLEMENTATION_READY；原 PR #1209 实施及工程复核，运行/产品验收 NOT_RUN。
- `research-existing-synthetic-read-candidate.md` — [已有合成原件的单件手工评估读取](research-existing-synthetic-read-candidate.md)；#1205 仅B000123457，隔离范围已确认，架构复核中/NOT_IMPLEMENTATION_READY；未实现，不开放AI、转入或发布。
- `research-commercial-assessment-v1.md` — [市场调研与商业选品 V1：独立架构提案](research-commercial-assessment-v1.md)；#1074 设计阶段，来源与产品决定待定，Admission NOT_READY，不授权生产代码或真实调用。
- `shein-us-one-page-trial-send.md` · [SHEIN 美国站一页试取单次发送](shein-us-one-page-trial-send.md) · #1074 的离线发送门禁、25 credits 上限与原单核账。
- `zenrows-shein-us-trial-adapter.md` · [ZenRows 发送与原单用量查证](zenrows-shein-us-trial-adapter.md) · #1074 一页试取适配器设计。
- `shein-us-page-publication.md` · [SHEIN 美国站一页响应到原版发布](shein-us-page-publication.md) · #1074 离线组合设计。
- `shein-us-trial-entry.md` · [SHEIN 美国站单页取数入口](shein-us-trial-entry.md) · #1073 研究项目的默认关闭入口和原单恢复。
- `shein-us-trial-installation.md` · [SHEIN 单页试取账本与运行依赖准备](shein-us-trial-installation.md) · 原 Product 安装、最小授权与离线批准接线。
- `shein-us-trial-runtime-wiring.md` · [SHEIN 美国站单页试取运行接线](shein-us-trial-runtime-wiring.md) · 私有批准与门禁、只读挂载、末次发送检查及原单核账。
- `shein-us-trial-workbench-entry.md` · [SHEIN 美国站首试工作台入口](shein-us-trial-workbench-entry.md) · 复用登录和原项目权限查看、提交及核对固定批准单。

<a id="foundation"></a>

## 系统边界与应用装配

- `installation-and-module-management.md` — [首次安装向导与现有实例模块管理](installation-and-module-management.md) · #1240 统一设计；独立R1及文字增量复核见PR #1241，实施和原实例操作未执行。
- `installation-failure-recovery.md` — [安装失败说明、归档与安全重试](installation-failure-recovery.md) · #1240 最小恢复增量；当前仅设计，独立评审与真实操作授权分开。
- `retained-startup-prerequisites.md` — [保留实例启动前置：OwnIntake v3 与采集价格快照](retained-startup-prerequisites.md) · #1233 最小方案独立R1通过，IMPLEMENTATION_READY / FROZEN；实施义务未执行，不授权原35444操作。
- `app-assembly-boundaries.md` — [App Assembly Boundaries](app-assembly-boundaries.md)
- `architecture-review-checklist.md` — [Architecture Review Checklist](architecture-review-checklist.md)
- `fallback-contract-consolidation.md` — [Fallback 合同收敛](fallback-contract-consolidation.md)
- `httpapi-assembly-boundaries.md` — [HTTP API Assembly Boundaries](httpapi-assembly-boundaries.md)
- `project-boundaries.md` — [Project Boundary Rules](project-boundaries.md)
- `project-target-architecture.md` — [Project Target Architecture](project-target-architecture.md)
- `temporal-boundaries.md` — [Temporal Boundaries](temporal-boundaries.md)

<a id="identity"></a>

## 身份、企业与授权

- `account-audit-filters.md` — [Account Audit 完整历史筛选 V1](account-audit-filters.md)
- `account-audit-summary.md` — [Account Audit 四项真实汇总 V1](account-audit-summary.md)
- `account-audit-usage-readers.md` — [Account Audit 只读用量来源接线 V1](account-audit-usage-readers.md)
- `account-facts-invitations.md` — [Account facts and invitations v1](account-facts-invitations.md)
- `auth-and-tenancy.md` — [Authentication and Organization Scope](auth-and-tenancy.md)
- `auth-instance-cookie-isolation.md` — [Auth.js instance cookie isolation — #706](auth-instance-cookie-isolation.md)
- `authentication-runtime.md` — [Zitadel Authentication Runtime Extraction Design](authentication-runtime.md)
- `browser-kernel-release-catalog.md` — [平台浏览器内核发行目录 V1](browser-kernel-release-catalog.md)
- `compose-tofu-preflight.md` — [Compose OpenTofu preflight boundary](compose-tofu-preflight.md)
- `enterprise-custom-roles.md` — [Enterprise custom roles v1 — Issue #598](enterprise-custom-roles.md)
- `local-tls-revocation-maintenance.md` — [本机 TLS 撤销维护边界（#965）](local-tls-revocation-maintenance.md)
- `member-directory-query.md` — [Member directory query v1](member-directory-query.md)
- `protected-member-role-native-cas.md` — [B1：受保护成员单业务角色的原生条件写入](protected-member-role-native-cas.md)
- `public-trial-identity-origin.md` — [Fixed HTTPS identity origin for the cloud trial](public-trial-identity-origin.md)
- `source-account-registry.md` — [Source Account SA1 contract](source-account-registry.md)
- `source-account-bff.md` — [Source Account SA-2 BFF contract](source-account-bff.md)

<a id="commercial"></a>

## 资金、资源与计费

- `alipay-wallet-topup-design.md` — [微信支付与支付宝企业钱包充值架构设计](alipay-wallet-topup-design.md)
- `commercial-retail-pricing.md` — [店铺与 AI 预付资源正式价格 V1](commercial-retail-pricing.md)
- `data-row-platform-pricing-zero-orders.md` — [DATA_ROW 平台调价与零价订单设计](data-row-platform-pricing-zero-orders.md) · D1–D4 已确认，正式架构准入 IMPLEMENTATION_READY；原 PR 离线实施，保留环境改价及产品验收未授权。
- `enterprise-storage-capacity.md` — [企业存储容量 V1](enterprise-storage-capacity.md)
- `local-ai-trial.md` — [Local 一次性 AI 试用来源 v1](local-ai-trial.md)
- `member-resource-allocation.md` — [Member resources and official store connection v1](member-resource-allocation.md)
- `minio-capacity-admission.md` — [本地 MinIO 企业容量身份准入 V1](minio-capacity-admission.md)
- `model-route-pricing-management.md` — [模型与费率管理 V1](model-route-pricing-management.md)
- `own-intake-provider-unbilled-recovery.md` — [#1165 供应商明确拒绝且未计费的原调用恢复](own-intake-provider-unbilled-recovery.md)
- `enterprise-shared-model-configuration.md` — [企业统一模型与参考价估算（已撤下精确厂商价格/完整硬上界前置；原单一账本及预算保持）](enterprise-shared-model-configuration.md)
- `bailian-deepseek-shared-text-admission.md` — [百炼DeepSeek共享文字准入（参考价估算，原预算保持）](bailian-deepseek-shared-text-admission.md)
- `ai-disabled-application-lifecycle.md` — [关闭新 AI 执行时保留完整应用、配置与历史读取](ai-disabled-application-lifecycle.md)
- `openai-flare-image-generation.md` — [Flare 图片参考估算冻结设计与实现（#1078，原报价与账本）](openai-flare-image-generation.md)
- `shared-channel-credentials.md` — [渠道凭据与模型定义分离（冻结合同，#1089已合入）](shared-channel-credentials.md)
- `eino-model-admission.md` — [Eino多厂商模型按需准入（冻结设计与实施合同；交付证据见PR #1093）](eino-model-admission.md)
- `channel-credentials-ui.md` — [企业渠道 Key UI 配置与轮换（D1–D3冻结、已获实施授权；工程证据见 PR #1096）](channel-credentials-ui.md)
- `referral-settlement-and-conduct.md` — [Referral settlement period and promotion conduct](referral-settlement-and-conduct.md)
- `referral-withdrawal-personal-kyc-contract.md` — [Referral Withdrawal Personal KYC Contract](referral-withdrawal-personal-kyc-contract.md)
- `settlement-reversal-locking.md` — [Immutable settlement reversal locking](settlement-reversal-locking.md)
- `subject-verification-tencent-esign-design.md` — [个人与企业主体认证：腾讯企业认证与阿里云个人认证](subject-verification-tencent-esign-design.md)
- `unified-base-prepaid-resources.md` — [统一基础方案与预付资源 V1](unified-base-prepaid-resources.md)

<a id="acquisition"></a>

## 来源采集与浏览器插件

- `1688-acquisition-figma.md` — [1688 在线采集、规则与待适配接入](1688-acquisition-figma.md)
- `1688-batch-import.md` — [1688 批量导入合同](1688-batch-import.md)
- `1688-context-locator.md` — [1688 window.context 定位合同](1688-context-locator.md)
- `1688-server-public-browser-acquisition.md` — [服务端匿名浏览器采集（1688 public browser acquisition）架构设计](1688-server-public-browser-acquisition.md)
- `1688-acquisition-rate-and-egress-proxy.md` — [设计增补：匿名采集速率治理与出口代理（#514）](1688-acquisition-rate-and-egress-proxy.md)
- `1688-acquisition-channel-pricing.md` — [1688 acquisition channel pricing — design basis for #592](1688-acquisition-channel-pricing.md)
- `plugin-capture-amazon-variants.md` — [Amazon 插件：可选采集同族其他变体](plugin-capture-amazon-variants.md)
- `plugin-capture-durable-drafts.md` — [商品采集插件：可恢复本地草稿规划与设计 v1](plugin-capture-durable-drafts.md)
- `plugin-capture-handoff-worker-restart.md` — [商品采集交接在扩展后台重建后恢复](plugin-capture-handoff-worker-restart.md)
- `plugin-capture-entry-flow.md` — [Product capture entry and original-tab return](plugin-capture-entry-flow.md)
- `plugin-capture-ingestion.md` — [插件采集增量架构：规格图片与 Amazon 接收保存 V1](plugin-capture-ingestion.md)
- `plugin-capture-listing-materials.md` — [插件上架资料补齐：高清相册、参数、正文/详情图与重量 V1](plugin-capture-listing-materials.md)
- `plugin-capture-page-entry.md` — [Product capture: source-page primary entry](plugin-capture-page-entry.md)
- `plugin-capture-session-return.md` — [Product capture: original-tab return across worker recreation](plugin-capture-session-return.md)
- `plugin-capture-source-quotes.md` — [插件资料补齐与来源报价只读展示 V1](plugin-capture-source-quotes.md)
- `source-business-flow.md` — [#30 / #307 完整来源业务闭环设计](source-business-flow.md)

<a id="product"></a>

## 商品、资产与草稿

- `own-product-sku-packaging.md` — [自有商品逐 SKU 含包装事实录入](own-product-sku-packaging.md)
- `completion-draft-history.md` — [待补全草稿保存历史与比较 v1](completion-draft-history.md)
- `completion-draft-restore.md` — [待补全草稿：恢复历史内容为新版本](completion-draft-restore.md)
- `completion-draft-structured.md` — [CompletionDraft 结构化候选 v1](completion-draft-structured.md)
- `completion-draft-internal-note.md` — [待补全草稿内部备注](completion-draft-internal-note.md)
- `market-research-selection.md` — [市场研究与选品：最小领域合同](market-research-selection.md)
- `market-research-evidence-analysis-v1.md` — [两个中心 V1：研究证据分析合同增量](market-research-evidence-analysis-v1.md)
- `amazon-opportunity-handoff-v1.md` — [Amazon 美国站调研机会与选品交接 V1](amazon-opportunity-handoff-v1.md)
- `issue1070-local-acceptance-fixture.md` — [#1070 本地验收项目合成资料导入](issue1070-local-acceptance-fixture.md)
- `own-product-ai-intake.md` — [自有商品 AI 录入 V1（含多 SKU）](own-product-ai-intake.md) · #1158 设计候选，实施准入以正文为准。
- `own-product-intake-openai-mini.md` — [自有商品 AI 整理 OpenAI Mini 单图接入（#1159）](own-product-intake-openai-mini.md)
- `own-product-intake-price-review.md` — [自有商品录入价格建议、逐行确认与不确定库存隔离](own-product-intake-price-review.md)
- `own-product-intake-stock-review.md` — [自有商品录入库存建议与逐行确认](own-product-intake-stock-review.md)
- `own-product-draft-entry.md` — [自有商品单件转入与独立待补全入口 v1](own-product-draft-entry.md)
- `own-product-edit.md` — [自有商品版本更新与独立编辑页 v1](own-product-edit.md)
- `own-product-entry.md` — [自有商品独立录入页与多变体 v1](own-product-entry.md)
- `own-product-existing-drafts.md` — [自有商品已有草稿与继续编辑 v1](own-product-existing-drafts.md)
- `own-product-history.md` — [自有商品历史读取、旧版详情和对比 v1](own-product-history.md)
- `own-product-optimization.md` — [自有商品无店铺依赖的内容优化与 Apply](own-product-optimization.md)
- `own-product-rich-description.md` — [自有商品描述富文本 v1](own-product-rich-description.md)
- `own-product-source-media-trial.md` — [自有商品来源图片的本地隔离试用](own-product-source-media-trial.md)
- `product-content-localization.md` — [商品内容完善与本地化智能体 v1](product-content-localization.md)
- `product-domain-contracts.md` — [Product Domain Contracts](product-domain-contracts.md)
- `rich-description-simple-editor-images.md` — [官方 Simple Editor 与描述图片独立上传](rich-description-simple-editor-images.md)
- `selected-application-multi-product.md` — [多商品优选申请 v1](selected-application-multi-product.md)
- `selected-application-overview.md` — [优选申请总览只读汇总 v1](selected-application-overview.md)
- `supply-unbound-completion-drafts.md` — [Supply 未绑定店铺待补全草稿](supply-unbound-completion-drafts.md)

<a id="store"></a>

## 店铺、平台与履约

- `amazon-condition-quotes-draft.md` — [Amazon 新品和二手来源报价：设计草案](amazon-condition-quotes-draft.md)
- `completion-shein-import.md` — [CompletionDraft → SHEIN Target 带入预览与确认](completion-shein-import.md)
- `erpnext-shein-fulfillment-fit.md` — [ERPNext 与 SHEIN 双履约适配验证](erpnext-shein-fulfillment-fit.md)
- `my-supply-chain-shein.md` — [我的供应链：SHEIN 美国站完整交付设计](my-supply-chain-shein.md)
- `supply-four-step-continuation.md` — [四步供应链与补全/审核后继续上传](supply-four-step-continuation.md)
- `order-procurement-fulfillment.md` — [订单、采购、发货与物流首版设计草案](order-procurement-fulfillment.md)
- `store-core-schema-alignment.md` — [own675 Store 核心合同对齐方案](store-core-schema-alignment.md)
- `report-center-shein-offline-package-source.md` — [Report Center：原 SHEIN 离线包诊断正常接线候选](report-center-shein-offline-package-source.md)
- `report-center-shein-target-source.md` — [Report Center：当前 SHEIN 已保存资料来源候选](report-center-shein-target-source.md)
- `shein-sales-attribute-mapping.md` — [SHEIN 单商品多颜色、多 SKU 销售属性映射](shein-sales-attribute-mapping.md)；#1158 当前冻结设计 IMPLEMENTATION_READY，独立 R1 无 BLOCKER，四项实现验证待后续授权；保留同一草稿三行，稀疏集合发布资格仍待确认。
- `controlled-local-platform-draft.md` — [原35444受控本地平台草稿接线](controlled-local-platform-draft.md)；#1158 固定试点合同，测试记录永久不可发布；实例级互斥的最新决定见共存设计，实施/运行证据在 Issue。
- `local-and-official-draft-coexistence.md` — [同一实例内测试草稿与正式刊登共存](local-and-official-draft-coexistence.md)；#1158 最小方案与独立架构评审，独立本地接口用途、原 owner/保存协议及正式发布准入保持；仅设计，不授权实现或原实例操作。
- `shein-color-split-drafts.md` — [SHEIN 按颜色拆分本地平台草稿](shein-color-split-drafts.md)；历史独立能力，已撤回作为本例必经流程的推荐，参见单商品映射设计。
- `shein-native-validator-contract.md` — [SHEIN 原生资料公共诊断与材料 Tool v2](shein-native-validator-contract.md)
- `shein-offline-stock-packaging.md` — [SHEIN 离线诊断的库存和 SKU 包装缺项](shein-offline-stock-packaging.md)
- `shein-offline-validation-input-binding.md` — [SHEIN 离线诊断输入、freshness 与组织范围读取设计](shein-offline-validation-input-binding.md)
- `shein-publication-flow.md` — [SHEIN 美国站：审核、首次上架与状态回读最小闭环](shein-publication-flow.md)
- `shein-submit-readiness-action-policy.md` — [SHEIN Submit Readiness Action Policy](shein-submit-readiness-action-policy.md)
- `store-browser-workbench-implementation.md` — [店铺浏览器接口实施交接](store-browser-workbench-implementation.md)
- `store-center-application.md` — [Store Center 接入 current application v1](store-center-application.md)
- `store-center-figma.md` — [Store Center：我的店铺完整 Figma 路径 v1](store-center-figma.md)
- `store-center-platform-observations.md` — [店铺中心：平台商品与订单物流只读首版](store-center-platform-observations.md)
- `store-observations-installation.md` — [在当前 Store 安装既有订单观察模块](store-observations-installation.md)
- `store-empty-observations-contract-alignment.md` — [四张空 Store 观测表的最小合同对齐](store-empty-observations-contract-alignment.md) · #1231 架构 IMPLEMENTATION_READY；实现/副本/原实例操作另行授权。
- `store-observations-preparation-runtime.md` — [Store observations internal preparation runtime](store-observations-preparation-runtime.md)
- `supply-market-choice-eligibility.md` — [优选申请资格读取：正常不可申请结果](supply-market-choice-eligibility.md)
- `supply-market.md` — [供应市场 v1 — 明确发布、优选申请与 SDS 定制成品](supply-market.md)
- `temu-official-contract-evidence.md` — [TEMU 官方接口与站点证据](temu-official-contract-evidence.md)
- `temu-platform.md` — [TEMU 三种模式与全部官方站点接入设计](temu-platform.md)
- `tiktok-pre-dispatch-no-send.md` — [TikTok 发送前未执行的终止与显式重试：冻结设计](tiktok-pre-dispatch-no-send.md)
- `tiktok-shop-us-contract-evidence.md` — [TikTok Shop 美国跨境：官方合同证据](tiktok-shop-us-contract-evidence.md)
- `tiktok-shop-us.md` — [TikTok Shop 美国跨境：店铺授权、商品草稿与受控刊登](tiktok-shop-us.md)
- `walmart-feed-recovery.md` — [Walmart Feed：明确拒绝的修复重试与有界续读](walmart-feed-recovery.md)
- `walmart-us-contract-evidence.md` — [Walmart 美国站官方合同证据与边界](walmart-us-contract-evidence.md)
- `walmart-us.md` — [Walmart 美国站：多商家授权、商品草稿与受控刊登](walmart-us.md)

<a id="agent"></a>

## Agent、Tool、Skill 与知识

- `agent-customization.md` — [智能体定制需求与人工进度 V1](agent-customization.md)
- `agent-knowledge-context.md` — [Agent + Knowledge Context V1](agent-knowledge-context.md)
- `knowledge-original-file-deletion.md` — [知识库原文件删除与容量释放 V1](knowledge-original-file-deletion.md)
- `ai-capability-platform.md` — [AI Capability & Agent Platform 设计](ai-capability-platform.md)
- `ai-provider-contracts.md` — [Provider-Neutral AI Contracts Design](ai-provider-contracts.md)
- `commerce-tool-canonical-inspection.md` — [B0 Canonical Product Inspection Tool 设计](commerce-tool-canonical-inspection.md)
- `commerce-tool-foundation.md` — [Phase 2A 商业工具（Commerce Tool）基础设计](commerce-tool-foundation.md)
- `product-image-facts-advice.md` — [图片事实检查与非执行性建议：C2-1 / C3-1（R1 冻结，实现见 #1076）](product-image-facts-advice.md)
- `product-image-visual-analysis.md` — [商品图片单图视觉分析：C2-2（限定工程检查已完成；运行与产品验收 NOT_RUN）](product-image-visual-analysis.md)
- `listing-candidate-category-rules.md` — [SHEIN 候选类目规则查询与只读解释：C3-2（R1 冻结，IMPLEMENTATION_READY）](listing-candidate-category-rules.md)
- `enterprise-agent-permission-modes.md` — [企业智能体权限与审批模式 V1](enterprise-agent-permission-modes.md)
- `official-knowledge-provenance.md` — [官方知识库 V2：产品内依据，不使用 GitHub 地址](official-knowledge-provenance.md)
- `official-knowledge.md` — [Official Knowledge V1 — Product Gate and design proposal](official-knowledge.md)
- `official-skill-product-entry.md` — [官方技能直接处理已有商品](official-skill-product-entry.md)
- `issue132-title-evaluation-admission.md` — [标题对照评测准入与预算边界（#132）](issue132-title-evaluation-admission.md)
- `issue132-title-pilot-local-entry.md` — [标题先导的本机操作入口（#132）](issue132-title-pilot-local-entry.md)
- `organization-agent-configuration.md` — [Organization Agent Configuration V1 — Slice D](organization-agent-configuration.md)
- `platform-listing-materials-agent.md` — [平台上架资料智能体 v1](platform-listing-materials-agent.md)
- `pod-design-agent.md` — [POD设计智能体 v1：从设计意图生成图案](pod-design-agent.md)
- `private-agent-delivery.md` — [指定企业私有智能体交付 V1](private-agent-delivery.md)
- `product-agent-google-interactions.md` — [Product Agent Google Interactions 标题路由增量合同](product-agent-google-interactions.md)
- `product-agent-runtime-contract.md` — [Product Agent 最小运行合同与接入依赖](product-agent-runtime-contract.md)
- `product-agent-text-provider-neutral.md` — [Product Agent 标题文本供应商可替换合同（候选）](product-agent-text-provider-neutral.md)
- `product-selection-comparison-agent.md` — [Product selection comparison Agent v1](product-selection-comparison-agent.md)
- `shop-operations-diagnosis-agent.md` — [Shop operations diagnosis Agent v1](shop-operations-diagnosis-agent.md)
- `skill-capability.md` — [Skill V1：三层归属与现有商品内容 Agent 接入](skill-capability.md)
- `tool-customization-figma.md` — [工具定制完整 Figma 表单与私有参考资料](tool-customization-figma.md)
- `tool-market.md` — [工具市场 v1 — 企业启用与人工定制需求](tool-market.md)

<a id="image"></a>

## 图片执行、预算与恢复

- `bailian-product-image-wiring.md` — [百炼商品图片接线 V1](bailian-product-image-wiring.md)
- `generic-image-template-entry.md` — [通用图片模板与技能入口带入](generic-image-template-entry.md)
- `grsai-vip-image-binding.md` — [GRSAI VIP 图片可选绑定 V1](grsai-vip-image-binding.md)
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
- `image-plan-optional-route.md` — [Single-product image plan optional route V1](image-plan-optional-route.md)
- `image-text-language-confirmation.md` — [Image target language, localization and confirmed copy](image-text-language-confirmation.md)
- `product-image-agent.md` — [商品图片智能体 V1 — 完整图片能力设计](product-image-agent.md)
- `source-white-background.md` — [来源商品白底图与资源结算合同](source-white-background.md)
- `vip-original-output-recovery.md` — [VIP original output recovery and paid-result presentation](vip-original-output-recovery.md)

<a id="workspace"></a>

## 工作台、项目与任务投影

- `ai-workbench-chat-business-task.md` — [AI Workbench Chat + BusinessTask V1 — Slice E](ai-workbench-chat-business-task.md)
- `ai-workbench-conversation-runtime.md` — [硕米 Chat 会话管理独立运行接线](ai-workbench-conversation-runtime.md)
- `ai-workbench-project-center.md` — [AI Workbench Project Center V1](ai-workbench-project-center.md)
- `ai-workbench-reports.md` — [AI工作台个人报告 v1](ai-workbench-reports.md)
- `ai-workbench-work-scope.md` — [AI 工作台：固定会话工作范围与任务筛选](ai-workbench-work-scope.md)
- `chat-task-object-goal-design-brief.md` — [Chat 通用任务创建与首批内容执行](chat-task-object-goal-design-brief.md)
- `chat-content-full-runtime-coexistence-candidate.md` — [内容 Chat 与完整商品智能体共存](chat-content-full-runtime-coexistence-candidate.md)
- `content-agent-chat-coexistence.md` — [商品内容标题试用与现有 Chat 共存的最小运行接线](content-agent-chat-coexistence.md)
- `completed-work-store-scope.md` — [#36 已完成本地资料的店铺归属投影](completed-work-store-scope.md)
- `product-workbench-trial-runtime.md` — [#36 可保留的本机隔离试用运行边界](product-workbench-trial-runtime.md)
- `notification-center.md` — [通知中心 V1：站内通知与业务状态投影](notification-center.md)
- `operations-cockpit-figma-read.md` — [运营驾驶舱四页 Figma 与只读投影补充](operations-cockpit-figma-read.md)
- `operations-cockpit.md` — [运营驾驶舱首版：人工经营事实与有据可查的经营投影](operations-cockpit.md)
- `cockpit-listing-task-metrics.md` — [运营驾驶舱首页：刊登与任务统计](cockpit-listing-task-metrics.md)
- `cockpit-walmart-feed-rate.md` — [运营驾驶舱：Walmart Feed 提交受理率](cockpit-walmart-feed-rate.md)
- `cockpit-tiktok-us-publish-rate.md` — [运营驾驶舱：TikTok Shop US 提交审核率](cockpit-tiktok-us-publish-rate.md)
- `cockpit-shein-order-amount.md` — [运营驾驶舱：SHEIN US 已保存订单商品金额（原币）](cockpit-shein-order-amount.md)
- `cockpit-shein-self-operated-orders.md` — [SHEIN US 普通自运营消费者订单及来源事实](cockpit-shein-self-operated-orders.md)
- `cockpit-shein-supply-orders.md` — [SHEIN 全托：按采购单号批次保存供货单与数量观察](cockpit-shein-supply-orders.md)
- `cockpit-shein-supply-business-times.md` — [SHEIN 全托：业务时间原值、来源与缺失状态保存](cockpit-shein-supply-business-times.md)
- `cockpit-shein-saved-supply-dates.md` — [已保存全托供货单按原始日期观察](cockpit-shein-saved-supply-dates.md)
- `cockpit-walmart-order-amount.md` — [Walmart US 消费者订单保存与单店商品金额](cockpit-walmart-order-amount.md)
- `cockpit-walmart-order-sync.md` — [Walmart US 消费者订单手动同步接线](cockpit-walmart-order-sync.md)
- `cockpit-tiktok-order-amount.md` — [TikTok Shop US 消费者订单保存与单店商品金额](cockpit-tiktok-order-amount.md)
- `cockpit-tiktok-order-sync.md` — [TikTok Shop US 跨境店消费者订单手动同步](cockpit-tiktok-order-sync.md)
- `cockpit-temu-order-amount.md` — [Temu US 本土/半托管消费者订单保存与单店商品金额](cockpit-temu-order-amount.md)
- `private-draft-offline-trial.md` — [指定企业平台草稿质检：隔离离线试用装配](private-draft-offline-trial.md)
- `project-center-overview-read.md` — [Project Center overview read V1](project-center-overview-read.md)
- `task-center-native-run-projection.md` — [任务中心：多来源、多能力的原生运行投影](task-center-native-run-projection.md)

<a id="services"></a>

## 数据与生态服务

- `data-services-native-authorization.md` — [Data Services native authorization — #619 / PR #631](data-services-native-authorization.md)
- `data-services.md` — [数据市场与 API 管理首版](data-services.md)
- `data-services-execution-lifecycle.md` — [关闭采集执行时保留数据服务配置、资料与历史](data-services-execution-lifecycle.md)
- `ecoservices-qualification-runtime.md` — [Ecoservices qualification-only runtime](ecoservices-qualification-runtime.md)
- `ecosystem-services.md` — [生态服务 V1：第三方入驻、服务交付与在线收款结算](ecosystem-services.md)
- `platform-support-chatwoot.md` — [Platform customer support through Chatwoot official APIs](platform-support-chatwoot.md)

Every architecture document is explicitly indexed above. 新增、合并或删除合同时同步本索引；当前规则写入已有 owner 的合同，不追加退休记录、历史索引或逐轮评审日志。

- `cockpit-temu-order-sync.md` — [Temu US 本土/半托管消费者订单手动同步](cockpit-temu-order-sync.md)
