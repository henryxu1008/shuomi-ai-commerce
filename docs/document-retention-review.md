# 文档保留核对记录

返回[文档组织与迁移规则](document-organization.md)。

**核对版本：** `main` 的 `1e89e8d56e81146d83ed006e93e492746891b28d`。下表记录 48 条已核实的**原路径保留**：原清单建议保留的 41 条文档和 3 项资产，以及 4 条候选路径与原路径相同的待审文档。每条均核对了原路径和至少一个既有引用；表中只列代表性依据，历史或受限材料仅写引用类型，不复制详情或增加直达链接。

本表独立于 [CSV 清单](document-inventory.csv)，不改变其处置建议、状态或字节内容。核对版本已迁移 272/353，保留不算迁移。另 33 条候选路径与原路径不同的待审项不在本表范围内。原路径保留不表示正文仍是现行权威，也不表示实机、部署或产品验收；后续如果归属、引用或可分享范围不清，记为待补证并先保留原处。

## 稳定入口与项目基线（原建议 `KEEP_PATH`）

| 原路径 | 保留理由 | 代表性既有引用 |
| --- | --- | --- |
| `docs/architecture/architecture-review-checklist.md` | 统一的架构评审入口，不归单个业务域。 | `docs/architecture/README.md` |
| `docs/architecture/project-boundaries.md` | 领域事实与责任边界的全局入口。 | `docs/README.md`、`AGENTS.md` |
| `docs/architecture/project-target-architecture.md` | 项目目标架构跨多个领域，保留全局入口。 | `docs/README.md` |
| `docs/architecture/README.md` | 架构总索引，从文档首页进入。 | `docs/README.md` |
| `docs/development/README.md` | 开发规则和源码导航的分层入口。 | `docs/README.md` |
| `docs/development/repository-structure.md` | 代码与目录地图，在多个文档层复用。 | `docs/README.md`、`docs/architecture/README.md` |
| `docs/engineering/issue-driven-development.md` | 全仓 Issue 驱动协作规则，不归某个业务交付。 | `AGENTS.md`、`docs/engineering/README.md` |
| `docs/engineering/README.md` | 工程规则和消费者接线的索引。 | `docs/README.md` |
| `docs/operations/README.md` | 安装、维护与用户指引的索引。 | `docs/README.md` |
| `docs/product/ai-commerce-agent-platform-strategy.md` | 产品方向入口，不与某个实现合同混放。 | `docs/product/README.md` |
| `docs/product/final-ui-ia-authority.md` | 产品流程与历史 UI/IA 说明的统一入口，保留权威顺序。 | `docs/README.md`、`docs/architecture/README.md` |
| `docs/product/greenfield-no-legacy-migration.md` | 全仓新建系统产品基线，存在仓库级引用。 | `AGENTS.md`、`docs/product/README.md` |
| `docs/product/README.md` | 各产品决定和流程的入口。 | `docs/README.md` |

## 跨域架构和平台合同（原建议 `KEEP_PATH`）

| 原路径 | 保留理由 | 代表性既有引用 |
| --- | --- | --- |
| `docs/architecture/commerce-tool-canonical-inspection.md` | 只读商业工具核验跨领域，保留架构索引入口。 | `docs/architecture/README.md` |
| `docs/architecture/commerce-tool-foundation.md` | 商业工具基础同时被工作台和业务领域消费。 | `docs/architecture/domains/workbench/README.md`、`docs/architecture/domains/catalog/product-image-facts-advice.md` |
| `docs/architecture/completed-work-store-scope.md` | 已完成工作与店铺范围的共享边界，保留工作台消费者引用。 | 工作台运行合同 |
| `docs/architecture/content-agent-chat-coexistence.md` | 会话与内容智能体共存是跨工作路径的合同。 | `docs/architecture/domains/workbench/task-center-native-run-projection.md` |
| `docs/architecture/data-services.md` | 数据服务总合同连到领域准入和工程交接，不并入一份消费者说明。 | `docs/architecture/domains/data-services/data-services-execution-lifecycle.md` |
| `docs/architecture/enterprise-shared-model-configuration.md` | 企业共享模型配置跨多个执行入口，保留统一定义。 | `docs/architecture/content-agent-chat-coexistence.md` |
| `docs/architecture/image-download-safety.md` | 图片读取安全边界被工作流复用，不搬成单功能的操作步骤。 | `docs/architecture/domains/ai/image-agent-workflow.md` |
| `docs/architecture/operations-cockpit.md` | 驾驶舱总设计承接多个诊断与指标消费者。 | `docs/architecture/domains/workbench/cockpit-listing-task-metrics.md` |
| `docs/architecture/platform-support-chatwoot.md` | 平台支持设计与运行说明分层，保留架构入口。 | 架构索引、部署说明 |
| `docs/architecture/pod-design-agent.md` | 设计智能体边界留在架构层，供相邻业务合同核对。 | `docs/architecture/domains/marketplace/order-procurement-fulfillment.md` |
| `docs/architecture/store-center-application.md` | 店铺中心接入总合同被多个店铺和工作台增量引用。 | `docs/architecture/operations-cockpit.md` |
| `docs/architecture/store-center-platform-observations.md` | 店铺只读观察合同同时服务平台和工作台。 | `docs/architecture/domains/marketplace/README.md` |
| `docs/architecture/store-core-schema-alignment.md` | 店铺核心结构的对齐依据独立于后续观察增量。 | 店铺增量架构合同、架构索引 |
| `docs/architecture/store-empty-observations-contract-alignment.md` | 保留店铺观测合同增量与已审边界，不混入安装指引。 | `docs/architecture/README.md` |
| `docs/architecture/store-observations-installation.md` | 店铺观测安装合同与消费者准备阶段分开保留。 | 工程运行合同、架构索引 |
| `docs/architecture/store-observations-preparation-runtime.md` | 店铺观测准备与运行边界已有安装和接线文档引用。 | 安装合同、部署说明 |

