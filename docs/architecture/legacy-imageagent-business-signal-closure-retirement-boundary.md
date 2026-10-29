# 旧 ImageAgent 业务 Signal 控制链完整退休设计

> #301：DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY。本 PR 当前只改本设计、architecture README 和 legacy-register 三文档；适用独立设计复核和用户确认前不改生产或测试源码。#301 保持 OPEN。

这批删除四个旧业务 Signal 入口及其专属排队、分派和测试尾部。当前四个业务命令通过带身份校验与回执的 **Temporal Update** 执行；恢复工作流使用的 **`effect_recovery_completed` Signal** 继续保留。名称带 `Signal` 的四个共享载荷仍是当前 Update 合同，不删除、不改名。

**待用户确认的回放边界：含四个旧业务 Signal 投递的历史不再承诺续跑，包括标为 V3 的这类历史；未含这些旧业务 Signal 的当前 Update 驱动有效 V3、恢复通知及其版本/幂等合同继续保留。** 没有读取或清空任何运行历史。本提案不是已生效的产品决定，设计确认前不得据此删代码。

## 1. 用户结果、范围与 Design Basis

- 父目标：[Issue #301](https://github.com/qq550723504/shuomi-ai-commerce/issues/301)，保持 OPEN。
- 使用者与场景：维护当前组织图组、Artwork 和采集单主图的用户及开发者；相同业务操作只经当前可确认、可恢复的命令入口，移除无仓库生产发送者的旧旁路和专属排队，不新增操作步骤。
- 用户授权：先设计完整闭包，不机械拆批；设计确认后再实施。不改 workflow/inventory，不操作既有实例、数据、凭据或进程，不增加内部试用前置。
- 调查基线：main `de55cf5f205f91fae246429be33eebf5bfee70e7`（已含 #996）；单 Writer、独立 worktree、一个主要分支/PR。
- 未来源码 **Design Basis: Independent Architecture**：改变 Temporal 命令接收与历史续跑边界，并精确替代前批 KEEP。现有 Update 和恢复事实 owner 不变，不能因代码看似闲置而省略这一决定。
- 当前三文档交付 **Design Basis: N/A / Ready**，不改变运行语义。适用独立 Architecture Review、用户确认 §1–§7（含 §4.2/§6.1）及 Issue 读回 `IMPLEMENTATION_READY` 前，正式源码保持 NOT_READY。
- 不做：新 API/UI、通用调度器、兼容 worker/alias、迁移/双读双写、Temporal 版本分叉、运行队列排空、读取或重写 history、数据库/权限/SQL/schema 变更、provider/canary 真执行、部署、上线或关单。GitHub workflow、完整 inventory、依赖、保护规则零改。

## 2. Authority、前批 KEEP 替代与 Must

适用 [AGENTS.md](../../AGENTS.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md) 和当前 [图像产品](product-image-agent-v1.md)。无 UI/IA/交互变化，Figma 增量 N/A。当前范围没有允许续跑旧 Signal 历史的兼容例外；本设计也不自行签发该历史的运行验收。

| 前批依据 | 本批提议替代的 KEEP | 仍保留 |
| --- | --- | --- |
| [#994 手工运行链](legacy-imageagent-manual-runtime-closure-retirement-boundary.md)、[#996 V2 合同尾部](legacy-imageagent-v2-contract-tail-closure-retirement-boundary.md) | “有效 V3 回放”不再笼统包含由四个旧业务 Signal 进入动作、触发持久化或 effect 的历史。完整退出这条 ingress，不保留版本化旧执行分支 | 当前 Update/Resume、Update 驱动的有效 V3、有效 recovery Signal、GetVersion marker/ID/取值、当前 JSON/hash/事务/恢复 |
| [2026-08-26 原设计](../superpowers/specs/2026-08-26-image-agent-workflow-design.md)、[原实施计划](../superpowers/plans/2026-08-26-image-agent-core-manual-workflow.md) | app 翻译为四个业务 Signal、正向重发/排队后执行的现行要求 | 当前业务不变量；原文、历史勾选和证据保留，只加限定失效注记 |
| [2026-08-29 effect finalization 计划](../superpowers/plans/2026-08-29-image-agent-external-effect-finalization.md) | 用业务 cancel Signal 驱动当前取消测试的写法 | started effect 必须终结后才能投影取消的安全要求，当前 Update 驱动与恢复通知继续有效 |

Must：组织/原始 member/业务任务/actor 归属不降低；当前同 ActionID 异载荷拒绝、单 pending owner、ack 持久化时点、响应丢失/Resume、UNKNOWN/fence、provider 不重复发送、计量/发布防重、绝对 deadline/取消/恢复不变。当前图像、AI/Resource/钱包、采集和 Store 观察同步/官方授权保留。旧 SHEIN 登录与身份回写已由 #985 退休，不恢复。

源码 hard-cut 不证明运行环境没有旧历史，也不授权停止、重置、终止、迁移或清理它们。发现具体当前外部可观察义务要求旧业务 Signal 续跑时，先报告用户；不得同时承诺完整删除入口和保留该执行历史，不自行加入兼容路径。

## 3. 实际消费者、调用链与根因

| 合同 → 实现 → 注入 → 消费者 | 核对结果与 owner |
| --- | --- |
| 当前 Service 命令 → `Client.ReplacePlan/RetrySlot/ApproveResults/Cancel` → `UpdateWorkflow` → `workflowUpdateState.register` 的四个 handler；`ResumeCommand` → `resume_command` | `worker.go` 的真实路径全用 Update。Retry/Cancel 等待 Accepted，其余当前命令等待 Completed；原身份/Validator/handler/回执逻辑保持 |
| 四个业务名称 → `GetSignalChannel` → `startSignalHandlers` 四个 coroutine → `dispatch*Signal` → 同一 handler | 旧接收旁路仍存在，但全仓生产 Go 中没有 `.SignalWorkflow(` / `SignalWithStartWorkflow(` 调用。`sdkWorkflowClient.SignalWorkflow` 只剩接口、测试 stub；没有发送者不等于已证明外部历史不存在 |
| 早期 retry Signal → `deferredRetries` + `signalIngressDeferred` → `dispatchDeferredRetries` → action/pending owner | 这是旧入口专属延迟执行队列；当前 retry Update 在业务阶段不允许时拒绝，不承诺自动等到 blocked 后执行。删队列及全部读写尾部，不给 Update 加队列 |
| 四个 `*Signal` struct + 四个名称常量 → Update Args/kind → `updateFingerprint`、pending receipt/Resume | 当前公共 wire 合同，仍有真实消费者。名字不能作为删除依据；原字段、顺序、tags、默认 JSON 键和 fingerprint 输入保持 |
| `EffectRecoveryWorkflow` → `SignalExternalWorkflow` → `effect_recovery_completed` → `dispatch/applyEffectRecoveryCompleted` → wake/父状态刷新 | 真实当前恢复链，必须保留。只从混合 helper 抽取原 recovery 接收循环到当前窄 helper；producer、回调及其身份/fence/intent/proof/settlement 校验不改 |

根因是当前命令已切 Update，但 workflow 仍保留另一套 ingress、入队和测试驱动。处理为 **RETIRE + SELECTIVE EXTRACTION**：删除旧入口，把仍有效的恢复接收循环留在现有 Temporal owner；不复制领域 Service，不新建第二事实源。

## 4. 合同、执行与持久化边界

### 4.1 删除与抽取顺序

1. 保留四个当前 Update handler/Validator 和 `resume_command` 的注册、作用域校验、业务准入、`initialActionPhase`、fingerprint、动作账本、pending/resume owner。`validateSignal` 虽然名字旧，仍是当前 Client 校验，不因名称删除或改写。
2. 从 `startSignalHandlers` 原样抽取 **只有 recovery** 的接收 coroutine，例如 `startEffectRecoveryHandler`。保留相同的 recovery channel、接收、`dispatchEffectRecoveryCompleted`、wake 和调用位置语义；删除原混合 helper，不包一层继续调用。
3. 删除 cancel/retry/replace/approve 的四个 `GetSignalChannel`、四个接收 coroutine，及 `dispatchReplaceSignal`、`dispatchRetrySignal`、`dispatchApprovalSignal`、`dispatchCancelSignal`、`signalCanEnter`、`signalCanEnterWithFingerprint`。
4. 删除 `deferredRetries`、`dispatchDeferredRetries`、`legacyRetryPending` 本地状态与专属条件。保留原 lifecycle timer、`WaitForCommands`、blocked/cancellation 等待、wake、pending Update 和恢复分支。
5. 删除私有 `signalIngressDeferred` 及 `prepareAction/runActionAttempt/validateAction` 中仅判断该值的子条件。保留 rejected/accepted/superseded、原 ledger 上限、拒绝墓碑、取消超限入口和同键异载荷拒绝。这三项是精确不可达尾部收敛，不重写当前 action 状态机。
6. 从自有 `sdkWorkflowClient` 删除无调用 `SignalWorkflow` 方法；保留 Start/Query/Update 全部签名与注册。删两个测试 SDK stub 的该方法、三个仅记录旧调用的字段，以及两条无发送时永远为空的断言；以窄接口/禁入 guard 替代。
7. `types.go` 仅去掉无声明承接的旧 Activity/migration 注释，说明四个共享 `*Signal` 是当前 Update 输入。四 struct、所有字段/tag/次序、`CommandAcknowledgement` 当前 alias、`ResumeCommandInput`、Activity/child/恢复载荷及常量逐字保持。

### 4.2 JSON/hash、回放与持久化 owner

| 边界 | 本批精确决定 |
| --- | --- |
| 当前 Update 输入 | `ApproveResultsSignal/RetrySlotSignal/ReplacePlanSignal/CancelSignal` 名称与声明不变；`approve_results/retry_slot/replace_plan/cancel` 继续是当前 Update 名称、kind 和 fingerprint 输入，保留常量和值。不得做字段改名、DTO 迁移、兼容 alias 或默认序列化调整 |
| 当前 JSON/hash | 保留原 payload 默认 Go 键、`Selection` 省略规则、嵌套值、nil/空、字段顺序、当前 hash 算法和既有有效值；原 `TestTemporalPayloadEncodingPreservesLegacyDefaultGoKeys` 不改。当前 plan/projection/commit/receipt 的保存与读取合同不变 |
| 私有 ingress 状态 | deferred 是 workflow 内存动作状态，不在 `CommandIngress` 的 Used/Limit/Exhausted/Reason 或 `PendingCommandReceipt.Status="pending"` 中持久化；队列无数据库 owner。删其源码重建路径会切断旧 Signal 驱动的历史续跑，不能据此声称没有历史影响 |
| **旧 Signal 历史** | **提请用户确认 hard-cut：** 含四业务 Signal 投递的历史不再承诺由本程序续跑，即使标为 V3。包含旧处理器只写内存账本/拒绝墓碑或排队而未发 effect 的情况，不能只按是否产生 Activity 判断。不给旧历史双执行分支、不新加 GetVersion fallback、不重写历史或迁移；不读取、终止、清理实际历史 |
| 当前 V3 回放 | 未含四旧业务 Signal 的 Update 驱动有效 V3、当前恢复通知、原 GetVersion change ID/顺序/有效取值、Activity/workflow/task queue/child/recovery IDs 不变。当前 actionID-only/run-scoped/bounded V3 发布键与有效 marker 保留；禁止把旧 Signal 改成重放时伪造 Update。现有版本 probe/fixture 只证明其覆盖的路径，不作为所有真实历史已验证的声明 |
| 实际持久化 | 当前组织 Repository 继续拥有 run/plan/slot/projection/commit；V3 ledger 继续拥有 effect/receipt/fence；Asset 事务继续拥有批准结果。schema/grants/SQL/迁移不改，不新增事实 owner，不访问现有表或记录 |
| 失败、并发与副作用 | 当前 CAS/commit fingerprint、绝对 deadline/取消、single pending owner、ActionID 同键异载荷、ack、UNKNOWN/响应丢失、Resume/finalization/恢复唯一责任方均保持；不新建重试、自动接管或 provider 补偿 |

Temporal 传输层仍可能接受一个无人消费的旧 Signal 并写入 history；不能把 SDK 发送成功当业务接受，也不能宣称服务器会按名字拒绝。源码边界是 **四旧 Signal 不进入当前动作账本、不消耗 ActionID、不触发写入/provider/发布、不排队后补执行**。不为返回拒绝回执再增加一套旧处理器。

## 5. 完整删除 / 保留 / 抽取矩阵

未来一次实施覆盖 **15 路径：3 生产、6 测试、6 文档**。当前 PR 只交三文档；不是已完成以下源码修改。没有整文件生产包删除，也不因此扩大到所有 Temporal Signal。

| 路径 | 决定与边界 |
| --- | --- |
| `internal/imageagent/temporal/workflow.go` | RETIRE 四业务 channel/分派/入队/提升/私有 deferred 尾部；EXTRACT 原 recovery 循环；KEEP 当前 Update/Resume/账本/校验/恢复与生命周期 |
| `internal/imageagent/temporal/worker.go` | RETIRE 窄 SDK port 的 `SignalWorkflow` 声明；其余当前执行/查询/Update/身份/等待阶段不改 |
| `internal/imageagent/temporal/types.go` | RETIRE 孤立旧注释；KEEP 共享输入和所有 wire 声明/常量，不改 JSON |
| `internal/imageagent/temporal/workflow_test.go` | 按 §6.1 转换全部旧业务 Signal 驱动和旧断言尾部；新增退休 Signal 无业务作用的负例；保留有效规则/安全/恢复断言 |
| `internal/imageagent/temporal/budget_authorization_test.go` | 仅将一个预算拒绝测试的 cancel 清理驱动改当前 Update，原预算/不派发断言保持 |
| `internal/imageagent/temporal/image_set_admission_test.go` | 仅将一个迟到图组测试的 cancel 清理驱动改当前 Update，原到期/不派发断言保持 |
| `internal/imageagent/temporal/manual_acceptance_test.go` | 仅删 `recordingRecoveryWorkflowClient.SignalWorkflow` 空 stub；现有 Update/恢复/十一槽位测试不改 |
| `internal/imageagent/temporal/organization_scope_test.go` | 仅删基于退役记录字段的 empty 断言；原不完整组织身份不能 Start/Query/Update 的断言保持 |
| `tests/legacy_imageagent_manual_runtime_retirement_test.go` | 新增本批完整 AST 禁入/保留 guard；原 #994/#996 函数原样保持，不改缺失错误处理/遍历/其它检查 |
| `docs/architecture/legacy-imageagent-business-signal-closure-retirement-boundary.md` | 本设计及实施状态，准确记录回放替代与批准例外 |
| `docs/architecture/README.md` | 当前索引和状态；历史正文保留 |
| `docs/refactoring/legacy-register.md` | 当前 EXTRACT/RETIRE 决定与 owner；历史/计数/勾选保留 |
| `docs/superpowers/specs/2026-08-26-image-agent-workflow-design.md` | 只追加四旧业务 Signal 现行要求被替代的顶部注记和本设计链接，历史正文不改 |
| `docs/superpowers/plans/2026-08-26-image-agent-core-manual-workflow.md` | 同上，限定旧 Signal 正向驱动/入队，原步骤与勾选保留 |
| `docs/superpowers/plans/2026-08-29-image-agent-external-effect-finalization.md` | 顶部注明取消驱动改用 Update，原 effect 终结安全要求仍有效；正文保留 |

### 5.1 显式 KEEP

`effect_recovery_workflow.go` 的真实 `SignalExternalWorkflow`、`EffectRecoveryCompletedSignal`、原 completion 校验和全部恢复测试；`approval_ack_test.go` 的丢失回执/同原动作 Resume；`history_replay_test.go` 与原 testdata/版本 probe；当前 domain、Service/HTTP/授权、Repository、schema/权限、worker 注册、WithWorker canary、AI/Resource/钱包/图像/采集/Store 观察与官方授权。不修改无关 assisted-planning 历史方案来恢复已拒绝的模式。

### 5.2 闭包停止条件

清单外出现实际消费者或原检查矛盾时，先检查同根 sibling，报告准确文件/行为/最小补丁和成本；需扩冻结清单、持久化/回放边界或原检查授权时先获用户准确决定。不得机械加批，不因 CI 失败放宽安全断言、不加 missing-skip、不修改 workflow/inventory/ceilings/lint 或绕过门禁。

## 6. 验证与验收

当前三文档只做适用 diff/链接/路径/索引核对及原 CI，不为文档制造业务 RED。后续获准实现先在现有测试 owner 加退休 guard/负例，确认旧源码真实 RED，再最小删除/抽取和改接；不新建 runner、验收平台或通用 callback 框架。

### 6.1 原检查精确转换

所有测试驱动改到现有 `testsuite.TestUpdateCallback`，明确检查 Accepted/Rejected/Completed 与预期错误；不吞错误、不复用 transport UpdateID 冒充业务 ActionID、不用发送成功替代业务完成。必要局部 helper 留在原测试文件，不新增工具/测试包。

以下是 `workflow_test.go` 本批涉及函数的完整分组；函数名允许随 Signal→Update 的真实含义改名。**有效断言保持**指当前业务/安全行为，不能将只属于旧 Signal 的排队、墓碑和续跑规则强加给 Update。

| 函数组 | 准确变更与保留 |
| --- | --- |
| `TestManualWorkflowExecutesEverySlotIndependently`、`TestManualWorkflowReplacesBlockedPlanAtExactRevision`、`TestManualWorkflowReplaceUpdatesRejectLostCommandWindowAndAckAfterPersistence`、`TestManualWorkflowReplaceUpdateResumesAfterRunTransitionFailure`、`TestManualWorkflowRetryUpdatesRejectLostCommandWindowAndAckAfterProjectionPersistence`、`TestManualWorkflowRetryUpdateResumesWithoutRerunningProviderAfterPersistenceFailure` | 替换旧 Signal 驱动/收尾为当前 Update；保留 slot/版本/落库时点、失败 Resume 和 provider 不重复断言 |
| `TestManualWorkflowMixedCommandsAtOneTickReserveExactlyOnePendingOwner`、`TestManualWorkflowUpdateActionIDIsIdempotentAndRejectsConflictingReuse`、`TestV3ProviderUnknownDeniesDirectRetry` | 替换仅有的旧驱动；同一 tick 单 owner、同键一致/冲突、UNKNOWN 不直重试保持 |
| `TestManualWorkflowWaitsForFinalApprovalBeforePublishing`、`TestManualWorkflowApprovalUpdateRequiresCanonicalDigestAndAcksAfterPublication`、`TestManualWorkflowApprovalUpdateResumesAfterCompletedStateFailureWithoutRepublishing`、`TestManualWorkflowDuplicateApprovalPublishesOnceWithStableKey`、`TestManualWorkflowIgnoresApprovalForStaleRevision`、`TestManualWorkflowRejectsApprovalWithWrongActorOrDigest`、`TestManualWorkflowExposesFinalResultDigestWhileAwaitingApproval` | 使用当前 approval Update，保留摘要/actor/版本拒绝、最终批准前零发布、稳定键/一次发布、回执/Resume 和 query 结果 |
| `TestManualWorkflowRejectsEarlyApprovalEvenWhenLaterResent`、`TestManualWorkflowRejectsApprovalQueuedWhileBlocked`、`TestManualWorkflowConsumesRejectedReplaceSignalActionID`、`TestManualWorkflowReplacementRejectsAssetsOutsideImmutableCatalogAndConsumesActionID` | 不再测试 Signal 排队/消费 ID；按原当前 Update validator/handler 分界断言早期/blocked/catalog/修订号准入。validator 拒绝不冒充 handler 墓碑；已准入 handler 的同键异载荷及拒绝墓碑仍由现有 Update 测试保持，不改生产语义 |
| `TestManualWorkflowRejectedLegacySignalTombstonesWorkflowUpdateActionID` | 删除“旧 Signal 抢占 ID”的正向期望，改为四退休 Signal 不改状态/写入/账本/派发，不延后执行，且相同业务 ActionID 的合法当前 Update 仍可成功的负例；不要用新 Signal 拒绝处理器实现它 |
| `TestManualWorkflowWrongActorRetrySignalDoesNotPoisonActionID`、`TestManualWorkflowDuplicateRetrySignalCreatesOneNewAttempt`、`TestManualWorkflowRemainsDurableWhileBlockedForLaterRetrySignal`、`TestManualWorkflowRejectsRetrySignalFromDifferentActor` | 改当前 retry Update；保留 actor 不污染、同动作一新 attempt、blocked 可等待后续合法命令、wrong actor 拒绝。planning/executing 的 Update 不自动排队；旧 deferred 专属期望退休 |
| `TestManualWorkflowRejectsCancelSignalFromDifferentActor`、`TestManualWorkflowWrongActorCancelSignalDoesNotPoisonActionID`、`TestManualWorkflowDrainsQueuedDuplicateCancelBeforeStartingChildren`、`TestManualWorkflowSerializesCancelAfterInFlightParentTransition`、`TestManualWorkflowCancellationStartsNoThirdChildAndKeepsCompletedSibling`、`TestManualWorkflowV3CancellationWaitsForStartedSlotFinalization`、`TestManualWorkflowCancelUpdateAcksAfterCancelledProjectionPersistence`、`TestManualWorkflowCancelUpdateResumesAfterTerminalPersistenceFailure` | 改当前 cancel Update，保留 actor、单取消写入/重复防重、已接受取消的派发边界、成功 sibling、started effect 先终结、持久化回执/Resume。原 Signal 排空实现细节退出，不新增超出当前 Update 生命周期的时序保证 |
| `TestSlotWorkflowUsesTemporalTechnicalRetryWithoutChangingSemanticAttempt`、`TestManualWorkflowQueryRecoversPersistedPartialSlotProgress`、`TestManualWorkflowBoundsConcurrentSlotChildren`、`TestManualWorkflowReplacePathUsesRealProjectionActivityAndRepository` | 改旧 Signal 收尾/命令；技术 retry 不改变语义 attempt、query 可恢复、并发上限、真实 projection/repository 原断言保留 |
| `TestTemporalClientRetryWaitsForAcceptedWhileOtherCommandsWaitForCompleted` | 只删退役 `signalName` 的空值断言；真实 Update options/name/stage/ID/Args/get 次数均保持。删除 `recordingSDKClient.SignalWorkflow` 与 signalWorkflowID/signalName/signalArg 三字段 |

另外四处测试尾部：

- `budget_authorization_test.go::TestImageAgentWorkflowBudgetDeniesRepairBeforeChild` 和 `image_set_admission_test.go::TestLateOriginalImageSetWorkflowClosesExpiredSlotsWithoutDispatch`：cancel 清理改当前 Update，原拒绝/不派发保持。
- `organization_scope_test.go::TestOrganizationClientRequiresCompleteBoundIdentityAtEveryIngress`：只删退役 `signalName` empty 断言；其它身份/零调用断言保持。
- `manual_acceptance_test.go::recordingRecoveryWorkflowClient.SignalWorkflow`：只删 stub。原恢复/worker restart/十一槽位流程保持，不把测试执行当真实环境验收。

在原 root guard 文件新增一项本批检查：用原 AST/文件解析机制确认四业务 channel、旧混合分派/queue/deferred/helper 和自有 SDK Signal 方法缺席；当前四 Update/Resume、Start/Query、恢复接收/producer/校验仍存在。不能禁所有 Signal、只靠一条名称 grep、忽略 parser/必需文件读取错误，或修改原 #994/#996 guard 的其它断言。

其它检查零改，包括原 payload JSON golden、已有 Update Validator/墓碑/上限/取消/UNKNOWN/deadline/ack/recovery 测试、`history_replay_test.go`/testdata、完整 inventory/计数/ceilings 和两套 workflow。若真实原 CI 暴露本表未涵盖的有效检查，按 §5.2 报告，不能临时改 expected 让检查通过。

### 6.2 实施验证与证据限度

必要本地检查覆盖 Temporal 相关原包/新负例、原 JSON/版本/recovery/安全断言及 root guard；按实际并发风险执行现有 race 测试，相关生产命令仅编译。稳定候选与最终合并候选使用原完整 CI；不为文档阶段跑业务测试，不人工重复 CI 全套。

新负例应覆盖旧消息在阶段变化前后都不能派发/污染动作，以及合法 Update 可正常推进；恢复 Signal 仍刷新父状态且拒绝错 scope/fence/intent。当前有效 wire 的既有 golden 和 fingerprint 原断言保持。现有版本 fixture/probe 不改写成新数据来伪造旧历史回放成功，不宣称已验证所有生产历史；真实环境升级/历史验收 NOT_RUN。

## 7. 复核、冻结、交付与停止点

1. 本阶段交三文档和准确设计 HEAD 的原 CI/适用独立设计复核。GitHub 原复核运行时不重复触发；确实启动失败最多一次同范围重试。未经用户本轮授权不默认新增 AI 互验线程，设计与源码最终复核授权分开。
2. finding 先按 AGENTS 分类，说明当前 Must、实际后果和阻塞层级。最多两轮正常架构复核；非 Blocker 不无限重开设计。§4.2 的旧 Signal 历史取舍必须向用户明确交接，不能以 review/CI 代签用户产品决定。
3. 适用复核完成、用户确认本设计且 #301 当前段读回 IMPLEMENTATION_READY 后，单 Writer 在同一主要 PR 整批实施15路径；不机械拆小批，不默认派生新任务。
4. 完整实现后一次集中源码最终复核，修复只复核相关增量；准确 HEAD、真实 diff、CI run/attempt 和批准证据写 PR。CI/AI 复核不是产品验收。当前阶段不签发 SOURCE_COMPLETE/可上线。
5. 合并、实际 merge SHA 的完整 push/main CI、部署/运行验收和 Issue 关闭分别授权。正常 expected-head 合并，不绕过门禁或改变保护；HEAD 改变依规则重新确认。
6. #301 始终 OPEN；只维护当前段和准确 PR 证据。既有历史后缀/勾选逐字保留。当前图像/AI/采集/Store 观察同步与官方授权继续保留，既有实例/数据/凭据/进程不动，没有新的内部试用前置。

```text
Legacy decision: RETIRE | EXTRACT
Reusable behavior: recovery completion receive loop; current Update inputs, validation and action lifecycle remain with their existing owner
Current owner: internal/imageagent/temporal; organization Repository / V3 ledger / Asset transaction retain current durable ownership
Cutover/deletion condition: design review + user confirmation of the old business-Signal history cut + IMPLEMENTATION_READY, then complete closure and original CI/final review in one PR; no runtime operation implied
```
