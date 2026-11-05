# #132 标题对照的模型准入、固定组与批次预算

状态 **DESIGN_CANDIDATE / NOT_IMPLEMENTATION_READY**
正式评测 **NOT_RUN**
依据 [Issue #132](https://github.com/henryxu1008/shuomi-ai-commerce/issues/132)、会话中已确认的预算与停止规则、[Product Agent 运行合同](product-agent-runtime-contract.md)、[企业统一模型与计费合同](enterprise-shared-model-configuration.md)及[共享渠道凭据合同](shared-channel-credentials.md)。源码核对基线为 `e4d440def298e36cb70c261de0c2bad9074a9503`。

## 1. 交付范围与设计分类

本批让评测执行者可以在同一份授权商品证据下，比较「固定读事实、生成一次」和「受限 Agent 选择工具及修复」两条标题路径。两组共用模型、版本化提示和输出协议、标题校验及计费。输出用于评测和人工审核，运行止于生成及校验。原 Review/Apply 权限、生产标题入口、企业模型选择、统一账本和既有调用恢复合同由原 owner 管理。固定组的受限路径仅供本轮评测；既有固定生成入口已退休，Legacy decision 为 N/A。

本地候选使用 [30 个合成商品清单](../../internal/app/httpapi/testdata/issue132_title_pilot_v1/manifest.json)的 12／10／8 分层。PS-002 单列，排除在本批 30 对与费用分配之外。人工标注只提供给评分者，模型输入来自获准的商品事实投影。30 对用于先导对照，正式评测规模及结论按 #132 原判定规则另定。

Design Basis 为 **Independent Architecture**。本轮新增固定组的受限模型执行路径、准确文本协议，以及跨 60 项运行的预算和停止约束。原标题、视觉单次运行或其他模型的 Ready 只覆盖各自批准范围。按 [Architecture First](../../AGENTS.md#architecture-first-开工准入)与 [Issue 准入规则](../engineering/issue-driven-development.md#31-派工前设计准入)，冻结文档与适用独立评审达到 `IMPLEMENTATION_READY` 后才可实施。

## 2. 两组共用的模型和发送准入

| 冻结项 | 候选合同 |
| --- | --- |
| 用途和版本 | `OperationProductAgentDecision`。OpenAI 官方标准直连的 `gpt-5.4-mini-2026-03-17`，准确快照而非别名。两组的组织、成员、连接 revision、endpoint、模型定义、适配策略、计量映射、价目和汇率版本一致。 |
| 请求与计量 | `POST /v1/chat/completions`，纯文本、非流式，`n=1`、`store=false`、`service_tier=default`、`reasoning_effort=none`。每次输入预算至多 16,384 token，含系统指令、证据、Agent 历史及原协议余量。请求显式锁定 `max_completion_tokens=2048`，覆盖完整输出用量。模型只返回现有 typed Action，服务商原生工具关闭。 |
| 冻结时点 | 准备时按现有配置 owner 生成执行快照及 profile digest；报价绑定实际输入摘要。发送前重验同一快照、用途、授权及当前路由。任何漂移、输入越界、计量映射或价格依据缺失都停止。资格变化后按新准备决定处理。 |

入口沿 [企业共享文本路由 reader](../../internal/app/httpapi/model_route_runtime.go)进入 [现有 AgentTextModel](../../internal/integration/agent/einomodel/model.go)和 [受治理执行器](../../internal/integration/aicapability/einomodel/executor.go)。当前通用 OpenAI 文本请求仍用 `max_tokens`；写入准确字段、显式关闭推理的现有策略属于单图用途。最小接线是在同一 Eino 组件新增**只对准确标题用途和模型生效**的适配策略，并在最终传输守卫核对精确端点、快照、请求结构、输出上限、禁止服务商工具及单次发送约束。运行时继续用现有 `QuoteText` 和受控传输。启用前须在隔离测试中捕获真实组装的请求字节，核对两组和 Agent 最长历史。输入估算需要覆盖该实际序列化形式；超界按拒绝处理。

OpenAI 公开文档提供准确快照、支持的请求字段、标准价格及收费维度的依据。本批实际渠道、用途资格、计量完整性与组织配置仍待核验。[准确快照及定价](https://developers.openai.com/api/docs/models/gpt-5.4-mini) · [Chat Completions 字段](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create)

## 3. 固定组复用标题模型与校验

1. 两组从现有授权 owner 获取同一组织的同一 `ProductKey + CatalogVersion + CatalogPublicationID` 与合法上下文。用准确输入和原工具审计创建各自运行，评分答案、占位图和任意 URL 不进入模型输入。每一对分别冻结既有标题指令、策略、快照及版本；Agent 的动态历史可随本组流程变化。
2. 固定组从现有 [Eino 固定预读与单次模式](../../internal/integration/agent/eino/runtime.go)开始。该模式当前仅接收领域提案；本批只给标题 Candidate 增加显式组合，走原标题 Validator 而不是构造虚假的领域 Proposal。固定顺序先读取 `canonicalinspect`；其投影能否承载所有必需事实和可核的证据 ID，需要组合回归确认。固定组在准备和运行均锁定 `ModelCalls=1`，第一次生成后只做校验；无候选、校验失败或预算拦截记录本例终态。Agent 组最多 6 次总模型调用、最多 2 次现有修复，且服从更小的现行限制。
3. 两组共用 [productAgentApplication.Validate](../../internal/app/httpapi/product_agent_application.go#L534)与原 Review 的证据/标题规则。运行状态经 `agentconfig.Guard` 和现有 Run Store 的唯一 Claim / CAS 写回。用内部、隔离的 #132 装配区分两组，沿原配置快照准备更小的限额。旧离线测试可以复用装配和断言思路，其假授权和内存采集回执只属于合成传输测试。付费先导必须从现行 owner 取合法样本绑定与身份；若现有合法入口装配不了 30 份样本，停在准入并报告具体缺口。

## 4. ¥38 批次上限和发送前拦截

| 分配 | 每项允许调用 | 每项封顶 | 30 项封顶 |
| --- | ---: | ---: | ---: |
| 固定组 | 1 | ¥0.18 | ¥5.40 |
| Agent 组 | 6 | ¥1.08 | ¥32.40 |
| 合计 | 最多 210 次 | 每对 ¥1.26 | ¥37.80 |

¥38 中的 ¥0.20 保持未分配。商品与组别额度固定。任务省下的额度不挪给其他任务，失败、换序和中断均不生成额外额度。用整数人民币微元比较每次 ¥0.18、任务 ¥0.18／¥1.08 与批次 ¥38。¥8/USD 是此前的规划假设。实际冻结价目、汇率、标准服务范围及收费维度必须使完整上界每次不超过 ¥0.18；若不满足、估算与冻结价目不一致，或应用配置无法强制执行，本批不开跑。

- **批次入口。** 预先锁定批次 ID、清单摘要、唯一组织/操作者/成员、固定配对顺序，以及每项的原上下文、配置摘要和请求键。只启用一个串行评测执行者。复用 Run Store 首项唯一 Claim 作为已开跑的证据。编排层应从原 Guard 的 `acquired` 结果取得首次领取证明；`StartWithClaimAttempt` 的布尔值只说明尝试过，不能用于判定抢占成功。首项已有回执、领取结果不明或读取失败时，该启动路径只查原回执并停止。其他项同样只接受当前批次原键和首次领取，发生重复或不确定便停批。进入下一项前确认上一项运行与所有调用的终态。
- **逐次发送。** 两组先走现有 Run `Usage.Reserve` 检查本项剩余调用次数、token 和人民币预算，再走现有调用与 AI 点数预留。现有 `GenerateWithGate` 最后发送门禁为两组共同检查批次身份、原槽位、已知前序结果以及本次报价不超过 ¥0.18 的接线点。校验同一调用的人民币报价能够保守覆盖冻结费率和汇率下的可收费维度，并让资源 owner 按同一冻结价目预留；任何金额、身份、用量上界或授权对不上就拒绝发送。批次额度由固定槽位和单项发送约束共同保证，不从进程内累计值推断预算，也不设第二套余额。
- **费用事实。** 人民币报价只参与限额比较。真实用量事实继续由原 AI Invocation 记录，实际点数结算由原资源 owner 依完整用量和冻结价目/汇率完成。Run、Invocation 和资源预留是不同回执；读到 Invocation 终态不能单独证明资源结算已成功。在线推进需确认实际装配了原资源结算器，并获得执行器终结与原 Run 提交的确认。重读时，需按原调用 ID 从资源 owner 核对对应结算或释放回执。目前未定位到给编排层按调用 ID 只读核对点数终结的现成接口；若核实需要补接线，只在原资源 owner 暴露所需的窄只读证据。进程中断后原 Run 可能没有写回最后一个调用 ID。找不回该调用的对应证据时继续标记 UNKNOWN。用量已观察且有结算确认时，即使标题不合格也照实记费。仅有权威未发送证据且预留终结已确认时才记零。

发送路径依据 [Run 预算](../../internal/agent/budget.go)、[最后发送门禁](../../internal/integration/aicapability/einomodel/executor.go#L153)、[调用记录及资源结算边界](../../internal/aicapability/store/gorm_invocation_recorder.go#L213)和[现有微点资源 owner](../../internal/integration/orgresource/model_invocation_repository.go)。

## 5. 停批、原键与恢复边界

任一费用 `UNKNOWN`，立即停止当前运行的后续模型调用与整批后续任务。包括可能已发送而超时或响应丢失、用量缺失/矛盾/超界、原运行结果或结算回执无法确认。终止控制信号必须传到编排层；不得仅根据标题结果或「没有异常日志」推进下一项。标记已完成项、原不确定项和未启动项，保存原批次、Run、Invocation、预留与已知费用证据。

本批中断后按原身份、原请求键及原调用 ID 进入只读核对。原 Run 仍为 Running、回执读取不明或 Invocation/资源 owner 证据不完整时维持不确定，并保留原预留。禁止重发、续跑、换键、把预留当实际扣费或手工释放。已有正式对账 owner 可按其原合同另行核对；即使原费用后来确认，本批也不会自动恢复。首项 Claim 提交结果不明时同样适用停批，不靠临时执行状态判定可再次启动。

## 6. 实施前和付费执行前的缺项

| 关口 | 需要补齐的证据 |
| --- | --- |
| **实施前** | 用户确认本文的三个接线与边界。按仓库规则记录 #132 本切片的 Design Basis、文档版本、唯一写入负责人和验证义务，完成适用独立架构评审并取得明确 `IMPLEMENTATION_READY`。优先核实真实的样本绑定入口、首项 Claim 的首次领取信号、两组共同的最终发送门禁，以及 Run 与资源结算回执的只读确认路径。缺少原 owner 能力时只报告该缺口和最小改动，避免在实现中临时引入新的状态 owner。 |
| **实施验证义务** | 用合成传输和隔离库证明两组同快照同 validator、固定组最多一次、Agent 的决策/修复计入 6 次；错用途/快照/协议、输入超限、报价超出 ¥0.18、项级预算耗尽、重复/并发领取均为零额外发送。覆盖「已发后缺用量」「调用终态写入成功但结算失败」「运行提交回执丢失」「进程中断后原键找回」并确认不启动后续槽位、不重复扣减和不自行释放。验证冻结样本摘要、共同证据、可核的实际请求和各拒绝原因。 |
| **付费先导前** | 另行确认专用组织/成员及用量额度、准确模型连接/用途/服务等级、冻结价格/汇率/计量上界、60 项绑定和冻结执行顺序。取得付费执行授权，并确认此前候选 P95 45 秒、单任务 60 秒及与现行更严时限的关系。环境及路由证据目前 `UNKNOWN`。 |

本稿只记录候选设计。离线回归、付费先导、正式评测均为 **NOT_RUN**。


## 7. R1 样本绑定修订候选

本节仅处理 R1 的样本绑定阻塞。R1 复核的原稿 SHA-256 为 `2351cd0668637ea8f5b7854600046298e4442cfc8a421e2415136fb67955663c`。以下方案待用户确认和原 Reviewer 增量复核，状态仍为 `NOT_IMPLEMENTATION_READY`。

### 7.1 需要确认的合同

建议允许 **仅 #132 冻结清单内的 30 个合成商品，使用现有 SRC/Catalog 的原发布回执进入内部标题评测**。标题评测明确识别独立的 `title_eval` 上下文，复用现有 `agent.Binding` 的字段、原标题 Agent、原授权、原回执和原计费。生产采集路径继续要求已发布的真实采集操作，自有商品的非空标题规则保留。按现行源码，#132 需要这条窄的来源准入：正常采集持久化校验规范 offer、URL 与采集 trace，自有商品导入拒绝 C 组的 8 份空标题，原预演的内存采集回执只属于离线替身。[真实采集绑定](../../internal/app/httpapi/product_agent_application.go#L450) · [采集持久化校验](../../internal/integration/persistence/product/acquisition/repository.go#L448) · [自有商品校验](../../internal/product/collection/own_product.go#L39)

### 7.2 从冻结样本到两组运行

1. **限定入口和身份。** 只有本次专用隔离环境的内部 #132 入口能装配此上下文。运行方从现行已验证会话和实时组织权限取得唯一的组织、操作者和成员，须同时具备原来源写入权限及标题 Agent 使用权限。由服务端固定批准的清单版本、清单及输入文件摘要、30 个 case ID 和原标题任务配置；调用者不能提供替代清单、任意 Catalog 版本、回执、身份、URL 或扩大批次。普通 HTTP 入口、其他 Agent 和普通标题执行路径不接受该上下文。使用的目标键固定为已有标题配置中的 `shein`，其用途是本次标题策略和本地事实读取；本批禁止外部平台读写及发布。
2. **原 owner 发布与留证。** 由现有 [受控发布器](../../internal/app/productsourcing/application.go#L24)和同一 PostgreSQL 事务的 [SRC/Catalog 发布器](../../internal/integration/persistence/product/sourcing/repository.go#L130)写入。使用现有 `controlled_snapshot/v1` producer。从冻结 case 派生中性 `manual_import / local-synthetic` 来源，保留 case ID、源版本、标题（含空值）、品牌、类目、描述、属性、缺失字段及原证据 ID／快照 ID／商品事实摘要。发布元数据记录冻结清单摘要和 case ID，原输入文件保持原样。仅派生发布时省略 `.test` URL 和占位图片，避免 Canonical 投影把它们带给模型；评分标注保留在评测侧。SRC owner 与 Catalog 原子写入证据、准确版本和原发布回执；不创建采集操作、第二套商品事实或第二份发布回执。
3. **固定绑定。** 组织、操作者、冻结清单摘要和 case ID 派生稳定的 `ProductKey` 和 `PublicationID`，固定初始基线 0。以原事务回执提供的 `CatalogVersion + CatalogPublicationID` 建立 `title_eval` 绑定，`ContextID` 指向原 SRC 发布 ID。两组共用同一绑定、证据和策略版本，每个组使用已冻结的独立运行原键。标题准入从现有 SRC reader 与 Catalog 按准确 ID 和版本重读，核验当前身份、清单归属、producer、原回执及输入、证据和快照摘要；核对本样本确实等于批准的派生投影。验证者使用原 [证据 ID 规则](../../internal/product/review/proposal/validation.go#L72)。既不选当前最新版本，也不接受来源相同但版本不同的替代回执。批次启动前 30 个绑定必须全部核验通过。
4. **失败和恢复。** 同键同输入复用 SRC 已有发布回执。提交响应丢失时仅走原 [Verify](../../internal/product/sourcing/publication.go#L169) 按同一命令核对。查无记录或核对失败时停止准备，不换键、不取当前版本、不启动整批。同键不同内容、越企业、撤权、摘要漂移、证据与 Catalog 不一致均拒绝。标题准备、首次领取、证据读取、模型最终发送和结果读取继续执行原权限校验，并携带同一内部评测作用域。评测只产出可人工评分的结果，权限不延伸到提交审核、采用、外部采集、刊登或其他真实业务操作。

### 7.3 本项验证和阶段边界

适用的隔离验证需覆盖 30 份原发布回执和 60 个运行槽位，两组的准确 Catalog 绑定一致，C 组 8 份原标题仍为空，缺品牌和其他缺失字段仍缺失，原证据能经过 `canonicalinspect` 进入现有标题校验。确认占位图、URL 和评分标注没有进入模型输入。拒绝清单外、任意发布 ID、假采集回执、版本替换、身份或租户漂移、撤权和普通入口直接使用该上下文；用原键验证响应丢失与重复准备。用隔离库及合成传输覆盖上述条件，付费先导另行授权。

R1 关于首次领取和原调用 ID 结算回执的两项实施验证义务仍适用，既有预算、结算 owner 和停批策略沿用前文。本次只新增样本来源准入候选，独立增量复核只检查 §7 对 R1 阻塞的处置及它与上述两项义务的交界。正式评测 **NOT_RUN**。
