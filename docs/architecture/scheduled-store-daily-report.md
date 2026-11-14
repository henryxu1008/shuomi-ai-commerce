# 定时生成单店经营日报：无模型执行与幂等保存

状态：**IMPLEMENTATION_READY（独立架构准入）/ DESIGN_ONLY（未实现）**。Design Basis：Independent Architecture。执行 Issue [#1279](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1279)。本轮只交付设计与独立评审，不写生产代码、不执行 schema、不更改原 35444、不调用模型、不发布商品、不处理或释放原预留。准入通过不扩大用户本轮授权，正式实施仍待后续任务。

## 1. 产品决定、范围与验收结果

2026-11-14 用户先确认“按业务类型组织定时任务，优先单店经营日报，保留现有计划”的最小方案，再明确要求独立评审后台权限、无模型约束、跨库保存和幂等恢复。Figma 仅辅助理解流程，不作为布局/菜单/视觉权威；当前 AGENTS.md 的产品先行、Delivery Batch、架构准入及评审停止规则适用。

用户路径：任务中心 → 定时任务 → 新建 → 店铺经营报告 → 当前权限内的一家店铺 → 一次或每天及有限次数 → 核对并明确授权“到点汇总已有数据，保存到我的报告” → 后台触发 → 计划记录查看结果 → 我的报告刷新重开同一份日报。现有商品内容计划仍可查、控制和执行，四种文本字段继续作为该类型的参数。

Must：单店、上一自然日日报；精确时区/名义执行槽；只读已有经营数据；缺项、零值及来源时间真实；无模型、无 AI 预留/结算；当前企业/用户/准确成员及店铺权限；同槽并发与重试只保存一次；跨库结果丢失可按原次恢复；现有计划、历史和未知占用保持。Should：复用计划时间规则、Temporal、报告查看/收藏/下载及已有 UI 组件。Out of Scope：周报、多店、新平台采集、AI分析、通知平台、发布/采购、自动填数据、新调度器/通用 Saga/第二账本/专项验收平台。

Threat Model：不信任浏览器传入的 org/member/role、计划成功或后台证明；不信任 Temporal payload 中的身份/业务输入；防跨租户、重建成员身份复用、撤权后新生成、重试覆盖/重复日报及不完整数据伪装完整。平台实时取数、数据商业真实性/人工录入质量不由本期证明；不同 owner 的读取不是全局 ACID 快照，报告必须展示各来源自己的时间/范围/缺项。没有新增安全风险豁免。

依据：[当前领域边界](project-boundaries.md)、[定时内容任务](ai-workbench-scheduled-tasks.md)、[个人报告](ai-workbench-reports.md)、[店铺经营诊断](shop-operations-diagnosis-agent.md)、[当前产品基线](../product/greenfield-no-legacy-migration.md)。个人报告合同只在本文件明确的新类型、后台保存及历史访问处扩展；原类型行为不变。旧合同中的 Figma UI 权威引用由当前用户决定取代。

## 2. 当前能力和真正缺口

调查基线 `main@e33501300a33495075a231710f00c9bd3a653d35`；相关定时/经营/报告后端代码与现有 35444 后端版本一致。原 35444 的只读配置调查显示已有 schedules、经营驾驶舱、报告中心，观察模块为 preparationOnly；诊断模型 policy 为零、正式 Supply 未配置。配置存在不是本轮运行/数据验收。本轮不再次访问或变更该实例。

- `aiworkbench.ScheduledPlanDraft.Selection.Valid()`、scheduled_plans 的 kind CHECK、正数 AI 预算及 `prepareScheduledExecution` 都强绑定商品内容。Dispatch/Reconcile 和 availability 又要求 Product Agent。因此增加下拉菜单不足，不能把日报伪装商品/AgentRun。
- `operationscockpit.Repository.Snapshot` 已有单库 repeatable-read、原金额/利润/覆盖计算及权限复核；`shopdiagnosis.NativeReader` 组合经营事实和保存观察，但其现有 Boundary 是 HTTP 身份绑定。报告只复用确定性数据和计算，不调用诊断模型，也不复制计算公式。
- `reportcenter` 只支持 TITLE_REVIEW / SHEIN_RECORD / SHEIN_TARGET_RECORD，现有授权依赖 HTTP Bearer，Snapshot 校验要求非空 ProductKey。它已有独立 Report DB、不可变文档和 command receipt。后台不能注入伪造 JWT 或用占位商品过校验。
- 原调度库和报告库独立；不存在跨库 FK 或共同事务。一次报告结果必须先在 Report DB 提交，再关联 occurrence，不可声称原子双写。

## 3. Owner、合同和实际接线

| 事实/合同 | 唯一 owner、候选实现 | 注入/消费者 |
| --- | --- | --- |
| 计划、控制修订、时间槽、次数、占用和结果引用 | `internal/aiworkbench`；原 `persistence/aiworkbench` | 原 Schedule HTTP / manager；原 Temporal worker |
| 时间触发 | `runtime/workbenchschedules` | callback 只给 plan ID / 名义槽；从 PG 重读计划，不接受 payload 提供身份/店铺 |
| 当前成员与企业状态 | 原 Zitadel exact service authorization、active user、organization status + Casbin | application 注入的窄 ReportOriginAuthorization；不是新增 IAM |
| 店铺授权、经营财务、已有观察 | 原 `storecenter` member-scoped repository、`operationscockpit` Snapshot、观察只读端口 | 定时日报只读消费者；业务事实不另存一套可编辑 owner |
| 报告快照、原保存命令和确定结果 | `reportcenter` + 原 `app/reportcenter` / persistence | 窄 `CaptureScheduledDaily` / `ReadScheduledCaptureReceipt`；现有报告页面 |

正式调用链：现有 ScheduleBackend → Workbench kind 分派 → `DispatchStoreDailyReport(planID, slot)` → PG 已认领 occurrence → 当前可信 origin → 原确定性经营 reader → Report owner 捕获/保存 → 精确 receipt → Workbench `CompleteScheduledReport` → 原计划历史链接及报告页。

这些是拟补窄端口，不是声称已有实现。旧内容分支继续调用原 DispatchScheduled/ReconcileScheduled；不迁移其代码到新的通用执行框架。application 负责装配端口，不直接读其它 owner 的业务表。

## 4. 无模型的 typed 计划与数据约束

新增明确 kind `store.operating-daily-report`。输入仅精确 store UUID、日报格式版本、periodPolicy=`previous_natural_day`、businessZone=`Asia/Shanghai`、一次/每天的原 ScheduleRule、有限最大次数和用户确认。企业、actor、member 由当前身份绑定；无自由 prompt、任意 tool、URL、SQL、商品、模型/凭据、预算或 publish 参数。未知字段拒绝。

首版报告时间规则只接受 `Asia/Shanghai`，界面明确北京时间。统计日由 **nominal slot** 转 UTC+8 后取前一个自然日；不是取重试/机器当前时间。跨午夜重试仍是同日，不能由客户端直接指定日期。周频率对新类型拒绝；旧内容任务的 once/daily/weekly/IANA 时区不变。最大次数沿原正整数限制，一次必须 1，周期不开放无限执行。

计划表最小扩展：新增 nullable typed `report_spec`；原 task_selection 仅报告 kind 可为空；kind-discriminated CHECK：内容 kind 必须原 Selection、原正数 AI budget、report_spec NULL；报告 kind 必须精确 report_spec、task_selection NULL、execution_spec 为空且无模型字段，budget_unit=`none`、两项预算严格为 0。未知 kind、混合载荷、日报带商品/AI预算、内容任务把预算置零均拒绝。**不放宽原 `ScheduleBudget` 的正数合同**，新增种类自己的 validator/limit projection。

原 BusinessTask/Agent/Review schema 不增加假商品日报。日报 occurrence 持结果引用，business_task_id 必须 NULL；原内容 occurrence 的合法 BusinessTask 来源约束保持。新增 occurrence 附属 `report_execution`（同 Workbench DB，主键/同企业 owner FK 到 occurrence），只存冻结 report-spec、业务日期、计划修订、确定性 request fingerprint、capture_started、结果报告引用/摘要和关闭证明，不存财务正文。此窄表是本次执行输入/回执，不是另一报告 owner。

无模型分支通过类型及注入限制，没有 Agent executor、Model gateway、OrgResource、平台 mutation 或通用工具端口。预算展示/报价在报告类型隐藏，核对只返回店铺、日期规则、下次时间、次数和保存授权，不生成报价/预留。报告启用不依赖模型 policy、商品资料或 AI 点余额；仍依赖原 Workbench/Temporal、Report DB、Store/Cockpit 和当前 IAM。

## 5. 后台权限与历史访问

### 5.1 创建、触发与保存

浏览器仍走现有 verified identity / effective organization / same-origin BFF。计划管理复用当前权限：`workbench.task.read` 与 `workbench.agent.use`（沿现有定时管理准入，不赋予模型能力），另需 `workbench.report.read/manage`、`workbench.cockpit.stores.read`、`workbench.store.read`、准确当前 store grant 和 active store；不要求 `local_agent.write`、产品写权限、AI诊断建议权限或模型配置。暂停/结束本人的报告计划可仅用当前计划管理权，即使后续丢失店铺/报告生成资格，也能停止未来触发；新建、改期、恢复则重新核全套生成资格。所有权不能由管理员读取其它 actor 的报告绕过。

后台身份只能从已持久化计划/occurrence 的 org/actor/member 铸造。复用 `ReadExactServiceProjectAuthorization`、`IsUserActive` 及当前企业停用检查；要求 exact authorization ID 等于冻结 member，用户重建同企业成员不得自动复活原计划。角色取当前 IAM，不读保存时角色，不持用户 JWT，不调用只支持 HTTP 的 binder。

不能直接调用 `productAgentOriginRoles`，它额外绑定商品写/Agent权限。新 ReportOriginAuthorization 在原授权适配层复用精确 IAM primitive，按本目的检查上述 permission。HTTP 请求不能进入后台分支：仅本地 application 注入的 typed occurrence handle 可构造调用；任意浏览器 body/header/context ID 不构成该 handle，HTTP 授权失败绝不 fallback。

经营数据复用原 Snapshot，但给同一 owner 增加明确的 **只读定时日报 Boundary**：只接受已经验证的 exact origin + occurrence，`Current` 限制为 stores-read（GoalsRead/维护/写均 false），`ReadStores` 使用当前 MemberScopedStoreRepository 的 Get，只允许计划中单店；`LockStores`/经营 Execute 在此消费者不可调用。防止 Snapshot 顺带读取组合目标/其它店铺。正常 HTTP authorizer 不变，也不把无期限 `AuthenticatedIdentity` 注入其原 binder。

在首次触发、数据读取前后、Report DB 保存前再次校验当前 origin、store grant 和报告管理权限；IAM/Store 不可用时不写新报告。已有不同 owner 的授权与撤权提交不承诺全局同事务；保存使用原“事务中最后 live check 后提交”的边界，提交前最后检查若已观察到撤权必须拒绝。暂停/结束仅阻止未来认领；已认领本次可完成只读报告，但成员/数据撤权不能被该规则绕过。

### 5.2 查看、列表和下载

新报告仍是当前企业内本人的私有不可变快照。新类型的列表/搜索/摘要、详情、下载和收藏均需当前 membership、report.read、cockpit.stores.read、store.read 和原单店授权；筛掉不可见条目时不能通过列表标题、数量、搜索、缓存或下载暴露原统计。管理收藏还需 report.manage。已有三种报告类型沿原个人历史导出合同，不追溯改变其权限。

普通历史读取不依赖计划仍 ACTIVE、调度在线、模型可用或经营事实当前版本；不因源数据编辑而覆盖/删除已保存快照。店铺撤权/停用、成员变更会拒绝新类型访问。原失败/未知执行在计划页只展示安全状态，不向失权者返回店名、报告标题或正文。内部恢复可读 exact receipt 的最少元数据以核对已发生结果，不授予用户正文读取或新写权。

## 6. 数据口径与报告内容

使用 `operationscockpit` 原 Snapshot(Module=stores, StoreIDs=[one], previous-day period)，保留原金额单位/币种、净收入/净利润/利润率/覆盖计算、源记录 ID/版本/区间。只投影选中店铺；不把整月记录按天均摊、不改公式。缺失金额/不完整覆盖为明确“未录入/资料不全”，不是零、健康或完整结果。明确成功的零值仍为零。无经营数据可保存“资料不全”日报，不生成无依据建议。

已保存平台观察按原权限及安全读取能力提供其独立采集时间/原范围/完整性，与财务统计分开标注，不充当昨日销售额/流量。沿既有 NativeReader 的安全限制：SHEIN 仅保存读；TEMU 避开会触发 QueryStore 的入口，标不可离线读取；其它原 reader 未准入则标未接入。任何路径不得调用 Sync、starter、平台取数或诊断模型。可选观察失权仅标该源不可用；核心财务/单店权限失败则本次阻断。

报告是确定性格式：店铺/统计日/生成时间；经营汇总及已知比较；覆盖与缺项；可用保存观察；来源/版本。原 Document 的 ≤128KiB、≤16 sections/256 fields 及字段长度界限保持。可省略超限的可选明细引用并明确省略，不截断金额/日期/来源归属；无法保留必需来源与缺项则本次受阻。来源各自的捕获时间保留，不宣称全局同时刻。

## 7. Report owner 的一次捕获和跨库恢复

### 7.1 固定身份与 Report DB 事务

新增 report kind `STORE_DAILY_REPORT`（满足现有 kind varchar 长度），允许此类型 ProductKey 严格为空；其他类型仍要求原商品字段。SourceID=准确 occurrence UUID，SourceVersion=`1`（本次格式版本，内容 digest 另存），StoreID=冻结 store。新增 nullable typed `scheduled_origin` 元数据（plan/occurrence/revision/member/slot/date/rendererVersion/requestFingerprint），只该 kind 可非空，准确来源与 report 字段互校；旧类型 NULL。报告库不对另一个 DB 建 FK。

`reportKey = UUIDv5(report-namespace, org + actor + occurrenceID)`；intent fingerprint 固定包含 org/actor/member、plan/occurrence/revision、nominal slot、store、统计日、业务口径/格式版本，**不包含读取时钟或可变化的源正文**。同 key 换店/周期/成员/格式冲突。Report owner 校验可信 handle/plan ref、当前权限及 source reader，不接受浏览器正文或“已授权”字段。

CaptureScheduledDaily 先按 fixed key/fingerprint 查原 receipt；若已保存，返回原报告，不重取数据。若未保存，在 Report DB 事务外读取和格式化来源；然后在 **原 Report DB 的一个事务**按 key 串行化、复查 receipt、当前权限，提交 immutable saved_report + command receipt。可复用现有 repository transaction / unique constraints，补一个窄目的方法，不复制独立报告存储。

并发两读可能看到不同源版本：Report DB 的同 key 串行化只允许首个成功提交的 snapshot；后来者在事务内先读回原 receipt，返回首个正文/digest，不因自己较新的临时读取冲突或覆盖结果。跨进程唯一约束是最后防线。保存响应包括 reportID、origin、requestFingerprint、snapshotDigest；Replay 不改 capturedAt、财务值、来源或报告ID。

### 7.2 Workbench claim / report commit / finalize

1. Workbench PG 事务：锁原 plan；先检查同槽是否已有 occurrence；验证精确 rule/revision、ACTIVE、registration MATCHED、catch-up、次数和无其它占用；原唯一 `(org,plan,slot)`；认领 occurrence + report_execution 的冻结输入 + active pointer + 次数。事务内不读 IAM/Store/Report DB，不造 BusinessTask。
2. 事务外获取当前后台授权。调用 Report owner 前，先在 Workbench 同一 occurrence 锁下通过 `BeginReportCapture` CAS 将 capture_started 从 false 改为 true，绑定原 key/fingerprint；必须确认该标记已提交才能调用捕获。该位只升不降，重试仍用原 key。若要以“尚未写入”收敛明确权限拒绝，必须在同一锁内确认 capture_started=false 并原子终态关闭 report_execution/occurrence；后到或旧在途 worker 的 BeginReportCapture 将拒绝。不能用单个 worker 的内存标记证明所有并发 writer 都没发送。首次认领后，不随计划改期/修订重算该次 spec、成员或统计日；到点后改计划不改变旧次输入。重复 callback 先读旧次，不能按新 revision 再认领。
3. Report DB 确认保存后，通过内部 `ReadScheduledCaptureReceipt` 核对同 org/actor/member/store/plan/revision/occurrence/slot/date/fingerprint。Workbench 事务 CAS 更新该 occurrence 为 RECONCILED、保存 reportID/digest/proof，并且只在 active pointer 仍指向自己时清空；绝不清空另一次占用。随后原计划页显示“已生成/资料不全”和查看报告入口。
4. receipt 证明和既有报告不一致则保持 UNKNOWN/占用，报冲突。Report DB 的 NotFound、超时、计划投影 ERROR 或用户刷新均不是“肯定未保存”的证明。

新类型的 `ReconcileScheduledReport` 只读取报告 exact metadata receipt，不借用 Agent/积分结算证明；原内容分支仍只用原 ReconcileScheduledTask 和原 Run/invocation/resource 证明。新增日报永远不访问或释放其它模型预留。

| 失败点 | 固定结果和后续动作 |
| --- | --- |
| claim 事务提交前失败 | 同槽正常重试；不存在半个 claim |
| claim 后、尚未尝试写报告，明确权限拒绝 | 原 occurrence 锁内 capture_started=false 才可原子终态关闭为 BLOCKED 并清本次占用；该终态阻止后到 BeginCapture，计次数不返还，不自动换店/放权 |
| Report DB 保存/响应结果不明 | UNKNOWN + 保持占用；原 key 查回执/重试捕获，只会得到同份报告；不换 key、不凭 NotFound 清占用 |
| 报告已提交、回写 Workbench 失败/进程重启 | 原 Temporal retry 或原有界 recovery scan 调用类型专用 reconcile，读原 receipt 后补关联；不读取新数据覆盖原报告 |
| 旧 writer 仍在途时暂停/结束/超时 | 保留 active occurrence，不能标“无结果”或清占用；只有 exact 保存证明，或尚未尝试写的确定证明可收敛 |
| 已尝试写后被撤权，尚无可核对 receipt | 不再创建报告；保留 UNKNOWN/占用；内部只读 metadata 核实可继续，不能用失权当取消证明 |
| 同槽并发/重试/稍后源财务变更 | 一次 occurrence、一次 Report DB command、同一 report；首个成功快照保持，未来新槽读新数据 |

复用原30秒有界恢复扫描/游标和 Temporal 重试，不新增调度器或全库扫描。新类型 scan 先按 exact key/fingerprint 读 Report receipt；已提交则只补原关联，无 receipt/可重试未终态且当前生成授权仍有效、新类型仍启用时，使用原 frozen occurrence / report key 继续 CaptureScheduledDaily，包括 BeginCapture 已提交但进程在调用 Report 前崩溃、超过 Temporal 重试窗口的情况。这是恢复已经认领的一次执行，不是创建过期新槽或重新计次数；仍不能由 NotFound 推断无在途写或清占用。权限拒绝/依赖未知时保持占用并记录安全原因。新报告类型被 feature-disable 后仍能只读核实已提交报告；不能再认领/捕获新报告。原跨槽占用、15分钟补触发窗口、过期槽记录跳过、不追补历史和次数语义保持。

## 8. 现有计划保留与显式安装边界

现有 kind 保留原 selector、spec bytes、fingerprint、Temporal Schedule ID/workflow/queue/args/memo digest、state、rule/revision、occurrence、Task/Run/Review和active pointer；新增可选字段在内容序列化中严格省略。现有 scheduleOptions 的内容分支构造投影必须保持原有字节与 hash，不能把新增类型字段默认写入而触发全部重登记。已有已暂停/结束/UNKNOWN计划不被此能力恢复或重建。原无关预留不进入任何新恢复入口。

只为新 kind 扩展原 owner 的 schema/validator/grants/readiness；当前行不 backfill、不改ID、不复制数据、不重排历史，不引入旧系统 compatibility adapter。现有当前数据的保留是本次用户明确决定。未来保留库安装只允许现有 installer/owner 入口的有界加法变更，各库各自事务；两库都通过验证且端口可用才显示新类型。serving 不跑 DDL。具体实例操作不在本轮授权内。

新类型单独 opt-in，关闭只影响新日报，不关闭现有内容、Chat、经营或报告模块。不新增安装/治理平台。安装部分完成时新类型不开放，旧内容路径继续合格可用，不用全功能启动失败来代替新能力 readiness。已有 verifier 的类型检查要同时支持当前两种合法 shape，原内容约束/权限缺失仍拒绝。

回退优先保持本次能识别两种 schema 的版本并关闭新类型创建/认领，已有结果只读恢复继续。**不承诺已有日报写入后可直接降级到旧二进制**；旧 verifier 不认识新 kind/表/约束。新事实存在时不删表/收窄约束/转换日报来硬退。原数据/服务不因方案退回而关闭，不为每次改动要求新备份/隔离项目。

## 9. UI、接口与有限范围

原 `/workbench/ai/tasks?view=schedules` 增加业务类型选择/筛选；仅展示真正可用类型。新 kind 使用 typed store/spec 和核对保存授权，内容分支仍原表单/quote，不调用内容报价来造日报。新日报的服务端权限准入在 body kind 分派后准确检查；未知类型拒绝，不能通过 HTTP原通用agent gate绕过 report/store gate。availability 返回按 kind 的真实 readiness；报告不因 a.agent==nil、模型 policy 未配置或 conversationOnly 而被错误挡住，但不自动扩大原 Workbench/企业 allowlist。

计划列表与安全历史由原管理器显示；日报 occurrence 只提供原报告链接，不假装 BusinessTask。报告页增加“经营报告”用途分类，详情与下载复用原文档 renderer，不显示模型/积分信息或内部数据库/receipt 术语。任务中心全量统计不把 occurrence/report 当商品任务重复计数；本批不新增一个独立可执行 Agent。

## 10. 实施验证义务（不是本轮已执行结果）

复用现有 Go/PG/Temporal/组件夹具和已完成 #1244 证据，不新建验收平台。四组实施验证：

1. **权限与无模型**：HTTP不能伪造后台身份；exact member变更/企业停用/单店撤权/报告管理撤权阻断新保存；无商品/模型权限仍可在报告权限下生成；无 ProductAgent 配置仍 readiness 正确；Spy/受控注入证明模型、平台 mutation、AI ledger调用均0。历史列表/搜索/计数/详情/下载撤权不泄露；旧报告类型无变化。
2. **约束与计划保留**：真实PG拒绝混合载荷/报告预算非0/旧内容预算0/假商品；保留已有一次/每天/每周、暂停、UNKNOWN 内容计划的全部ID/规则/摘要/登记hash和原active占用。验证内容字节不变、没有全量重登记、原预留不触碰；两库安装/能力关闭不误关既有模块。
3. **跨库与恢复**：同槽并发、Report提交后丢响应、Workbench finalize失败/重启、BeginCapture提交后调用Report前崩溃且超过Temporal重试窗口、源数据在并发读取间变化、改期与老次重试、撤权后只读核实、旧写仍在途不清占用；必须一 occurrence/一 report、首个snapshot不变、仅精确本次CAS；实际两库事务与原稳定receipt，不能用mock“全局提交”替代。
4. **用户路径与口径**：浏览器一次/每日单店核对 → 原后台触发 → 计划历史同份报告 → 我的报告刷新/下载；UTC+8前一天、跨午夜重试、显式0/缺失/部分覆盖与观察原周期。隔离验证用已有夹具/已有环境，不调用模型或发布商品；原35444启用/数据验证需后续授权，本轮 NOT_RUN。

实施前仅能声明设计准入；实现结束为 In Review，用户/指定验证者确认产品可用。实现、合并、部署和产品验收分开，不用CI或设计评审冒充运行通过。

## 11. 独立评审记录

### 实施增量：SHEIN 可选已保存订单观察（2026-11-14）

依据 [当前用户决定](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1279#issuecomment-6877937169)，本批将 `observations.Service.SavedOrderAmounts` 的现有只读结果投影进独立“已保存订单观察”章节，不使用 NativeReader 不按日的订单摘要。该来源仍由 Store Observations 负责，限 SHEIN 美国站半托管/普通自营；按原槽 UTC+8 前一天半开区间、订单列表创建时间和订单号去重，包含原状态，金额只取 productTotalPrice/orderCurrency 原币分组，不换汇、不充当销售额/实收/利润。沿用现有 250 候选上限及 complete / missing_fields / coverage_incomplete / limited 结果，不扩容。

后台复用现有 `storeobservationsapp.Authorization`、`OfficialObservationAccess` 和 MemberScopedStoreRepository：以原 frozen org/actor/member 重新读取 exact IAM，并要求既有订单只读权限；只构造同一店铺的 Directory（不会列举其它店铺）。先核对当前核心权限和 Store 平台为 SHEIN，才进入该来源；不进入 TEMU QueryStore 或其它平台分支。`SavedOrderAmounts` 只调用 Repository / Access.Open(sync=false) / handle.Check，不调用 QueryObservedOrders、Starter、Sync 或平台 API；新注入只暴露此读取函数，没有同步入口。原凭据仅在现有 OfficialObservationAccess 内解密并验证，不进入报告。

源失权、未配置、未同步、覆盖不足、最新同步未完成、未知时间、金额/币种缺失、超限都明确标记该可选来源不可用/不完整；不让它阻断具有核心权限的财务日报。只有 complete 才显示完整当日计数和原币总额；缺时间时不把 matched 数标完整，缺金额/币种不显示完整金额。保存前再次读取源当前权限；若已失权则去除临时订单数据，改为不可用提示，再执行核心权限的保存闸门。历史读取仍按本设计核心权限规则，不新增另一历史事实源或资金口径。本窄增量需独立复核后才进入正式接线，原设计其余部分冻结不变。

2026-11-14 第一轮，独立 Reviewer `daily_report_architecture_review` 只读核对设计候选 `920de860697a32f6859f4e3d6d4db6e0db2c0ae0` 及当前实际 ScheduledPlan/Temporal、HTTP-only report/cockpit authorizer、Report DB 事务和原内容预算约束。结论 **IMPLEMENTATION_READY**，BLOCKER=0、IMPLEMENTATION_TEST=1，无其它有效 finding。

Finding：BeginCapture 已提交而 Report 调用前崩溃，若 Temporal 重试窗口耗尽，单纯查 receipt 的恢复扫描无法完成本次报告。Product requirement affected：本次已认领执行可在崩溃/重启后恢复，不能永久卡住核心路径。Classification：**IMPLEMENTATION_TEST**，属于原固定身份/幂等协议内的恢复接线细化，无需新恢复器或重开全局架构。Action：§7 明确原扫描按当前授权/开关和原 frozen occurrence/key 继续同次 Capture；§10第3组补上该崩溃窗口。Reviewer 已读取该两段实际增量，确认覆盖且无新 finding，维持 IMPLEMENTATION_READY。运行验证仍是实施阶段义务，不记为本轮 PASS。

该设计现冻结。按 AGENTS finding 分类及停止规则，只有新实证 BLOCKER 可重开；实施细节在上述四组验证收敛，不因普通 main/元数据变化重复审核未变设计。本轮不派 Writer 写生产代码。准确最终 HEAD、CI 和独立评审证据继续记录在设计 PR，不在本文滚动更新。

## 12. 本批实施接线与启用入口

上文“本轮不派 Writer”描述设计评审阶段；用户随后明确授权在同一 #1280 实施本方案。新增代码沿用 Workbench / Store / Cockpit / Report 的现有 owner，不包含模型、平台同步或刊登调用。

- `internal/app/scheduledreport` 是有界组合消费者：exact IAM + active user + 企业状态、当前单店授权、Cockpit 只读 Snapshot、Report Capture。Report Center 的日报历史权限接线独立于 Chat/模型是否启用；执行端仍只有原 Temporal plan/slot 回调与原 30 秒恢复扫描。
- 配置在既有 `aiWorkbench.schedules` 下显式设置 `storeDailyReports: true`。无模型模式仍使用 `conversationOnly: true`，不填规划或内容模型策略；既有完整内容模式可与日报共存。两库扩展、权限或端口未准备好，只不开放新日报创建，不关闭既有内容、Chat 或报告能力。关闭新类型后只回查该次已保存回执，不再次捕获。
- Workbench owner 复用 `ai-workbench-schema-init --store-daily-report-only --runtime-role ai_workbench_runtime --expected-database <当前库名> --expected-server-address <数据库实际地址> --expected-server-port <数据库实际端口>`。使用既有私密 `AI_WORKBENCH_SCHEMA_DSN` 交接；入口核对准确目标，事务内增加当前原生 schema 分支及新表权限，不改计划、任务或余额，不重登记现有计划。
- Report owner 复用 `report-center-schema-init --store-daily-report-only --dsn-file <本机私密 owner DSN 文件> --confirm-database reports`。仅安装当前 Report schema 的类型与来源扩展，保持原报告及既有权限；serving 不运行 DDL。
- 新报告执行的 key 仅由原 occurrence 身份生成。Report DB 的事务在原键锁后先回查胜出结果，再检查权限、移除已失权的可选来源、重算最终正文摘要并提交；失败重试不能用临时读取的更新正文覆盖旧报告。Workbench 先提交 capture_started，再调用 Report；原扫描能够继续崩溃前已认领但尚未调用 Report 的执行。
- 日报来源权限适用于历史列表、搜索、统计、详情、收藏与下载。下载日报前重新读取报告，不直接导出已失权的页面缓存。既有三类报告保持原来源合同；时间统一按原 UTC 接口格式输出。
- 复用既有 Postgres/HTTP/Playwright 夹具验证；浏览器夹具的身份会话与名义 Temporal 回调受控，保存、当前单店权限、财务读取、两库事务及结果回查走真实代码。正式 Temporal 调度器未更改，沿用已完成的原调度验证；真实平台接入及原35444启用均不属于本批执行授权。

SHEIN 可选来源的独立增量复核结论为 IMPLEMENTATION_READY，0 BLOCKER。原 reviewer 的两项 IMPLEMENTATION_TEST 均纳入实现：保存前 `orders.read` + 原店铺/连接 binding 的实时检查及最终字节脱敏；不按状态额外过滤、不改原订单状态，不承诺没有提供的状态分组。正文独立保留同步记录、原范围、取得时间及完整性，原币不换汇；缺失与超限不当作零。
