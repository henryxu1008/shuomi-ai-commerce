# 项目文档

这里仅维护当前系统的产品依据、领域合同、开发入口和操作说明。任务范围、实现进度、准确版本和验证结果在对应 GitHub Issue / PR 中维护。

| 你要做什么 | 阅读入口 |
| --- | --- |
| 确认产品目标、页面和交互 | [产品依据](product/README.md)，先读 [UI / IA Authority](product/final-ui-ia-authority.md) |
| 确认事实 owner、权限、事务和恢复 | [架构合同](architecture/README.md)，先读 [系统边界](architecture/project-boundaries.md) |
| 找源码、命令、配置与开发方式 | [开发入口](development/README.md) |
| 安装、配置或使用功能 | [操作说明](operations/README.md) |
| 接单、评审或核对消费者接线 | [工程规则与接线](engineering/README.md) |

开发按[全新系统产品基线](product/greenfield-no-legacy-migration.md)进行：全新安装、空业务数据和当前模型。已完成的退休工程不再是后续任务的前置；当前产品、安全、权限、事务、资源和恢复要求继续适用。

文档描述的设计准入、源码实现、实例启用和用户验收是不同状态。具体功能是否开放，以对应合同与执行 Issue 为准。

维护时先更新现有 owner 的文档，避免同一规则多处复制。已被替代的方案和已完成的过程材料从当前树删除；追溯使用 Git，不建立新的历史目录或默认阅读入口。
