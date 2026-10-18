# 任务中心：多来源、多能力的原生运行投影

Status: IMPLEMENTATION_READY（Reuse Existing Architecture，非新独立架构）。执行 [#798](https://github.com/qq550723504/task-processor/issues/798) 当前续作；本文件先于生产代码提交。实现及用户验收仍待完成。

## 产品结果与依据

当前企业有权限的运营人员，在任务中心统一查找本人已发起的内容、图片、上架资料、选品、经营诊断和 POD 等现有运行，查看真实阶段与原业务入口。任务中心的读取不再以 Chat planner 可执行为前提。不是启用上述全部能力的承诺：只有原目录明确 `canReadRuns` 且原读取成功的记录才展示。

2026-10-18 用户明确批准先设计再实现，任务不限定采集商品或标题优化，保留数据并注意其它会话。来源是获取事实的方式；任务对象是获授权的商品/业务范围及原版本引用；目标由当前 capability/Agent 表达。不能将 `acquisition` 的实现分支当成全产品来源模型，更不能把其它上下文默认为采集。

Product/Figma Authority：沿 [最终 UI/IA](../product/final-ui-ia-authority.md)，Figma `tg48P46SSXl6TBy9lZwg63` 的任务中心 [427:3005](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=427-3005)，本轮已重新读取设计 context 与截图。复用 #800 的概览、筛选、列表/详情、响应式和静态资源。Figma 示例任务与数字不成为运行数据。

复用 [Chat/BusinessTask](ai-workbench-chat-business-task-v1.md)、[Agent Runtime](product-agent-runtime-contract.md)、[商品内容](product-content-localization-v1.md)、[四种权限模式](enterprise-agent-permission-modes-v1.md)、[内容与 Chat 共存](content-agent-chat-coexistence-v1.md)。本批不撤销共存设计中 storage-only Chat 与 title-only 试用的执行限制，也不重新启用独立标题 Agent。

## 有界交付与明确非目标

复用智能体目录和最近任务 API，在 Console 内存中组织带 owner 引用的条目；这是现有合同的消费者，不是新增公共领域协议或新持久化 Task。新功能只有查询、筛选、详情说明和原业务导航。

不改变 Chat 创建/规划/确认合同，不新增业务能力、通用编排、跨 owner 状态同步、新 IAM/数据库/队列/账本/恢复器；不执行模型、生成、审核、Apply、发布或任何业务写入。用户后续要从 Chat 创建其它目标时，应以对应当前 capability/对象合同单独设计，不能由本读取接线冒充已经实现。

共享实例、配置、凭据、卷、原始任务及会话完全保留。本批只在自有 worktree 开发和隔离验证；运行更新需准确候选和另行授权。图片运行与真实试用仍由原图片会话负责，不写其组件或工作区。

## 现有 owner 与实际接线

| 事实/合同 | 当前实现与注入 | Console 消费 |
| --- | --- | --- |
| BusinessTask 意图与身份 | `internal/aiworkbench`，原 Task HTTP/BFF | #800 的 TaskBrowser/TaskDetail，保留原可用性和动作 |
| 可读能力目录 | `internal/agentconfig/httpapi` 的 market 与 `canReadRuns`；`agent_configuration_module.go` 的 RunsAvailable | 原 `configurationRequest` + `catalogPageSchema`；不以 canUse、启用状态、provider readiness 代替读取权限 |
| 内容/标题/上架资料运行 | `agentconfigpersistence.Recent`，准确开始快照/member、原 Run 与 binding 授权 | 原 `recentRunsSchema`；不增加采集来源过滤，保留 `contextKind`/operationId/productKey/requestKey/runId |
| 选品/经营诊断运行 | 上述原 owner 与其 inspector 对范围/历史重新授权 | 原 `selectionRecentSchema` / `diagnosisRecentSchema`，不虚构商品或平台 |
| 图片/POD 运行 | 原图片 Run 与 `ListImageSets` / `ListArtworkRuns`，当前企业、actor、member 过滤 | 原 `imageSetRecentSchema` / `artworkRecentSchema`，保持原运行与上下文身份 |
| 待确认提案/采用 | 原 Review/Catalog 与 #800 ProposalSession | 保留唯一审核 owner、精确 revision/CAS、UNKNOWN 写锁与单独 Apply |

原 BFF 仍负责 cookie/token、expected user/org、严格路由、响应大小与 no-store。新增消费者只使用这些现成接口。没有新的后端 API、SQL join、schema 或跨库聚合；最近运行查询内部原有 join 仍归原配置/运行 owner。

Console 条目的引用包含 owner 类别、必要 agentId 与真实 id。React key 和选择身份包含 owner/agentId，防止不同来源同名 ID 覆盖。它们只存在浏览器内存，不写库、localStorage、历史 BusinessTask 或 Chat。

运行不是 BusinessTask；不会为它构造 TaskID/ConversationID。与原 Review/BusinessTask 只有准确现成引用才能关联；不能按商品名称/近似时间去重。缺乏关联时保留“运行记录”和“待确认事项”两种明确类型，不推断不同记录就是同一任务。

## 授权、发现与隔离

只有原 Agent 配置 BFF 有实际可信 origin，当前身份具备目录读取和 Agent 使用权限时，才尝试原目录。只有目录的 `canReadRuns=true` 才请求该能力的 recent-runs。目录失败、无可读能力与成功空列表分别表达，不伪造成功空结果。

所有 query key、选择和缓存生命周期包含当前 user/org/roles/permissions。切换企业、退出、权限变化或上下文阻塞时卸载旧消费者、取消请求，不显示旧数据；迟到响应不能覆盖新 scope。每个 recent 请求继续经过原 live authorization 与 member/binding 检查。撤权响应立即隐藏相应来源已有页，不能继续显示缓存中的商品上下文。

源上下文不是可信浏览器快照。展示仅使用原 owner 已授权 DTO。链接只拼接现有工作区及经过既有 schema 验证的引用，导航后仍由原工作区重新授权。未知 context 没有 acquisition fallback；只能前往该原智能体入口，并明确具体上下文尚无直达链接。

## 状态、分页与统计

运行阶段不等于业务采用状态。保留原 phase/status 文案；文本 `human_review_required` 只表示运行已产生建议，不能推断 Review 尚未接受或已经应用，因此不将它计为新的待确认事项，也不计为已完成业务任务。需要用户处理的实际提案仍由 Review 列表给出。文本 `stopped` 表示运行停止，不能推断成功或具体错误原因。

明确有依据的分组：文本 running→执行中、interrupted→已暂停、stopped→异常/停止；图片/POD 的 planning/executing/evaluating/repairing→执行中，awaiting_plan_approval/awaiting_final_approval→待确认，completed→已完成，blocked/failed→异常，cancelled→已停止。图片完成只表示原图片运行完成/素材结果，不能写成已发布或已采用到商品。

建议已生成、取消/停止等无法适用当前五种业务筛选的记录在“全部”保留原阶段。状态不明时保留未知，数量不能当作完整可判断结果。概览沿“已加载”范围，明确统计覆盖已知状态；失败或不可判断使对应概览不可用，不显示全局/今日数量。

目录与每个能力沿原 cursor 独立分页；不将一个 owner 的 cursor 交给另一个。显示已加载记录和“加载更多”，不自动遍历全历史。分页中被原 owner 授权过滤的空页仍可继续；不把空 items 当作没有 next。刷新清除所有分页和选择。原 Chat Task 与 Review 的分页机制保持。

每个来源有独立 loading/error/next 提示。某来源暂不可用不会把其它可读来源假装完整；来源错误重试只重新读取该来源。身份/授权错误先清除旧数据。原 #800 Review/Task 未完成或 UNKNOWN 写意图存在时，禁止切换选择、来源/状态导航、刷新或通过原业务链接逃离该锁。

## 原入口与写入边界

详情展示能力名称、原阶段、时间、真实对象/上下文、运行引用和已知范围。已有准确直达合同的自有商品、源商品、上架资料沿其原入口；其他上下文只能指向该能力的既有工作区，不伪造错误的采集链接。后者明确提示在原任务历史中按运行引用选择。

本批没有新写按钮，不将图片原业务面板嵌入任务中心，从而不复制其写锁、报价、幂等、恢复与权限所有权。用户在原业务入口操作后刷新原读取即可获得新状态。原 Review/BusinessTask 已有写操作仍完全归原组件负责。

## 设计准入与必要验证

Design Basis: Reuse Existing Architecture。完全复用既有目录/运行/审核授权读取与 UI 组织，不改变产品执行语义、公共领域合同、canonical owner、授权、状态机、事务、计费、副作用或恢复边界；无需新的独立高风险架构评审。若实现必须改变上述边界，停止该部分并重新分类，不能在本文件内临时发明后端。

Legacy decision: N/A。本批不消费 RETIRE owner、旧 Task Dashboard，不做 backfill/ID 映射/双读双写/兼容。保留历史数据不等于给历史数据新建任务。

实施按 TDD 验证：storage-only Chat 下多能力任务发现；canReadRuns 而非 canUse；own_product/多来源上下文无采集 fallback；对应严格原 schema；真实 phase/status 映射与不伪造完成/待确认；独立分页/局部失败；当前 scope/撤权/迟到响应；原 Review UNKNOWN 锁不被新增交互绕过。复用原组件、TanStack Query、Zod 与原测试框架，不建 runner 或新验证平台。

开发检查为相关 Vitest、typecheck、定向 lint/build，稳定候选交现有 PR CI 和一次独立最终增量检查。交付原入口、使用步骤、覆盖和未接入项；实现者/CI/AI review 不签发产品验收。本批真实模型、共享运行切换及用户验收均 NOT_RUN。
