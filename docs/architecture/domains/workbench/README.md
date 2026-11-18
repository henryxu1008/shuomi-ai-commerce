# AI Workbench：会话、任务、计划与报告

> **阅读索引，不新增 Agent/Task/Report 或资源账本。** 权威是[项目事实边界](../../project-boundaries.md)、[目标架构](../../project-target-architecture.md)及以下原合同。Refs [#1351](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1351)。

## 根据用户目标查阅

| 业务入口 | 当前合同/说明 | 事实 Owner |
| --- | --- | --- |
| 硕米 Chat 保存会话、规划和确认单次业务任务 | [Chat + BusinessTask](../../ai-workbench-chat-business-task.md)、[会话单独运行](ai-workbench-conversation-runtime.md)、[完整商品智能体与内容 Chat 共存](chat-content-full-runtime-coexistence-candidate.md) | `internal/aiworkbench`；执行结果归原 Agent/Tool owner |
| 项目中心组织现有任务和资料 | [Project Center](ai-workbench-project-center.md)、[固定工作范围](../../ai-workbench-work-scope.md) | `internal/aiworkbench/projectcenter`，私有项目链接不扩权 |
| 在 Chat 创建个人标题技能、选择已启用的标题技能或执行项目操作 | [硕米 Chat 选择 Agent 技能与项目操作](../../chat-project-actions.md) | Skill 持有技能版本，AI Workbench 持有创作命令、会话与任务，标题运行与计量归原服务，Project Center 持有项目与关系 |
| 在 Chat 创建并选用个人文字技能 | [硕米 Chat 创建并使用个人文字技能](../../chat-general-text-skills.md)，#1429 设计已进入实施准入 | Skill 持有私有技能和准确版本，AI Workbench 持有创作与消息命令 |
| 任务中心查看已有 BusinessTask / AgentRun、提案和审核 | [原生运行投影](task-center-native-run-projection.md)、[任务中心接线](../../../engineering/unified-task-center.md) | Workbench 只作授权投影；原运行、Review/Apply 与 Temporal 保持各自 Owner |
| 新建店铺经营日报定时任务，控制已有计划 | [定时计划合同](../../ai-workbench-scheduled-tasks.md)、[店铺日报](../../scheduled-store-daily-report.md)、[编辑/复制增量](../../scheduled-plan-edit-copy.md)、[用户操作](../../../operations/user-guides/ai-workbench-scheduled-tasks.md) | `internal/aiworkbench` 的计划/occurrence，Temporal 原触发，`internal/reportcenter` 保存日报 |
| 保存、收藏和下载个人报告 | [个人报告合同](../../ai-workbench-reports.md)、[报告交接](../../../engineering/integration/report-center-runtime-handoff.md) | `internal/reportcenter` 的不可变报告/收藏，不是来源事实或审核 |
| Agent 能否调用 Tool、选择模型与预算 | [Commerce Tool Foundation](../../commerce-tool-foundation.md)、[AI Capability](../../ai-capability-platform.md)、[模型价格路由](../../model-route-pricing-management.md) | `internal/commercetool`、`internal/aicapability`、原 Model/Resource/Review owners |

- 内容 Chat 的空结构安装与文本用途准备见[定向准备合同](chat-local-open-preparation.md)；冻结空 scope 成功不代表完整 serving schema 或真实模型就绪。

## 当前优先适用的范围

2026-11-15 起，**新建定时计划只允许店铺经营日报**，日报可以复制新建。商品内容计划**新建、复制和 Chat「设为定时任务」已撤下**；已有商品内容计划仍可查看、编辑、控制并读取原任务、执行历史、审计及未决预留。早期 [定时主合同](../../ai-workbench-scheduled-tasks.md)中的商品内容新建场景是旧阶段产品范围，当前启用以 [#1297](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1297)、[编辑/复制增量](../../scheduled-plan-edit-copy.md)及实际实例配置为准。

- **BusinessTask 不是 AgentRun 或 Temporal Workflow。** 用户任务只关联准确的原运行，不能按名称或时间凭空合并；原执行 owner 负责状态、取消与 UNKNOWN。
- **Chat 不代表已有模型/工具的执行权限。** 保存会话、计划建议、执行批准、人工 Review/Apply、扣费、平台发布属于不同门禁，不能从 UI 可见性推断已经开放。
- **Project 和 Report 不是实时商品或店铺事实。** 私有项目资源链接不授予访问权；报告是某一时点的授权快照，不能替代当前来源、审批或提交事实。
- **计划保存≠已安排真实执行，任务完成≠已采用或发布。** 时区、修订、权限、额度/计量、原键恢复和准确运行结果都由原合同验证。

最新 UI/菜单仅以[当前业务流程决定](../../../product/final-ui-ia-authority.md)为准，Figma 仅供参考。真实能力与产品验收参见 [#298](https://github.com/henryxu1008/shuomi-ai-commerce/issues/298)、[路线图 #137](https://github.com/henryxu1008/shuomi-ai-commerce/issues/137)及具体执行 Issue。原冻结合同保留、不作批量迁移。
