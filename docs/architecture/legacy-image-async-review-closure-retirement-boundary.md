> 本批实施准入：用户已确认准确设计 df8973a9906c9c6f7def63aa87efc64ff70b438a 的§1–§7（含§4.2/§6.1），原CI与独立设计复核已齐。当前 USER_CONFIRMED / IMPLEMENTATION_READY / IMPLEMENTING；同一PR整批61路径，不机械拆批。合并/运行/关单未授权，#301 OPEN。以下冻结设计原文逐字保留。

# 旧图片异步协议、GRSAI 通用执行与模型审核尾部完整退休设计

> Design Basis: **Independent Architecture**。当前 **DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY**；用户仅授权设计，尚未授权实施。本次文档工作为 N/A / Ready，只提交本文、architecture README 和 legacy-register。起始 main：`f359224f9a7ee08bde28ab554b7aee020036e593`（#1033 已合并）。一个主要 PR，三组一起，不机械拆批。#301 保持 OPEN。

最终设计候选仅同步 main `d1035558e44883db3ff3b33e7965e76a2e889e10`（#1036：1688 插件内联脚本输入预算，三个无关文件）；未新增本批生产实现，消费者/owner/删除与保留边界不变。首轮设计复核唯一 IMPLEMENTATION_TEST 是遗漏的百炼测试 getter 消费者，已纳入 T30/§6.1；完整范围校准为61路径。

## §1 这次具体清理什么

使用者仍从当前商品/图像入口生成单主图、图组和 Artwork，Store 仍做观察同步。本批不增加用户步骤；交付给维护者的是：这些当前路径只依赖实际需要的同步能力，不再背负已经取消的异步任务和模型审核协议。

| 完整闭包 | 退休的东西 | 留下的实际能力 |
| --- | --- | --- |
| 旧图片异步协议 | Submit/Query DTO、异步 sentinel、宽 ImageGenerator、被迫实现的 unsupported 方法及旧装配 | 当前 exact-route 编辑窄合同；当前图片请求/结果、模型与凭据版本绑定 |
| GRSAI 通用执行链 | 通用 Generate/Edit、轮询、重试、远程来源重抓、旧结果下载、JobError、专属配置/日志及检查工具尾部 | GRSAI GPT-image-2.5/VIP 单次同步提交、Artwork、精确来源字节、内存上限、observer 在结果下载之前落账 |
| 旧模型审核尾部 | Review/Scene 常量、InvocationRecord 的三个可写审核结果字段及旧审核回放/学习测试 | 当前六类 native text 调用、真实用量失败结算、当前账本/恢复；账本所需的三个固定零值编码槽位 |

**不会把当前图片生成或当前文本 AI 退休。** 百炼 Qwen、原 Amazon/1688 采集、当前图像/Artwork/资产批准、身份权限、钱包/Resource、取消/UNKNOWN/恢复和 Store 观察同步保留。已退休的独立 SHEIN 登录/身份回写链不恢复。

Out of Scope：新功能/UI/provider、路由/计费机制重做、新持久化事实或 schema、数据迁移/扫描清理、兼容 alias/fallback/双读双写、新 runner/验收平台、真实环境操作、部署、合并或关单。workflow、所有 inventory、ceilings、保护规则、依赖零改。不操作既有实例、数据、凭据、缓存、Profile 或进程，不增加内部试用前置。

## §2 Authority、旧 KEEP 的替代与风险边界

