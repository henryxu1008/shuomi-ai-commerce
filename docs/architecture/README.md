# 当前系统架构合同

返回[文档首页](../README.md)。先读 `project-boundaries.md` 和 `project-target-architecture.md`，再按业务领域选择合同；源码入口见[Repository Structure](../development/repository-structure.md#current-entrypoint-map)。

产品基线为 [PD-GREENFIELD-NO-LEGACY-MIGRATION-2026-09-08](../product/greenfield-no-legacy-migration.md)。页面与交互的**当前用户决定及业务流程参考**见[产品流程与历史 UI/IA 说明](../product/final-ui-ia-authority.md)，Figma 不再是最终 UI/IA 权威；领域事实和运行边界由下列合同负责。合同中的设计准入、历史提交与旧阶段状态不代替当前源码完成、实例启用或真实用户验收。文档分域与候选迁移计划见[文档组织与迁移规则](../document-organization.md)及[逐文件清单](../document-inventory.csv)；未审查前不按文件名前缀搬移合同。

## 商品来源、标准商品与研究（分域阅读导航）

这三个导航只负责**找到当前权威合同**，不会取代 [领域事实归属](project-boundaries.md) 和下面的详细设计，也不要求将已有根级合同搬迁：

- [Sourcing：来源证据、采集通道与受控发布](domains/sourcing/README.md) — 来源身份、1688/Amazon 当前采集边界、SRC 与 Catalog 交接
- [Catalog：不可变商品版本、Collection 与 Asset](domains/catalog/README.md) — canonical facts、人工素材批准、商品资料与 Listing 边界
- [Research：市场证据、供货核对、经营评估与人工选品](domains/research/README.md) — 研究与已保存商品事实分离、当前评估增量、人工转入

精确实现/CI/实例/客户验收继续以各 Issue/PR 当前记录为准；历史设计的 Figma 截图与固定两 ASIN 测试不自动成为新的业务需求或全流程验收。

## 研究与商业评估规划

- `domains/workbench/scheduled-plan-deletion.md` — [定时计划删除](domains/workbench/scheduled-plan-deletion.md)；#1288，保留结果与在途任务的列表移除；独立检查 IMPLEMENTATION_READY，沿用计划控制与恢复。

- `domains/workbench/scheduled-store-daily-report.md` — [定时生成单店经营日报](domains/workbench/scheduled-store-daily-report.md)；#1279 独立架构及恢复增量复核 IMPLEMENTATION_READY；设计与后续实施记录；#1279 记录原实例单次实测，不代表实时完整数据或用户签收，不开放模型或平台发布。

- `domains/research/research-supplier-review-v1.md` · [单候选人工核对已保存 1688 货源](domains/research/research-supplier-review-v1.md) · #1074 独立设计与合成工程验收，真实联调及采购未执行。
- `domains/research/research-manual-cost-preview-v1.md` · [单候选人工费用参考试算](domains/research/research-manual-cost-preview-v1.md) · #1074 历史冻结设计；现行保存边界见 #1205。
- `domains/research/research-commercial-assessment-persistence-v1.md` — [保存经营评估与人工决定引用](domains/research/research-commercial-assessment-persistence-v1.md)；#1205 D1–D3 已确认，冻结设计 IMPLEMENTATION_READY；后续工程收口，指定合成保存/找回/引用通过；完整现场及产品验收 NOT_RUN。
- `research-existing-synthetic-read-candidate.md` — [已有合成原件的单件手工评估读取](research-existing-synthetic-read-candidate.md)；#1205 仅B000123457，隔离范围已确认，架构复核中/NOT_IMPLEMENTATION_READY；未实现，不开放AI、转入或发布。
- `research-commercial-assessment-v1.md` — [市场调研与商业选品 V1：独立架构提案](research-commercial-assessment-v1.md)；#1074 设计阶段，来源与产品决定待定，Admission NOT_READY，不授权生产代码或真实调用。
- `shein-us-one-page-trial-send.md` · [SHEIN 美国站一页试取单次发送](shein-us-one-page-trial-send.md) · #1074 的离线发送门禁、25 credits 上限与原单核账。
- `zenrows-shein-us-trial-adapter.md` · [ZenRows 发送与原单用量查证](zenrows-shein-us-trial-adapter.md) · #1074 一页试取适配器设计。
- `domains/data-services/shein-us-page-publication.md` · [SHEIN 美国站一页响应到原版发布](domains/data-services/shein-us-page-publication.md) · #1074 离线组合设计。
- `domains/research/shein-us-trial-entry.md` · [SHEIN 美国站单页取数入口](domains/research/shein-us-trial-entry.md) · #1073 研究项目的默认关闭入口和原单恢复。
- `shein-us-trial-installation.md` · [SHEIN 单页试取账本与运行依赖准备](shein-us-trial-installation.md) · 原 Product 安装、最小授权与离线批准接线。
- `shein-us-trial-runtime-wiring.md` · [SHEIN 美国站单页试取运行接线](shein-us-trial-runtime-wiring.md) · 私有批准与门禁、只读挂载、末次发送检查及原单核账。
- `domains/research/shein-us-trial-workbench-entry.md` · [SHEIN 美国站首试工作台入口](domains/research/shein-us-trial-workbench-entry.md) · 复用登录和原项目权限查看、提交及核对固定批准单。

<a id="foundation"></a>

## 系统边界与应用装配

- `installation-and-module-management.md` — [首次安装向导与现有实例模块管理](installation-and-module-management.md) · #1240 统一设计；独立评审、实施与运行证据见PR #1241。
- `installation-failure-recovery.md` — [安装失败说明、归档与安全重试](installation-failure-recovery.md) · #1240 最小恢复增量；独立R1通过、IMPLEMENTATION_READY，实施及真实恢复另行授权。
- `domains/workbench/chat-35444-retained-wiring-design.md` — [35444 保留库店铺查询接线设计](domains/workbench/chat-35444-retained-wiring-design.md) · 独立架构评审 IMPLEMENTATION_READY；实施仅限获准的保留实例及其隔离副本，原实例更新仍需单独放行。
- `domains/workbench/chat-retained-minimal-admission-proposal.md` — [原 35444 Chat 接线的开发试用最小准入](domains/workbench/chat-retained-minimal-admission-proposal.md) · 新用户决定及独立架构复核；复用已有证据，保留目标、数据、权限和原预留保护。
- `domains/workbench/chat-retained-1260-trial.md` — [保留试用实例隔离接线方案](domains/workbench/chat-retained-1260-trial.md) · 隔离演练依据；不构成原实例更新授权。

- `retained-startup-prerequisites.md` — [保留实例启动前置：OwnIntake v3 与采集价格快照](retained-startup-prerequisites.md) · #1233 最小方案独立R1通过，IMPLEMENTATION_READY / FROZEN；实施义务未执行，不授权原35444操作。
- `platform/runtime/app-assembly-boundaries.md` — [App Assembly Boundaries](platform/runtime/app-assembly-boundaries.md)
- `architecture-review-checklist.md` — [Architecture Review Checklist](architecture-review-checklist.md)
- `platform/runtime/fallback-contract-consolidation.md` — [Fallback 合同收敛](platform/runtime/fallback-contract-consolidation.md)
- `platform/runtime/httpapi-assembly-boundaries.md` — [HTTP API Assembly Boundaries](platform/runtime/httpapi-assembly-boundaries.md)
- `project-boundaries.md` — [Project Boundary Rules](project-boundaries.md)
- `project-target-architecture.md` — [Project Target Architecture](project-target-architecture.md)
- `platform/runtime/temporal-boundaries.md` — [Temporal Boundaries](platform/runtime/temporal-boundaries.md)

<a id="identity"></a>

## 身份、企业与授权

- `platform/identity/account-audit-filters.md` — [Account Audit 完整历史筛选 V1](platform/identity/account-audit-filters.md)
- `platform/identity/account-audit-summary.md` — [Account Audit 四项真实汇总 V1](platform/identity/account-audit-summary.md)
- `platform/identity/account-audit-usage-readers.md` — [Account Audit 只读用量来源接线 V1](platform/identity/account-audit-usage-readers.md)
- `platform/identity/account-region-catalog.md` — [编辑资料预置地区目录 V1](platform/identity/account-region-catalog.md)
- `platform/identity/account-facts-invitations.md` — [Account facts and invitations v1](platform/identity/account-facts-invitations.md)
- `platform/identity/auth-and-tenancy.md` — [Authentication and Organization Scope](platform/identity/auth-and-tenancy.md)
- `platform/identity/auth-instance-cookie-isolation.md` — [Auth.js instance cookie isolation — #706](platform/identity/auth-instance-cookie-isolation.md)
- `platform/identity/authentication-runtime.md` — [Zitadel Authentication Runtime Extraction Design](platform/identity/authentication-runtime.md)
- `browser-kernel-release-catalog.md` — [平台浏览器内核发行目录 V1](browser-kernel-release-catalog.md)
- `platform/runtime/compose-tofu-preflight.md` — [Compose OpenTofu preflight boundary](platform/runtime/compose-tofu-preflight.md)
- `platform/identity/enterprise-custom-roles.md` — [Enterprise custom roles v1 — Issue #598](platform/identity/enterprise-custom-roles.md)
- `platform/runtime/local-tls-revocation-maintenance.md` — [本机 TLS 撤销维护边界（#965）](platform/runtime/local-tls-revocation-maintenance.md)
- `platform/identity/member-directory-query.md` — [Member directory query v1](platform/identity/member-directory-query.md)
- `platform/identity/protected-member-role-native-cas.md` — [B1：受保护成员单业务角色的原生条件写入](platform/identity/protected-member-role-native-cas.md)
- `platform/identity/public-trial-identity-origin.md` — [Fixed HTTPS identity origin for the cloud trial](platform/identity/public-trial-identity-origin.md)
- `platform/runtime/retained-lan-access.md` — [Retained trial LAN access and scoped identity admission](platform/runtime/retained-lan-access.md)
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
- `domains/catalog/own-intake-provider-unbilled-recovery.md` — [#1165 供应商明确拒绝且未计费的原调用恢复](domains/catalog/own-intake-provider-unbilled-recovery.md)
- `enterprise-shared-model-configuration.md` — [企业统一模型与参考价估算（已撤下精确厂商价格/完整硬上界前置；原单一账本及预算保持）](enterprise-shared-model-configuration.md)
- `automatic-usd-fx.md` — [平台美元参考汇率自动更新](automatic-usd-fx.md)；#1249，用户决定改为周更，周更增量 IMPLEMENTATION_READY，现场启用须单独授权。
- `domains/ai/bailian-deepseek-shared-text-admission.md` — [百炼DeepSeek共享文字准入（参考价估算，原预算保持）](domains/ai/bailian-deepseek-shared-text-admission.md)
- `domains/ai/ai-disabled-application-lifecycle.md` — [关闭新 AI 执行时保留完整应用、配置与历史读取](domains/ai/ai-disabled-application-lifecycle.md)
- `openai-flare-image-generation.md` — [Flare 图片参考估算冻结设计与实现（#1078，原报价与账本）](openai-flare-image-generation.md)
- `domains/ai/shared-channel-credentials.md` — [渠道凭据与模型定义分离（冻结合同，#1089已合入）](domains/ai/shared-channel-credentials.md)
- `domains/ai/eino-model-admission.md` — [Eino多厂商模型按需准入（冻结设计与实施合同；交付证据见PR #1093）](domains/ai/eino-model-admission.md)
- `domains/ai/channel-credentials-ui.md` — [企业渠道 Key UI 配置与轮换（D1–D3冻结、已获实施授权；工程证据见 PR #1096）](domains/ai/channel-credentials-ui.md)
- `referral-settlement-and-conduct.md` — [Referral settlement period and promotion conduct](referral-settlement-and-conduct.md)
- `referral-withdrawal-personal-kyc-contract.md` — [Referral Withdrawal Personal KYC Contract](referral-withdrawal-personal-kyc-contract.md)
- `settlement-reversal-locking.md` — [Immutable settlement reversal locking](settlement-reversal-locking.md)
- `platform/identity/subject-verification-tencent-esign-design.md` — [个人与企业主体认证：腾讯企业认证与阿里云个人认证](platform/identity/subject-verification-tencent-esign-design.md)
- `unified-base-prepaid-resources.md` — [统一基础方案与预付资源 V1](unified-base-prepaid-resources.md)
- `usage-event-business-object.md` — [用量明细的关联业务对象](usage-event-business-object.md)

<a id="acquisition"></a>

## 来源采集与浏览器插件

- `1688-acquisition-figma.md` — [1688 在线采集、规则与待适配接入](1688-acquisition-figma.md)
- `domains/sourcing/1688-batch-import.md` — [1688 批量导入合同](domains/sourcing/1688-batch-import.md)
- `domains/sourcing/1688-context-locator.md` — [1688 window.context 定位合同](domains/sourcing/1688-context-locator.md)
- `1688-server-public-browser-acquisition.md` — [服务端匿名浏览器采集（1688 public browser acquisition）架构设计](1688-server-public-browser-acquisition.md)
- `1688-acquisition-rate-and-egress-proxy.md` — [设计增补：匿名采集速率治理与出口代理（#514）](1688-acquisition-rate-and-egress-proxy.md)
- `1688-acquisition-channel-pricing.md` — [1688 acquisition channel pricing — design basis for #592](1688-acquisition-channel-pricing.md)
- `plugin-capture-amazon-variants.md` — [Amazon 插件：可选采集同族其他变体](plugin-capture-amazon-variants.md)
- `plugin-capture-durable-drafts.md` — [商品采集插件：可恢复本地草稿规划与设计 v1](plugin-capture-durable-drafts.md)
- `domains/sourcing/plugin-capture-handoff-worker-restart.md` — [商品采集交接在扩展后台重建后恢复](domains/sourcing/plugin-capture-handoff-worker-restart.md)
- `plugin-capture-entry-flow.md` — [Product capture entry and original-tab return](plugin-capture-entry-flow.md)
- `plugin-capture-ingestion.md` — [插件采集增量架构：规格图片与 Amazon 接收保存 V1](plugin-capture-ingestion.md)
- `plugin-capture-listing-materials.md` — [插件上架资料补齐：高清相册、参数、正文/详情图与重量 V1](plugin-capture-listing-materials.md)
- `plugin-capture-page-entry.md` — [Product capture: source-page primary entry](plugin-capture-page-entry.md)
- `plugin-capture-session-return.md` — [Product capture: original-tab return across worker recreation](plugin-capture-session-return.md)
- `domains/sourcing/plugin-capture-source-quotes.md` — [插件资料补齐与来源报价只读展示 V1](domains/sourcing/plugin-capture-source-quotes.md)
- `source-business-flow.md` — [#30 / #307 完整来源业务闭环设计](source-business-flow.md)

<a id="product"></a>

## 商品、资产与草稿

- [Listing 领域导航：候选草稿、四阶段刊登、人工审核和原提交回执](domains/listing/README.md) — 只定位原合同，不新建 Listing/Review/Product 事实 Owner


- `domains/catalog/own-product-sku-packaging.md` — [自有商品逐 SKU 含包装事实录入](domains/catalog/own-product-sku-packaging.md)
- `domains/listing/completion-draft-history.md` — [待补全草稿保存历史与比较 v1](domains/listing/completion-draft-history.md)
- `domains/listing/completion-draft-restore.md` — [待补全草稿：恢复历史内容为新版本](domains/listing/completion-draft-restore.md)
- `domains/listing/completion-draft-structured.md` — [CompletionDraft 结构化候选 v1](domains/listing/completion-draft-structured.md)
- `domains/listing/completion-draft-internal-note.md` — [待补全草稿内部备注](domains/listing/completion-draft-internal-note.md)
- `domains/research/market-research-selection.md` — [市场研究与选品：最小领域合同](domains/research/market-research-selection.md)
- `domains/research/market-research-evidence-analysis-v1.md` — [两个中心 V1：研究证据分析合同增量](domains/research/market-research-evidence-analysis-v1.md)
- `amazon-opportunity-handoff-v1.md` — [Amazon 美国站调研机会与选品交接 V1](amazon-opportunity-handoff-v1.md)
- `amazon-us-zenrows-first-page.md` — [Amazon 美国站 ZenRows 单页试用](amazon-us-zenrows-first-page.md)
- `issue1070-local-acceptance-fixture.md` — [#1070 本地验收项目合成资料导入](issue1070-local-acceptance-fixture.md)
- `domains/catalog/own-product-ai-intake.md` — [自有商品 AI 录入 V1（含多 SKU）](domains/catalog/own-product-ai-intake.md) · #1158 设计候选，实施准入以正文为准。
- `domains/catalog/own-product-intake-openai-mini.md` — [自有商品 AI 整理 OpenAI Mini 单图接入（#1159）](domains/catalog/own-product-intake-openai-mini.md)
- `domains/catalog/own-product-intake-price-review.md` — [自有商品录入价格建议与人工确认](domains/catalog/own-product-intake-price-review.md)
- `domains/catalog/own-product-intake-stock-review.md` — [自有商品录入库存建议与逐行确认](domains/catalog/own-product-intake-stock-review.md)
- `domains/listing/own-product-draft-entry.md` — [自有商品单件转入与独立待补全入口 v1](domains/listing/own-product-draft-entry.md)
- `domains/catalog/own-product-edit.md` — [自有商品版本更新与独立编辑页 v1](domains/catalog/own-product-edit.md)
- `domains/catalog/own-product-entry.md` — [自有商品独立录入页与多变体 v1](domains/catalog/own-product-entry.md)
- `domains/listing/own-product-existing-drafts.md` — [自有商品已有草稿与继续编辑 v1](domains/listing/own-product-existing-drafts.md)
- `domains/catalog/own-product-history.md` — [自有商品历史读取、旧版详情和对比 v1](domains/catalog/own-product-history.md)
- `domains/catalog/own-product-optimization.md` — [自有商品无店铺依赖的内容优化与 Apply](domains/catalog/own-product-optimization.md)
- `domains/catalog/own-product-rich-description.md` — [自有商品描述富文本 v1](domains/catalog/own-product-rich-description.md)
- `domains/catalog/own-product-source-media-trial.md` — [自有商品来源图片的本地隔离试用](domains/catalog/own-product-source-media-trial.md)
- `domains/catalog/product-content-localization.md` — [商品内容完善与本地化智能体 v1](domains/catalog/product-content-localization.md)
- `product-domain-contracts.md` — [Product Domain Contracts](product-domain-contracts.md)
- `domains/catalog/rich-description-simple-editor-images.md` — [官方 Simple Editor 与描述图片独立上传](domains/catalog/rich-description-simple-editor-images.md)
- `domains/supply-market/selected-application-multi-product.md` — [多商品优选申请 v1](domains/supply-market/selected-application-multi-product.md)
- `domains/supply-market/selected-application-overview.md` — [优选申请总览只读汇总 v1](domains/supply-market/selected-application-overview.md)
- `domains/listing/supply-unbound-completion-drafts.md` — [Supply 未绑定店铺待补全草稿](domains/listing/supply-unbound-completion-drafts.md)

<a id="store"></a>

## 店铺、平台与履约

- [Marketplace 领域导航：SHEIN、TEMU、TikTok Shop、Walmart 的现行协议和发布门禁](domains/marketplace/README.md) — Store 授权、Listing 提交与平台 adapter 权责分离


- `domains/sourcing/amazon-condition-quotes-draft.md` — [Amazon 新品和二手来源报价：设计草案](domains/sourcing/amazon-condition-quotes-draft.md)
- `domains/listing/completion-shein-import.md` — [CompletionDraft → SHEIN Target 带入预览与确认](domains/listing/completion-shein-import.md)
- `domains/marketplace/erpnext-shein-fulfillment-fit.md` — [ERPNext 与 SHEIN 双履约适配验证](domains/marketplace/erpnext-shein-fulfillment-fit.md)
- `my-supply-chain-shein.md` — [我的供应链：SHEIN 美国站完整交付设计](my-supply-chain-shein.md)
- `supply-four-step-continuation.md` — [四步供应链与补全/审核后继续上传](supply-four-step-continuation.md)
- `domains/marketplace/order-procurement-fulfillment.md` — [订单、采购、发货与物流首版设计草案](domains/marketplace/order-procurement-fulfillment.md)
- `store-core-schema-alignment.md` — [own675 Store 核心合同对齐方案](store-core-schema-alignment.md)
- `domains/workbench/report-center-shein-offline-package-source.md` — [Report Center：原 SHEIN 离线包诊断正常接线候选](domains/workbench/report-center-shein-offline-package-source.md)
- `domains/workbench/report-center-shein-target-source.md` — [Report Center：当前 SHEIN 已保存资料来源候选](domains/workbench/report-center-shein-target-source.md)
- `domains/listing/shein-sales-attribute-mapping.md` — [SHEIN 单商品多颜色、多 SKU 销售属性映射](domains/listing/shein-sales-attribute-mapping.md)；#1158 当前冻结设计 IMPLEMENTATION_READY，独立 R1 无 BLOCKER，四项实现验证待后续授权；保留同一草稿三行，稀疏集合发布资格仍待确认。
- `domains/listing/controlled-local-platform-draft.md` — [原35444受控本地平台草稿接线](domains/listing/controlled-local-platform-draft.md)；#1158 固定试点合同，测试记录永久不可发布；实例级互斥的最新决定见共存设计，实施/运行证据在 Issue。
- `domains/listing/local-and-official-draft-coexistence.md` — [同一实例内测试草稿与正式刊登共存](domains/listing/local-and-official-draft-coexistence.md)；#1158 最小方案与独立架构评审，独立本地接口用途、原 owner/保存协议及正式发布准入保持；设计与后续实现/运行阶段分列见原 Issue/PR，不据文档迁移授予操作权限。
- `domains/listing/shein-color-split-drafts.md` — [SHEIN 按颜色拆分本地平台草稿](domains/listing/shein-color-split-drafts.md)；历史独立能力，已撤回作为本例必经流程的推荐，参见单商品映射设计。
- `domains/marketplace/shein-native-validator-contract.md` — [SHEIN 原生资料公共诊断与材料 Tool v2](domains/marketplace/shein-native-validator-contract.md)
- `domains/marketplace/shein-offline-stock-packaging.md` — [SHEIN 离线诊断的库存和 SKU 包装缺项](domains/marketplace/shein-offline-stock-packaging.md)
- `shein-offline-validation-input-binding.md` — [SHEIN 离线诊断输入、freshness 与组织范围读取设计](shein-offline-validation-input-binding.md)
- `domains/listing/shein-publication-flow.md` — [SHEIN 美国站：审核、首次上架与状态回读最小闭环](domains/listing/shein-publication-flow.md)
- `domains/marketplace/shein-submit-readiness-action-policy.md` — [SHEIN Submit Readiness Action Policy](domains/marketplace/shein-submit-readiness-action-policy.md)
- `store-browser-workbench-implementation.md` — [店铺浏览器接口实施交接](store-browser-workbench-implementation.md)
- `store-center-application.md` — [Store Center 接入 current application v1](store-center-application.md)
- `store-center-figma.md` — [Store Center：我的店铺完整 Figma 路径 v1](store-center-figma.md)
- `store-center-platform-observations.md` — [店铺中心：平台商品与订单物流只读首版](store-center-platform-observations.md)
- `store-observations-installation.md` — [在当前 Store 安装既有订单观察模块](store-observations-installation.md)
- `store-empty-observations-contract-alignment.md` — [四张空 Store 观测表的最小合同对齐](store-empty-observations-contract-alignment.md) · #1231 架构 IMPLEMENTATION_READY；实现/副本/原实例操作另行授权。
- `store-observations-preparation-runtime.md` — [Store observations internal preparation runtime](store-observations-preparation-runtime.md)
- `domains/supply-market/supply-market-choice-eligibility.md` — [优选申请资格读取：正常不可申请结果](domains/supply-market/supply-market-choice-eligibility.md)
- `domains/supply-market/supply-market-card-selection.md` — [供应市场列表价格、选入状态与公开商家名称](domains/supply-market/supply-market-card-selection.md)；#1292，独立设计及认证名称修订IMPLEMENTATION_READY。
- `platform/geography/supply-region-catalog.md` — [账户资料与供应市场共用中国省市目录](platform/geography/supply-region-catalog.md)
- `supply-market-categories.md` — [硕米自营与硕米优选共用商品类目](supply-market-categories.md) · #1377，独立架构已确认，IMPLEMENTATION_READY。
- `domains/supply-market/supply-market-filters.md` — [自营与优选商品筛选](domains/supply-market/supply-market-filters.md)；#1292，只读市场查询增量，独立设计复核IMPLEMENTATION_READY。
- `domains/supply-market/supply-market.md` — [供应市场 v1 — 明确发布、优选申请与 SDS 定制成品](domains/supply-market/supply-market.md)
- `domains/marketplace/temu-official-contract-evidence.md` — [TEMU 官方接口与站点证据](domains/marketplace/temu-official-contract-evidence.md)
- `domains/marketplace/temu-platform.md` — [TEMU 三种模式与全部官方站点接入设计](domains/marketplace/temu-platform.md)
- `domains/listing/tiktok-pre-dispatch-no-send.md` — [TikTok 发送前未执行的终止与显式重试：冻结设计](domains/listing/tiktok-pre-dispatch-no-send.md)
- `domains/marketplace/tiktok-shop-us-contract-evidence.md` — [TikTok Shop 美国跨境：官方合同证据](domains/marketplace/tiktok-shop-us-contract-evidence.md)
- `domains/marketplace/tiktok-shop-us.md` — [TikTok Shop 美国跨境：店铺授权、商品草稿与受控刊登](domains/marketplace/tiktok-shop-us.md)
- `domains/listing/walmart-feed-recovery.md` — [Walmart Feed：明确拒绝的修复重试与有界续读](domains/listing/walmart-feed-recovery.md)
- `domains/marketplace/walmart-us-contract-evidence.md` — [Walmart 美国站官方合同证据与边界](domains/marketplace/walmart-us-contract-evidence.md)
- `domains/marketplace/walmart-us.md` — [Walmart 美国站：多商家授权、商品草稿与受控刊登](domains/marketplace/walmart-us.md)

<a id="agent"></a>

## Agent、Tool、Skill 与知识

- [AI 领域候选导航：#132 标题对照与先导评测](domains/ai/README.md) — 两份独立设计尚未准入；真实模型/费用执行未授权

- `domains/ai/agent-customization.md` — [智能体定制需求与人工进度 V1](domains/ai/agent-customization.md)
- `domains/workbench/agent-knowledge-context.md` — [Agent + Knowledge Context V1](domains/workbench/agent-knowledge-context.md)
- `domains/workbench/knowledge-original-file-deletion.md` — [知识库原文件删除与容量释放 V1](domains/workbench/knowledge-original-file-deletion.md)
- `ai-capability-platform.md` — [AI Capability & Agent Platform 设计](ai-capability-platform.md)
- `domains/ai/model-and-services.md` — [硕米「模型与服务」设计](domains/ai/model-and-services.md) · 首版范围已确认，设计已评审。
- `domains/ai/model-connectivity-test.md` — [模型与服务连通性设计](domains/ai/model-connectivity-test.md) · #1420 只读模型资料查询。
- `domains/ai/ai-provider-contracts.md` — [Provider-Neutral AI Contracts Design](domains/ai/ai-provider-contracts.md)
- `commerce-tool-canonical-inspection.md` — [B0 Canonical Product Inspection Tool 设计](commerce-tool-canonical-inspection.md)
- `commerce-tool-foundation.md` — [Phase 2A 商业工具（Commerce Tool）基础设计](commerce-tool-foundation.md)
- `product-image-facts-advice.md` — [图片事实检查与非执行性建议：C2-1 / C3-1（R1 冻结，实现见 #1076）](product-image-facts-advice.md)
- `product-image-visual-analysis.md` — [商品图片单图视觉分析：C2-2（限定工程检查已完成；运行与产品验收 NOT_RUN）](product-image-visual-analysis.md)
- `domains/listing/listing-candidate-category-rules.md` — [SHEIN 候选类目规则查询与只读解释：C3-2（R1 冻结，IMPLEMENTATION_READY）](domains/listing/listing-candidate-category-rules.md)
- `enterprise-agent-permission-modes.md` — [企业智能体权限与审批模式 V1](enterprise-agent-permission-modes.md)
- `domains/workbench/official-knowledge-provenance.md` — [官方知识库 V2：产品内依据，不使用 GitHub 地址](domains/workbench/official-knowledge-provenance.md)
- `domains/workbench/official-knowledge.md` — [Official Knowledge V1 — Product Gate and design proposal](domains/workbench/official-knowledge.md)
- `domains/ai/official-skill-product-entry.md` — [官方技能直接处理已有商品](domains/ai/official-skill-product-entry.md)
- `domains/ai/official-skill-automatic-preparation.md` — [官方技能模板自动准备](domains/ai/official-skill-automatic-preparation.md)
- `domains/workbench/task-center-summary.md` — [任务中心统一统计](domains/workbench/task-center-summary.md)
- `domains/ai/issue132-title-evaluation-admission.md` — [标题对照评测准入与预算边界（#132）](domains/ai/issue132-title-evaluation-admission.md)
- `domains/ai/issue132-title-pilot-local-entry.md` — [标题先导的本机操作入口（#132）](domains/ai/issue132-title-pilot-local-entry.md)
- `domains/workbench/organization-agent-configuration.md` — [Organization Agent Configuration V1 — Slice D](domains/workbench/organization-agent-configuration.md)
- `domains/listing/platform-listing-materials-agent.md` — [平台上架资料智能体 v1](domains/listing/platform-listing-materials-agent.md)
- `pod-design-agent.md` — [POD设计智能体 v1：从设计意图生成图案](pod-design-agent.md)
- `domains/ai/private-agent-delivery.md` — [指定企业私有智能体交付 V1](domains/ai/private-agent-delivery.md)
- `domains/ai/product-agent-google-interactions.md` — [Product Agent Google Interactions 标题路由增量合同](domains/ai/product-agent-google-interactions.md)
- `product-agent-runtime-contract.md` — [Product Agent 最小运行合同与接入依赖](product-agent-runtime-contract.md)
- `domains/ai/product-agent-text-provider-neutral.md` — [Product Agent 标题文本供应商可替换合同（候选）](domains/ai/product-agent-text-provider-neutral.md)
- `domains/research/product-selection-comparison-agent.md` — [Product selection comparison Agent v1](domains/research/product-selection-comparison-agent.md)
- `domains/workbench/shop-operations-diagnosis-agent.md` — [Shop operations diagnosis Agent v1](domains/workbench/shop-operations-diagnosis-agent.md)
- `skill-capability.md` — [Skill V1：三层归属与现有商品内容 Agent 接入](skill-capability.md)
- `skill-script-sandbox.md` — [技能脚本执行：Linux Docker + OpenSandbox + gVisor 接入设计](skill-script-sandbox.md) · IMPLEMENTATION_READY，正式执行 #1293/PR #1298；§23 补齐独立WSL执行profile的D-IT1/init-proc真实验证并修复metadata label限制。结论仅限该profile，业务部署仍核对实际挂载/公网路由；未开放用户试用、未授权合并/业务部署，Kubernetes非前置。
- `paused-title-script-review.md` — [暂停新启动后的商品标题脚本人工审核](paused-title-script-review.md) · #1355
- `domains/workbench/tool-customization-figma.md` — [工具定制完整 Figma 表单与私有参考资料](domains/workbench/tool-customization-figma.md)
- `tool-market.md` — [工具市场 v1 — 企业启用与人工定制需求](tool-market.md)

<a id="image"></a>

## 图片执行、预算与恢复

- `domains/ai/bailian-product-image-wiring.md` — [百炼商品图片接线 V1](domains/ai/bailian-product-image-wiring.md)
- `domains/ai/generic-image-template-entry.md` — [通用图片模板与技能入口带入](domains/ai/generic-image-template-entry.md)
- `domains/ai/grsai-vip-image-binding.md` — [GRSAI VIP 图片可选绑定 V1](domains/ai/grsai-vip-image-binding.md)
- `domains/ai/image-agent-budget-authorization.md` — [Image Agent Budget Authorization Design](domains/ai/image-agent-budget-authorization.md)
- `domains/ai/image-agent-candidate-release-gate.md` — [Image Agent Candidate Release Gate Design](domains/ai/image-agent-candidate-release-gate.md)
- `domains/ai/image-agent-effect-finalization.md` — [Image Agent external-effect finalization design](domains/ai/image-agent-effect-finalization.md)
- `domains/ai/image-agent-effect-policy.md` — [Image Agent V3 Effect Policy Design](domains/ai/image-agent-effect-policy.md)
- `domains/ai/image-agent-effect-recovery.md` — [Image Agent External Effect Recovery Workflow Design](domains/ai/image-agent-effect-recovery.md)
- `domains/ai/image-agent-effect-terminalization.md` — [Image Agent Effect Terminalization Design](domains/ai/image-agent-effect-terminalization.md)
- `domains/ai/image-agent-recovery.md` — [Image Agent Recovery and Compatibility Design](domains/ai/image-agent-recovery.md)
- `domains/ai/image-agent-workflow.md` — [Image Agent Workflow](domains/ai/image-agent-workflow.md)
- `domains/ai/image-api-event-read-permissions.md` — [Image API event-read permission correction](domains/ai/image-api-event-read-permissions.md)
- `image-download-safety.md` — [Image Download Safety Boundary Design](image-download-safety.md)
- `domains/ai/image-plan-optional-route.md` — [Single-product image plan optional route V1](domains/ai/image-plan-optional-route.md)
- `domains/ai/image-text-language-confirmation.md` — [Image target language, localization and confirmed copy](domains/ai/image-text-language-confirmation.md)
- `product-image-agent.md` — [商品图片智能体 V1 — 完整图片能力设计](product-image-agent.md)
- `source-white-background.md` — [来源商品白底图与资源结算合同](source-white-background.md)
- `domains/ai/vip-original-output-recovery.md` — [VIP original output recovery and paid-result presentation](domains/ai/vip-original-output-recovery.md)

<a id="workspace"></a>

## 工作台、项目与任务投影

- [AI Workbench 领域导航：Chat、BusinessTask、Project、原生运行、定时计划与报告](domains/workbench/README.md) — 已有商品内容计划保留，当前新建定时任务只开放店铺经营日报


- `domains/workbench/chat-visible-store-query.md` — [硕米 Chat 按当前权限查询店铺概况](domains/workbench/chat-visible-store-query.md) · #1259 独立架构评审通过；正文记录已获实施及隔离验证授权，不含真实模型调用、原 35444 操作或释放原预留；本索引不新增实际试用授权。
- `domains/workbench/chat-selected-store-context.md` — [硕米 Chat 选择店铺背景](domains/workbench/chat-selected-store-context.md) · 已连接店铺选择及权限校验。
- `chat-project-actions.md` — [硕米 Chat 选择 Agent 技能与项目操作](chat-project-actions.md) · #1396，覆盖个人标题技能的创作、计费及恢复，后续标题技能版本交接和项目操作。
- `chat-general-text-skills.md` — [硕米 Chat 创建并使用个人文字技能](chat-general-text-skills.md) · #1429，通用文字创作与 Chat 选用已独立复核并达到 IMPLEMENTATION_READY。
- `domains/workbench/ai-workbench-chat-business-task.md` — [AI Workbench Chat + BusinessTask V1 — Slice E](domains/workbench/ai-workbench-chat-business-task.md)
- `domains/workbench/ai-workbench-conversation-runtime.md` — [硕米 Chat 会话管理独立运行接线](domains/workbench/ai-workbench-conversation-runtime.md)
- `domains/workbench/ai-workbench-project-center.md` — [AI Workbench Project Center V1](domains/workbench/ai-workbench-project-center.md)
- `domains/workbench/ai-workbench-reports.md` — [AI工作台个人报告 v1](domains/workbench/ai-workbench-reports.md)
- `domains/workbench/ai-workbench-work-scope.md` — [AI 工作台：固定会话工作范围与任务筛选](domains/workbench/ai-workbench-work-scope.md)
- `domains/workbench/chat-selected-product-context.md` — [Chat 规划读取已选商品资料](domains/workbench/chat-selected-product-context.md)
- `domains/workbench/ai-workbench-scheduled-tasks.md` — [AI 工作台预约与周期任务 V1：架构合同](domains/workbench/ai-workbench-scheduled-tasks.md) · #1244 已记录后续合并、授权部署和单次真实调用；原设计与隔离证据保留，新建仅日报，用户验收不由历史测试代签。
- `domains/workbench/scheduled-plan-edit-copy.md` — [定时计划编辑与复制](domains/workbench/scheduled-plan-edit-copy.md) · #1297 IMPLEMENTATION_READY；复用原计划 owner，保留在途任务与历史，保存前重新核对时间和预算。
- `domains/workbench/chat-compose-before-create.md` — [内容型 Chat 创建前输入与背景](domains/workbench/chat-compose-before-create.md) · #1257 内容型会话设计；标题型范围接线不在本轮。
- `domains/workbench/chat-voice-input.md` — [Chat 语音输入与会话入口](domains/workbench/chat-voice-input.md) · #1257 独立架构复核通过，真实语音识别待用户试用。
- `domains/workbench/chat-task-object-goal-design-brief.md` — [Chat 通用任务创建与首批内容执行](domains/workbench/chat-task-object-goal-design-brief.md)
- `domains/workbench/chat-content-full-runtime-coexistence-candidate.md` — [内容 Chat 与完整商品智能体共存](domains/workbench/chat-content-full-runtime-coexistence-candidate.md)
- `domains/workbench/chat-local-open-preparation.md` — [Chat 定向结构与 OpenAI 文本用途准备](domains/workbench/chat-local-open-preparation.md)
- `content-agent-chat-coexistence.md` — [商品内容标题试用与现有 Chat 共存的最小运行接线](content-agent-chat-coexistence.md)
- `completed-work-store-scope.md` — [#36 已完成本地资料的店铺归属投影](completed-work-store-scope.md)
- `product-workbench-trial-runtime.md` — [#36 可保留的本机隔离试用运行边界](product-workbench-trial-runtime.md)
- `domains/workbench/notification-center.md` — [通知中心 V1：站内通知与业务状态投影](domains/workbench/notification-center.md)
- `domains/workbench/operations-cockpit-figma-read.md` — [运营驾驶舱四页 Figma 与只读投影补充](domains/workbench/operations-cockpit-figma-read.md)
- `operations-cockpit.md` — [运营驾驶舱首版：人工经营事实与有据可查的经营投影](operations-cockpit.md)
- `domains/workbench/cockpit-listing-task-metrics.md` — [运营驾驶舱首页：刊登与任务统计](domains/workbench/cockpit-listing-task-metrics.md)
- `domains/workbench/cockpit-walmart-feed-rate.md` — [运营驾驶舱：Walmart Feed 提交受理率](domains/workbench/cockpit-walmart-feed-rate.md)
- `domains/workbench/cockpit-tiktok-us-publish-rate.md` — [运营驾驶舱：TikTok Shop US 提交审核率](domains/workbench/cockpit-tiktok-us-publish-rate.md)
- `domains/workbench/cockpit-shein-order-amount.md` — [运营驾驶舱：SHEIN US 已保存订单商品金额（原币）](domains/workbench/cockpit-shein-order-amount.md)
- `domains/workbench/cockpit-shein-self-operated-orders.md` — [SHEIN US 普通自运营消费者订单及来源事实](domains/workbench/cockpit-shein-self-operated-orders.md)
- `domains/workbench/cockpit-shein-supply-orders.md` — [SHEIN 全托：按采购单号批次保存供货单与数量观察](domains/workbench/cockpit-shein-supply-orders.md)
- `domains/workbench/cockpit-shein-supply-business-times.md` — [SHEIN 全托：业务时间原值、来源与缺失状态保存](domains/workbench/cockpit-shein-supply-business-times.md)
- `domains/workbench/cockpit-shein-saved-supply-dates.md` — [已保存全托供货单按原始日期观察](domains/workbench/cockpit-shein-saved-supply-dates.md)
- `domains/workbench/cockpit-walmart-order-amount.md` — [Walmart US 消费者订单保存与单店商品金额](domains/workbench/cockpit-walmart-order-amount.md)
- `domains/workbench/cockpit-walmart-order-sync.md` — [Walmart US 消费者订单手动同步接线](domains/workbench/cockpit-walmart-order-sync.md)
- `domains/workbench/cockpit-tiktok-order-amount.md` — [TikTok Shop US 消费者订单保存与单店商品金额](domains/workbench/cockpit-tiktok-order-amount.md)
- `domains/workbench/cockpit-tiktok-order-sync.md` — [TikTok Shop US 跨境店消费者订单手动同步](domains/workbench/cockpit-tiktok-order-sync.md)
- `domains/workbench/cockpit-temu-order-amount.md` — [Temu US 本土/半托管消费者订单保存与单店商品金额](domains/workbench/cockpit-temu-order-amount.md)
- `domains/ai/private-draft-offline-trial.md` — [指定企业平台草稿质检：隔离离线试用装配](domains/ai/private-draft-offline-trial.md)
- `domains/workbench/project-center-overview-read.md` — [Project Center overview read V1](domains/workbench/project-center-overview-read.md)
- `domains/workbench/task-center-native-run-projection.md` — [任务中心：多来源、多能力的原生运行投影](domains/workbench/task-center-native-run-projection.md)

<a id="services"></a>

## 数据与生态服务

- `domains/data-services/data-services-native-authorization.md` — [Data Services native authorization — #619 / PR #631](domains/data-services/data-services-native-authorization.md)
- `data-services.md` — [数据市场与 API 管理首版](data-services.md)
- `domains/data-services/data-services-execution-lifecycle.md` — [关闭采集执行时保留数据服务配置、资料与历史](domains/data-services/data-services-execution-lifecycle.md)
- `ecoservices-qualification-runtime.md` — [Ecoservices qualification-only runtime](ecoservices-qualification-runtime.md)
- `ecosystem-services.md` — [生态服务 V1：第三方入驻、服务交付与在线收款结算](ecosystem-services.md)
- `platform-support-chatwoot.md` — [Platform customer support through Chatwoot official APIs](platform-support-chatwoot.md)

Every architecture document is explicitly indexed above. 新增、合并或删除合同时同步本索引；当前规则写入已有 owner 的合同，不追加退休记录、历史索引或逐轮评审日志。

- `domains/workbench/cockpit-temu-order-sync.md` — [Temu US 本土/半托管消费者订单手动同步](domains/workbench/cockpit-temu-order-sync.md)
