# 旧 ImageAgent 历史执行协议与便利合同完整退休设计

> #301：DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY。本次仅提交本文、架构索引和 Legacy 登记三文档。源码 Design Basis 为 Independent Architecture；准确设计候选的适用复核和用户确认完成前，不修改正式生产路径。#301 保持 OPEN。

这批清理的是 **当前 ImageAgent 内部为了续跑旧执行历史而留下的版本分支**，不是删除当前图像功能。提案保留当前版本 1，把已到达版本读取点、却只能得到旧 `DefaultVersion` 的历史排除出续跑支持范围；同时删除旧失败投影 Activity、旧摘要/发布键选择和四个无实际业务消费者的便利方法。当前图组、Artwork、采集单主图、Update/Resume、恢复通知、Store 观察同步继续保留。

**需要用户确认的准确取舍：历史标为 V3、甚至属于组织模式，也不自动代表本批继续支持。若回放在本批任一原版本读取点得到旧默认版本，当前程序将不再承诺它能继续推进。** 本文不声称已证明没有这些历史，也不授权读取、停止、重置、终止或迁移实际历史。

## 1. 用户结果、范围与 Design Basis

- 父目标：[#301](https://github.com/henryxu1008/shuomi-ai-commerce/issues/301)，保持 OPEN。使用者是维护当前图像、采集和审核能力的开发者；交付结果是当前执行只走一套协议，不再靠旧历史选择器决定安全、摘要、发布和投影行为，不增加用户操作。
- 当前用户授权只覆盖完整闭包调查、设计、必要文档检查、提交/推送和设计 PR；不包含源码实施、合并、运行验收、部署或关单。单 Writer、一个主要分支/PR，不机械拆批。
- 调查基线：main `46a817ef1b872f69515669942d18435216c2e4b7`，已含 #998。接单、真实消费者及 owner 核对：[记录](https://github.com/henryxu1008/shuomi-ai-commerce/issues/301#issuecomment-6495206627)。
- **源码 Design Basis: Independent Architecture。** 本批改变 Temporal 历史可续跑边界，精确替代前批 KEEP；即使不新增表或用户入口，也不能按死代码机械删除。本文是待确认设计，不是已成立的新产品决定。
- 当前三文档：**Design Basis: N/A / Ready**，仅设计交付。源码必须在适用独立设计复核、准确例外处置及用户确认 §1–§7 后显式进入 `IMPLEMENTATION_READY`，才可开工。
- 不做：新图像能力、协议迁移/兼容 worker、别名/fallback、版本平台、第二事实源、通用验收工具、运行队列清理、历史抓取、provider/canary 真执行、数据库/schema/SQL/权限/计费改造。GitHub workflow、inventory、依赖和保护规则零改；不操作既有实例、数据、凭据或进程，不增加内部试用前置。

## 2. Authority、前批 KEEP 替代与 Must

适用 [AGENTS.md](../../AGENTS.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md) 和 [图像产品架构](product-image-agent-v1.md)。无 UI/IA/交互变化，Figma 增量 N/A。当前 Product/Threat Model 不因此扩张；不把生产历史调查或专项运行验收变成内部试用前置。

| 原依据 | 本批待确认替代 | 仍然有效 |
| --- | --- | --- |
| [#994 手工运行链](legacy-imageagent-manual-runtime-closure-retirement-boundary.md)、[#996 V2 合同尾部](legacy-imageagent-v2-contract-tail-closure-retirement-boundary.md)、[#998 业务 Signal](legacy-imageagent-business-signal-closure-retirement-boundary.md) | 广义“有效 V3 回放”不再包括在本批原版本读取点落入旧默认分支的历史；ActionID-only、旧 run-scoped 发布键、无 execution identity 投影、旧失败投影及非 finalization 子协议退出 | 当前版本 1 的 Update/Resume、组织身份/租户、有效 JSON/hash、execution-scoped commit、当前有界发布键和当前恢复通知；#998 已退出的四旧业务 Signal 不恢复 |
| 2026-08-27 recovery、08-28 budget、08-29 finalization/recovery 的 spec/plan（§5 列出八文件） | 旧版本双分支、默认关闭预算/恢复、旧终态 fence 时点和旧回放要求的现行地位 | 当前 durable effect、quote、绝对 deadline、取消/finalization、UNKNOWN、恢复唯一 owner、安全及幂等要求；原文和历史勾选保留，只追加窄失效注记 |

Must：当前组织/member/actor 归属、同 ActionID 异载荷拒绝、单 pending owner、回执持久化时点、响应丢失/Resume、CAS/ledger/fence、provider 不重复发送、资源/结算/发布防重、当前绝对 deadline/取消及恢复不变。当前图像、AI/Resource/钱包、采集和 Store 观察同步/官方授权保留。旧 SHEIN 登录与身份回写已由 #985 退休，不恢复。

Legacy decision: **RETIRE + SELECTIVE EXTRACTION**。可复用行为是当前值的版本检查、当前纯摘要计算与已有 Normalize 校验；留在现有 Temporal/图像 owner，不包装旧选择器，不引入兼容类别。

## 3. 实际消费者、owner 与根因

| 合同 → 实现 → 注入 → 消费者 | 当前事实与处理 |
| --- | --- |
| `Client.StartManual` → 公共 `WorkflowInput` → `ImageAgentWorkflow` | Client 不填 `BudgetAuthorization`，公共载荷的零值仍会出现；入口根据版本点设置当前运行值。不能在入口把所有 false 都拒绝，也不能把私有协议位混同公共数据合同 |
| 父 workflow → `workflowActivityWire` / `activityWireForWorkflow` → effect owner、摘要、发布、child/恢复派发 | 一个私有混合选择器仍承载 V2/V3、发布键、digest、恢复和旧默认分支。当前 parent 路径已选 V3，根因是历史执行兼容仍留在当前 owner，不能只删除一两个方法名 |
| 父 workflow → `ImageSlotWorkflowV3` → `ExecuteSlotV3` / finalization → recovery | 当前 parent 产生 finalization=true 的子协议；旧 false 子协议可退休。保留当前 child/Activity 名称、task queue、IDs、字段序列化及有效取消/恢复 |
| `RecoverEffectV3` → durable `effect.Quote.Fingerprint` → `ExecuteSlotV3ActivityInput.BudgetAuthorization` | 该布尔值由当前持久化事实生成，可能是 false；这是当前恢复合同，继续保留。不能伪造 quote、迁移数据或把 false 一律解释为旧历史 |
| workflow 异常 → failure marker → V1/V2 `PersistWorkflowFailure*` → 同一组织 Repository | V1 wrapper 使用固定旧 commit；V2 execution-scoped commit 是当前路径。退休 V1 DTO/name/注册/wrapper，保留 V2 与共享私有事务实现 |
| `EffectRecoveryWorkflow` → reconcile Activity → parent-signal marker → recovery Signal | marker 在 Activity 后，不得提升到入口；保留当前 reconcile、通知、settlement/proof/fence 和父状态刷新 |
| `ValidateStagingManifest` / `ValidateFinalManifest` | 无业务生产调用；前者只有 `manual_acceptance_test.go` fixture 调用，改用现有 `NormalizeStagingManifest` 并返回原错误。保留 Normalize 及全部实际范围校验 |
| `BudgetPolicy.HasProviderLimits` / `Budget.EnabledLimitNames` | 全仓只有定义，无实际消费者；删除。`Policy/Allows`、checked math、limit 常量/字段及零 EnabledLimits 的当前解释不变 |

`Service.Start/RestartFailed` 仍服务采集单主图及 ImageSet；canary 有真实 CLI 消费者，均不是本批退休对象。公共类型/方法是否保留按真实调用和当前产品合同判断，不能按 V1/V2/V3 或 Signal 名称判断。

## 4. 当前协议、JSON 与持久化边界

### 4.1 保留 SDK 机制，退出旧选择器

仓库固定 `go.temporal.io/sdk v1.43.0`。复用 SDK 的 `workflow.GetVersion(ctx, existingID, 1, 1)`，保留原 change ID、当前值 1 和读取位置，不发明协议注册表、输入版本标记或第二 owner。SDK 的[版本 API 文档](https://pkg.go.dev/go.temporal.io/sdk@v1.43.0/workflow#GetVersion)明确支持在只剩当前分支时保留版本调用并提高最低支持版本；[官方 Go 版本说明](https://docs.temporal.io/develop/go/workflows/versioning)用于解释确定性边界。

**14 个既有 ID、15 个调用点全部保留**，每点最小/最大值变为 1，移除旧分支而不删除 marker：

| 原位置及原先后关系 | change ID 常量；值逐字保留 |
| --- | --- |
| 父入口，在原校验/调用位置依次读取 | `commandIngressPlanPolicyPatch`、`budgetAuthorizationPatch`、`externalEffectFinalizationPatch`、`projectionExecutionCommitPatch` |
| effect owner 构造中，原 requests channel 建立之后、重复 external marker 和 coroutine 之前 | `slotExecutionWireV3Patch`、`approvalActionIDV3Patch`、`approvalPublicationWireV3Patch`、`resultDigestV3Patch`、`approvalPublicationScopePatch`、`approvalPublicationKeyLengthPatch`、`activityWireV2Patch`、`effectRecoveryStartWireV1Patch` |
| effect owner 原位置再次读取，不能合并/提前 | `externalEffectFinalizationPatch`（同一 ID 的第二调用点） |
| 父 workflow 内部执行返回错误之后 | `workflowFailureProjectionPatch` |
| recovery 的 reconcile Activity 完成之后 | `effectRecoveryParentSignalPatch` |

中间八点可抽成窄的当前协议断言 helper，但必须位于原 channel 建立之后，保持个别调用顺序。原有两处条件 marker 在全部当前 1 分支中都会到达；提案只支持此当前分支，不改变当前读取序列。不为了通用校验把失败/恢复 marker 提升到入口。

删除私有 `workflowActivityWire`、`activityWireForWorkflow`、effect owner 的 `.activities`、私有 `enforceIngressPlanPolicy/externalEffectFinalization` 和相应复制位。直接使用当前 Activity 常量，严格校验、当前取消/恢复/fence 无条件采用当前语义。`projectionExecutionID` 继续是运行时身份，不能为空；无 execution ID 的旧 commit/hash fallback 退出，不能改成对空值算 hash。

保留 `approvalActionPublicationKey` / 导出 `ApprovalActionPublicationKey` 当前有界 SHA256 算法；删 `approvalPublicationKeyForWire`、`legacyApprovalActionPublicationKey` 及 ActionID-only/旧 run-scoped 分支。`resultDigestForWire` 只抽取当前纯计算到窄 helper：ImageSet/Artwork 仍走原 `VisualResultDigest`，其余仍走原 `ResultDigestV3`，算法/排序不变。

### 4.2 回放、载荷与事实 owner 的准确取舍

| 边界 | 提案与限制 |
| --- | --- |
| 旧历史判定 | **在已到达的原 GetVersion 点**，旧默认版本不在支持区间，SDK 拒绝该次回放/Workflow Task 继续推进。可能表现为版本错误和重复任务失败，不承诺自动转成业务 terminal failure，也不加 catch 后写 `run.failed` 的补偿 |
| 未到达 marker | 不要求一份历史已经包含所有 14 个 marker。条件/晚到点尚未发生，不等于 legacy；新执行首次正常到达时由 SDK 记录当前值。不能按“缺 marker”扫描历史并统一拒绝 |
| 检测与副作用时点 | 不承诺所有旧历史都在所有 effect 之前拒绝。失败 marker 在内层执行之后，恢复 marker 在 durable reconcile 之后；先前已完成/已调度的共享当前 Activities 仍由原 lease/deadline/UNKNOWN owner 处理事实，源码切换不会神奇撤回它们 |
| 旧子协议 | `SlotWorkflowV3Input.ExternalEffectFinalization=false` 不再支持。保留公共字段/JSON，在 child 原 `validateWorkflowScope` 之后、安排新 Activity 之前以现有 validation error 拒绝，组织/queue/租户拒绝优先级不变。当前 parent 始终传 true；不新加 child marker、兼容 wrapper 或 state owner。只拒绝这一旧协议位，不能同时要求 BudgetAuthorization=true |
| 当前预算数据 | 公共 `WorkflowInput` 的初始 false 仍接受，在原 budget marker 后设置当前父值。child/共享 `ExecuteSlotV3` 的 BudgetAuthorization 及其校验分支继续保留，尤其 recovery 依据已保存 quote 的 false；不修改 `activities_execution_v3.go`、`activities_review_v3.go` 或 `activities_recovery_v3.go` |
| 当前 JSON/hash/版本 | 公共 WorkflowInput、child、ExecuteSlotV3、EffectRecovery、四个共享 `*Signal`、Update/Resume 载荷的声明顺序/tag/默认 JSON/nil空值不变；当前 Activity/child/task queue/恢复 IDs、hash、版本常量和值不变。私有运行位不进入序列化；V1 failure 专属 DTO/name 退出，不加 alias |
| 当前持久化 | 组织 Repository 继续拥有 run/plan/slot/projection/commit；V3 ledger 拥有 effect/receipt/fence，Asset/Resource/钱包仍由当前事务 owner 持有。V2 failure wrapper 与共享私有事务保持。无 schema/SQL/migration、旧记录改写或数据清理 |
| 未完成任务与 rollout | 删除 V1 failure registration 后，旧队列里的 V1 Activity 可能无人处理并继续重试；不保留拒绝 wrapper/旧 worker，也不自行 cancel/drain。实际旧历史/队列是否存在为 UNKNOWN，真实部署、续跑和排空为 NOT_RUN。源码交付不等于 rollout 已验证，不增加内部试用前置 |

如果发现具体当前外部义务要求继续执行这些旧历史，应报告用户重新决定；不能一边承诺完整 hard-cut，一边默默保留旧执行分支。真实环境处置需要另行授权，本批不访问它来替代产品决定。

## 5. 完整删除 / 保留 / 抽取矩阵

未来同一批一次覆盖 **33 路径：8 生产、12 Go 测试、2 普通合成 JSON fixture、11 文档**。本设计 PR 只含其中三文档；下列是待确认的完整实施边界，不是已执行状态。无新生产文件/表/owner。独立初审发现遗漏的 direct-child 测试，分类 `IMPLEMENTATION_TEST`，在设计确认前收齐同根调用与原安全断言改接，不增加生产机制。

| 路径 | 决定及准确边界 |
| --- | --- |
| `internal/imageagent/temporal/workflow.go` | RETIRE 旧 wire/selector、两私有协议位、旧 commit/key/digest/failure/终态 fence 分支及全部调用尾部；EXTRACT 当前纯 digest/窄 marker 检查；KEEP §4 15调用点、Update/Resume/actor/ledger/ack/CAS/UNKNOWN/current cancellation/finalization/recovery |
| `internal/imageagent/temporal/slot_workflow.go` | RETIRE false-finalization 子协议及 `effectPhaseForFinalizationWire` 空 phase 分支；当前 options 去历史布尔参数，保留 heartbeat/WaitForCancellation。KEEP BudgetAuthorization 数据分支、绝对 provider window、`addFinalizationGrace` 饱和运算及 #932 安全边界 |
| `internal/imageagent/temporal/effect_recovery_workflow.go` | 原 late marker 改当前区间，移除旧不通知分支；原 reconcile/Signal/wake/proof/settlement/重试 owner 和时点不变 |
| `internal/imageagent/temporal/types.go` | RETIRE 两私有位、V1 `PersistWorkflowFailureActivityInput` 和 `activityPersistWorkflowFailure`；所有当前公开载荷/字段/常量逐字保留 |
| `internal/imageagent/temporal/activity_registration.go` | 仅删 V1 failure 注册，V2 和其余注册/模式隔离不变 |
| `internal/imageagent/temporal/activities_persistence.go` | 仅删 V1 `Activities.PersistWorkflowFailure` wrapper；保留 V2/共享私有实现的授权、事务、CAS、幂等 |
| `internal/imageagent/artifact.go` | RETIRE 两 Validate 包装；Normalize 和当前验证算法/合同不变 |
| `internal/imageagent/budget.go` | RETIRE `HasProviderLimits`、`EnabledLimitNames`；当前 Policy/Allows/checked math/限制合同不变 |
| `internal/imageagent/temporal/workflow_test.go` | §6.1 原测试改接当前协议，退旧历史正向断言；保留所有仍有效安全/业务断言 |
| `internal/imageagent/temporal/history_replay_test.go` | 退旧正向 probe、旧 wire/key 构造；复用原 fixture parser，增加两合成 prefix 的普通 SDK 回放测试，不新建 runner |
| `internal/imageagent/temporal/budget_authorization_test.go` | 转 current marker/options；退 false-finalization 正向模式，补 child 拒绝；预算/绝对 deadline/溢出/取消有效断言不变 |
| `internal/imageagent/temporal/approval_ack_test.go` | 一个原 marker mock 区间校准；lost ACK/取消不能越过已发布批准断言不变 |
| `internal/imageagent/temporal/image_set_execution_test.go` | 仅现有 partial-result 测试改用当前纯 digest helper；全部 effect 关闭前不得提供结果断言不变 |
| `internal/imageagent/temporal/image_set_policy_test.go` | 原 `nativePolicyAdoptionWorkflow` fixture 去私有位/混合摘要；真实 policy/adoption/receipt 断言不变 |
| `internal/imageagent/temporal/image_set_selection_test.go` | 原 `selectedSetApprovalWorkflow` 同类 fixture 改接；Selection/digest/pending 断言不变 |
| `internal/imageagent/temporal/manual_acceptance_test.go` | `EnsureStaged` fixture 改现有 Normalize 并原样传错；原验收断言不变 |
| `internal/imageagent/temporal/effect_recovery_workflow_test.go` | 追加 late marker 旧版本拒绝测试；原当前成功通知/UNKNOWN/无重复派发/fence 断言零改 |
| `internal/imageagent/temporal/slot_effect_v3_activity_test.go` | 仅一个 direct-child fixture 显式 true finalization，原 BudgetAuthorization=false 保留；其它函数零改 |
| `internal/imageagent/temporal/slot_effect_activity_test.go` | 两个当前安全测试仅 child fixture 补 true；旧 false 成功/空 phase 正向模式改入口拒绝且零 Activity，原 true 传播/终结 phase 断言保留，见 §6.1 |
| `tests/legacy_imageagent_manual_runtime_retirement_test.go` | 追加本批完整闭包 AST/调用顺序/保留 guard；#994/#996/#998 原函数零改，不减安全断言 |
| `internal/imageagent/temporal/testdata/current-organization-parent-prefix.json` | 新增普通单测合成 SDK history prefix；baseline/candidate 都必须回放成功，非真实历史 |
| `internal/imageagent/temporal/testdata/retired-default-version-parent-prefix.json` | 新增普通单测合成旧默认 prefix；baseline 成功、candidate 必须在预计 SDK 版本点失败，不接受解析错误冒充拒绝 |
| `docs/architecture/legacy-imageagent-history-protocol-closure-retirement-boundary.md` | 本文：边界/矩阵/验证/状态；准确 HEAD、CI 和 review 证据写 PR |
| `docs/architecture/README.md` | 最小当前索引，保留原正文/历史 |
| `docs/refactoring/legacy-register.md` | 最小当前登记，保留历史/计数/勾选和旧证据 |
| `docs/superpowers/specs/2026-08-27-image-agent-recovery-compatibility-design.md` | 只追加本批旧历史要求失效及当前保留注记，原正文不改 |
| `docs/superpowers/plans/2026-08-27-image-agent-recovery-compatibility.md` | 同上，原步骤/勾选不改 |
| `docs/superpowers/specs/2026-08-28-image-agent-budget-authorization-design.md` | 同上，区分旧父协议和仍有效预算事实 |
| `docs/superpowers/plans/2026-08-28-image-agent-budget-authorization.md` | 同上，当前预算安全要求保留 |
| `docs/superpowers/specs/2026-08-29-image-agent-external-effect-finalization-design.md` | 同上，取消/绝对 deadline/finalization 不降级 |
| `docs/superpowers/plans/2026-08-29-image-agent-external-effect-finalization.md` | 同上，旧正向模式失效，不改历史步骤 |
| `docs/superpowers/specs/2026-08-29-image-agent-external-recovery-workflow-design.md` | 同上，当前 durable recovery/Signal owner 保留 |
| `docs/superpowers/plans/2026-08-29-image-agent-external-recovery-workflow.md` | 同上，原恢复安全要求/历史证据保留 |

清单外新增实际消费者、检查变化、公共载荷变化或必须扩大生产抽取时，先提交具体根因和最小补丁提案，再取得准确例外；不能扩成“所有旧 V3”或顺手整理其它包。

## 6. 实施与必要验证

设计阶段只检查文档路径/索引、33路径分类及 diff，不运行真实 workflow/provider/服务。源码获准后先建立能捕获目标缺口的测试，再做最小删除/抽取；范围内在同一主要 PR 连续完成，不以测试文件数拆批。

### 6.1 原检查处置与有效断言保留

| 原检查/fixture | 允许转换 | 不能降低 |
| --- | --- | --- |
| `workflow_test.go` 的 `newWorkflowEnv`、`newV3BlockWorkflowEnv` 及下列使用既有 marker 的当前流程测试 | 同 ID mock 的支持范围从 Default..1 改 1..1；旧简化 fixture 改成当前真实 terminalization/proof。当前流程不能靠返回 Default 绕过 finalization | 原 actor/org/幂等/预算/账本/pending/取消/无第三 child/保留 sibling/回执/UNKNOWN/恢复断言 |
| `TestManualWorkflowUpdateActionIDIsIdempotentAndRejectsConflictingReuse`、`TestV3InitialChildFailurePersistsRecoverableBlockedSlotIdentity`、`TestManualWorkflowRejectsCancelAfterApprovalPublicationBeforeProjection`、`TestManualWorkflowCancelSupersedesFailedApprovalCommand`、`TestManualWorkflowWrongActorCancelUpdateDoesNotPoisonActionID` | 上述当前 marker/fixture 改接；私有 wire/位改为当前 helper 与执行身份 | 原身份、同键异载荷、批准发布边界、blocked slot、取消 action owner |
| `TestManualWorkflowCancellationStartsNoThirdChildAndKeepsCompletedSibling`、`TestManualWorkflowV3CancellationWaitsForStartedSlotFinalization`、`TestManualWorkflowKeepsBlockedCancellationOpenWithAccurateReceipt`、`TestManualWorkflowDoesNotProjectCancelledForUnterminalizedEffect`、`TestManualWorkflowCancelUpdateAcksAfterCancelledProjectionPersistence`、`TestManualWorkflowCancelUpdateResumesAfterTerminalPersistenceFailure` | 同上；若原 delayed fixture 假设 legacy 不等待，可改成明确未发送/已取消且有 `ProviderNotDispatched` 证明的当前 child。不能把付费已成功结果伪装为可丢弃 | 已开始 effect 必须终结；成功 sibling 保留；未终结不投影 cancelled；持久化后 ACK/Resume、原调用与防重断言 |
| `TestManualWorkflowStartsExternalRecoveryForUnterminalizedEffect`、`TestManualWorkflowKeepsBlockedWhenRecoveryStartFails`、`TestManualWorkflowPersistsSecondaryRecoveryStartFailureWithRealRepository`、`TestManualWorkflowStartsIndependentRecoveryOwnersForEachBlockedEffect` | current1 marker 与当前恢复 fixture | recovery owner/真实仓库状态/失败保存/独立 effect 身份/无重复发送 |
| `TestManualWorkflowQueryRecoversPersistedPartialSlotProgress`、`TestManualWorkflowReplacePathUsesRealProjectionActivityAndRepository`、`TestReplacementBusinessValidationEnforcesBlockedActionPolicyForNewHistories`、`TestRecoveryHandoffBlocksRejectReplaceAndRetryForNewHistories` | current1、去私有协议位/旧选择器 | 当前严格规则/真实投影/持久化 partial progress/handoff 拒绝 |
| `TestWorkflowEffectOwnerFencesEveryTerminalRunStatusBeforeExecution` | 该旧默认协议的“持久化成功前就 fence”正向要求退休；转换为当前成功后 fence 的验证，并保留 `TestWorkflowEffectOwnerDoesNotFenceFailedTerminalPersistence` | 不能因持久化失败提前封死当前 owner，当前 terminal fence 必须有效 |
| `TestPersistWorkflowFailureProjectsTerminalStateIdempotently`、`TestRestartedExecutionCanRepersistProjectionAndReplayedSlotResult`、`TestRunProjectionCommitIDIsScopedToTemporalExecution` | V1 调用改 V2 + execution-scoped commit，去无执行身份 fallback 的正向断言，补缺身份失败 | tenant/terminal/idempotency/重启执行身份隔离和已保存结果可重投影 |
| `TestRegisterWorkerUsesExactModeBoundWorkflowAndActivitySets` | 仅从 expected 删除 V1 failure 注册；断言其缺席 | 其它精确注册集/模式隔离原样保持 |
| `workflow_test.go` 的 `TestImageSlotWorkflowFailsClosedForMismatchedOrEmptyExecutorResult` | direct child fixture 仅补 `ExternalEffectFinalization:true` | 四类非法返回、`invalid_slot_result`、结果 slot/attempt 原断言全部保持 |
| `TestRetiredActivityWireFailsBeforeAnyActivity`、`TestRetiredApprovalWireHasNoDigestOrPublicationKeyFallback` | 退不可再构造的私有 selector，用实际 SDK 版本边界和当前 key/digest guard 接替 | 不能保留已删类型只供测试；真实 SDK 错误需辨因，无静默 fallback/新 Activity |
| `history_replay_test.go` 的旧 `*Probe`、`markerVersion`、marker组合/PreFinalization/LegacyScoped/RecoveryStart 默认关闭等正向测试 | 退休旧支持要求；当前 V3/digest/bounded-key 有效断言改接当前 helper/SDK fixture。复用 `readHistoryFixture`、`historyVersionChangeIDs`、`scheduledActivityNames` 等通用解析能力 | 当前 key 长度/算法、SDK 顺序、当前1回放；普通 mock/probe 不能声称真实 history 回放 |
| `TestImageSlotWorkflowV3VersionsCancellationDrainActivityOptions`、`TestImageAgentWorkflowBudgetDeniesRepairBeforeChild` | 去旧 options 模式；current1 + 原预算拒绝场景；新 false-finalization child 入口拒绝 | heartbeat/WaitForCancellation、预算拒绝无派发、原 deadline/grace/溢出安全断言逐字保留 |
| `budget_authorization_test.go` 的 `TestImageSlotWorkflowV3BudgetDeadlineBlocksWithoutActivity` | 仅 child fixture 补 `ExternalEffectFinalization:true` | `BudgetElapsedCode` 和零 Activity 原断言保持，不调整 deadline/预算值 |
| `slot_effect_activity_test.go` 的 `TestImageSlotWorkflowV3PreservesInvalidPersistedPolicyClassification`、`TestImageSlotWorkflowV3RejectsInvalidMainCandidateCount` | 仅 child fixture 补 `ExternalEffectFinalization:true`，不改 BudgetAuthorization | 原错误分类、候选数、blocked/result 断言全部保持 |
| 同文件 `TestImageSlotWorkflowV3ThreadsExternalEffectFinalizationToActivityInput` | 退旧 false 成功/空 phase 正向要求；false 改验证入口拒绝且 Activity 零调用 | 当前 true 传播、accepted 与 `PublicationComplete` 原断言保持 |
| `TestManualWorkflowApprovalLostACKCannotBeSupersededByCancel` | 仅既有 marker mock 的范围校准 | 全部 lost ACK/已批准后取消不得 supersede 原断言 |
| `TestSetWorkflowOffersKnownPartialResultsOnlyAfterEveryEffectCloses`、`nativePolicyAdoptionWorkflow`、`selectedSetApprovalWorkflow`、`EnsureStaged` | §5 纯 helper/fixture 改接 | 原所有有效断言不变 |
| `TestImageSlotWorkflowV3StartsFreshActivityAfterFinalInnerAttemptLosesPublicationResponse` | direct child 只补 `ExternalEffectFinalization:true`；BudgetAuthorization=false 与原持久化 fixture 保留 | 原零 GenerateCalls、Finalize/Completion/claim-owner 数量、lease/retry 身份、无重新生成不变；下层 “WithoutWire” Activity 取消安全测试零改 |
| recovery 新测试与本批追加 guard | 检查 late marker 保持 reconcile 后、旧版本不再通知；覆盖 14ID/15点/顺序、旧声明/注册/四便利方法缺席、当前公共载荷与安全路径保留 | 不改变原三个退休批次 guard，不加 missing-skip，不改 workflow/inventory/其它检查 |

上述表允许为明确当前场景建立准确的 mock/proof，不授权放宽业务断言或扩大状态机。未列函数、测试框架和原检查逻辑零改；公共 DTO/JSON 原检查不改。发现表外的真实同根失败必须先报告准确例外。

### 6.2 回放、故障与证据

1. **普通 SDK 回放 RED→GREEN：** 当前 repo 无 tracked history JSON。新增 §5 两个明确标为合成的 parent prefix，直接调用 pinned SDK `WorkflowReplayer`，不建 runner/服务器/采集工具。先在 baseline main46a 验证两份都格式正确且能回放；candidate 当前1成功，旧默认在预计版本点被拒绝。解析、未知 Activity、字段错误等不算预期拒绝。若无法构造有效 baseline fixture，报告具体阻碍，不改检查掩盖。
2. **单测不冒充回放：** testsuite 的 GetVersion mock 可能绕过 SDK 区间验证；普通 mock 只验证当前控制流/副作用。用真实 SDK Replayer 区分版本拒绝，late recovery 用有界单测核对原时点和拒绝后的无通知，不宣称覆盖所有历史。
3. **当前事实与安全：** 复用原相关包测试；当前执行身份、V2 failure idempotency、批准 digest/key、预算 false 恢复、取消/绝对 deadline、UNKNOWN/恢复与 ACK/Resume 必需通过。必要 race 只覆盖本次受影响 workflow/owner；不重新建设故障矩阵或重复跑无关全量测试。
4. **原 CI：** 稳定源码候选运行原 Task Processor CI，准确 HEAD/attempt/整 run 结果记录在 PR。开发测试/合成 fixture、独立代码复核、源码 CI、真实运行验收明确分开，不能互相替代。当前入口做必要仅编译，不启动其进程或执行 canary/provider。
5. **既有数据与平台：** schema/持久化/依赖/workflow/inventory 零改核对；真实历史、队列、当前部署、provider、用户试用及运行续跑均 NOT_RUN，旧任务存在性 UNKNOWN。不借测试访问既有 DB/Redis/Temporal、凭据、文件或实例。

## 7. 交付、复核与停止点

1. 当前先交付三文档、准确设计 HEAD、原 CI 与适用独立设计结论。新范围只有 design，源码状态 `SOURCE_NOT_STARTED / NOT_READY`。用户确认必须包含 §4.2 的历史 hard-cut/旧 child 取舍，不能把设计 PR CI 绿色当确认。
2. GitHub 自动复核若启动失败且没有正在运行的复核，可唯一重试；有适用复核正在运行不重复触发。**未经用户本轮授权不默认新增 AI 互验线程；设计与源码最终复核授权分开。** 无有效适用结论则如实待复核，不把 Completed/空 reviews 当通过。
3. 用户确认及适用准入后，同一主要 PR 将状态改为 `IMPLEMENTATION_READY / IMPLEMENTING`，按33路径整批实施。设计冻结后不重开无关架构；finding 先分类，真实 Must 缺陷才阻止交付，准确清单外例外先报告。
4. 源码形成稳定候选后一次完整最终交付检查：实际完整 diff、SDK 当前/拒绝边界、调用顺序、共享预算 false、JSON/hash、原有效断言、所有当前能力和已有证据；修复后只查增量。开发自检/另一 AI 复核均不等于用户产品验收。
5. 当前没有 merge/runtime/关单授权；若用户后续授权正常合并，绑定最终准确 HEAD/expected-head，遵守保护规则，再确认实际 merge SHA 的 push/main **整 run** 成功。变化 HEAD 按现行准确授权规则办理，不 bypass/强推/改保护。
6. #301 当前段最小更新，历史后缀从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起逐字保持，历史勾选不动、正文 ≤65536 字符、始终 OPEN。运行/真实历史处置和关单另行授权，不在本批附带。
