# AI 工作台定时任务 V1：计划、触发与原业务任务交接

> **状态：DESIGN_CANDIDATE / NOT_READY**。Design Basis：Independent Architecture。跟踪 Issue [#1244](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1244)。本文件是待独立架构评审的设计候选；**不构成 IMPLEMENTATION_READY、已实现、已部署或真实执行验收**。
>
> 源码调查基线：`main@58111fce176ee0974f8d4893dec0cf02b1e43fe3`（2026-11-11）。正式实施前核对已合入的业务与导航增量。

## 1. 产品结果、边界和用户路径

有权使用当前企业的 AI 工作台的成员，可以将**已经支持的明确商品内容任务**安排为一次预约或有边界的每天／每周计划；关闭浏览器后由后台按计划产生一次独立业务任务。任务中心展示计划、下一次时间、触发记录、原 BusinessTask 与实际 AgentRun/Review 的结果；用户继续沿原人工审核/采用路径处理建议。

第一条可运行验收：选择一件有权访问、来源与 Catalog 版本可核实的已保存商品 → 选择内容本地化的字段及目标语言 → 看到确切时区、预约时间、对象与预算 → 显式保存计划 → 关闭浏览器 → 到点后台复核权限并生成唯一业务任务 → 打开原任务中心找到结果和提案 → 人工处理。

**本批 Must**：精确时区；持久化预约及每日/每周规则；确认过的对象／业务动作／权限／预算边界；同触发不重复执行、不重复扣费；后端重启后可恢复；企业及成员隔离；撤权后不派发新执行；用户可查每次触发结果与真实失败原因；手动 BusinessTask 原合同及 Review/Apply 保持不变。

**本批 Should**：用同一计划管理服务支持任务中心表单及 Chat 的“设为定时任务”；用户可编辑、暂停、恢复、终止；优先复用现有 UI 与 Temporal。

**非范围**：任意 Agent/Prompt/Tool 的自动授权；市场、店铺新取数和连接器；外部上架、采购、付款或自动 Apply；新模型／新报告 owner；邮件／短信等通用通知；第二调度器或通用规则平台；历史数据迁移或 legacy 兼容。Chat 的自然语言日期语义解析仅能生成**候选**，不承担执行授权，属于可延期增强。

批准依据：用户对 #1244 的需求与本次开始实现的明确意图；现有 `ai-workbench-chat-business-task.md`、`ai-workbench-work-scope.md`、`product-content-localization.md`、`product-agent-runtime-contract.md`、`temporal-boundaries.md`、`project-boundaries.md`、`organization-agent-configuration.md`；仓库现行 `AGENTS.md` 覆盖旧 Figma UI/IA 权威描述。Figma 仅供理解业务流程。

## 2. 现状与缺口

- `internal/aiworkbench/service.go` 的 `Confirm` 经当前用户身份、当前提案及 `PreparedTask` 固化 BusinessTask 并立即 `Start`。同一个 proposal 只能确认一次，不能由定时器定期重放。
- `internal/aiworkbench/selection.go` 已支持 `product.content.localize`；工作台会话及执行范围受限于原商品、项目／店铺和当前组织权限。
- `internal/integration/persistence/aiworkbench/schema.sql` 的原 BusinessTask 有 Chat/Message/Proposal 祖先约束。不能为了定时绕过它，或伪造最新消息／伪造提案。
- 现有 `internal/platform/temporal/client.go` 与 Temporal SDK 可复用；当前代码未提供面向用户的 Workbench Schedule owner。
- 请求执行路径依赖当前 HTTP 身份；现有 Agent worker 的无 JWT 复核模式是可借鉴端口，不等于定时任务权限已经接好。

以上说明：**定时计划不是复用一次 Chat Confirm 的重复按钮**，而是新建可验证的、有限授权的持久计划，再由它每次产生精确的一次性执行意图。

## 3. Fact owner、合同、注入和消费者

| 合同/事实 owner | 实施位置（候选） | 消费者/禁止越权 |
| --- | --- | --- |
| 用户计划、修订、时间、状态、授权范围及每个触发身份 | `internal/aiworkbench` 的 `ScheduledPlan` / `ScheduledOccurrence` 及其仓储端口 | 工作台 Schedule HTTP、Chat 的结构化“设为计划”操作；不拥有 AI 任务结果 |
| Workbench 自有计划／触发持久化及登记命令 | `internal/integration/persistence/aiworkbench` 中原 ai_workbench schema 的显式 installer 与仓储 | Workbench Schedule Service；不保存 JWT、原 Owner 密钥或第二份资源账本 |
| 定时触发 | 原 `internal/platform/temporal` client 的有界 adapter + application worker 注入 | 调用 Workbench 后台入口；Temporal 的 Schedule/Workflow 不是执行资格的权威来源 |
| 已确认业务任务与执行 | 原 BusinessTask / AgentRuntime / Review / Catalog / Resource owners | 每次触发只交接一份经授权执行意图，不重新实现 Agent/Review/扣费系统 |
| 当前成员、对象、模型及预算资格 | 原 Identity/WorkbenchContext、Project/Store/Catalog/SRC、AgentConfig/Resource 只读及准入端口 | 首次创建及每次到点重新判定；不复用过期 token 或请求缓存 |

**建议在当前 Workbench owner 扩展一种 `schedule_occurrence` 类型的 server-created 业务意图**。仅后台可信端口能生成；每个发生记录携带已确认计划的精确修订、业务动作、商品身份、版本解析规则、源摘要、预算限制及原可追踪的运行父关联。原手动 Chat Confirm 的 proposal、message、同键与 T1/T2 不变量不变。对新增来源的 BusinessTask 必须使用明确的来源判别与约束，不允许创建无父凭证的任意任务。最终采用**同库单事务**确保 occurrence 与 BusinessTask 的绑定原子性；若原 BusinessTask/Agent Run 跨 owner，则采用原有 durable receipt/claim 协议，禁止伪造跨库原子性。

## 4. 输入、身份、快照和授权

- **计划意图**：plan_id、organization_id、创建者及有效 member_id、工作范围、`ContentTaskKind`、有界 typed 商品选择、冻结的用户目标、来源/版本规则、每次运行参数和资源上限、计划限制、创建时授权确认摘要、schedule revision。
- **每次运行的实际证据**：scheduled_at_utc、固定的 plan revision、最终解析的 Catalog publication/version/digest、模型配置和原预算快照引用、唯一 occurrence_id 和 BusinessTask_id。计划输入与每次实际证据不能混为一个“最新商品”事实。
- 一次性预约初版使用**准确指定的商品及源版本**；每天／每周需要在保存时由用户明确选择“保持固定版本”或“每次按当前授权重新读取这件商品的最新已保存版本”。**禁止自动换商品、扩大数据范围或使用模型推断商品归属**。
- 每轮在创建执行意图前及原资源外部发送准入处分别复核实际当前授权：同企业/原成员仍有权、当前工作范围仍允许、商品及来源可读、Agent 当前允许、模型配置仍在准入范围、额度与时间预算有效。撤销权限或依赖不可用时 fail closed，保存真实阻断原因，不退回缓存授权或系统管理员权限。
- 保存“允许在约定时间内重复执行有限任务”的明确用户授权；**计划创建不是允许无人审核 Apply/Publish**。新执行仍按原 Review/Apply 权限和流程，不能把旧确认当成未来任何动作的无限授权。
- 身份证据仅保存有界稳定业务标识，不保存 access token、refresh token、provider credential 或重用用户 HTTP request；worker 使用原受控、无 JWT 的授权端口。

## 5. 调度、事务与失败恢复

### 5.1 时间与状态

时间规则：`once`（含 UTC 准确时间）、`daily` / `weekly`（明确 IANA 时区及本地时间、周几），起始/结束期限、最多执行次数或预算上限。所有计算出的 occurrence slot 使用 UTC，UI 展示时区和未来具体时间。设计及代码需证明 DST 缺失与重复本地时刻不会产生重复计费触发。

计划最小状态：`PENDING_REGISTRATION`, `ACTIVE`, `PAUSED`, `CLOSED`, `REGISTRATION_UNKNOWN`（名称可在实现时定稿）；**实际业务运行状态从原 AgentRun/Review 读取**。每次发生记录区分 `PENDING`, `CLAIMED`, `DISPATCHED`, `SKIPPED`, `BLOCKED`, `UNKNOWN`，只有已取到原结果的投影才能显示业务完成。编辑产生新 revision；更新中的旧修订触发不能绕过状态检查。

### 5.2 PostgreSQL → Temporal

选择 **Workbench PostgreSQL 为持久计划意图及触发资格的唯一权威来源**；Temporal Schedule（周期）和 StartDelay（一次性，若锁定服务版本满足编辑/暂停语义）只执行触发。建议同库事务写计划 revision + 待登记命令，并由唯一有界的 sync/repair owner 以稳定外部 schedule_id 登记或更新 Temporal；远端响应丢失按精确 ID 与配置版本回读校验。**不可因 DB 提交成功而直接显示 ACTIVE**；待登记/不确定结果真实展示且不可假装计划正常执行。

创建登记、更新、暂停、取消与 Temporal 不存在双写事务：每次在数据库中先序列化用户决定；worker 收到触发后必须重新读取有效 revision 和当前状态。注册完毕后反向持久化确认。若持久化确认丢失，使用原稳定身份核验恢复，禁止创建第二个调度器或自动改写已确认的执行规则。

### 5.3 唯一发生与外部副作用

一个计划的一个预定槽只有一个 occurrence 身份：`(organization_id, plan_id, scheduled_at_utc)`，并记录首次绑定的 revision + frozen payload digest；同槽的重放/重复回调/故障恢复只读同一 occurrence，载荷不同必须冲突，不因编辑换键二次执行。使用 PostgreSQL 唯一索引和原子 claim/CAS 约束并发；用户手动显式新增一次执行需单独准确身份及明确授权，不混用自动预定槽。

发生记录领取、原 BusinessTask 构造和关联必须遵循当前 Workbench 的数据库事务与恢复口径。到 Agent 调度及付费模型的交接失败，按**原 Task/Run/invocation 证据**判定结果：仅证明尚未发生外部发送才可恢复原意图；`UNKNOWN` 不得换 key 重新发送、释放预算或按零费用结算。

Temporal 的 overlap policy 只覆盖它追踪的 Workflow 生命周期。若 Workflow 提交 BusinessTask 后结束，后续 AgentRun 仍可能在运行；**真实不重叠必须通过 Workbench occurrence/原 AgentRun 判定**。首版建议同计划在前次仍运行时跳过并记录，本次不积压；`PENDING_REVIEW` 与 `UNKNOWN` 是否占用在独立评审中定稿。服务恢复按有效 catch-up 时间窗、预定次数与预算边界处理旧触发；过窗保留漏跑事实，不批量生成旧付费调用。

## 6. API / UI 和交付路径

API 候选：`POST /schedules`、`GET /schedules`、`GET /schedules/{id}`、`PATCH /schedules/{id}`、`POST /schedules/{id}/pause`、`POST /schedules/{id}/resume`、`POST /schedules/{id}/close`、`GET /schedules/{id}/occurrences`；实际前缀及路由遵从现有 Workbench HTTP 规范。所有写入均含有界 operation key、expected revision 与服务器校验的 scope；GET 只读当前成员有权的计划/结果，分页游标绑定企业、操作者及筛选。用户不能通过请求载荷指定 actor/org/member 或绕开当前范围。

UI 在“AI 工作台 → 任务中心”提供计划区、创建/编辑表单、明确下次执行时间、当前登记/暂停状态与每次运行详情。Chat 仅复用该创建合同，在用户确认结构化卡片后保存；不增加独立的导航体系，也不依赖 Figma 画板的菜单层级。计划的暂停、终止只禁止**未来**新执行；已经启动的实际业务任务按原取消、人工审核和结果处理合同执行。

**实施次序（同一 Delivery Batch、一个主要 PR）：**
1. 本设计与独立适用评审达到 `IMPLEMENTATION_READY`，然后改正式生产代码；
2. 先完成一次性内容本地化：schema/port/后台授权/一次触发/Task 及结果找回；
3. 扩展周期 Schedule、暂停/恢复/编辑/结束、漏跑与追踪；
4. 接入任务中心和 Chat 创建入口，完成定向测试、最终独立检查及准确 HEAD CI。

## 7. 可验证的不变量

1. 相同预定槽重复通知 2 次，跨 worker 并发和进程重启后仍只有一个 occurrence、一个对应的 BusinessTask 和原预算预留；相同键不同内容报冲突。
2. DB 成功/Temporal 失败、Temporal 成功/响应丢失、暂停与到点竞争、编辑与登记竞争均可用原键读回，计划状态不虚假。
3. 撤销成员或项目/店铺权限、商品变更/缺失、Agent 被禁用、额度不足均无新外部发送；无权限用户无法读取计划、结果或源快照。
4. 有效时区、DST 缺失/重复、重启补执行、离线超窗、到点并发均按明确规则保留单次准确身份；不误补跑过期付费任务。
5. 第一次 AgentRun 未完成时下一槽不得产生并发模型运行；`UNKNOWN` 使用原执行身份查证，不自动付费重试。
6. 实际执行的来源版本/摘要及配置、预算、用量回执可追溯；原任务中心可见且能打开原人工处理路径，派发成功不被误报为业务完成。
7. 手动 Chat Confirm、原 Review/Apply、Catalog、Resource 不回归。不能通过计划选择扩大权限或自动对外刊登。
8. 实施沿现有 installer、测试/CI，必要的真实隔离 PostgreSQL / Temporal 证据与代码级单元回归分别记录。当前全部为 **NOT_RUN**。

## 8. 设计评审待定点 / 当前权限

独立 reviewer 需冻结：一次性调度编辑/取消与 SDK 版本行为；计划/触发 schema 的约束、CAS 与唯一索引；计划授权有效期及每次预算累计的**真实原资源 owner**；周期的 latest/fixed 选择；真实 AgentRun 占用至何状态；跨 Workbench/Agent owner 的交接与 UNKNOWN；最小限度的 server-created BusinessTask 扩展及注入/consumer。

Threat Model 仅覆盖本批新增的后台执行权、跨租户访问、持久来源及原资源副作用；不扩大原 Accepted Risk、不建设第二权限体系。**当前状态 DESIGN_CANDIDATE / NOT_READY；没有独立架构评审批准，不能将本文件作为 Writer 改生产代码的授权。** 不授权真实模型调用、保留实例部署、真实业务数据修改、合并或 Issue 关闭。
