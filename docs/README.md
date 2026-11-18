# 项目文档

这里维护当前系统的**产品依据、领域合同、开发入口、工程协作及运行说明**。文档的目录分类不决定事实 owner；需求实施进度、准确提交/CI、部署和用户验收记录在对应 [GitHub Issue](https://github.com/henryxu1008/shuomi-ai-commerce/issues) / PR，不在此复制为另一份任务状态。

## 按目标查找

| 你需要解决什么问题 | 首选阅读入口 | 当前权威与边界 |
| --- | --- | --- |
| 产品战略、使用者及业务决策 | [产品索引](product/README.md)、[产品战略](product/ai-commerce-agent-platform-strategy.md) | 最新用户明确决定与已批准产品合同优先 |
| 市场调研、智能选品及商品刊登流程 | [研究与选品操作](product/flows/market-research-selection-centers-v1.md)、[业务流程与历史导航决定](product/final-ui-ia-authority.md) | Figma 只供理解业务流程，不决定最终 UI/IA |
| 事实 owner、授权、事务及恢复 | [架构合同索引](architecture/README.md)、[领域边界](architecture/project-boundaries.md)、[目标架构](architecture/project-target-architecture.md) | 领域合同决定谁拥有事实，不由页面或文件名推断 |
| 找代码、配置、构建和本地调试 | [开发入口](development/README.md)、[源码地图](development/repository-structure.md) | 源码和当前命令为准，准确实例配置另核 |
| 接单、架构准入、评审与交接 | [工程索引](engineering/README.md)、[Issue 驱动规范](engineering/issue-driven-development.md)、[AGENTS.md](../AGENTS.md) | 既有规则只维护一个权威正文 |
| 安装、使用、故障排查与恢复 | [运行与操作说明](operations/README.md) | 工程交付、实例就绪和客户验收分别确认 |
| 查阅文档目录设计或准备迁移 | [文档组织与迁移规则](document-organization.md)、[逐文件候选清单（CSV）](document-inventory.csv) | 清单仅是候选分类，不授权直接搬移或删除 |

## 文档维护的停止线

- 文档遵循 [全新系统产品基线](product/greenfield-no-legacy-migration.md)，不为恢复旧技术路线增加兼容、迁移或第二事实源。当前产品、领域、安全及资金边界仍需遵守。
- 某项功能的设计准入、源码/CI、实例开放和用户验收是四种不同证据，不得用目录中的设计或历史运行说明冒充当前上线。
- 先调整**原 owner 的现有文档**，其他层引用它，不复写同一规则。跨层同名文档可能分别是产品决定、架构合同和操作指南，需读内容才能决定是否重叠。
- 不批量搬迁 350 份 Markdown，不按文件名前缀断言 fact owner；对高度引用的核心路径先保持稳定。已确认退休的过程材料由 Git 历史追溯，不新建长期 `archive/`。
- 文件名以小写英文及连字符表达主题；目录已代表文档类型，不额外堆叠 Issue 编号、实例编号、日期与开发阶段。旧文件名需要经审查后再逐步调整。

**当前执行顺序**单独参考 [#137](https://github.com/henryxu1008/shuomi-ai-commerce/issues/137)，该排序不替代每个执行 Issue 的 Design Basis、授权与验收范围。
