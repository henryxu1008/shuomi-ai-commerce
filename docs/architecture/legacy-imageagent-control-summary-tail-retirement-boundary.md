> #1044 / #301 用户已明确确认冻结设计 `44fa04b6fb8cc151974fe28ce80c3233f7b9af4b` 的 §1–§7（含 §4.2、§6.1）：**USER_CONFIRMED / IMPLEMENTATION_READY**。按下方14路径在本同一主要 PR 整批实施；七对象退出，四个原测试函数的有效断言直接转接现有批准业务校验及当前摘要。仅窄顶注校准实施阶段，以下完整设计正文冻结保留；最终源码复核、准确实现 HEAD/原 CI 与用户确认见 PR，不以设计证据代替。Store 组暂缓，当前能力/wire/hash/marker/资金/恢复保持；不合并、不部署、不操作既有实例或数据、不关闭 #301，无新试用前置。

# ImageAgent 旧控制、摘要合同与测试尾部完整退休边界

## §1 用户结果、Design Basis 与授权阶段

用户已授权先做完整设计：一次覆盖七个对象、相关测试和旧 KEEP 替代，同一主要 PR，不机械拆批；安排一名独立 Reviewer，设计与原 CI 通过后交用户确认。**确认前不实施。** Store 旧来源准入/审计辅助那组明确暂不做。

