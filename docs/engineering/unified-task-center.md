# 统一任务中心：业务任务与待确认事项（#798）

## 当前产品结果与依据

使用者从 AI 工作台 → 任务中心进入 `/workbench/ai/tasks`，在同一页搜索已加载记录、切换状态、选择左侧任务并在右侧查看详情。标题及商品内容建议归入待确认流程；原 Product Review 决策组件直接嵌入详情，不再增加 AI 工作台二级“标题审核”入口。只有审核服务已配置而 Chat 未启用的应用，也直接读取原审核集合，不需要启用模型。

Design Basis: **Reuse Existing Architecture / IMPLEMENTATION_READY**。用户于 2026-10-18 批准本轮范围，当前执行依据为 [#798](https://github.com/qq550723504/task-processor/issues/798)。复用 [最终 UI/IA](../product/final-ui-ia-authority.md)、[BusinessTask §9/10/14/15](../architecture/ai-workbench-chat-business-task-v1.md)、[原审核合同](product-title-review-ui.md) 及 [独立入口装配](../architecture/issue-36-local-trial-runtime.md)。没有新增权限、事实 owner、状态机、schema、计费或恢复协议。

## Figma 与实现映射

本轮实读 Figma file `tg48P46SSXl6TBy9lZwg63` / [任务中心 427:3005](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=427-3005) 的 design context 和截图。复用已存在的 Console Shell、Card/Button、CSS 语义变量及 task-center 双栏样式。

| Figma | 当前消费者与范围 |
| --- | --- |
| 560:380/386/392 状态概览 | 待你处理、执行中、已完成；只统计已加载且可投影的记录，失败/状态不可读显示未知。接口无完成时间，因此不称“今日完成”。 |
| 560:398 工作范围 | 明确本人业务任务和原审核授权范围；没有实际店铺关联/筛选合同，不画可用的店铺下拉。 |
| 560:404–420 状态筛选 | 原任务状态路径保留；暂停单独筛选原 PAUSED 投影，不增加暂停命令。 |
| 560:375 搜索 | 搜索已加载任务标题、商品和摘要；没有服务端全量搜索承诺。 |
| 560:425 左列表 / 564:363 右详情 | TaskBrowser 投影业务任务与独立待确认事项；选择行只读，任务详情复用 TaskDetail，审核复用 ProposalSession。 |
| 564:372/376 等待事项与建议 | 原 Review 的对比、依据、冻结权限、接受/编辑/拒绝和单独 Apply。 |
| 560:377 发起任务 | 仅在当前 Chat 准入可用时连接原新会话入口。 |

原 Figma 的任务名、数量、步骤百分比、预计耗时和经营能力均为示例，不是业务事实。390px 等窄屏沿现有响应式单列；明暗主题沿现有 tokens。本批不交付经营诊断、潜力商品分析或运营计划。

## 现有 owner 与交互不变量

- BusinessTask 持有确认意图；Agent 持有运行/恢复；Product Review 持有提案/决策/回执；Catalog 唯一写入商品事实。界面没有第二状态机或本地持久账本。
- 业务任务作为主记录。准确 reviewId 已对应到当前可见任务时，不重复列出同一提案；未加载分页不推断关联或全局待办数量。直接提案链接仍由原 Review 单独授权。
- 提案与 Task 分页继续分别使用现有游标；部分来源失败不伪装成空集合或 0。
- 精确修订、人工编辑撤销旧批准、接受与 Apply 分开、只读模式、权限拒绝后重新读取、CAS/UNKNOWN 和原幂等键完全复用原审核实现。
- 嵌入审核会话与任务详情的读取生命周期分开，任务刷新/读取失败不能丢失待核实写意图。提交中或结果未知时，新页面的行选择、搜索、分页、刷新及状态切换锁定；不会自动重放写操作。
- Task 启动、继续和提交审核同样锁定列表。结果未知后，只能显式“核实原操作结果”，沿现有 Task API 重放原幂等键及冻结输入；Resume 的修订变化不替换原反馈/修订，也不因一次 GET 就认定未提交。复用原 scope 内的 sessionStorage 操作键，不新增恢复服务或事实 owner。
- 用户/企业/角色/权限变化按原 scope 重挂载并清理详情；早期 URL 选择不会在切回企业后自动复活。迟到响应不能流入新 scope。
- 本地资料准备历史继续由原 Listing 只读入口读取，不被伪装成 BusinessTask。旧 Task-first Dashboard 不恢复。

## 入口与验证

正常应用沿既有启动/登录流程，使用 `/workbench/ai/tasks` 或 `/workbench/ai/tasks/pending`。当前 `/workbench/ai/tasks/pending/other?proposal_id=...` 直接链接仍进入同一待确认页面，面包屑与侧栏归任务中心；URL 标识不是授权。

首页与待确认页的 `proposal_id` 链接在刷新后恢复原提案定位，“我的报告”也只从这三条同源路径提取准确 ID，继续由原来源服务重新授权。独立提案的已应用/已拒绝结果通过详情链接回看；已完成业务任务列表仍由 BusinessTask 投影，未新增全量终态提案集合。

开发核对复用现有 Vitest/Next 构建及原 Product Review 隔离浏览器 fixture，不新建验收平台。定向测试覆盖无 Chat 的直接审核、任务关联去重、搜索、未知统计、原审核写合同、scope/撤权和旧入口归属。最终准确 HEAD、CI 与独立复核证据维护在主要 PR。

共享保留实例安装、真实身份/客户数据/模型/平台操作和用户产品验收不在本批授权内，保持 NOT_RUN；原运行 owner 与数据保留范围不变。
