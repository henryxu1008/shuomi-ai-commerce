# 操作与运行说明

返回[文档首页](../README.md)。以下按用户操作组织源码交付和候选使用说明。是否在具体实例启用，以该实例交付记录和对应 Issue / PR 为准；本索引不签发部署、真实模型调用或数据操作权限。

## 安装、配置与构建

- 新建独立本地环境：[Account Compose](../../deployments/docker/account-compose/README.md)、[模块组合](../../deployments/docker/account-compose/MODULES.md)。
- 当前后端命令与 owner：[Repository Structure](../development/repository-structure.md#current-entrypoint-map)。
- 镜像构建：[显式 Git 版本构建](listingkit-image-build.md)。
- 登录隔离：[实例 cookie 隔离](auth-instance-cookie-isolation.md)。
- 模型治理：[模型与费率管理](model-route-pricing-v1.md)。
- 浏览器内核服务：[内核发行目录交接](browser-kernel-catalog.md)。

## 采集、商品资料与草稿

- 采集与资料：[浏览器插件](../../extensions/product-capture/README.md)、[来源素材](source-materials.md)、[来源图试用](own-product-source-media-trial-v1.md)、[1688 采集计划](../runbooks/acquisition-plans-local.md)。
- 自有商品：[录入](own-product-entry-v1.md)、[历史](own-product-history-v1.md)、[富文本描述](own-product-rich-description-v1.md)。
- 待补全草稿：[单件转入](own-product-draft-entry-v1.md)、[已有草稿](own-product-existing-drafts-v1.md)、[未绑定店铺草稿](supply-unbound-completion-drafts.md)。
- 草稿编辑：[属性与 SKU](completion-draft-structured-v1.md)、[保存历史](completion-draft-history-v1.md)、[恢复为新版本](completion-draft-restore-v1.md)。
- 自有商品优选：[多商品申请](selected-application-single-product.md)，原文件路径保留，交互范围已更新。
- SHEIN：[供应链操作与启动](my-supply-chain-shein-v1.md)、[导入试用安装](own675-shein-import-trial-installation-v1.md)、[离线资料说明](own675-offline-materials/README.md)。

## AI 工作与企业配置

- 标题与图片：[Product Agent](product-agent-trial.md)、[图片智能体](product-image-agent-v1.md)、[Chat 到业务任务](ai-workbench-chat-trial.md)。
- 工作组织：[项目中心](../runbooks/project-center-v1.md)、[统一任务中心](../engineering/unified-task-center.md)。
- 配置与授权：[企业智能体](organization-agent-configuration-v1.md)、[执行权限模式](enterprise-agent-permission-modes-v1.md)、[Skill](skill-capability-v1.md)。
- 定制交付：[智能体定制](agent-customization-v1-handoff.md)。
- 试用与成员资源：[Local 非现金 AI 试用](local-ai-trial-v1.md)、[成员资源](member-resources-v1.md)。

## 店铺、工具与服务

- 店铺读取：[商品与订单观察接线](../engineering/store-observations-integration-handoff.md)。
- 平台专项：[TEMU](temu-store-connection.md)、[TikTok Shop 美国站](tiktok-shop-us.md)、[Walmart 美国站](walmart-us-v1.md)。各文档的候选、未开放和恢复限制继续适用。
- 平台服务：[工具市场](current-tool-market.md)、[数据服务](../engineering/data-services-runtime-handoff.md)、[生态服务](ecosystem-services-v1.md)。
- 阅读与诊断：[官方知识](../engineering/official-knowledge-handoff.md)、[个人报告](../engineering/report-center-runtime-handoff.md)、[经营诊断](../engineering/shop-operations-diagnosis-handoff.md)、[运营驾驶舱](../engineering/operations-cockpit-runtime-handoff.md)。
- 通知：[通知中心](notification-center-trial.md)。

## 保留的专项边界与历史运行记录

以下材料保留有效门禁、环境隔离边界或历史证据，不能作为当前应用的通用部署说明，也不能据此恢复已退休的 CLI / Task 路径：

- [RUN-1 身份与路由环境](current-application-run1.md)、[环境拆分记录](acceptance-environment-split.md)：限定验收环境；FULL 延期和旧 schema 入口退休说明继续适用。
- [旧 ListingKit 发布候选](listingkit-release-candidate-runbook.md)、[客户试用 checklist](listingkit-customer-trial-acceptance-checklist.md)：历史发布/Task 材料；生产 `NOT_READY` 及仍有效的专项商业、安全和恢复门禁保留。
- [旧 Listing Control Plane](listing-control-plane-runbook.md)、[独立 SHEIN 登录 worker](shein-login-worker-current.md)：源码退休记录，无当前运行入口。
- [旧 SourceAccount preflight](source-account-ownership-preflight.md)、[Store 历史 hard-cut](store-service-history-hardcut.md)：历史处置记录，不是可重复执行的数据操作授权。

历史文件路径中的 `current`、`runbook` 或 `ListingKit` 不决定其适用性；以正文明确的范围、替代说明和当前合同为准。
