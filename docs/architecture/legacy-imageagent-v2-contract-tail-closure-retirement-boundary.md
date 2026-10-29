> 本轮用户批准的准确尾部例外（§5.2/§6.1）：按 [PR 提案6488918931](https://github.com/qq550723504/shuomi-ai-commerce/pull/996#issuecomment-6488918931) 将遗漏 `store/slot_effect_v3_repository_test.go` 纳入，删除 V2 JSON-only 保活测试、两项旧字段拒绝改接当前 decoder；HTTP 仅两行 expected 对齐原 durable ErrValidation/GET400，完成回放409与无泄露/无副作用断言保留。生产映射、workflow/inventory及其余检查不改。原33路径精确补为34（14生产/15测试/5文档），不扩大产品范围、不新增兼容层。

> #301 / #996 当前实施：USER_CONFIRMED / IMPLEMENTATION_READY / SOURCE_IMPLEMENTED / IN_REVIEW。用户确认冻结设计25258f5eb的§1–§7（含§4.2、§6.1），同一主要PR收齐34路径；旧V2候选/批准/默认装配与专属尾部按本设计退休。当前durable JSON/hash/有效V3回放恢复、真实组织worker/provider、图像/采集和Store观察同步保留；GitHub workflow/完整inventory/依赖/保护及既有实例/数据/凭据/进程零改，无新试用前置，#301 OPEN。实施/原CI/源码最终复核证据写PR，设计复核不替代源码复核；合并/部署/关单另授权。

# 旧 ImageAgent V2 合同、批准链与默认装配残留完整退休设计

本批收净 #994 退出旧手工任务后留下的三组源码：**V2 候选/投影合同、旧批准摘要与发布键、无人调用的默认能力装配**。当前图组、Artwork、采集单主图和 Store 观察同步继续使用当前 owner；不是删除当前 ImageAgent 或 V3 恢复。

## 1. 用户结果、范围与 Design Basis

- 父目标：[Issue #301](https://github.com/qq550723504/shuomi-ai-commerce/issues/301)，保持 **OPEN**。
- 本轮产品决定：用户授权先按 Independent Architecture 形成完整设计，明确 JSON/hash/版本、持久化及 §6.1，设计确认后再实施；不机械拆批。
- 使用者与场景：当前组织图组、Artwork、采集单主图的调用者与维护者。交付结果是当前图像路径只依赖有效持久化对象合同，不再通过测试或默认构造保活已退出的 V2；不新增产品入口。
- 调查基线：main `0269376046c468e4a51b7f2fd4178110871cdb1e`，已含 #994；本分支从该提交开始，同一 Writer、独立 worktree，一个主要 PR。
- **Design Basis：Independent Architecture。** 本批改变混合候选合同和旧载荷的源码接收边界，替代前批 KEEP，横跨 Domain、Temporal、App 和 Repository 消费者；不能仅以死代码删除视作机械重构。
- 当前交付只修改本设计、architecture README、legacy-register 三份文档：`Design Basis: N/A`（不改变运行语义）/ `Ready`。未来源码状态 **DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY**；适用独立设计复核、用户确认及 Issue 读回 `IMPLEMENTATION_READY` 前不得修改正式生产/测试源码。
- 不做：GitHub workflow、完整 inventory、依赖/保护规则变更；既有实例、数据库、对象、Temporal history、凭据、文件或进程操作；部署、队列排空、迁移/重写/清理、真实 provider/canary 执行、产品验收或关单。没有新的内部试用前置。

## 2. Authority、前批 KEEP 替代与风险边界

适用 [AGENTS.md](../../AGENTS.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md) 与 [全新系统基线](../product/greenfield-no-legacy-migration.md)。本批没有 UI/IA/交互变化，Figma 增量 N/A；当前产品能力继续按已有入口和授权开放范围，不因退休设计宣称生成可用或上线。

| 既有依据 | 本批替代的限定 | 继续有效 |
| --- | --- | --- |
| [#994 手工运行链](legacy-imageagent-manual-runtime-closure-retirement-boundary.md) §4.1/§4.2/§5/§6.1 | 冻结 V2 `AssetCandidate` URL/Metadata、旧摘要/键和仅供其成功的 probe 不再保活；默认 capabilities 的泛称 KEEP 不覆盖无调用 wrapper | 当前组织入口、最小值合同、V3 载荷/版本/恢复、当前表、WithWorker canary、图像/采集/Store |
| [#977 HTTP 运行链](legacy-imageagent-http-runtime-closure-retirement-boundary.md) | 全部 V2 合同/回放一律保持的旧实现限定 | 当前 HTTP 严格输入、身份/权限、typed Service、安全输出及真实恢复断言 |
| [2026-08-27 recovery design](../superpowers/specs/2026-08-27-image-agent-recovery-compatibility-design.md) 与 [plan](../superpowers/plans/2026-08-27-image-agent-recovery-compatibility.md) | 保留 V2 candidate-ID-only digest、V2 wire/URL 投影及正向旧协议测试的执行要求 | 当前 V3 staging/finalization、有效版本 marker、批准/恢复/防重/取消/预算不变量；历史正文与证据保留 |
| [当前图像产品](product-image-agent-v1.md)、[图像文本确认](image-text-language-confirmation-v1.md)、[有效方法版本](official-skills-and-image-method-v1.md) | 无 | 当前图组/Artwork/方法/文案与已保存有效 JSON；不能把名称里的 v2 当退休条件 |

Must：不降低组织/原始 member/业务任务归属；不把未验证 URL 当发布资产；不重复 provider dispatch、扣费或批准提交；保留已授权当前 V3 的 JSON/hash/版本/绝对 deadline/取消/UNKNOWN/恢复。范围外：旧 V2 续跑或兼容、历史数据承接、真实环境升级验收。源码退休不是观察到旧运行态已经清空，也不是接受当前安全要求的例外。

## 3. 实际消费者与根因

| 组 | 真实调用与遗留 | 当前正确 owner |
| --- | --- | --- |
| 候选与投影 | `ProductImageSlotExecutor.BuildSlotResult`、Artwork builder → 当前 `AssetCandidate`/`SlotExecutionResult` → `NewSlotEffectV3PublishedResult` → V3 published result；恢复/父 workflow 将 durable result 投影回候选。生产 builder 均产生对象标识。`projection.go` 仍允许 URL-only，`projection_json.go` 仍有 URL/Metadata 双格式；store 的 JSON clone/decode/hash 消费此合同 | `internal/imageagent` 当前 durable candidate / projection；staging/provider 临时材料继续属原 owner |
| 当前拒绝消费者 | `assetpublication/publisher.go` 和 `httpapi/acquisitionImagePublishedURL` 仍显式拒绝候选 URL/Metadata；并非可以直接删除的无用安全断言。后者最终 URL 由 scoped ObjectKey 经过当前 URL resolver 得出 | 当前 domain 校验、Asset 发布 owner、HTTP 只读投影；拒绝断言按 §6.1 改接 |
| 摘要与批准 | 真正 `runImageAgentWorkflow` 已在任何 Activity 前拒绝非 V3 slot/approval。`resultDigestForWire`、`approvalPublicationKeyForWire` 却仍有 V2 fallback，`resultDigest` 已无调用；旧 probe 绕过真实入口，仍把混合旧 marker 视为成功 | 当前 `ResultDigestV3`/`VisualResultDigest`，版本化 V3 幂等键与原批准事务 |
| 默认装配 | resolver 仍声明/赋值 `BuildCapabilities`，全仓没有调用；只剩测试 stub。`buildProductionImageCapabilities` 只被该赋值和两测试引用。真正 resolver 调用 `BuildOrganizationCapabilities` | 当前组织 credential admission → `BuildOrganizationImageCapabilities` → 现有 typed ports/provider |
| 专属尾部 | `hasCandidateAsset` 无调用；V2 golden、旧 probe、URL happy fixture及旧文档继续保活旧设计 | 现有混合测试文件保留有效断言；不新增兼容 alias、替代 worker 或通用验证框架 |

根因是入口 hard-cut 后，共享类型、序列化和测试还同时承载两种设计。处理方式是收窄现有当前 owner，删除旧分支；不再创建新的过渡 owner、wrapper、双读或第二事实源。不同包同名 `sourcing.AssetCandidate`、授权来源 `AuthorizedAsset`、provider `GeneratedAsset` 仍实际使用 URL/Metadata，不属于本批。

## 4. 当前合同、持久化与版本边界

### 4.1 最小合同与删除次序

1. 在现有 `internal/imageagent` 内明确当前值合同：`AssetCandidate` 保留 Artwork、GenerationProof、AssetID、SourceAssetID、Width、Height、Operations、DurableAsset 八字段；删除 URL、Metadata。`SlotExecutionResult` 仍是三字段进程内 builder/聚合结果，保留名称与当前消费者，不新建 alias/复制一套合同。当前字段原 tag 保持；不再承诺该进程内值的旧 V2 直接 JSON。
2. 保留 `SlotEffectV3PublishedResult`、`SlotEffectV3AssetCandidate`、`SlotProjection`、`RunProjection` 和当前 Activity/child 输入输出。只有这些当前 wire / projection owner 的已用 JSON 属于保持对象；旧 V2 `SlotExecutionResult` 载荷没有继续接收义务。
3. `validateProjectionCandidate` 不再以安全 URL 代替 durable identity。所有非空候选继续经当前 Normalize/slot-bound 对象标识校验；`NewSlotEffectV3PublishedResult` 保留其余校验/clone/规范化，删除已无法构造的两旧字段检查。HTTP 与 Asset publisher 同样只删除旧字段读取，保留当前 durable 校验、scope、摘要、来源和提交前拒绝。
4. 删除 `ResultDigestV2`、无调用 `resultDigest`、旧计划级 `publicationKey`、`hasCandidateAsset`。保留共享 `resultDigestSHA256` 和所有当前算法。ForWire helper 不再返回 V2 摘要/旧键：不合法 wire 明确失败（摘要返回原冲突错误，键返回空值），真正入口的 effect 前拒绝不变；不把旧 wire 默认为 V3。
5. 删除 `BuildCapabilities` 字段/默认赋值、闲置 `buildProductionImageCapabilities`。保留 `BuildAI`、`BuildOrganizationCapabilities`、当前 credential/logger/authorizer、`buildImageCapabilities`、`newRoutedOpenAIProductImageProvider`、policy resolver、usage quote/provider 等全部共享实现；它们仍被组织能力装配使用。

### 4.2 JSON、hash、版本与持久化 owner

| 边界 | 保持 / 本批精确变化 |
| --- | --- |
| 当前 projection JSON | 当前 durable candidate 的键名、顺序、类型、值和省略规则逐字节保持：Artwork/GenerationProof、AssetID/SourceAssetID、DurableAsset、Width/Height/Operations；Closure/Slot/Attempt/Candidates/ErrorCode 和 nil/空数组区分不变。保留自定义 projection serializer，不用直接 marshal 进程内 candidate 取代它 |
| 退休载荷 | serializer 删除 URL/Metadata 输出分支。decoder 只在候选对象层用现有 `encoding/json`/RawMessage 识别并拒绝两退休键，再解到当前 typed shape；不是接受后丢弃。URL-only、Metadata-only、与有效 durable 混合、空/null 的旧键均拒绝；覆盖大小写及 JSON 转义键，不能由重复键后值覆盖规避。无别名、迁移或全局 decoder 框架，不顺带改变其它未知字段/数字解析规则 |
| 当前 V3 JSON/hash | published result、generation receipt/proof、final/staging manifest、有效 V3 Activity/child、图组/Artwork/方法/文案合同不改。对有效 durable 的完整 RunProjection/ProjectionCommit 与现有 fingerprint 做前后字节/摘要对照，不能只检查字段存在 |
| 摘要与批准幂等 | `ResultDigestV3`、`VisualResultDigest` 及 SHA256 算法、顺序和已有有效值不改。V3 actionID-only、旧 run-scoped key、bounded SHA key 都可能是有效 V3 replay：保留 `legacyApprovalActionPublicationKey`、现有 marker 条件与字节；它不是本批旧计划级 V2 key |
| Temporal 版本 | 所有现有 GetVersion change ID、调用顺序及有效取值保持；不改 Activity/workflow 名、队列、child/recovery ID。非 V3 slot 或不完整 approval marker 仍在真实入口、failure projection 和任何 Activity 前退出。不得删除 marker 后把旧历史转新 wire |
| 实际持久化 | 当前组织 Repository 是 run/plan/slot/projection/commit 的 owner；V3 ledger 是 effect/receipt/fence owner；Asset transaction 是批准结果 owner。表名 `image_agent_v2_*` 中当前仍用的表、V3 表、schema/权限/SQL/迁移全部不改。新安装不新增表/字段；既有表/载荷不读、不写、不重写、不 Drop |
| 失败/并发/恢复 | CAS/commit fingerprint、事务、action 同键异载荷、approval ack、租约/fence、响应丢失、dispatch/settlement、UNKNOWN、取消/绝对 deadline、finalization 与恢复唯一责任方不变。不新增自动接管、重试 provider 或补偿 |

`cloneProjection` 对合法当前值仍可正常 round-trip；不得因新 decoder 把合法内存值变成零值。旧存量投影在源码 decoder 被拒绝，不提供迁移或兼容入口；不把这一设计当作已执行的数据库清理。

### 4.3 保留能力与副作用

图组、Artwork、采集单主图的现有入口/准入，当前 AI/图像 provider、定价/Resource/钱包、来源采集与浏览器/debug、有效规则与共享合同、Store 观察同步/官方授权、隔离 WithWorker canary 保持。旧 SHEIN 登录/身份回写已由 #985 退休，不恢复。不存在新的网络/provider/DB 操作入口、凭据读取或 worker 默认接管。

## 5. 完整删除 / 保留矩阵

未来源码一次收齐下列 **34 路径**；不是本轮文档交付的已执行清单。14 个混合生产文件、15 个测试文件、5 份文档；没有整包删除，也不增加运行工具。

### 5.1 混合生产文件（14）

| 路径 | 本批准确改动；其它当前行为保持 |
| --- | --- |
| internal/imageagent/model.go | `AssetCandidate` 仅删除 URL/Metadata；其它类型/字段/tag 不改 |
| internal/imageagent/ports.go | 校准四处仍声称 frozen V2 port 存在的注释；接口/输入输出声明保持 |
| internal/imageagent/projection.go | `validateProjectionCandidate` 退出 URL fallback；其余 snapshot/scope/cursor/recovery 检查不改 |
| internal/imageagent/projection_json.go | 保留当前 durable JSON 形状；删除双格式写入/接受，增加候选层退休字段拒绝 |
| internal/imageagent/result_digest.go | 仅删除 `ResultDigestV2`，保留 V3 与共享 SHA256 |
| internal/imageagent/slot_effect_v3.go | 只删转换函数两旧字段读取并校准相关注释；其余 Normalize/manifest/receipt 不改 |
| internal/imageagent/assetpublication/publisher.go | 只删候选两旧字段读取；完整当前 publication 校验/事务保持 |
| internal/imageagent/temporal/workflow.go | 删除旧摘要/键 fallback、无调用 resultDigest；ForWire 无效输入不回退；只收窄已被入口阻断的 V2 summarize 分支。真实入口/有效 marker/当前 V3 算法和其它协议不改 |
| internal/imageagent/temporal/types.go | 删除旧 `publicationKey`；校准 SlotWorkflowResult/SlotWorkflowV3Input 的过期注释；值合同/ID 字节不变 |
| internal/imageagent/temporal/slot_workflow.go | 仅删除无调用 `hasCandidateAsset` |
| internal/imageagent/temporal/activity_registration.go | 仅校准 PublishApprovedV3 已注册但注释仍称未注册的尾部；注册逻辑不改 |
| internal/app/worker/imageagent/dependencies.go | 仅删 BuildCapabilities 字段/赋值；组织/AI/资源/权限/存储生命周期保持 |
| internal/app/worker/imageagent/capabilities.go | 仅删 buildProductionImageCapabilities；实际 typed/provider/policy/quote 实现保持 |
| internal/app/httpapi/imageagent_acquisition_module.go | 删除不可构造旧 candidate.URL 读取；scoped durable→URL、trial policy、HTTP/批准/UNKNOWN 行为不变 |

### 5.2 测试与文档（20）

测试 15 文件：`internal/imageagent/{projection,result_digest}_test.go`；`internal/imageagent/temporal/{history_replay,slot_effect_activity,slot_effect_v3_activity,workflow}_test.go`；`internal/imageagent/store/{repository_contract,image_set,slot_effect_v3_repository}_test.go`；`internal/app/worker/imageagent/{capabilities,dependencies}_test.go`；`internal/app/httpapi/{imageagent_acquisition_module,image_set_worker_authorization}_test.go`；`internal/app/runtime/image_agent_temporal_runtime_test.go`；`tests/legacy_imageagent_manual_runtime_retirement_test.go`。每项允许转换在 §6.1，不能以文件在矩阵内为由改其它函数。后三组额外尾部来自全仓旧 ExecuteSlot/PublishSlot 方法反查，不为已退出接口保留无人调用的测试实现。

文档 5 文件：本设计、`docs/architecture/README.md`、`docs/refactoring/legacy-register.md`、`docs/superpowers/specs/2026-08-27-image-agent-recovery-compatibility-design.md`、`docs/superpowers/plans/2026-08-27-image-agent-recovery-compatibility.md`。后两份未来仅追加准确的 V2 执行要求 superseded 顶注，保留历史正文/数字/勾选，明确有效 V3 仍归当前 owner。已整体 superseded 的本地验收文档不重复改写。

若编译/原 CI/复核发现矩阵外真实消费者或 §6.1 外必须变化的有效断言，先定位同根 sibling、记录最小准确补丁并报告用户；不顺手放宽检查或静默扩大冻结清单。当前调查没有读取任何既有凭据/实例数据，也不声称全仓其它无 import 包已获退休授权。

## 6. 验证方法与保持证据

本轮三文档只做差异/链接/范围检查，运行既有原 CI；不制造业务测试 RED、不运行 provider/环境。未来源码先用现有测试补能捕获旧字段/分支仍在的失败用例，确认真实 RED，再按 §4 完成收窄并 GREEN；不另建 runner、验收平台或验证工具。

### 6.1 原检查精确转换

| 原检查 / 函数 | 准确允许变化 | 必须保持 |
| --- | --- | --- |
| projection_test.go：`TestValidateProjectionSnapshotAcceptsOnlyValidatedDurableV3Candidates`、`TestSlotProjectionJSONPreservesV3DurableIdentityWithoutChangingV2CandidateWire` | 去掉已删字段的 typed 赋值/旧 V2 成功例；改用原始候选 JSON 验证两退休字段（含 mixed/empty/null/转义/重复键）被拒绝，typed 缺失/错误 durable 仍失败；增加合法完整 projection 字节/hash对照 | 原有效 durable round-trip、无 URL/Metadata 输出、scope/ObjectKey/hash 检查；其它三个 recoverable-effect 检查原样 |
| result_digest_test.go：`TestResultDigestV2MatchesCommittedHistoricalValue` | 删除 V2-only golden 和仅服务它的 import | V3 revision/role/object/hash/order 全部原断言保持，增加原 V3 值冻结对照不改算法 |
| history_replay_test.go：`approvalProtocolProbe`、`workflowWireProbe`、`TestApprovalMarkerCombinationsSelectOnlyCompleteProtocols` | probe 在使用摘要/发布前应用真实入口同一 wire 准入；旧混合 marker 成功例变拒绝且 Activity/provider=0；删除重复 V3 注册和旧 digest/key expected | 完整有效 V3 组合仍验证原摘要/键/发布 Activity，不能只删旧用例而不证明入口拒绝 |
| history_replay_test.go：`TestRecoveryStartMarkerDefaultsOffForLegacyV3Histories`、`TestRecoveryStartMarkerEnablesExternalRecoveryForNewV3Histories` | 两例把无效的三项 approval marker fixture 改成有效 V3；继续分别设置 recovery-start DefaultVersion / 1 | 原 recovery off/on 结果、版本预期/调用与原 current 名称保持；其它 pre-finalization/new-wire/run-scoped/bounded-key 测试不改断言，尤其 LegacyScopedApprovalHistory 是有效 V3，不能删除 |
| workflow_test.go | 仅新增本批 ForWire/完整入口无旧 fallback 的回归；删除 revisionFailingExecutor、identityCheckingExecutor 的四个无调用 ExecuteSlot/PublishSlot 方法，保留当前 GenerateSlot/BuildSlotResult；原 `TestRetiredActivityWireFailsBeforeAnyActivity` 原样保留 | 全部原授权、身份断言、预算、绝对 deadline/取消、响应丢失、UNKNOWN、projection/recovery/approve 断言原样；不删当前 publisher 的委托逻辑 |
| store/repository_contract_test.go：`testProjectionCommitRejectsMismatchedSlotAttemptIdentityAtomically`、`testProjectionCommitRollbackHidesEventSnapshotAndNormalizedWrites` | 前者 URL 成功 fixture 改当前 scoped durable；后者危险 URL fixture 改缺失/非法 durable；仍触发原被测错误层 | 原错误分类、事务回滚、event/snapshot/normalized writes 不可见和每项 identity 拒绝，不以提前失败掩盖原测试目标 |
| store/image_set_test.go：`TestSetSlotClosurePreservesUnstartedWorkInNormalizedAndProjectedState` | unverified URL fixture 改当前缺失/错误 durable，不增加新持久化字段 | 原未派发 work、normalized/projected 一致性和拒绝无验证资产断言 |
| store/slot_effect_v3_repository_test.go：`TestSlotExecutionResultV2JSONRemainsFrozen`、`TestSlotEffectV3PublishedResultRejectsLegacyCandidateFields` | 用户批准准确例外：删除旧 V2-only JSON 保活测试；两项 URL/Metadata 拒绝改接当前 projection 原始 JSON decoder，仍断言 ErrValidation | 该文件其余 V3 ledger、原并发/fence/权限/恢复/allowlist 全部逐字保持 |
| temporal/slot_effect_activity_test.go：`recordingRecoverableSlotExecutor`、`activityGeneratedOutput` | 删除完全无消费者的旧 recoverable executor 结构、五方法及其专属 generated helper/import；不造新的 staged 替身保活它 | 原六个当前 V3 测试及全部断言保持；其余仍被当前消费者使用的 GeneratedAsset 临时材料/过滤行为不动 |
| temporal/slot_effect_v3_activity_test.go：`recordingStagedExecutor`；runtime/image_agent_temporal_runtime_test.go：`runtimeSlotExecutor`；httpapi/image_set_worker_authorization_test.go：`fullImageWorkerSourceProbe` | 仅删除对应零调用的旧 ExecuteSlot/PublishSlot 方法（分别2/2/1）；不删共享结构或当前 staged 方法 | 所有原测试/计数/身份/失败/取消/权限断言保持；当前生成/构建接口断言继续有效 |
| worker/capabilities_test.go：`TestBuildProductionImageCapabilitiesQuotesCurrentRouteAndRejectsStaleAuthorization`、`TestProductionSourceEditDoesNotResolveReviewRoute` | 两 setup 直接使用已有 routed provider + embedded resolver + buildImageCapabilities，替代闲置 wrapper；不新增替代 wrapper | 原准确路由/配置版本/stale authorization/quote/reviewCalls=0 全部断言 |
| worker/dependencies_test.go | 仅删除 BuildCapabilities 的旧 t.Fatal stub | 真实 BuildOrganizationCapabilities 的调用、mode拒绝、open/dial/provider=0、context/close/权限/资源断言保持 |
| httpapi/imageagent_acquisition_module_test.go：`TestAcquisitionImageTrialResultDerivesOnlyExactPublishedAssetURL`、`TestAcquisitionImageRoutesRejectForgedURLCandidateOnReadAndCompletedReplay` | typed 旧 URL 赋值改缺失/伪造 durable；旧 URL 可被 store 读取的正向 require 取消，退休 JSON 拒绝由 domain 原始输入测试承担 | 用户批准两行 expected：伪造 durable 返回原 ErrValidation/GET400；完成回放仍409。原 URL 不泄露、approve=0、immutable owner read=0、精确 resolver/组织作用域/trial/不可重解释批准响应。其它 HTTP/JSON/授权/计量断言保持 |
| tests/legacy_imageagent_manual_runtime_retirement_test.go | 原函数全保持；同文件增加本批 AST 缺席/现合同形状检查，覆盖两字段/五闲置函数/BuildCapabilities 和旧 probe fallback | 原 #994 文件/声明禁入、parser 失败关闭；保留当前文件必须存在，不加 missing-skip、不降低检查逻辑 |
| 其它原检查、两套 GitHub workflow、完整 inventory | 零改；本批不预授任何 CURRENT 行例外 | 历史、计数、ceilings、lint/ratchet/安全和必需检查全部保持 |

源码必要自检仅覆盖受影响 domain/Temporal/store/assetpublication/worker/HTTP 原测试及原退休 guard；编译当前 application、image worker、schema-init 等实际入口，不启动。已有数据库测试仅按原隔离 fixture 方式执行；无任务资源准备授权时记 NOT_RUN，不连接既有数据库。未变化有效 V3 证据可引用，不能冒充新 HEAD 执行。

## 7. 评审、准入与交付停止点

- 设计复核核对真实调用、完整闭包、当前 JSON/hash/版本/持久化、§6.1 及有效回放/恢复保留；finding 先分类并指出当前 Must/具体后果。沿用最多两轮正常架构评审，不重开无关架构。
- 使用现有 GitHub 复核与原 CI；准确 HEAD 评审/CI 证据留同一 PR，不把 Running/启动失败/空 reviews 当通过。未经用户本轮授权不默认新增 AI 互验线程；设计复核和源码最终交付复核授权分开。相同 SHA 正在复核时不重复触发。
- 本轮停在可复核设计和用户确认点。适用设计复核完成且用户确认后，Issue 当前段才能标 `IMPLEMENTATION_READY`；此前未来生产源码 NOT_READY。后续仍用同一主要 PR 完成 §1–§7，不拆机械小批。
- 源码交付最终检查完整 diff、JSON/hash/版本对照、原有效断言、全部保留能力；修复后只复核相应增量。实现/原 CI/独立复核不等于运行或产品验收。
- 合并须用户授权准确最终 HEAD 并正常 expected-head merge；之后只认实际 merge SHA 的完整 push/main CI。部署、真实运行、实例/数据/凭据/进程操作和 #301 关单均未授权。#301 始终 OPEN，不增加内部试用前置。