依据用户本轮明确授权、[全新安装决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[项目边界](project-boundaries.md)、[当前来源闭环](source-business-closure-v1.md)、[当前主图执行](../engineering/2026-09-25-issue487-main-image-execution-design.md)、[当前图像能力](product-image-agent-v1.md)、[GRSAI VIP 绑定](grsai-vip-image-binding-v1.md)、[百炼图像接线](bailian-product-image-wiring-v1.md)、[模型路由定价](model-route-pricing-management-v1.md)及[当前预付资源](unified-base-prepaid-resources-v1.md)。无 UI/IA/命名/交互变化，Figma N/A；不改变当前 Product/Figma Authority。

| 前批 KEEP | 这次为何可替代 | 仍有效的部分 |
| --- | --- | --- |
| [#983 中立合同](legacy-asset-source-openai-tail-closure-retirement-boundary.md)：ImageAsync DTO、ImageGenerator/RouteBoundImageGenerator、GRSAI 编译断言/sentinel | 源码搜索和实际装配证明无当前异步调用；宽接口只强迫空实现 | 有效 DTO 的字段/tag/顺序、ImageRouteSelection、Usage、模块依赖边界 |
| [#1025 便利链](legacy-task-openai-convenience-closure-retirement-boundary.md)及[#958 bootstrap](legacy-idle-provider-bootstrap-closure-retirement-boundary.md)：ImageGenerator/GetDefaultModel 保留 | GetDefaultModel 的真实用途只是 GRSAI pin 时核对模型；在同一基础设施 owner 内直接核对具体 Client 配置即可，不保留宽 port | 模型不匹配在发送前拒绝；组织凭据/配置版本及实际编辑能力 |
| [#980 图像策略合同](legacy-ai-routing-image-policy-closure-retirement-boundary.md)：Scene/Review 常量和 InvocationRecord 精确形状 | #1033 已删除最后模型审核生产者；三个业务字段只剩 ORM 往返及账本散列 | 三个零值对当前 native 账本 JSON 的影响必须抽取保留，见 §4.2；其它原字段/顺序和值不变 |
| [#893 日志门面](legacy-logger-facade-closure-retirement-boundary.md)、[#934 helper](legacy-idle-adapter-helper-closure-retirement-boundary.md)、[#1022 S3 尾部](legacy-auth-route-s3-operation-closure-retirement-boundary.md)：GRSAI 本地 Logger/SSA KEEP | logSubmitDiagnostic 只被将退出的通用方法调用；现役 Once 路径没有此调用。其生产 SSA rule 是最后一条，不能用空规则运行假装保留有效门禁 | 当前 App/worker 日志、全局日志禁入和来源发布的真实 AST 安全检查；共享 AST helper 先 EXTRACT |
| [#1033 通用图像整链](legacy-imageagent-generic-execution-closure-retirement-boundary.md)：当前同步供应、有效 DTO、原账本 | 不恢复已删链路，只收齐其仍留在中立合同/GRSAI/InvocationRecord 的专属尾部 | SourceGenerationExecutor、SourceMainExecutor、ImageResultBuilder、AcquisitionMainPublisher、当前 JSON/hash/版本/批准/恢复 |

Must：当前 happy path、有效平台协议、组织/凭据绑定、精确来源、单次发送、观察先落账、未知不自动接管、原账本指纹/事务和 Store 观察不退化。Should：最小 EXTRACT 到当前 owner，专属尾部整退，不留空骨架。

Threat Model 不扩张：防止错误路由/跨组织、来源重抓漂移、外部生成重复发送、未观察成功被当作失败释放、账本指纹变化导致重复或冲突结算、旧审核事实进入当前恢复。未获准做真实环境验收、旧任务续跑或新验收平台；这些保持 NOT_RUN，不能写成 PASS。无证据表明静态问题已造成真实事故。

## §3 实际消费者与 owner

调查以起始 main 的生产调用和装配为准，测试/历史文档不构成运行消费者。

| 合同 → 实现 → 注入 → 消费者 | 核对结果 / 当前 owner |
| --- | --- |
| `ai.ImageGenerator` → GRSAI/百炼/stub → `worker/imageagent/generation_provider.go` → `openai.SourceImageAdapter` | SourceImageAdapter 实际只调用 EditImageWithRoute；百炼 Artwork 单独调用 GenerateImageWithRoute。宽接口的 Generate/Edit/async 不是其业务需求 |
| `grsai.NewSynchronousSourceImageAdapter` → synchronousProductImageClient.EditImage → Client.EditImageOnce | 生产里那一处 `.EditImage` 是此同步 wrapper 的转发，不能误认成通用 Client.EditImage 的消费者。wrapper/宽接口退出后直接绑定 Once + observer |
| `grsai.Client.GenerateImageOnce` → generationProviderFactory.artwork → 当前 Artwork executor | 当前真实文生图；与 EditImageOnce 共享严格同步 wire/观察，不轮询 |
| 百炼 `EditImageWithRoute` / `GenerateImageWithRoute` → 当前来源编辑/Artwork | 当前协议和 observer、有效准入/诊断全部保留；只退 forced-wide 空方法 |
| `InvocationRecord` → Eino native executor → GormInvocationRecorder → orgresource GormModelInvocationRepository | 当前 Product Agent、Content、Materials、Selection、ShopDiagnosis、AIWorkbenchChatPlan 六类操作；仍需要 UsageObservedFailed、原始用量账本、预算和显式恢复 |
| 三个 Review 字段 → `invocationRowFromRecord` / `invocationRecordFromRow`；`modelPointFingerprint` / finalize 的 `modelPointHash(fact)` | 无当前写入审核结果的 producer，但有两处真实账本散列消费者，不能直接删字段了事 |
| `logSubmitDiagnostic` → 通用 Submit/Edit/Generate | Once 不调用它；Logger/AdaptLogrus/配置注入和唯一生产 logger SSA rule 随最后消费者退出 |
| provider logger SSA 的 `providerSyntaxParents` → `tests/issue30_publication_guard_scan_test.go` | 唯一仍在旧测试工具文件外使用的 helper。原 AST 栈遍历抽回该当前来源发布检查文件，再退休专属 SSA 文件，不能连带删除当前发布安全断言 |

没有生产 SubmitImageGeneration/SubmitImageEdit/QueryImageGeneration/SupportsAsync 调用；没有当前模型 Review producer；ImageGenerator/GetDefaultModel 的有效模型核对不得遗失。具体消费者名和检查转换见 §5/§6.1。普通人工审核的 `catalog.Review`、canonical/readiness projection 的 `ReviewReasons` 是当前合同，完全不在本次退休禁令内。

需要指出的架构问题：宽接口使同步 owner 继承异步空方法；账本直接散列整个业务结构，使闲置业务字段也成为资金幂等身份的一部分；专属测试工具又被当前发布 guard 借用。分别以窄合同、当前账本私有编码和共享 helper 归属抽取解决，不另造框架或第二事实源。

## §4 抽取与切换边界

### §4.1 最小当前实现与外部副作用

1. `internal/ai` 新增且只暴露 `RouteBoundImageEditor`：唯一方法 `EditImageWithRoute(context.Context, *ImageEditRequest, ImageRouteSelection) (*ImageResponse, error)`。不嵌入旧 ImageGenerator，不提供旧名 alias。有效 Usage/ImageGenerateRequest/ImageEditRequest/ImageInlineReference/ImageData/ImageResponse/ImageRouteSelection 原声明的字段、类型、tag、顺序和含义保持；含 MaxRetries、URL/Mask 槽位也不顺手删除。
2. `openai.SourceImageAdapterConfig.ImageClient` 和 App build/factory/test callback 改接此窄 port；原 typed-nil 拒绝与其它依赖/路线/报价/结果校验保持。旧运行期宽接口 type assertion 被静态窄合同替代，不以重新添加无 route 的 Edit 方法接受错误 delegate。
3. GRSAI 的 `SourceImageAdapterConfig.Client` 改为本包具体 `*Client`；只开放现有 NewSynchronousSourceImageAdapter 装配。合并其内部有效 pin/quote/delegate 逻辑，退休普通 NewSourceImageAdapter 和 synchronousProductImageClient。pin 持有 Client + 必需 observer，直接 EditImageOnce；发送前仍精确比较模型、credentialReference、configurationVersion，并直接核对 Client.cfg.Model。GetDefaultModel 不作为公共兼容方法保留。Build 的 typed-nil delegate、缺依赖和 quote route 漂移拒绝不变。
4. Client 保留当前 NewClient、GenerateImageOnce、EditImageOnce 和严格同步 POST；通用 GenerateImage/EditImage、Submit/Query、poll/retry/结果下载全部退出。HTTP POST 无重放/无 redirect 变成该唯一发送路径的无条件行为，而非可关闭的 disableSubmitReplay 分支。不修改原 caller deadline/Timeout、授权、价格、observer 写入和 finalization 的 10s WithoutCancel 边界。
5. 从混合 helper 只抽取精确字节→base64 行为和跨请求字节 semaphore。保留原编码/MIME 判定、主图在前、已批准参考图原顺序、最多八来源、每项与总量原上限、取消时可中断 acquire、once 释放和约1MiB单位的原计量。默认512MiB和原最小预算折算不变，不顺手调优。退休来源 URL 重抓/下载槽位及 MaxReferenceMaterializationConcurrency/PollInterval/MaxAttempts；保留 MaxReferenceMaterializedBytes。当前输出 URL 的 public HTTPS/SSRF、受限下载和图像验证仍由 SourceImageAdapter/httpimage owner 执行，不随旧下载方法删除。
6. 当前 request 仍为 `model/prompt/images/aspectRatio/quality/replyType`，无源 Artwork 不带 images；2.5=auto、VIP=medium、1024、replyType=json。响应仅接受原 succeeded/非空id/空error 形状，1MiB上限保持。running、丢响应、截断、HTTP错误、另一路 JSON 形状都按原 UNKNOWN 返回，不 poll、不重发、不自动释放。保留 resultItem 的 URL/Content 字段及顺序：其 JSON 已用于 result digest，不能因 Content 暂无业务读取就删。观察在下载之前，RequestID 与 ResponseID 可不同/缺失的当前语义保持。
7. 百炼只退出 GetDefaultModel、无 route Generate/Edit 和 async unsupported 方法；当前路由绑定两方法、原参数、no-effect proof、观察和取消/UNKNOWN 行为零改。GRSAI 专属日志无当前 Once 使用，不制造新日志调用来保活；App/worker 正常日志与 httpimage 日志/安全规则保留。

### §4.2 JSON、账本指纹、原始表与历史取舍

**两处散列都要保持，只有保留 reserve hash 不够。** 原代码：`modelPointFingerprint` 归一化 dispatch 事实后散列整个 InvocationRecord；finalize 又把 StartedAt/FinishedAt 转 UTC、截到微秒后直接 `modelPointHash(fact)` 作为结算 proof。

设计：从公共 InvocationRecord 退出 ReviewScore、ReviewNeedsHumanReview、ReviewReasons 及旧回放注释；由 orgresource owner 的私有 `modelPointFactHash` 编码当前 InvocationRecord，并在原末尾位置显式附加以下**只能由编码器写零值**的字段：

```json
"ReviewScore":0,"ReviewNeedsHumanReview":false,"ReviewReasons":null
```

编码载体只在 `internal/integration/orgresource/model_invocation_fingerprint.go` 内使用，可用匿名嵌入当前 InvocationRecord 的私有 struct 加上述三个固定槽位；不导出可写 Review 合同，不给 InvocationRecord 增加全局 MarshalJSON/兼容 decoder，不从旧表恢复这些值。此编码是当前资金幂等合同的选择性抽取，不是旧模型审核兼容。

| 当前不变量 | 具体保持方式 |
| --- | --- |
| 预留 fingerprint | 原 StartedAt UTC/微秒归一化；原终态/usage/error/provider/output 清零过程和 dispatched 值保持，仅最后一步改用私有当前事实编码 |
| finalize proof | 原两时间 UTC/微秒归一化和全部真实终态字段保持；同一私有编码保留三个零槽位，不把 proof 错改为 dispatch fingerprint |
| JSON/hash/版本 | 原所有其它字段、嵌套 tariff、顺序、默认大写 key、nil 与空数组区别、时间序列化、SHA256/hex、model_invocation_v1、model-point reserve/finalize key、receipt 和资源事务版本保持。generic modelPointHash 对其它 string/receipt 的调用不改 |
| 当前账本和事务 | 原组织/member/run/input/tariff/bound、锁、同键不同载荷拒绝、预留/终结/债务/月度归属、幂等 replay、无发送释放/UNKNOWN、不重复 provider 及财务恢复 owner 不改；不新建账本或公共 hash 框架 |
| 原始物理事实 | `ai_invocations` 的 invocationRow 全字段/gorm tag/index、Review 三列、schema/Goose/grants/安装验证和已有记录零改。新当前写入仍为0/false/`"null"`；不因业务结构变小就删除列或改变新安装 schema |
| 停止旧审核业务恢复 | 拒绝 operation=`productimage_review`、capability=`productimage.scene_generation` 的新写入/当前读取与恢复；物理三列带非零 score/true human/非空且非字面`null` reasons 的行也不能被投影成当前事实后继续结算/释放。SQL零值空字符串只表示无载荷，不解析旧 reasons，`[]` 不改写成 null |

Store 内用一处当前 row→业务事实边界统一拒绝旧审核身份/载荷；FindInvocation、ReadModelInvocation、RecordInvocation 的 existing/winner/terminal replay 和 ResolveDispatchedInvocation 全部覆盖。旧行必须在任何覆盖、重放财务动作或释放之前失败关闭，不能先丢掉 Review 字段再继续处理。原身份过滤、缺席/冲突、terminal 不可覆盖、CAS 和错误处理保持；仅删除退出的 Review 映射/对比赋值，固定零列仍由当前 mapper 生成。

原始表不是业务回放许可：旧模型审核历史不再续跑/重构为当前调用，不发送、不自动释放、不读/改实际表，不处理旧记录。若发现当前受支持消费者实际要求带非零 Review 的事实或其它外部 JSON 兼容义务，先停下报告，不能新增迁移/fallback。当前 native producer 不写这三项，InvocationRecord 无当前 HTTP/持久化 JSON DTO 消费；当前实际散列消费正是上述两处。

`InvocationUsageObservedFailed` 及六类当前 text operation 保留；从 SupportsObservedUsageFailure 只移除旧 Review 分支，修正注释。native 输出格式失败仍按已观察真实 tokens 结算，不能全部改成 ordinary failure/释放；原始 Token/计费/diagnostics 不退出。catalog/readiness 的人工审核、资产批准 ReviewUsage 的现有拒绝条件和全部其它 JSON/hash/版本不变。

实施时在改字段前，使用起始代码的合成 native 事实固定 reserve 和 finalize 的原始 JSON字节与SHA256；包括非零 usage/终态、非UTC输入和纳秒时间、nil/嵌套tariff。改后逐字节/逐hash比对两条路径，并保留原真实仓库结算/恢复测试。设计阶段不把未执行的等价测试写成 PASS，不读取真实账本。

## §5 完整文件矩阵与切换条件

Legacy decision: **EXTRACT | RETIRE**。Reusable behavior：窄 route 编辑、精确字节/同步提交、当前账本编码、当前来源发布 AST 父节点遍历。Current owner：ai/现役 provider、orgresource、当前来源发布 guard。Cutover/deletion condition：同一 PR 先抽取并完成当前接线，再删除所有旧声明、装配、测试/文档专属尾部；不留 alias/回退入口。

下列是未来整批实施允许范围；不是要求每个保留文件产生 diff。当前设计提交仅 D1–D3。文件/检查外的新消费者先报告并补精确边界，不顺手改保护门禁。范围统计按唯一真实路径，不按花括号或文件数拆 PR。

### §5.1 生产：16 路径

| ID | 路径 | EXTRACT / RETIRE 与保留边界 |
| --- | --- | --- |
| P1 | `internal/ai/contracts.go` | 新窄 Editor；退宽两接口、两 async DTO/sentinel/专属 import；七有效值合同零改 |
| P2 | `internal/integration/bailian/image.go` | 仅退强迫空方法；当前两个 WithRoute 方法及 HTTP 路径保持 |
| P3 | `internal/integration/grsai/client.go` | 按§4.1保留最小同步 Client/config/POST/字节预算/URL构造；整退通用重试/轮询/下载/日志配置/helper |
| P4 | `internal/integration/grsai/error.go` | 整退 JobError；当前同步 UNKNOWN 语义不替换为旧错误分类 |
| P5 | `internal/integration/grsai/logger.go` | 整退 Logger/noop/AdaptLogrus；无新假调用保活 |
| P6 | `internal/integration/grsai/source_image_adapter.go` | 抽取有效 pin/route/quote/delegate；退普通 factory 与其空方法，改唯一同步装配 |
| P7 | `internal/integration/grsai/synchronous_edit.go` | 退宽 wrapper/通用 helper 依赖；保留 Once、observer、digest和精确协议，使用当前最小transport |
| P8 | `internal/integration/openai/source_image_adapter.go` | 窄 Editor 注入/调用；当前有效规则/抓取/校验保持 |
| P9 | `internal/app/worker/imageagent/generation_provider.go` | 改窄 callback；去 GRSAI MaxAttempts/Logger 注入；供应、凭据、路由/Artwork及App日志保持 |
| P10 | `internal/aicapability/model.go` | 仅退 CapabilityProductImageScene / OperationProductImageReview |
| P11 | `internal/aicapability/invocation.go` | 仅退三个可写审核结果字段/注释与 observed-failed 的 Review 分支；六 native 和全部其它合同保持 |
| P12 | `internal/aicapability/usage_settlement.go` | 只修正旧 Review 注释；当前实际计费算法/校验零改 |
| P13 | `internal/aicapability/store/gorm_invocation_recorder.go` | 退业务审核映射，按§4.2固定零值和拒绝旧事实；物理row/事务/CAS保持 |
| P14 | `internal/aicapability/store/model_point_fact.go` | 改接同一受限mapper，保留原六操作查询/定价校验及schema验证 |
| P15 | `internal/integration/orgresource/model_invocation_repository.go` | reserve/finalize 两散列改接当前私有编码；其它hash/账本逻辑零改 |
| P16 | `internal/integration/orgresource/model_invocation_fingerprint.go`（新增） | 最小当前事实零槽位编码，只服务原两散列，无第二事实源 |

### §5.2 测试与专属工具：30 路径

| ID | 路径 | 处置 |
| --- | --- | --- |
| T1 | `internal/ai/contracts_test.go` | 退旧async JSON/stub；校验有效DTO和最小Editor合同 |
| T2 | `internal/ai/import_boundary_test.go` | 旧GRSAI宽接口编译断言退出，改当前Editor消费者/百炼实现断言；原不跨provider import检查保留 |
| T3 | `internal/integration/grsai/client_test.go` | 旧通用正向/重试/poll/URL重抓保活退出；有效POST不重放、URL构造、字节预算断言改接真实Once，见§6.1 |
| T4 | `internal/integration/grsai/client_integration_test.go` | 整退旧live通用Generate试验；不读取其环境变量或发请求 |
| T5 | `internal/integration/grsai/logger_test.go` | 随专属日志整退 |
| T6 | `internal/integration/grsai/synchronous_edit_test.go` | 窄callback/配置改接；原实际HTTP/observer/UNKNOWN/单次发送断言保持 |
| T7 | `internal/integration/grsai/synchronous_references_test.go` | 去MaxAttempts；退普通Edit拒绝测试，保留两项Once有效来源顺序/限额/无发送 |
| T8 | `internal/integration/grsai/synchronous_artwork_test.go` | 只去MaxAttempts；原两模型实际生成、无原图/不重发保持 |
| T9 | `internal/integration/grsai/synchronous_vip_test.go` | 只去MaxAttempts；原VIP quality和单次发送保持 |
| T10 | `internal/integration/grsai/source_image_adapter_test.go` | 改唯一同步factory的真实client/合成HTTP；原模型、路线、quote、typed-nil/delegate及无发送断言保留 |
| T11 | `internal/integration/openai/source_image_adapter_test.go` | stub/helper改窄Editor，退强迫空方法；保留当前图像/路线/结果断言，无法表达的宽接口delegate由静态窄合同替代 |
| T12 | `internal/app/worker/imageagent/grsai_synchronous_edit_test.go` | 只改窄callback，来源provenance和精确bytes断言保持 |
| T13 | `internal/imageagent/assetpublication/image_set_adapter_test.go` | 只改窄callback，distinct/missing RequestID 的原选择发布断言保持 |
| T14 | `internal/imageagent/temporal/generation_output_recovery_test.go` | 只去GRSAI MaxAttempts；原恢复/不重发/选择发布断言保持 |
| T15 | `tests/imageagent_organization_integration_test.go` | 只去MaxAttempts/改窄callback；原Temporal预算首次发送前拒绝断言与原隔离环境前提保持，不调用已有环境 |
| T16 | `internal/aicapability/productimage_scene_test.go` | 两项旧enum保活整退，缺席由T29及原guard强制 |
| T17 | `internal/aicapability/usage_settlement_test.go` | 旧Review positive改六native observed-failure；非支持操作拒绝/ordinary failure不计费保留 |
| T18 | `internal/aicapability/store/gorm_invocation_recorder_test.go` | 五项Review fixture改当前native/原溯源断言；增旧事实拒绝/原始行不变，见§6.1 |
| T19 | `internal/integration/orgresource/model_invocation_fingerprint_test.go`（新增） | 起始代码合成事实的两个JSON/hash金值与真实调用接线等价；非零terminal与dispatch不可混同 |
| T20 | `tests/listingkit_backend_retirement_test.go` | 只转换§6.1列出的四函数，原其它guard不变 |
| T21 | `tests/integration_provider_logger_contract_test.go` | 退两GRSAI专属Logger保活函数；真实constructor inventory/全局日志禁入/httpimage检查原样保持 |
| T22 | `tests/provider_logger_ssa_test.go` | 仅剩一条GRSAI生产rule随最后消费者退场，整退专属测试工具，不留空rules假PASS |
| T23 | `tests/provider_logger_ssa_review_test.go` | 专属工具的合成回归整退 |
| T24 | `tests/provider_logger_ssa_nil_analysis_test.go` | 专属nil分析整退，无当前生产用途 |
| T25 | `tests/provider_logger_ssa_managed_use_analysis_test.go` | 先抽出唯一当前helper，再整退其余专属分析 |
| T26 | `tests/provider_logger_ssa_loading_test.go` | 专属加载验证整退；通用模块编译门禁不动 |
| T27 | `tests/provider_logger_ssa_analysis_test.go` | 专属SSA分析整退；不改其它AST/导入/安全检查算法 |
| T28 | `tests/issue30_publication_guard_scan_test.go` | 原样抽入AST父节点遍历，可改名publicationSyntaxParents并改其唯一调用；发布全部原断言/算法/失败关闭不变 |
| T29 | `tests/legacy_image_async_review_retirement_test.go`（新增） | 本闭包原路径/声明明确缺席，当前port/两hash/物理row/有效协议/共享helper owner和文档入口保留；不建新通用检查器 |
| T30 | `internal/integration/bailian/image_test.go` | 仅退出现役Qwen helper中的GetDefaultModel保活断言；原真实POST模型值、精确字节、单次发送/observer及其它Qwen测试保持 |

### §5.3 文档：15 路径

| ID | 路径 | 处置 |
| --- | --- | --- |
| D1 | `docs/architecture/legacy-image-async-review-closure-retirement-boundary.md` | 当前设计/冻结/实施证据入口 |
| D2 | `docs/architecture/README.md` | 当前短说明及反引号文件名显式索引，历史逐字保留 |
| D3 | `docs/refactoring/legacy-register.md` | 当前三组 EXTRACT/RETIRE，旧登记逐字保留 |
| D4 | `docs/architecture/legacy-asset-source-openai-tail-closure-retirement-boundary.md` | 后续实施时追加准确KEEP替代注记，原文逐字作后缀 |
| D5 | `docs/architecture/legacy-task-openai-convenience-closure-retirement-boundary.md` | 同上，仅旧宽接口/模型便利方法 |
| D6 | `docs/architecture/legacy-ai-routing-image-policy-closure-retirement-boundary.md` | 同上，仅Scene/Review/三业务字段，以当前零槽位编码替代 |
| D7 | `docs/architecture/legacy-logger-facade-closure-retirement-boundary.md` | 同上，仅GRSAI专属日志及生产SSA，当前共享安全规则保持 |
| D8 | `docs/architecture/legacy-idle-adapter-helper-closure-retirement-boundary.md` | 同上，仅相关GRSAI/logger KEEP |
| D9 | `docs/architecture/legacy-auth-route-s3-operation-closure-retirement-boundary.md` | 同上，不恢复S3/旧认证 |
| D10 | `docs/architecture/legacy-idle-provider-bootstrap-closure-retirement-boundary.md` | 同上，仅旧ImageGenerator及GRSAI日志KEEP |
| D11 | `docs/architecture/legacy-imageagent-generic-execution-closure-retirement-boundary.md` | 同上，仅本次剩余合同/日志/审核映射，不重开#1033 |
| D12 | `docs/superpowers/plans/2026-06-20-listingkit-async-studio-image-generation.md` | 追加旧异步协议已退出的历史注记，不删除历史方案 |
| D13 | `docs/superpowers/plans/2026-08-06-ai-async-job-binding.md` | 同上，不恢复job接受/查询/回放 |
| D14 | `docs/superpowers/plans/2026-08-06-ai-capability-phase-1.md` | 同上，仅本次异步/旧审核范围，当前native账本仍有效 |
| D15 | `docs/superpowers/plans/2026-08-21-provider-neutral-ai-contracts.md` | 同上，当前值DTO与窄Editor取代旧宽合同 |

合计 **61 唯一路径**（16生产＋30测试/工具＋15文档）；三个新增生产/测试文件与本文均显式列明。没有 workflow/inventory 改动，没有部署文件/schema/Goose/依赖变化。并非把所有含 Review/logger/async 名字的文件一并删除。

## §6 验证、当前断言与证据边界

设计阶段：核对源码调用、装配和 sibling paths，核对导航/相对链接/历史后缀及三文档diff，运行适用现有文档/索引检查；原PR CI和准确HEAD设计复核记录在PR，不放滚动SHA进AGENTS。未来业务测试/真实provider/DB/Temporal环境均未执行，不据此签发运行验收。

实施阶段在同一PR：先固定当前两账本JSON/hash和有效路径fixture，再写能捕获退场遗漏的guard并确认真实RED；最小抽取/切换/删除后GREEN。运行受影响ai、GRSAI、百炼、SourceImageAdapter、aicapability/store、orgresource、当前主图/图组/恢复原测试及相关race，命令入口只编译；完整原CI留给稳定候选。不重复手工跑CI已覆盖的无关检查，不启动保留环境、provider、轨迹/缓存操作或新工具平台。

### §6.1 原检查的精确处置

| 检查位置 | 允许转换 / 必须保留 |
| --- | --- |
| T1 `TestContractsMarshalStableJSON` / `TestContractsExposeMinimalInterfaces`及contractImageStub | 退async-only JSON和强迫宽stub；改有效值DTO原JSON及单方法Editor编译/精确方法集，durationPointer/intPointer如仍有真实消费者则保留 |
| T2 `TestProviderAdaptersDoNotImportOpenAIForSharedContracts` | 只移旧宽接口编译保活，追加当前Editor断言；原对grsai/client.go的import禁令/必读/报错全部保持 |
| T3 `TestSourceEditDisablesGRSAISubmitReplay` | 改真实Once/窄pin；保留EOF/截断/307/308/HTTP错误的至多一次POST、无redirect；原running允许一次query改零query；旧普通调用恢复3次重试的正向部分退出，不把其 expected 数随意放宽 |
| T3 `TestReferenceMaterializationBudgetCapsConcurrentReferences` / `TestBuildSubmitURLUsesGenerateEndpoint` | 原字节预算阻塞/释放、URL构造断言保持，只改精简后的参数签名；默认预算/取消/once释放不弱化 |
| T3 其余旧通用测试 | `TestReferenceMaterializationBudgetReleasesDownloadSlotPerReference`；全部 `TestClientEditImage*`、`TestClientSubmitImage*`、`TestClientQueryImage*`、`TestClientGenerateImage*` 随旧方法退场；真实Once的精确字节/顺序/unsafe input/不重发/UNKNOWN覆盖保持在T6–T10，不保留旧URL重抓、poll/瞬时重试和旧moderation分类契约 |
| T4/T5/T16 | 分别整退live旧Generate、旧logger、两个旧Scene/Review enum保活；不把它们标为SKIPPED/PASS，不触发真实调用 |
| T6–T15 | 仅§5列出的窄类型、MaxAttempts、factory/stub改接；T7只退`TestOrdinaryGRSAIEditRejectsExactAdditionalReferences`；其它现役原assertion保持。T11无route delegate旧fixture不可赋给新静态port，不新增运行fallback；typed-nil和其它依赖拒绝仍测试 |
| T17 `TestObservedFailedReviewSettlesActualTokensWithoutChangingOrdinaryFailure` | 改六native table；真实tokens结算、普通失败不计费、非支持operation拒绝原样保留，新增旧Review literal拒绝，不扩大到其它operation |
| T18 五函数 | `TestGormInvocationRecorderObservedFailedReviewSettlesAndReplaysWithoutRelease`、`TestGormInvocationRecoveryRequiresObservedReviewUsageForFailedOutput`、`TestObservedFailedReviewSettlementFailureKeepsDurableTerminalForRetry`、`TestObservedFailedReviewRequiresExistingDispatchedReview`、`TestObservedFailedReviewRecoveryPreservesDispatchedProvenance`：有效安全行为改当前native fixture并可相应重命名；溯源、无重复provider、无释放、无持久dispatch拒绝、usage缺失拒绝、持久终态/settlement重试/身份冲突全部保持。纯旧审核正向不再成立，由旧literal/非零列拒绝和raw-row未改写替代 |
| T18/T19新增 | 旧row拒绝覆盖Find/ReadModel/Record existing&winner&terminal replay/Resolve入口，断言零settle/release/无覆盖；两账本实际调用使用同一当前编码，起始金值字节等价、微秒/时区/nil/非零terminal及同键冲突验证；原orgresource结算/回收/债务/月归属测试不改 |
| T20 `TestLegacyTaskOpenAIConvenienceClosureIsRetired` | 必需ImageGenerator改RouteBoundImageEditor，旧无route/宽声明改明确禁入；其余原task/value/JSON/规则断言保持 |
| T20 `TestLegacyIdleProviderBootstrapClosureIsRetired` | ImageGenerator非alias保活改当前Editor非alias；旧宽类型明确缺席，其余config/owner/禁止包装检查保持 |
| T20 `TestLegacyAIRoutingImagePolicyClosureIsRetired` | 必需/enum表只移Scene/Review两项；InvocationRecord精确shape只退三个字段及专属注释；其余字段顺序/值/PolicySceneDefaults/GenerationMetadata/导入等保持，另校验当前私有账本零槽位 |
| T20 `TestLegacyAssetSourceOpenAITailClosureIsRetired` | shape表移两async和宽两接口，添单方法Editor；七有效值合同shape和原旧openai别名/声明禁入、当前catalog/JSON/hash断言保持；禁令中历史名字无需删除 |
| T21 | 退`TestIntegrationProviderLoggerMethodSetsMatch`、`TestGRSAIConfigRetainsExplicitLocalLogger`（已无provider本地logger消费者）；只清失效import。`scanProviderConstructors`/当前GRSAI构造inventory、`TestProductionLLMManagerAdapterCallersPassLogger`的旧入口禁入、`TestIntegrationProviderLoggingUsesOnlyLocalExplicitPorts`、`TestHTTPImageResolverIsImmutableFunction`全部保持 |
| T22–T28 | 专属SSA生产rule归零后连同工具/合成测试整退，不留空检查假PASS，不把通用模块编译/identity/issue30发布检查一起删；先原样抽出`providerSyntaxParents`并只改其本地名字/唯一调用，原发布扫描/所有安全断言保留 |
| T29及文档 | 使用仓库既有AST/声明/文件缺席helper检验精确闭包；保留当前解码、生产图像注册、JSON/hash/批准/恢复及Store guard，不加missing-skip、不全仓禁止Review字样。新doc显式索引，D4–D15原Git blob作逐字后缀 |
| T30 `TestQwenImageEditSendsFrozenBytesOnceAndObservesBeforeReturning`所调`testQwenImageEditSendsFrozenBytes` | 仅删除`require.Equal(t, model, c.GetDefaultModel())`的旧getter保活行；原`require.Equal(t, model, payload.Model)`真实HTTP模型断言、两个模型循环、精确bytes/参数、单次发送/observer与全部其它Qwen测试保持，不保留getter/alias或替换成空断言 |

其它原安全断言/检查算法、workflow、inventory、ceilings不变。发现漏列消费者或真实CI失败时先定位同根sibling并给出最小准确补丁；超出本矩阵不默默修改，也不能把生产缺陷用降低检查掩盖。设计已明确可退的专属检查不要求留下空循环/假调用；有效当前断言必须在正确owner继续执行。

## §7 准入、复核、交付与停止点

当前 DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY。需先完成适用独立设计复核、分类 findings，达到 IMPLEMENTATION_READY，且用户确认本文§1–§7后才可改生产路径。设计确认和执行授权不包含合并/部署/真实环境操作/Issue关闭。#301始终OPEN。

一个Writer、一个主要PR；最多两轮正常Architecture Review。具体当前Must缺陷按AGENTS分类；非Blocker实现细节进入IMPLEMENTATION_TEST，保留在本批，不扩为新框架或新试用前置。冻结后只核对新增真实差异和组合风险，未变化证据可明确复用，不能冒充新SHA直接执行结果。

**未经用户本轮授权不默认新增 AI 互验线程。** 优先既有GitHub适用复核；启动失败最多一次明确重试，不重复触发Running评审。若仍无结论，报告具体状态并请求一次只读设计复核；设计与源码最终交付复核分别绑定准确HEAD、授权分开。不能把Completed/failed summary或空reviews当作审核PASS。

设计交付：三文档、完整删除/保留矩阵、原检查处置、准确HEAD/原CI/独立设计结论，交用户确认。未来实施交付：同PR完整闭包、当前入口/调用不变说明、两个账本JSON/hash证据、适用原测试/CI及只读最终交付检查；运行/平台/旧历史数据验收保持NOT_RUN。实际main合并须另获最终HEAD/expected-head正常合并授权，再核对实际merge SHA的完整push/main CI，不能拿PR CI或Required CI Gate单项代替。

停止条件：当前能力或账本原指纹不能保持；发现现役非零Review事实/外部兼容义务；需新schema/owner/恢复协议或重复外部副作用；需改workflow/inventory/既有资源。先报告根因、准确范围和最小处置，由用户决定；不自行兼容、迁移、删表、清理或扩试用门禁。
