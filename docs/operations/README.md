# 操作与运行说明

返回[文档首页](../README.md)。以下按用户操作组织源码交付和候选使用说明。是否在具体实例启用，以该实例交付记录和对应 Issue / PR 为准；本索引不签发部署、真实模型调用或数据操作权限。

## 从业务流程找到操作步骤

先从[产品业务流程入口](../product/flows/README.md)确定你要完成的操作，再读取本目录对应的安装、使用与恢复说明。**文档中的命令是适用范围内的操作方法，不授予真实实例变更、账号登录、采集或付费执行权限**；需先核对该实例的实际配置、当前权限和原操作回执。

<a id="installation"></a>

## 安装、配置与构建

- 本机安装工具：[安装与模块管理](runbooks/local-installation-management.md)，按准确实例授权执行。

- 新建独立本地环境：[Account Compose](../../deployments/docker/account-compose/README.md)、[模块组合](../../deployments/docker/account-compose/modules.md)。
- 当前后端命令与 owner：[Repository Structure](../development/repository-structure.md#current-entrypoint-map)。
- 标题技能脚本：[使用与私有执行配置](skill-script-sandbox.md)；默认关闭，D-IT1/execd init-proc 检查是试用开放前置。
- 镜像构建：[显式 Git 版本构建](application-image-build.md)。
- 登录隔离：[实例 cookie 隔离](runbooks/auth-instance-cookie-isolation.md)。
- 模型治理：[模型与费率管理](runbooks/model-route-pricing.md)。
- Research 经营评估：[当前安装的评估表准备](runbooks/research-assessment-install.md)，实际数据库变更仍须单独授权。
- 浏览器内核服务：[内核发行目录交接](browser-kernel-catalog.md)。

<a id="products"></a>

## 采集、商品资料与草稿

- 采集与资料：[浏览器插件](../../extensions/product-capture/README.md)、[来源素材](runbooks/source-materials.md)、[来源图试用](runbooks/own-product-source-media-trial.md)、[1688 采集计划](user-guides/acquisition-plans-local.md)。
- 自有商品：[录入](user-guides/own-product-entry.md)、[历史](user-guides/own-product-history.md)、[富文本描述](runbooks/own-product-rich-description.md)。
- 待补全草稿：[单件转入](user-guides/own-product-draft-entry.md)、[已有草稿](user-guides/own-product-existing-drafts.md)、[未绑定店铺草稿](user-guides/supply-unbound-completion-drafts.md)。
- 草稿编辑：[属性与 SKU](user-guides/completion-draft-structured.md)、[保存历史](user-guides/completion-draft-history.md)、[恢复为新版本](user-guides/completion-draft-restore.md)。
- 自有商品优选：[多商品申请](user-guides/selected-application-multi-product.md)，以当前多商品申请合同为准。
- SHEIN：[供应链操作与启动](my-supply-chain-shein.md)、[导入试用安装](runbooks/shein-import-installation.md)。

<a id="ai-work"></a>

## AI 工作与企业配置

- 标题与图片：[Product Agent](product-agent-trial.md)、[图片智能体](product-image-agent.md)、[Chat 到业务任务](runbooks/ai-workbench-chat-trial.md)。
- 预约与周期内容建议：[定时任务 V1](user-guides/ai-workbench-scheduled-tasks.md)。
- 工作组织：[项目中心](user-guides/project-center.md)、[统一任务中心](../engineering/unified-task-center.md)。
- 配置与授权：[企业智能体](user-guides/organization-agent-configuration.md)、[执行权限模式](enterprise-agent-permission-modes.md)、[Skill](user-guides/skill-capability.md)。
- 定制交付：[智能体定制](user-guides/agent-customization-handoff.md)。
- 试用与成员资源：[Local 非现金 AI 试用](local-ai-trial.md)、[成员资源](runbooks/member-resources.md)。

<a id="stores"></a>

## 店铺、工具与服务

- 店铺读取：[商品与订单观察接线](../engineering/integration/store-observations-integration-handoff.md)。
- 订单、采购与仓库履约：[首版运行与保存/恢复](user-guides/order-fulfillment.md)、[独立合成试用](runbooks/order-fulfillment-synthetic-trial.md)。合成环境、真实平台联调和生产分别取证；环境操作按当前授权执行。
- 平台专项：[TEMU](user-guides/temu-store-connection.md)、[TikTok Shop 美国站](user-guides/tiktok-shop-us.md)、[Walmart 美国站](user-guides/walmart-us.md)。各文档的候选、未开放和恢复限制继续适用。
- 平台服务：[工具市场](user-guides/tool-market.md)、[数据服务](../engineering/integration/data-services-runtime-handoff.md)、[生态服务](user-guides/ecosystem-services.md)。
- 阅读与诊断：[官方知识](../engineering/integration/official-knowledge-handoff.md)、[个人报告](../engineering/integration/report-center-runtime-handoff.md)、[经营诊断](../engineering/integration/shop-operations-diagnosis-handoff.md)、[运营驾驶舱](../engineering/integration/operations-cockpit-runtime-handoff.md)。
- 通知：[通知中心](runbooks/notification-center-trial.md)。

<a id="limited-scope"></a>

## 有界环境

[RUN-1 身份与路由装配](runbooks/isolated-application-acceptance.md)仅描述其明确范围；不作为完整产品验收或生产发布证明。专项功能未开放项见各自合同与当前执行 Issue。