## 工程规则与交接（原建议 `KEEP_PATH`）

| 原路径 | 保留理由 | 代表性既有引用 |
| --- | --- | --- |
| `docs/engineering/1688-public-acquisition.md` | 保留采集工程交接与原历史范围，不作为当前启用凭证。 | 工程索引、产品决策说明 |
| `docs/engineering/administrator-module-projection.md` | 保留管理员模块展示接线的依据。 | `docs/engineering/README.md`、`docs/engineering/store-browser-permission-combination.md` |
| `docs/engineering/code-health-rules.md` | 保留工程索引引用的代码健康规则入口。 | `docs/engineering/README.md` |
| `docs/engineering/console-foundation.md` | 保留 Console 基础交接，不改写产品流程权威。 | `docs/engineering/README.md` |
| `docs/engineering/enterprise-storage-capacity-runbook.md` | 保留容量工程交接，仍与容量架构合同分离。 | 工程索引、部署说明 |
| `docs/engineering/knowledge-context-consumption.md` | 保留知识上下文如何被工程消费者接入的说明。 | 工程索引、操作说明 |
| `docs/engineering/new-run-ledger-installer-design.md` | 保留执行账本安装边界的历史工程依据，不恢复旧操作。 | 历史工程预览与只读核对 |
| `docs/engineering/referral-registration.md` | 保留个人推广注册交接，仍由原接线说明引用。 | `docs/engineering/integration/referrals-runtime.md` |
| `docs/engineering/selection-shared-model-wiring-preview.md` | 保留选品模型接线的预览与适用范围，不当作当前运行验收。 | 产品流程、历史工程交接 |
| `docs/engineering/store-browser-permission-combination.md` | 保留店铺浏览器权限组合与角色展示的工程说明。 | `docs/engineering/README.md` |
| `docs/engineering/unified-task-center.md` | 保留统一任务中心的工程交接及分层入口。 | `docs/engineering/README.md`、`docs/architecture/domains/workbench/README.md` |
| `docs/engineering/zitadel-official-provider-boundary.md` | 保留官方身份适配与授权边界交接，不把工程记录当作新授权。 | 工程索引、身份架构合同 |

## 证据与参考资产（原建议 `KEEP_ASSET`）

| 原路径 | 保留理由 | 代表性既有引用 |
| --- | --- | --- |
| `docs/architecture/evidence/erpnext-fulfillment-fit.json` | 现有履约适配文档引用的 JSON 证据，原件与说明分开保留。 | `docs/architecture/domains/marketplace/erpnext-shein-fulfillment-fit.md` |
| `docs/architecture/evidence/shein-us-data-v1-p1-p2.json` | 现有研究设计引用的静态 JSON 证据，不代表新的真实取数。 | 研究设计与离线试用合同 |
| `docs/development/shuomi-design-system-assets.csv` | 设计系统基线使用的 CSV 参考资产，继续作为独立数据表。 | `docs/development/ui/shuomi-design-system-baseline.md` |

## 四条同路径待审项（原建议 `REVIEW_*`）

| 原路径 | 保留理由 | 代表性既有引用 |
| --- | --- | --- |
| `docs/architecture/store-center-figma.md` | 保留店铺中心设计与固定验收测试路径；不外推 #641 验收。 | `docs/architecture/README.md`、`tests/architecture_docs_test.go` |
| `docs/architecture/tool-market.md` | 架构合同和同名操作指南职责不同；#613 工程关闭不证明当前安装或实机状态。 | `docs/operations/user-guides/tool-market.md` |
| `docs/engineering/issue1070-first-trial-walkthrough.md` | 保留原离线走查和合成样例证据，不替代端到端验收。 | 既有合成验收合同 |
| `docs/engineering/market-research-unified-update-plan.md` | 保留原路径，仅作历史工程交接证据。 | 历史工程交接 |
