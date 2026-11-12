# AI 工作台定时任务 V1：计划、触发与原业务任务交接

> **状态：IMPLEMENTATION_READY（架构准入）/ IMPLEMENTATION_IN_PROGRESS（实现待完成）**。Design Basis：Independent Architecture。跟踪 Issue [#1244](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1244)。用户已明确裁定：准确 HEAD 的 Codex 独立评审完成、无新 findings 且 PR 获 Codex 👍 即视为通过。本文件在 HEAD `b1ff3377bbfae782fcafcebb665fbe250afe3294` 的第二轮评审无新发现，[PR #1245](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1245) 收到 bot 👍，准确 CI [42994325637](https://github.com/henryxu1008/shuomi-ai-commerce/actions/runs/42994325637) SUCCESS；依用户本次明确决定，架构 **IMPLEMENTATION_READY**。这仅允许在原边界内开始实现，**不代表生产代码已实现、已部署或真实业务验收通过**。旧四条 BLOCKER 原始评论继续可追溯，设计修正在 §3.2，运行验证仍是实施义务。
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

### 3.1 当前 BusinessTask 数据模型的准入级问题与最小修正

固定版本 `internal/integration/persistence/aiworkbench/schema.sql` 中，`business_tasks.conversation_id`、`source_message_id`、`proposal_id` 全是 `NOT NULL`，并有三条相关复合外键；`task_action_receipts` 又通过任务 `owner_user_id` 绑定原 actor。这不是可通过新增一个“计划表”就自然复用的入口。工作台当前 `ExecutionPort.Prepare(ctx, ExecutionProposal, key)` 依赖真实 Chat 提案内容，不能直接由无会话的 Cron 调用。

候选方案（待独立审查与落实具体列级 DDL）：
1. **先新增 `scheduled_plans` / `scheduled_occurrences`**，以不可由浏览器伪造的 `(organization_id, owner_user_id, plan_id, occurrence_id)` 保留已确认计划及准确当次执行祖先；发生记录对其计划有同租户/创建人复合 FK，且唯一约束按 `(organization_id, plan_id, scheduled_at_utc)`。计划中的业务动作、目标和允许的费用上界不可由 Temporal payload 改写。
2. **在原 `business_tasks` 中加入显式互斥来源类型** `CHAT_PROPOSAL | SCHEDULE_OCCURRENCE`，并由 schema CHECK 约束：Chat 来源必须拥有原三个非空 Chat/Message/Proposal 祖先并延续现有复合 FK；Schedule 来源只能带有效 `scheduled_occurrence_id` 和其同企业/actor 的复合 FK，不带伪造的 Chat 祖先。为此将旧三列按受控来源允许 nullable（不移除原 Chat 的检查和外键）；不对旧行做迁移/回填，已有行仍由原默认/严格路径写入。具体缺省值、旧行为保持及约束组合须在正式评审中确定。
3. 服务端新增窄 `PrepareScheduledOccurrence` 端口：只接可信已领取 occurrence 的冻结 typed 业务输入、实时授权及原来源/配置/预算引用，调用**原内容 Agent 的准入与 Start**。它不能接受浏览器自带的“已经审批”字段，也不能复用原 `Confirm` 绕过 `GetProposal`。与原提案对应的计划来源必须是显式确认的计划事实，而不是将计划保存等同于已运行的提案。
4. `business_tasks` 中此来源的 `execution_request_key` 使用 occurrence 派生的稳定 UUID，用户重试和 Worker 重启不重新生成；原 Chat task action receipt 及投影只做最小分支扩展，不伪造 Conversation/Proposal 数据。读取需按来源走严格合法关联；不能直接把原 Chat 详情用 null 字段“凑出来”。
5. 只在**同一个现有 Workbench PostgreSQL**中，以一个事务完成 occurrence 领取/校验、合法 BusinessTask 新建和准确关联；其它 RunDB/ProductDB 的副作用继续按原 owner 的持久回执、权限与 UNKNOWN 协议。若实际数据库装配不共享此事务，则必须先重新评估而非通过多阶段写入假称原子提交。

**本节是设计候选而非已实施 schema**。独立审查应特别复核原复合外键、来源 CHECK、当前 `BusinessTask` DTO、原手动 Confirm 与 TaskProjection/TaskActionReceipt 的相容性、只读权限以及空库 installer。任何不能保持现有手动任务不变量的变更都是本批实施阻塞，不能以只做页面或调度脚本代替。

## 3.2 四项独立评审 BLOCKER 的冻结处置（#1245，2026-11-11）

本节是对准确提交 `bba71503` 的独立评审提出的四项实施级 BLOCKER 的最小设计修正；不以普通 Code Review 代替下一轮明确的架构准入结论。下面的硬约束高于前文仍在使用“建议/待定”的候选措辞。实现如无法满足，必须停止对应业务路径，而非退回只做调度或页面。

### B1：定时来源一律只允许人工采用

**冻结：首版只允许用户确认的 `default` 或 `read_only` 业务执行模式；对于 `auto_approve`、`full_access`，即使原权限准许，也不创建/恢复定时任务。** 本批不授权无人值守 Apply。创建计划、恢复计划、每次 Prepare、每次 Start/Resume 都检查其冻结模式与当前生效模式；如模式变化到不兼容，记录 BLOCKED，不自行降级、换模式或重发模型。业务请求冻结 `ManualReviewOnly=true`，原 Request/Run 序列化与指纹绑定该位，且 `checkManualReviewMode` 必须接入本来源首次运行和恢复；不允许客户端传 `false` 覆盖。即使运行产生可采用候选，完成阶段也必须拒绝所有定时来源自动 `policy.Adopt`，继续等待原人工 Review。用户后来显式打开详情进行人工 Apply，仍走原当前授权。原手动 Chat Confirm 的模式矩阵与自动采用行为不改。验收：default/read_only 可按原能力保存建议，auto_approve/full_access 四组合均没有定时发送或策略采用；模式在准备后改变时无新发送；重启后的 Run/Resume 不会剥离该位。

依据：`product_agent_execution_mode.go` 的 `finishTitleCandidate` 会在 auto_approve/full_access 进入 `policy.Adopt`，不能仅依赖 Workbench `startTask` 的原路径默认为人工。依赖原 `agent.Request.ManualReviewOnly` 与 `checkManualReviewMode`，不新建审批引擎。

### B2：真正的后台无 JWT 入口与可信授权

**禁止定时 worker 调用 HTTP-only `workbenchExecution.Start` / `startTask` 并注入伪造的 HTTP context。** 该路径当前通过 `freshIdentity` 需要来自 HTTP 的身份、capability 与 bearer token，不能由定时任务直接调用。

候选实际接线合同冻结为：

`Temporal activity (only schedule ID, slot, revision) → aiworkbench ScheduledExecutionService.Dispatch(ctx, trustedOccurrenceRef) → Workbench PG exact occurrence/plan owner → trusted ExecutionIdentity{OrganizationID, ActorID, MemberID} → productAgentOriginAuthorization(ctx, identity) + 当前对象/来源/工作范围/Agent 配置权限端口 → 原 bounded AgentRequest 准备/Claim/Start/Resume → 原 Review/Resource owner`。

- `Dispatch` 是 app 通过**本地服务端注入**给 worker 的窄端口，不暴露为可匿名 HTTP endpoint；仅接受 PG 中已领取且与调用源一致的 occurrence 句柄，worker 传入的 organization/actor/mode/资源不构成授权。
- 实际身份由 Workbench PostgreSQL 计划和 occurrence 的有界不可变记录铸造，成员绑定不得从 Chat 缓存/过期 session/JWT/Temporal payload 派生。复用 `productAgentOriginAuthorization` 的服务身份令牌读取**准确企业授权及 member ID**、用户 active 状态与 `workbench.agent.use` 和商品写入权限；项目/店铺/商品/来源由原各领域端口按本次具体目的重新授权。
- 将目前 `startTask` 中的 HTTP 身份获取与**原业务准备/原 run-store Claim/Start** 拆成窄的、显式接收已授权 `ExecutionIdentity` 的 application 层内核；手动 HTTP 入口提供原 `freshIdentity`，后台入口使用上面的 current service authorization。双方调用同一个受治理的模型、资源、配置和 Review 路径，不能复制一个不会复核原回执的独立 runner。
- 在 Prepare、Start 及实际 Model/Tool 发送前使用既有对应 owner 的当前授权；若发送前权限变更，禁止新调用但保留已发送原键的 UNKNOWN/用量核实。服务 IAM 不可用一律 UNAVAILABLE，不因是预定任务而提升权限。任何 Run 恢复也必须重新校验计划当前状态及本次允许的身份、来源和模式；已在途调用不假装撤回。

### B3：跨时间槽唯一占用、释放及 UNKNOWN

**计划行有且仅有一个 `active_occurrence_id`（nullable），以及配套 CAS/修订，作为跨时间槽的持久占用。** 同一 Workbench PostgreSQL 事务必须完成计划行 `SELECT ... FOR UPDATE` 或条件更新、时间槽唯一检查、新 occurrence 创建、`active_occurrence_id` 的原子绑定及合法 BusinessTask 创建；持锁期间不调用 IAM、ProductDB、RunDB 或模型。非 ACTIVE 的计划或仍有 active occurrence 时，本槽记 SKIPPED（保留 slot 记录，不能偷偷重试为新 run）；不能因为 RunDB 查询失败、投影 ERROR 或 worker 重启就清空占用。

占用自 `CLAIMED`（即使还无 Run）开始，直到已核实**业务 AgentRun 不会再产生运行/费用副作用**才可释放：
- 正在准备/未证实派发、运行中、暂停/Interrupted 可 Resume、任何 `UNKNOWN`（包括 response lost、超时、usage/model/tool 未明）一律保持占用。
- `HumanReviewRequired` 且原 Run 已明确停止发模型/Tool、原生成与用量结算已明确不再 UNKNOWN 时，**可释放执行占用**，即使 Review 仍等待用户决定；审核继续由原 Review owner 负责，下一周期不得自动 Apply。
- `Stopped` 只有在准确原 Run/request/usage/receipt 证明无需再发外部请求、且预算已结算/受原 owner 持有时才能释放；`TaskProjection.State=ERROR` **不是**释放证明。
- 恢复释放必须由单一 Workbench `ReconcileOccurrence` 入口执行：先按精确 occurrence→BusinessTask→AgentRun/invocation/资源回执只读核验，构造证明；再对同一 plan row + `active_occurrence_id` + 原 revision 用 CAS 清空并记录原证明摘要/决策。无法获取确切身份/凭据或读依赖失败保持占用和 `UNKNOWN`，必须允许可见的人工恢复/诊断，不自动换键重发。
- 两个不同 slot 并发、暂停与到点竞态及多 worker 恢复，均在同一个 plan row 锁序列化；数据库唯一键保证同槽一个 occurrence，外部发送继续受原 AgentRun/invocation idempotency 保护。修订改变不释放已开始的 occurrence；已占用运行可完整读回。

### B4：费用上限不依赖未经证实的累计扣费接口

**V1 收窄为强制有限执行次数 `max_occurrences` 和经原 Agent/Commercial 许可的单次最坏费用上界 `per_run_upper_bound`，且必须同时保存 `max_occurrences * per_run_upper_bound` 作为用户确认的总理论上界。** 不提供“累计实际花费达到 XX 自动停”的承诺，不实现第二套计费账本。最多运行次数包括已认领并可能发送的 UNKNOWN，不能因为执行失败/HTTP 响应丢失而返还一次免费名额。最大次数和单次上界必须受现有限制并安全计算（定点数或严格整数，不能浮点溢出）；无法计算保守合法上界的模型或配置不得用于定时计划，显示真实限制，不捏造金额硬上限。

每次正常执行沿原 Resource/Commercial owner 做准确配置、报价、预留、预算、实际 usage 与原调用回执；计划只持不可变 `per_run_upper_bound` 与配额次数、已占用次数和原 invocation refs 的窄投影，不拥有扣费结果。周期规则必须有明确截止日期或最大槽次数；如原调用只能估算不能证明硬上界，则 V1 不承诺跨周期金额上限而改为仅显示**参考估算+次数硬限制**，经用户单独明确确认后方能开放该类模型；未确认则 fail closed。UNKNOWN 保留原预留并占用计数与运行锁，不按零费用处理。

## 3.3 明确的实施门禁

上述四项均按本轮 reviewer 的 `BLOCKER` 分类。正式 Writer 还必须给出：
- 同一个 Workbench 数据库/当前 installer 中的 source CHECK、复合 FK、唯一 slot + plan 占用 CAS 草案及测试；
- 独立的后台 application 注入路径、原内容运行执行身份与最近发送门禁；
- 模式矩阵及定时 `ManualReviewOnly` 防自动采用的运行验证；
- 按原 Run/资源真实 receipt 核验的释放条件及 UNKNOWN 恢复；
- 单次预算上界来源与有限次数的真实配置校验。

架构评审需明确上面已冻结的选择是否覆盖现有 Must。**架构已按用户批准的 👍 规则达到 `IMPLEMENTATION_READY`；可以在本 Delivery Batch 开始正式实现。** 所有实现期风险仍需实证验证，不能将 CI 或设计准入冒充真实执行成功。

## 4. 输入、身份、快照和授权

- **计划意图**：plan_id、organization_id、创建者及有效 member_id、工作范围、`ContentTaskKind`、有界 typed 商品选择、冻结的用户目标、来源/版本规则、每次运行参数和资源上限、计划限制、创建时授权确认摘要、schedule revision。
- **每次运行的实际证据**：scheduled_at_utc、固定的 plan revision、最终解析的 Catalog publication/version/digest、模型配置和原预算快照引用、唯一 occurrence_id 和 BusinessTask_id。计划输入与每次实际证据不能混为一个“最新商品”事实。
- 一次性预约初版使用**准确指定的商品及源版本**；每天／每周需要在保存时由用户明确选择“保持固定版本”或“每次按当前授权重新读取这件商品的最新已保存版本”；两种规则均受 §3.2 的有限次数和当前授权限制。**禁止自动换商品、扩大数据范围或使用模型推断商品归属**。
- 每轮在创建执行意图前及原资源外部发送准入处分别复核实际当前授权：同企业/原成员仍有权、当前工作范围仍允许、商品及来源可读、Agent 当前允许、模型配置仍在准入范围、额度与时间预算有效。撤销权限或依赖不可用时 fail closed，保存真实阻断原因，不退回缓存授权或系统管理员权限。
- 保存“允许在约定时间内重复执行有限任务”的明确用户授权；**计划创建不是允许无人审核 Apply/Publish**。新执行仍按原 Review/Apply 权限和流程，不能把旧确认当成未来任何动作的无限授权。
- 身份证据仅保存有界稳定业务标识，不保存 access token、refresh token、provider credential 或重用用户 HTTP request；worker 使用原受控、无 JWT 的授权端口。

## 5. 调度、事务与失败恢复

### 5.1 时间与状态

时间规则：`once`（含 UTC 准确时间）、`daily` / `weekly`（明确 IANA 时区及本地时间、周几），起始/结束期限以及**必须配置的最大执行次数**；金额型累计停止条件不属于 V1。所有计算出的 occurrence slot 使用 UTC，UI 展示时区和未来具体时间。设计及代码需证明 DST 缺失与重复本地时刻不会产生重复计费触发。

计划最小状态：`PENDING_REGISTRATION`, `ACTIVE`, `PAUSED`, `CLOSED`, `REGISTRATION_UNKNOWN`（名称可在实现时定稿）；**实际业务运行状态从原 AgentRun/Review 读取**。每次发生记录区分 `PENDING`, `CLAIMED`, `DISPATCHED`, `SKIPPED`, `BLOCKED`, `UNKNOWN`，只有已取到原结果的投影才能显示业务完成。编辑产生新 revision；更新中的旧修订触发不能绕过状态检查。

### 5.2 PostgreSQL → Temporal

选择 **Workbench PostgreSQL 为持久计划意图及触发资格的唯一权威来源**；Temporal Schedule（周期）和 StartDelay（一次性，若锁定服务版本满足编辑/暂停语义）只执行触发。建议同库事务写计划 revision + 待登记命令，并由唯一有界的 sync/repair owner 以稳定外部 schedule_id 登记或更新 Temporal；远端响应丢失按精确 ID 与配置版本回读校验。**不可因 DB 提交成功而直接显示 ACTIVE**；待登记/不确定结果真实展示且不可假装计划正常执行。

创建登记、更新、暂停、取消与 Temporal 不存在双写事务：每次在数据库中先序列化用户决定；worker 收到触发后必须重新读取有效 revision 和当前状态。注册完毕后反向持久化确认。若持久化确认丢失，使用原稳定身份核验恢复，禁止创建第二个调度器或自动改写已确认的执行规则。

### 5.3 唯一发生与外部副作用

一个计划的一个预定槽只有一个 occurrence 身份：`(organization_id, plan_id, scheduled_at_utc)`，并记录首次绑定的 revision + frozen payload digest；同槽的重放/重复回调/故障恢复只读同一 occurrence，载荷不同必须冲突，不因编辑换键二次执行。使用 PostgreSQL 唯一索引和原子 claim/CAS 约束并发；用户手动显式新增一次执行需单独准确身份及明确授权，不混用自动预定槽。

发生记录领取、原 BusinessTask 构造和关联必须遵循当前 Workbench 的数据库事务与恢复口径。到 Agent 调度及付费模型的交接失败，按**原 Task/Run/invocation 证据**判定结果：仅证明尚未发生外部发送才可恢复原意图；`UNKNOWN` 不得换 key 重新发送、释放预算或按零费用结算。

Temporal 的 overlap policy 只覆盖它追踪的 Workflow 生命周期。若 Workflow 提交 BusinessTask 后结束，后续 AgentRun 仍可能在运行；**真实不重叠必须通过 Workbench occurrence/原 AgentRun 判定**。按 §3.2 B3 的计划行 CAS 锁约束，同计划前次实际执行仍占用时跳过本槽并记录，不积压；已确认不再发生新外部副作用且原用量可核实的 PENDING_REVIEW 可释放，所有 UNKNOWN 始终占用至原回执核实。服务恢复按有效 catch-up 时间窗、预定次数与预算边界处理旧触发；过窗保留漏跑事实，不批量生成旧付费调用。

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
8. 实施沿现有 installer、测试/CI，真实隔离 PostgreSQL / Temporal 证据与代码级单元回归分别记录；本批具体记录见 §9。真实 provider、部署与用户产品验收保持 **NOT_RUN**。

## 8. 设计评审待定点 / 当前权限

独立 reviewer 需冻结：一次性调度编辑/取消与 SDK 版本行为；计划/触发 schema 的约束、CAS 与唯一索引；计划授权有效期及有限次数/单次最坏费用界的原资源事实依据；周期的 latest/fixed 选择；§3.2 B3 的原回执释放证明可执行性；跨 Workbench/Agent owner 的交接与 UNKNOWN；最小限度的 server-created BusinessTask 扩展及注入/consumer。

Threat Model 仅覆盖本批新增的后台执行权、跨租户访问、持久来源及原资源副作用；不扩大原 Accepted Risk、不建设第二权限体系。**当前架构状态 IMPLEMENTATION_READY（按用户本次确定的 👍 准入规则）；实现候选及证据见 §9，产品验收未完成。** 不授权真实模型调用、保留实例部署、真实业务数据修改、合并或 Issue 关闭。

## 9. 当前实现映射与验证边界（2026-11-12）

- Workbench 单一事实 owner：`internal/integration/persistence/aiworkbench/scheduled_*`；计划、控制命令、发生记录及显式 `SCHEDULE_OCCURRENCE` BusinessTask 来源在原库事务内关联，双向组合 FK 防止同任务绑定另一个 occurrence。无伪造 Chat 祖先或第二账本。
- Temporal adapter：`internal/app/runtime/workbenchschedules`，一次和周期统一使用 SDK Schedules；typed `TemporalScheduledStartTime` 和 `TemporalScheduledById` 绑定名义时刻，精确稳定 ID、revision 和意图摘要回读。使用 ALLOW_ALL 让每个名义回调交给 Workbench 作真实占用判断并保留跳过事实，不用 Temporal workflow 结束推断原 Run 结束。
- 执行内核：`internal/app/httpapi/ai_workbench_scheduled_execution.go` 只接受进程内后台 capability，从持久计划取身份，经当前组织状态、原 service IAM、Scope/Collection/Source、配置与发送前 gate；调用与手动任务相同的 Prepare/Start、Agent、invocation、积分及 Review。`read_only` 只对合法定时来源开放原人工建议路径。
- B3 释放：精确 task/request/run 绑定、无 pending/UNKNOWN、原 invocation 已知计量及原积分服务同键结算成功后才签发释放摘要。Run/计量未知不会清占用或返还次数。恢复扫描使用有界游标，不用释放未决任务换取扫描前进。
- B4 上限使用原 `OfficialModelTariff.ReserveTextMicropoints` × 原 `ModelCalls`，固定 `ai-micropoint-v1` 单位并检查总量溢出；UI 以 AI 积分展示，不混用参考 USD/CNY 的 `CostMicros`。
- API、BFF 与 UI：`ai_workbench_scheduled_routes.go`、`lib/contracts/ai-workbench.ts`、`schedule-manager.tsx` / `schedule-form.tsx`。创建及控制保存原操作键；503/超时/响应丢失不视为未提交。修改只允许时间规则，保留原 scope、来源、配置、模式、次数与预算。
- 真 PostgreSQL 的原子失败回滚、来源约束、CAS、未决公平扫描、提交后最终读回 503、两种来源和 Chat 项目范围、撤权/停用及 UNKNOWN 测试已执行；真 Temporal dev server 已验证登记、暂停/恢复、准确 nominal slot 和旧 revision 不覆盖。模型 transport 为合成隔离 fixture，无真实 provider 调用。准确 HEAD、最终 CI 和独立增量复核记录在 PR。

操作、显式 opt-in 配置及未开放能力见[定时任务运行说明](../operations/ai-workbench-scheduled-tasks.md)。本节不改变冻结架构准入，也不签发用户验收或实例部署结论。