- Execution Issue：[#301](https://github.com/henryxu1008/shuomi-ai-commerce/issues/301)，保持 OPEN。
- 唯一 Writer：聊天 `01a194f8-c65c-7231-9e0d-acb39710885c`；独立工作区 `issue301-browser-tail-946/task-processor`；分支 `codex/issue301-image-control-summary-retirement`。
- 调查与开工基线：main `161b1de17e1deb769fd661162a9fb38786ab003e`，已含 #1041。已检查当前 Issue、聊天、worktree 与开放 PR，无同范围 Writer；#1038/#1043 归原 owner。
- Design Basis：**Independent Architecture**。本设计精确限定旧公开 Go 类型及旧 KEEP 的退休边界；不重开全局图像架构。当前 **DESIGN_ONLY / DESIGNING / SOURCE_NOT_STARTED / NOT_READY**；评审和 CI 不自动授予实施权限。
- 本轮只交付 §5 的 D1–D4 四份设计/导航文档；P/T 及 D5–D8 都是待确认实施范围，当前零修改。

使用者是维护当前单主图、图组、Artwork 和回归测试的开发者。可见交付是七对象到实际消费者、当前 owner、有效断言和旧 KEEP 的完整映射；未来实施后，测试直接验证当前路径，不再靠旧摘要或转发壳保活。**本轮没有证据表明当前用户业务被这些尾部阻塞**，不新增产品功能、试用前置或验收工具。

Authority：本轮用户决定、[全新系统产品基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[项目边界](project-boundaries.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[当前图像产品边界](product-image-agent-v1.md)、[Artwork](pod-design-agent-v1.md)，以及已批准的 [#998 业务 Signal 退休](legacy-imageagent-business-signal-closure-retirement-boundary.md)、[#1001 历史协议退休](legacy-imageagent-history-protocol-closure-retirement-boundary.md)、[#1033 通用执行退休](legacy-imageagent-generic-execution-closure-retirement-boundary.md)。旧文档按其最新精确替代注记解释，不恢复已取消行为。无 UI/IA/页面或交互改动，Figma 增量 **N/A**。

Must：当前身份/组织/成员归属、命令幂等与回执、显式选择/版本/digest、UNKNOWN 不重发、取消/恢复、当前 JSON/hash/wire/marker、持久化及资金边界保持；七对象和专属测试调用完整退出；有效测试断言归当前 owner。Should：复用现有函数、fixture、AST guard、原 CI。Out of Scope：Store 那组、当前功能扩展、状态机/恢复/公共 wire 重写、新持久化或数据迁移、新框架/runner、部署、真实资源操作和关单。未新增 Accepted Risk。

## §2 前批 KEEP 精确替代与 Legacy 决策

| 原依据 | 用户未来确认本设计后替代的准确范围 | 继续有效的边界 |
| --- | --- | --- |
| #998 设计 §4.1 明文要求保留 `initialActionPhase` | 只替代这个已无调用的私有 switch helper 的 KEEP；当前各 handler 已直接向 `prepareAction` 传入原 phase 常量 | 四个 Update/Validator、Resume、恢复 Signal、phase 常量和值、action/pending/ack/recovery 全保留 |
| #1001/#1033 的当前摘要与有效 V3 保留 | 只退出 `summarizeResultsV3 → slotProjectionsV3` 这个无生产入口的旧子图及无当前调用的校验/克隆壳，不撤回其业务安全要求 | `summarizeCurrentResults → summarizeResults/slotProjections`、当前 child DTO、marker、真实恢复/发布与结果准入保持 |
| 原 core-manual 计划的 `AssetRef`、assisted-planning 的 `[]AssetRef`、automatic-repair 的候选/兄弟 `AssetRef` | 只替代七对象中的孤立 `imageagent.AssetRef` Go 值声明及“仍须实现/保活这个旧值”的建议 | 当前 `AuthorizedAsset`、`AssetCandidate`、`StagedAssetRef`、durable identity、ImageSet/Artwork 和已有规则保持；不恢复旧规划/自动修复管线 |
| 两个当前测试文件仍通过旧函数构造断言 | 退出旧转发/摘要调用，改为 §6.1 的实际生产函数和当前局部输入 | 全部身份、版本、digest、显式选择、错误码、投影恢复、AllowedActions 和直接重试拒绝断言保留 |

Legacy decision：**RETIRE**。Reusable behavior：批准业务校验和当前摘要/投影已经在正确 owner 内实现，本批原样复用，不新增生产 EXTRACT、alias、转发壳或第二事实源。Current owner：ImageAgent domain/Service、既有 Temporal action/summary owner、Repository/Asset/Resource 各自原 owner。Cutover/deletion condition：用户确认 §1–§7（含 §4.2/§6.1）后，在同一 PR 收齐 §5，原必要检查及源码最终复核完成；合并另行授权。

## §3 七对象与真实调用闭包

基线检查覆盖 tracked Go 的 internal/cmd/tests/tools/scripts（含测试和 build-tag 文件），并核对直接设计引用。声明命中不算消费者；不按 `V3`、`Signal`、`Asset` 等名称批量退休。

| ID / 对象 | 当前位置、实际消费者 | 决定及当前替代 |
| --- | --- | --- |
| O1 `AssetRef` | `internal/imageagent/model.go`；Go 中只有声明，无字段、参数、构造或测试引用；三份旧计划有历史描述 | 删除孤立类型；当前资产模型不改 |
| O2 `clonePendingCommand` | `internal/imageagent/service.go`；仅声明，生产/测试均无调用 | 删除私有浅拷贝 helper；`PendingCommandReceipt` 及原仓库/Service 行为保持 |
| O3 `initialActionPhase` | `temporal/workflow.go`；仅声明，生产/测试均无调用 | 删除旧 phase 选择壳；各真实 handler 对 `prepareAction` 的原 phase 实参零改 |
| O4 `validateUpdateIdentity` | 同文件；仅声明，生产/测试均无调用 | 删除组合壳；仍被真实 handler 使用的 `validateCommandOwner`、`validateCommandRevision` 保留 |
| O5 `validateApproveResultsBusiness` | 同文件；生产无调用；T1/T2 共八个调用表达式，只转发到下一列 | 删除旧 wrapper；测试直接调用 `validateResultAdoptionBusiness(resultAdoptionIntent(signal))` |
| O6 `summarizeResultsV3` | 同文件；仅 T1 的两个测试函数调用，生产无调用 | 测试改接 `summarizeCurrentResults` 和 `SlotWorkflowResult`；不新建摘要转换器 |
| O7 `slotProjectionsV3` | 同文件；唯一调用方是 O6，无其它生产/测试调用 | 与 O6 整退；真实 `slotProjections`、`SlotWorkflowV3Result`/child 合同原样保留 |

当前批准路径：Service/Client → Temporal Update → `workflowUpdateState.register` → `handleApproveResults` / native policy adoption → `handleResultAdoption` → 原 owner/revision 校验、`prepareAction`、`validateResultAdoptionBusiness`、`runActionAttempt`、原发布与持久化。当前 summary 在初始执行、retry 和恢复刷新中调用 `summarizeCurrentResults`，由当前 `SlotWorkflowResult` 与 `slotProjections` 构造投影。

架构问题是**当前实现已有自己的 owner，但旧未消费辅助及测试调用仍与它并存**。O5 是一行转发，不据此虚构当前安全缺陷；O6/O7 的测试也不能代表当前生产摘要已执行。本批让有效断言直接落在实际函数，不创建新的 workflow/service/事实 owner。

## §4 合同、事实、执行与副作用边界

### §4.1 未来实施顺序与不变量

1. 按 T3 加入有界禁入 guard，先在七对象仍存在时记录真实 RED；设计阶段不提前写该测试或删源码。
2. 将 T1/T2 的有效调用改接现有当前函数。O5 使用当前 `resultAdoptionIntent` 类型转换，不新增 wrapper；O6 测试使用当前 `SlotWorkflowResult.Execution` 携带原 slot ID/attempt，原 status/error code 等输入语义不变。
3. 删除 O1–O7；只有 O6→O7 内部边同时退出，不留下改名版本、测试文件中的同名重建或旧结果转换层。P3 其余函数体、handler 注册、phase 常量/实参、marker 调用位置与顺序零改。
4. T1 的仓库 round-trip 保留原 memory/GORM 两个分支及其 InitializeRun/GetProjection、scope/member/错误/AllowedActions/retry 拒绝断言；不新增 fixture、表或 repository。
5. 当前 Update/Resume 的 owner 校验、同 ActionID 同载荷/冲突、单 pending、持久化 ACK、response-loss/Resume、取消/UNKNOWN/publication fence/lease/deadline 保持；不利用这批清理修补或重写状态机。

### §4.2 JSON/hash、持久化与历史边界

| 边界 | 明确规则 |
| --- | --- |
| 公共序列化 | O1 未进入当前字段/参数/编码；本批不删任何 current struct 字段。`WorkflowInput`、四个 `*Signal`、`ResumeCommandInput`、`SlotWorkflowV3Input/Result`、`ExecuteSlotV3`、恢复/Activity/child DTO、`PendingCommandReceipt`、`CommandAcknowledgement` 的名称/字段/顺序/tags/nil空值均保持 |
| 当前摘要和 fingerprint | 不修改 `summarizeCurrentResults`、`summarizeResults`、`slotProjections`、`projectionResultDigest`、`updateFingerprint`、visual digest、publication keys、版本常量或编码。旧摘要仅由局部测试调用，不承诺保留其 Go API；T1 仍核对原错误码和允许操作 |
| 事实 owner / 事务 | Repository 仍拥有 run/plan/slot/projection/commit/pending receipt；Asset/Resource/钱包/V3 effect ledger 仍沿原事务/锁/receipt/fence。无表、列、Goose、SQL、grants、物理记录或当前模型 hash 变化 |
| 回放和注册 | 本批不新增/改变历史支持范围，不重开 #994/#996/#998/#1001/#1033 的 hard-cut。删除的是无运行入口函数；原 GetVersion marker、取值/顺序、workflow/Activity/task queue/child/recovery ID 与注册保持。真实历史/rollout 未验证，不能写为 PASS |
| 失败、重试和外部 effect | 七对象无当前业务入口、无新的 provider/DB/消息动作；真实成功/失败/响应丢失/重启/取消/并发继续由原 action/recovery owner 处理。不会新加自动重试、补偿、对账、UNKNOWN 解释或资金释放逻辑 |
| 真实资源 | 不读取/重写既有历史、数据库、凭据、配置、缓存、Profile、队列或对象；不停止/重启/部署/清理实例。运行与产品验收 NOT_RUN，物理运行状态 UNKNOWN |

如发现七对象有清单外真实消费者或当前外部可观察契约，应停止该部分并给出证据；不得用旧文档/测试推定兼容义务或临时增加 alias/fallback。

## §5 同一主要 PR 的完整范围矩阵

未来确认后一次覆盖 **14 唯一路径：3 生产、3 测试、8 文档**。小切片仅作提交/验证边界；不把源码、测试、旧依据拆成多个 PR。本轮设计阶段只提交 D1–D4。

### §5.1 生产：3 路径

| ID | 路径 | 准确允许差异 |
| --- | --- | --- |
| P1 | `internal/imageagent/model.go` | 只删 O1；当前全部其它模型和序列化字段不变 |
| P2 | `internal/imageagent/service.go` | 只删 O2；身份、Service 方法、receipt 和全部其它 helper 不变 |
| P3 | `internal/imageagent/temporal/workflow.go` | 只删 O3–O7；无其它生产逻辑、常量、marker、handler 或 wire 改动 |

### §5.2 测试：3 路径

| ID | 路径 | 准确允许差异 |
| --- | --- | --- |
| T1 | `internal/imageagent/temporal/workflow_test.go` | 仅 §6.1 的三个原测试函数中 O5/O6 调用和当前局部结果输入换接；全部其它函数零改 |
| T2 | `internal/imageagent/temporal/image_set_selection_test.go` | 仅首个 `TestSetApprovalRequiresExplicitCompleteSelection` 的 O5 调用换接；原其它函数零改 |
| T3 | `tests/legacy_imageagent_control_summary_retirement_test.go`（未来新增） | 七对象完整声明/调用禁入、当前 owner 必需及冻结文档后缀 guard；复用现有 AST/读取模式，不新增工具或检查框架 |

### §5.3 文档：8 路径

| ID | 路径 | 当前设计 / 未来实施处置 |
| --- | --- | --- |
| D1 | `docs/architecture/legacy-imageagent-control-summary-tail-retirement-boundary.md` | 本完整设计；确认后正文冻结，后续仅加准确当前顶注 |
| D2 | `docs/refactoring/legacy-register.md` | 仅当前七对象设计/退休顶注；原正文/计数/勾选完整保留 |
| D3 | `docs/architecture/project-boundaries.md` | 仅精确当前 owner/旧 KEEP 导航顶注；原边界完整保留 |
| D4 | `docs/architecture/README.md` | 新设计的显式反引号文件名索引，保留原导航完整后缀 |
| D5 | `docs/architecture/legacy-imageagent-business-signal-closure-retirement-boundary.md` | 确认实施时才加窄顶注，仅替代 `initialActionPhase` 的旧 KEEP；其余当前 Update/recovery 不变 |
| D6 | [2026-08-26-image-agent-core-manual-workflow.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-26-image-agent-core-manual-workflow.md) | 确认实施时才加旧 `AssetRef` 准确保留/实施建议失效注记 |
| D7 | [2026-08-26-image-agent-assisted-planning.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-26-image-agent-assisted-planning.md) | 同上，只针对孤立旧 `AssetRef`，不恢复/实现旧规划管线 |
| D8 | [2026-08-26-image-agent-automatic-repair.md（历史）](https://github.com/henryxu1008/shuomi-ai-commerce/blob/f936702227f871e122d478276868e63a6efcc4fc/docs/archive/implementation-plans/2026-08-26-image-agent-automatic-repair.md) | 同上，只针对孤立旧 `AssetRef`，不恢复/实现旧自动修复 |

D2–D8 的原 Git blob 仅允许追加最前方窄顶注/导航，历史后缀逐字保留（工作区核对可归一化 CRLF→LF）；不全局 replace 历史。用户确认后的准确设计 HEAD 将作为 D1–D8 冻结后缀依据；滚动 HEAD/CI/评审状态写 PR，不为证据改历史正文。

### §5.4 KEEP / 零差异

当前单主图、图组/Artwork、当前产品/Asset/SourceSelection/授权/预算/计费、Temporal SDK/Update/Resume/recovery、现有图像和资金/UNKNOWN/取消/恢复、所有 current wire、有效历史 fixture、JSON/hash、当前 Store 服务/观察及 #1041 资源结算全部保持。`SlotWorkflowV3Result`、`StagedAssetRef` 等当前模型不能按旧名称禁入。

Store 的 `source_access.go`/专属测试、`audit.go` 及其闲置辅助明确不做；不接管 #1038/#1043。全部 workflow、完整 inventory（CURRENT/历史/计数/ceilings/算法）、既有其它 guard、依赖/锁/保护、schema/配置/部署文件零改。

## §6 验证与验收方法

本轮设计只核对对象/消费者、14 路径唯一性、引用/链接、历史后缀、四文档准确 diff，并运行原适用文档检查及原 PR CI。不改源码或测试，不运行应用/provider/容器，不读取真实配置/凭据/数据。一名独立设计 Reviewer 只读核对完整设计与真实调用。

用户确认后实施才执行 TDD：先 T3 在原七对象存在时 RED，再原样换接 T1/T2、删除 O1–O7，运行相关 domain/Temporal 和当前回归至 GREEN，稳定候选执行原 CI与适用最终源码复核。已有有效证据只复用未变化代码，不人工重复完整 CI，不新增故障矩阵、验收 session 或 runner；任何真实运行/产品验收保持 NOT_RUN。

### §6.1 原有效检查的准确转接

| 原测试 / guard | 允许改动 | 原有效断言 / 不允许变化 |
| --- | --- | --- |
| T1 `TestApprovalUpdateValidatorKeepsCompletionChecksInHandler` | 四处 O5 调用改 `state.validateResultAdoptionBusiness(resultAdoptionIntent(signal))` | run/actor/action shape 与 owner 校验不变；原 revision/state/digest 空白/不符错误类型、capacity 和其它子测试/输入/断言保持。不是把业务校验移进 Update Validator |
| T2 `TestSetApprovalRequiresExplicitCompleteSelection` | 四处同样换接当前生产业务函数 | 首次没有显式选择必须拒绝；合法完整选择成功；run/revision/result 替换与 source version 漂移拒绝；原 fixture、CloneImageSetCommand、scope/selection/digest 和全部断言保持 |
| T1 `TestSummarizeV3ResultsPreservesPublicationUnknownBlockCode` | `summarizeResultsV3` 改 `summarizeCurrentResults`；局部 `[]SlotWorkflowV3Result{Published: ...}` 改 `[]SlotWorkflowResult{Execution: ...}`，原 slot ID/attempt/status/error code 保持 | 原 publication-unknown Block.Code 和 AllowedActions（edit-plan/cancel）断言不变；不只测试新的 helper 或删除错误后果 |
| T1 `TestInvalidPersistedV3PolicySurvivesProjectionRefreshAndRejectsDirectRetry` | 同样换接当前摘要/局部结果输入，使用已有 current Execution slot ID/attempt | 原三类 phase/policy/future-policy 输入与规范化期望、memory/GORM 两仓库、InitializeRun/GetProjection、原组织/member/scope、refreshed Block.Code/AllowedActions 和直接 retry 拒绝全部保持；原其它语句不改 |
| T3 新 guard | 明确 O1–O7 在原 owner 及整个 ImageAgent Go 子树均无声明/调用，包含测试/build-tag文件；失败关闭，不允许改名转发或移入 `_test.go` 保活 | 解析器/必需读取失败不得 skip；按独立标识符检查，不误禁 `StagedAssetRef` 或整个 V3。当前摘要/业务校验/handler/phase/child/receipt 必需；D1–D8 原文冻结后缀保持 |
| 原其它全部测试 / guard / golden / replay fixture | **零改** | 特别保留实际生产初始摘要错误码/AllowedActions、handler late-state校验、ActionID冲突、单pending、ACK/Resume、UNKNOWN/取消/恢复、绝对deadline、计费/审批幂等、JSON/hash和marker回放断言；不改 expected 来绕过失败 |

T3 不为设计阶段补造空实现。发现清单外检查要求七对象继续存在，先说明准确冲突与最小补丁；不能加 skip、伪造消费者或降低原安全断言。未证明当前 Must 无法满足的理论完善按规则分类，不自动扩成新系统。

## §7 准入、评审、交付与停止点

1. 当前四文档设计在同一主要 PR 交付，Issue 记录本轮接单和范围。当前未进入正式源码 Ready；设计和原 CI 即使通过也仍为 USER_CONFIRMATION_PENDING / SOURCE_NOT_STARTED。
2. 用户已授权本轮一名独立设计 Reviewer：稳定候选后只读检查全设计、七对象消费者、旧 KEEP 精确替代、§4.2/§5/§6.1、当前能力和 Store 暂缓边界。不得自行新增第二名 Reviewer/Writer、工具或子任务线程。
3. Finding 先按 AGENTS 的 BLOCKER / IMPLEMENTATION_TEST / BACKLOG / ACCEPTED_RISK / NOT_APPLICABLE 分类，指出当前 Must/具体后果/阻塞层级；最多两轮正常设计 review。修复后同一 Reviewer 只复核相关增量，不重复未变化设计。
4. 设计复核、准确候选原 PR CI 收敛后交用户确认 **§1–§7（含 §4.2/§6.1）**。没有用户明确确认，不登记 USER_CONFIRMED/IMPLEMENTATION_READY，不改 P/T/D5–D8。
5. 未来确认后在同一 PR 整批实施；必要自检/提交/推送/CI修复可在批准范围连续完成。源码最终独立复核、合并/实际 main CI、部署、运行/产品验收和关单分别判断，设计 review 不替代源码最终复核。
6. #301 OPEN；不部署、不操作既有实例/数据/凭据/缓存/进程、不关闭父 Issue。不增加内部试用前置，不把本维护宣称为用户业务或上线验收。

停止条件：新增当前消费者/外部义务；需要改变 current wire/JSON/hash/marker/回放、身份/权限、事实/schema/事务、资金/UNKNOWN/恢复/副作用；有效断言无法原样转接；需要清单外生产文件或 workflow/inventory/保护/依赖变化；需要触碰 Store 暂缓组或既有资源。应给出根因、当前证据和最小差异交用户决定，不自行扩范围或兼容。
