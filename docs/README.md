# 项目文档

从这里查找硕米智能引擎的产品依据、开发入口和使用说明。具体任务的当前范围与进度以执行 Issue / PR 为准；文档中的设计准入、源码交付、实例开放和用户验收是不同状态。

## 按目的阅读

| 你要做什么 | 阅读入口 |
| --- | --- |
| 了解产品、启动项目 | [项目 README](../README.md)、[新建本地 Account Compose](../deployments/docker/account-compose/README.md)、[可选模块](../deployments/docker/account-compose/MODULES.md) |
| 确认页面、导航和产品语义 | [产品文档](product/README.md)、[当前 UI / IA Authority](product/final-ui-ia-authority.md) |
| 查找领域合同、权限、事务与恢复边界 | [架构索引](architecture/README.md)、[Project Boundaries](architecture/project-boundaries.md) |
| 找代码入口、开发和构建方法 | [开发说明](development/README.md)、[源码入口地图](development/repository-structure.md#current-entrypoint-map)、[前端](../web/listingkit-ui/README.md)、[采集插件](../extensions/product-capture/README.md) |
| 查找使用步骤、配置和运行交接 | [操作与运行说明](operations/README.md) |
| 执行任务、提交 PR | [AGENTS.md](../AGENTS.md)、[Issue 驱动派工](engineering/issue-driven-development.md)、[Issue 模板](../.github/ISSUE_TEMPLATE/execution-task.md)、[PR 模板](../.github/pull_request_template.md) |
| 判断旧代码的归属与退出边界 | [重构文档](refactoring/README.md)、[Legacy Hard-Cut](refactoring/legacy-hard-cut-policy.md)、[Legacy Register](refactoring/legacy-register.md) |
| 追溯旧设计、实施计划或验证结果 | [设计与计划资料](superpowers/README.md)、[历史重构材料](refactoring/README.md#historical-records)、[工程证据目录](engineering/evidence)、[产品验证记录](product/validation/runs/README.md) |

当前工作排序见 [Roadmap #137](https://github.com/henryxu1008/shuomi-ai-commerce/issues/137) 及具体执行 Issue / PR。本页不复制运行地址、账号、滚动 SHA 或验收结果。

## 如何判断文档是否适用

- **当前依据**：适用的 Product Decision、Figma Authority、架构合同和执行 Issue；按批准范围及明确的替代关系阅读。`IMPLEMENTATION_READY` 表示设计准入，不能据此认定功能已部署。
- **使用与交付说明**：说明已交付源码或候选版本的入口、配置与操作；按文内的版本、启用条件和限制使用，实例开放状态另行确认。
- **历史材料**：已标 `SUPERSEDED`、`RETIRED`、`HISTORICAL` 或绑定旧基线的快照，只用于追溯；旧命令、未勾选计划和历史 PASS 不恢复当前执行权限。
- **混合文档**：先读顶部的准确替代说明。只退出明确被替代的部分，仍有效的合同、商业门禁、安全和恢复边界继续有效；文件名带 ListingKit 或日期不代表整份过期。

新业务遵守[全新系统产品基线](product/greenfield-no-legacy-migration.md)。不要从旧文档恢复兼容层、旧数据迁移、双读双写或第二事实源。

## 文档维护

同一事实维护一个当前版本，其他位置引用。产品与合同放在各自目录，运行交付放在操作说明或对应工程交接，任务进度和准确验证证据放在 Issue / PR。新增架构文档继续登记到[架构索引](architecture/README.md)。

清理以当前用途和明确替代关系为准：已完成的普通实施步骤、阶段scope/checkpoint、被替代的版本草稿和重复旧入口退出当前树，完整原文通过固定Git版本追溯；过程勾选或历史引用不自动要求每份计划常驻工作树。仍作为当前合同消费、未被撤销的准入、现有guard保护的冻结正文、必要实际验证及有效运维边界保留；直接历史引用可改为准确Git版本链接，不改变原判断和结果，不降低测试或保护规则。

根目录不再保留旧Amazon Task Workbench API和旧刊登数据库连接代理总结。完整原文分别见[旧API协议](https://github.com/henryxu1008/shuomi-ai-commerce/blob/cfd08cf5425f0cc4fb05db6c7df4bc265cfb7f93/docs/amazon-listing-workbench-api.md)与[连接池历史总结](https://github.com/henryxu1008/shuomi-ai-commerce/blob/cfd08cf5425f0cc4fb05db6c7df4bc265cfb7f93/docs/database-connection-optimization.md)；当前API、数据库和运行合同从上表各owner入口查找，不执行旧示例。
