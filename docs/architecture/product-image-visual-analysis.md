# 商品图片单图视觉分析：C2-2 最小设计候选

状态：**IMPLEMENTATION_READY / FROZEN（仅限本次 C2-2 单图分析架构）**。Design Basis：Independent Architecture。
Refs #134 / #128；当前唯一负责人、原独立工作区继续，无新增 Writer / Reviewer / 派工。
用户已[认可 D1–D3](https://github.com/henryxu1008/shuomi-ai-commerce/issues/134#issuecomment-6592979123)，
后续授权原 Writer 沿原工作区和 PR 继续开发、复用原唯一 Reviewer 做必要准入与最终检查。
原唯一 Reviewer 已对准确设计 HEAD `75730fc9ca6fe1becb045adb3e54ea3286fb6549` 完成范围内 R1，
结论 DESIGN_REVIEW_PASS / IMPLEMENTATION_READY，无设计级 BLOCKER；
[准入及三项 IMPLEMENTATION_TEST 回执](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1078#issuecomment-6602880445)。
本文件为冻结实施基线，尚不表示视觉实现完成或产品验收通过；依既有授权先回归再最小实现。
不新增人员或 PR，不包含合并、部署、关单或真实调用权限。

2026-11-03 用户进一步确定[企业统一模型配置与官方价折算](enterprise-shared-model-configuration.md)：
先有默认渠道，企业在一处统一配置文本/图片生成/多模态默认模型，所有成员共用，
各业务读取已配置模型。GPT/Gemini/Qwen 均保留支持目标，按实际模型官方价折算 AI 点数。
这替代“先仅支持一家”及“成员每次分析另选模型”的建议；本用途读取企业多模态默认配置。
公开型号/价目信息已有本会话只读调查；Mini 文本/多模态、Flare 生图准确快照及汇率/微点规则
已获后续产品认可；统一配置、官网价/冻结 FX、微点及原账本安全基础已完成独立工程检查。
基础准确 HEAD/CI/回执沿 PR 记录，本用途不重复审核未变基础。
C2-2 的实际图片输入、保守报价、确认与结果接线已获本用途架构准入，尚须实施；D1–D3 边界不变。
Flare 完整费用上界/官方直连执行仍 NOT_READY；三类全部合格才原子发布，不改为部分或 pending current。

## 1. 用户结果、依据与阶段

建议交付：商品操作人员从现商品图片流程选择一张当前有权读取的图片，
确认本次图片和费用上界后获得主体/构图/明显质量疑点及非执行性建议。
示例结果是“主体可能被裁切，请核对或换图”，不是商品属性、图片合格或发布许可。
分析结束不改图、不生成、不保存、不批准、不回填商品资料；后续操作仍走原入口。

依据：

- [原 C2-1 / C2-2 阶段决定](https://github.com/henryxu1008/shuomi-ai-commerce/issues/134#issuecomment-6575806435)：
  C2-2 延期/未实现；重新启动需要明确范围、模型/费用、证据、授权及 UNKNOWN 合同。
- [本轮设计整理接续](https://github.com/henryxu1008/shuomi-ai-commerce/issues/134#issuecomment-6592753393)。
- [D1–D3 产品认可与 Draft PR 授权](https://github.com/henryxu1008/shuomi-ai-commerce/issues/134#issuecomment-6592979123)。
- [C2-1 冻结设计](product-image-facts-advice.md)、[Commerce Tool](commerce-tool-foundation.md)、
  [Agent 运行合同](product-agent-runtime-contract.md)、[文本模型治理](product-agent-text-provider-neutral.md)、
  [模型与费率管理](model-route-pricing-management.md)、[业务职责](../product/business-agent-responsibilities.md)。
- [当前领域边界](project-boundaries.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)。
- [UI / IA Authority](../product/final-ui-ia-authority.md)：Figma `tg48P46SSXl6TBy9lZwg63` 非归档 `31:463`。
  复用商品图片职责入口和既有结果组件，不新增顶层智能体/工作台。
  本轮没有重新读取 Figma；实际入口位置、控件与布局尚未冻结，不能从历史截图宣称视觉准入/验收。

原调查基线：main `c6f53e1a928862e556bbf0a1883c2af03dbedfb7`；新增接缝以原 PR 当前有限安全基础为准。
旧 SDK/历史设计的视觉能力描述不作为当前正式链路已可用的证据。
当前 C2-1 / C3-2 的 Ready、CI、旧版本决定不自动覆盖本新增带图模型与计量合同。

### 1.1 已确认的产品决定

| 决定 | 推荐候选 | 当前状态 |
| --- | --- | --- |
| D1：分析目的 | 主体可见性、构图/裁切、背景和明显清晰度疑点，输出只读意见；不做精确 OCR、商品材质/规格回填、侵权判定或平台合规保证。 | 已产品认可。 |
| D2：图片范围 | 一次一张，选自当前商品准确版本的已有来源图或已批准素材，身份分开；不接受任意 URL、新上传或多图比较。 | 已产品认可；不把来源图伪装成批准资产，也不要求先批准才能分析。 |
| D3：调用/费用 | 用户显式确认获准模型接收本次图片及费用上界；一次视觉调用直接产生说明和建议，不追加第二次解释/自动重试，不提高原总调用/费用上限。 | 已产品认可；不授权本轮真实调用，模型、费率、图像计量依据缺失仍不可执行。 |
| 企业共享配置 | 本次分析使用企业在统一入口选择的多模态默认模型；成员确认实际模型和费用，不另存个人/页面模型配置。 | 已产品决定；共享选择/官网价/微点基础已复核，新增视觉消费仍须本用途准入。 |

Must：准确图片/版本和实际内容绑定；fresh 身份/资源授权；一次受治理派发与真实计量；
明确失败/未知；模型判断不变成事实/批准；结果只读；原键和历史可读。
Should：原 owner、Registry、Model/ledger、Run 和页面复用，只补这个用户结果所需的接缝。
Out of Scope：批量/多图比较、OCR/属性回填、自动选图/生成/改图/批准/发布、
平台视觉合规/侵权或真实性裁定、模型市场/BYOK/自动切换、通用付费 Tool 网关、
新运行/计费/审批平台、迁移/兼容/旧运行改造、部署和真实操作。
没有新增 Accepted Risk；延期/未知不写成 PASS。

## 2. 已有能力与实际缺口

| 已核能力 | 可复用 | 不能据此宣称的能力 |
| --- | --- | --- |
| `assetinspect.ImageAdviceDefinition` / `asset.AnalyzeImageFacts` | 批准库存的身份、尺寸、角色、位置与规则检查；现结果面板。 | `product.image.analyze@v1` 不读图片字节，不评价图像内容或质量。 |
| Catalog `IdentifyImages`、Source/Asset reader | 当前版本的来源/图片引用和批准谱系。 | ID/URL/ReferenceHash 不普遍证明实际图片字节未变。 |
| `PublicImageProbe` / `httpimage` / `ImmutableObjectStore.ReadObject` | 安全有界读取、格式/尺寸/内容哈希、已有不可变对象读取。 | Probe 不是视觉模型，原生图片校验入口也不是通用分析命令。 |
| `TextInputIdentity` / `QuoteText` / governed Eino executor | 组织路由、配置/价格快照、claim、资源预留、单次 handoff、usage/UNKNOWN。 | 现正式输入是 System/Prompt 字符串，文本字节上界不是视觉 token/计费证明。 |
| 当前生图/编辑适配器与 ImageAgent | 既有图片操作/资源/批准流程。 | GenerateSlot、PrepareImageConfiguration、InitializeRun 不能包装为只读分析。 |
| Eino Tool 准入 | 同一 Registry 的已批准 read/compute + unmetered + SideEffectNone。 | 不能把真实视觉调用藏在该免费 Tool 中，也不能只改提示词/allowlist。 |

当前阻碍是三个确需代码/合同的接缝：实际图像内容输入与证据、视觉报价/usage 映射、
独立只读结果及消费者。属于 Independent Architecture，不是只差旧证据映射。

## 3. 最小技术候选与 owner

推荐 A：**原事实 Tool 保持；在原受治理 Model 层增加有界图片输入，单次调用返回 typed 只读结果。**
不把 C1 的文本路径决定自动扩展为本次批准；D1–D3 已单独认可，本候选仍须适用架构复核。
替代 B 是模型可调用的付费视觉 Tool，涉及嵌套 invocation/预算与 metered gateway；
目前的单张显式分析场景没有证明需要该扩展，不作为首阶段实施路线。

| 合同 → 实现 → 注入 → 消费 | 拟复用 owner / 窄改动 |
| --- | --- |
| 用户选图 → 原 Catalog/Source/Asset 授权 → 当前图片读取 port → 原应用确认 | 原 owner 返回准确选择与内容，模型不供应可信参数；不在 App 新建规则/事实 owner。 |
| 配置/请求/模式/报价快照 → AgentConfig 原 Prepare 事务 → 分析执行定义 → 显式确认 | 单独的分析用途与准入，不借生图计划/许可；现图片职责入口不增加顶层智能体。 |
| 确定性事实预读 → 原 Registry/Gateway/audit → 同一 Agent Run → 一次带图 Model | 原运行器只补本用途固定预读接线；不能先让模型付费挑 Tool，再隐藏一次看图调用。 |
| 带图 Quote/dispatch/usage → 原 capability/Model executor → 原 Invocation ledger/资源 owner | 复用成熟 Eino/供应商 SDK；不新建模型调用/计费系统，不直接从领域调用 provider。 |
| typed 视觉结果 → 固定领域 validator → 原 Run/checkpoint → 原图片结果组件 | 历史只读，不交标题 Review、Catalog 写入或 ImageAgent 批准。 |

候选分析执行定义独立于当前 `product.image.agent` 生图定义，并归同一图片职责显示。
候选命名 `product.image.analysis@v1.0.0`、operation `product.image.analysis`、
结果 `image-analysis-result-v1`；这些不是已注册或可用状态。
不改变当前 title/content/materials/生图的版本/历史/权限资格；不自动给所有 Agent 加能力。
分析的新定义、精确 schema/模式快照与单用途 runtime 接线必须一起冻结，不能只加目录卡片。

### 3.1 准确接缝与执行身份

内部执行 ref 为 `product.image.analysis@v1.0.0`，但不建立第二 activation、template 或模块。
它唯一映射到既有 `product.image.agent` 的 activation/epoch/revision 和 registered-agent mode subject。
原 `ResolvedModeSubject` 要求 Subject.ID 与 Definition.ID 一致，故 mode 仍绑定 image→image，
不能伪填 image→analysis 或放宽通用 Valid。分析 ref/版本另进入原配置 Snapshot 的
`image-analysis-config-v1` 段和 Prepare 指纹，由本用途 factory/final guard 验证固定映射。
Snapshot 的激活与模式身份仍为原 image 定义；Run 的执行定义为 analysis，二者责任明确。
原 Store/Guard 对本用途作精确版本分支，不把通用文本快照或生图配置格式当成分析格式。

入口按 WorkbenchAgentUse、LocalAgentWrite（原本地 Run 命令）和准确 source/asset 读权限
单独投影分析资格；配置沿已有 WorkbenchAgentConfigure。不得借商品业务写权限、
生图 permission/permit/template/整体 readiness，也不放宽原生图或 Flare 门禁。
HTTP、Prepare、实际预读、新执行和最后 handoff 均重核 verified org/actor/effective member 与资源。
历史 GET 按当前读权限披露原记录，不因读失败修改结果或重新调用。

| contract → implementation → injection → consumer | 原 owner 与本次窄接线 |
| --- | --- |
| Catalog IdentifyImages/VersionedSnapshotReader、Asset BoundedApprovedInventoryReader → 原应用选图 | source 绑定准确 Catalog 原 index ID；approved 绑定准确库存摘要/Asset ID。缺库存不当空，不按今日索引重解释旧图。 |
| 原 SourceSelection/SourceByteReader、GeneratedMaterialReader → 准备与最终内存 buffer | 两种来源分别授权，安全读取实际 bytes 并哈希；不复制 blob、不从浏览器或模型接 URL。 |
| AgentConfig StartCommand/Snapshot/Prepare/Guard → 原 Run DB → analysis factory | 原 JSON 快照增加版本化分析段，参与指纹；原 Request 保持可比较，只持 ConfigurationSnapshotRef，不塞 slice/pointer。原 Prepare 行锁/事务无新表/新 owner。 |
| SharedCategoryReader → 原 managed route resolver → analysis ModelProfile | 新 operation 读 multimodal；SharedCatalog 核实际 serializer/mapper/保守界证据，不能只删除 VISION_CONSUMER_NOT_IMPLEMENTED 提示。 |
| 原 TextInputIdentity/QuoteText → 同一 Executor/GuardedClient → 原 Invocation/资源 owner | 增加仅该 operation 可用的 typed 单图描述和临时 bytes 输入，保持原文本无图 hash/预算不变；不复制 Generate/账本/结算事务。 |
| 原 Registry/Gateway/audit → Eino graph → 一次 Model → typed DomainProposal | 本用途固定真实预读，不先调用模型选 Tool；固定 DomainValidator，只读记录到原 Run/checkpoint，不进入 title Review/Apply 或图片批准。 |
| 原 HTTP/BFF → own-product-images-page 与图片职责组件 | 页面提交原 ID/版本/key，显示模型、费用和一次收图确认；只读消费原结果，不新增每页模型/价格配置。 |

`product.image.analyze` 现有 Tool 只计算 Catalog/ApprovedInventory 事实，不读取图片内容。
其成功不冒充视觉 bytes 预读。图像内容哈希由上述真实 reader 计算；预读输出/审计保留有限身份、
hash 与元信息，固定事实 Tool 的实际输出仍沿原定义/audit，二者在快照/报价中准确绑定。
现 Eino graph 从 model 开始，校验失败会 repair、resume 会回 model；本用途须窄接固定预读顺序，
预读计原 Steps/timeout/audit 而不消费 ModelCalls/付费许可，然后最多一次 GovernedModel。
预读失败、无效结果、已尝试调用、UNKNOWN 均不得经 repair/resume 再发送；其他用途行为不变。

分析的配置段完整选择/执行 ref/内容/quote 参与原 Prepare fingerprint，保留 Snapshot 8192 字节上限；
不保存 bytes/base64/Key。analysis 原键为 org/actor/analysis-definition/key，跨 context 查找旧回执，
换图、换上下文或异载荷同键冲突；复用 Materials 的窄查键/锁行为，不改其他用途的键合同。
原 Request/Run 只引用该 immutable snapshot，保留原 claim/fence/checkpoint 和 2MiB Run 上限。
新调用次数固定至多一次，Tokens/CostMicros/运行时取原应用配置、实际 route 与模式允许的更小上限；
原 Guard 为上述固定映射验证 ceiling，不引入可编辑的第二预算或放宽原限制。

Legacy decision: **EXTRACT | RETIRE**。复用当前 source/asset 读取、配置回执、Run/guard 与原 Model/资源事务；
不依赖退休 ProductImage task/queue、固定点价或旧 current，无兼容/迁移或第二事实源。

## 4. 图片选择、实际字节与资源边界

服务端选择绑定至少包含：组织/成员、ProductKey、publication/准确 Catalog version、
来源谱系及适用 apply receipt、source kind、原图片 ID；批准素材另含库存摘要/批准身份。
来源图片 ID 复用原 Catalog owner 对该准确版本的标识，不按“当前第几张”跨版本重解释。
普通选图不新增 Store 前置；本候选不评价平台特定视觉规则。

1. 通过原 owner 授权读取所选图片。已有持久化对象优先走授权 owner 的 ReadObject；
   来源是外部链接时，沿 `httpimage` 的公共地址/DNS/redirect/字节限制，不接受浏览器/模型任意 URL。
2. 图片读取 owner 验证实际格式、像素/字节上界并计算实际 SHA256；已有 ArtifactHash 时交叉核对。
   引用摘要、文件名和批准状态不代替内容哈希。通用读取只抽取已有合格纯行为，
   不整体调用带 Store/save 语义的原生 Probe/准备任务。
3. 本候选不裁剪、压缩或重编码；首阶段建议只接受已支持的 JPEG/PNG。
   图片二进制不进入文本 Prompt、Tool JSON、日志或工作流历史；实际 provider wire 使用受控图片 part，
   不让 provider 另取原 URL/签名链接或读取内部凭据。
4. 单图 bytes、pixels、序列化后的总 wire 与响应受各自准入上界的更小值限制，读取前/解码前/发送前分别检查。
   当前文本 profile 总请求上界不超过 1MiB，本候选不默默提高；base64/协议开销必须计入。
   超限明确不可分析，不自动变更图像、扩上限或按截断内容宣称已覆盖。

Prepare 冻结所选图片的原身份、实际内容 SHA256、媒体类型/字节/宽高、配置/价格及报价摘要。
此时真实 Run 尚未创建：冻结同一已准入 profile 的完整费用上界，原 Run owner 在 Start 时生成真实 ID；
实际 Invocation Quote 再绑定真实 Run、完整输入及同一图片，必须落在该 profile 和已确认费用上界内。
准备不使用虚构 Run ID，不创建第二调用/预留事实，也不为页面报价提前扣款。
原图链接可能变化，不保证来源图片都有不可变对象；真正发送前再次取得相同内容并验 hash，
不一致则发送前拒绝，不能借同 key 分析今天的新图。首阶段不为此复制全部图片或新增持久化事实源。
已提交结果的历史找回不下载今日图片，也不重做分析；保留证据不可读时说明缺失，不能补造旧图。

## 5. 准备、确认与一次执行

1. 用户在原图片职责流程选一张已有图片；原有效 Product/source 权限和准确版本必需。
2. 原 reader 读取并固定实际内容，原配置 owner 形成不可变分析准备/报价回执；此时无模型派发。
3. 页面显示图片身份/版本、分析范围、获准模型、费用上界和不确定性，用户显式确认。
   准备不等于执行许可；未确认不外发图片给模型，不预留付费调用成功状态。
4. 同一原 key Start 先找回已存在准备/运行/结果；新执行重查身份、图片权限、模式、配置和报价资格。
5. 本用途沿原 Registry 实际预读及持久化 audit，随后在原 Run 下执行一次带图 Model；
   固定预读不伪造“模型请求过该 Tool”。如现 runtime 无该接缝，只能按本用途窄补，不能另起状态 owner。
6. 固定 validator 检查结果结构/绑定/引用/只读字段，Run 保存实际结果或失败；所有模式均不采用/批准/发布。

公共命令身份为 org/actor/分析执行定义/request key。改变图片、来源种类、版本、目的或有效配置等意图应冲突，
不能靠新 ContextKind/ContextID 静默建立第二运行；沿原 Prepare 事务/行锁解决，保留其他用途自身合同。
原准备回执在尚未 claim 时也可按原键读回；读回不绕过停用、撤权、过期价格或新执行准入。

## 6. 视觉模型与费用：准入必须补齐

当前模型/价格配置按[企业共享合同](enterprise-shared-model-configuration.md)读取多模态类别，
不能直接按原标题 operation 或生图 points/image 推定。模型/渠道变化须在原新准备及最后发送核验；
旧请求不静默换模，已提交结果仍按原事实读取。该依赖同时覆盖官方价、汇率及细点数单位，
不在本用途创建独立可编辑点数费率或第二配置源。

当前 `QuoteText` 只绑定文本。新窄图片输入/Quote 必须将实际图像摘要/媒体参数、模型/配置、
prompt/output schema、完整输入及计量策略纳入同一原 input hash；不能将 base64 文本长度当视觉 token 定价。
准确模型/协议固定为下述 Mini / OpenAI Chat Completions 候选，不因 SDK 存在图片方法就自动准入。

每个开放 route 必须明确：受控 endpoint/凭据引用、图像输入支持及大小/像素限制、
最坏图片+文本输入用量上界、真实 usage 完整性、厂商官网价/FX/单位版本和返回结果关联。
输入/输出 token、图片单位或其他收费项如何合并，必须由该 route 的实际依据确定。
缺上界、usage 映射、模型/费率或配置时保持 DEPENDENCY_MISSING / NOT_READY；缺用量不按零结算。
本轮只读核官方协议/计量/价格文档，没有读取保留实例配置/凭据或发出模型/业务平台调用；现场准备 UNKNOWN。

候选预算：一次显式请求最多一次视觉 handoff，直接返回说明/建议；不再调用模型改写解释、
不做模型 repair/自动重试/切换供应商。固定 validator 失败保留该次实际用量和失败，不追加收费修复。
不提高原总调用/费用/预留上限；报价超过现额度/预算/用户确认上界即拒绝。
实际 quote/claim/预留/guarded send/usage/结算、原键与 UNKNOWN 继续原 ledger/资源 owner，
没有第二个 AI 许可、计费账本或可重放的 provider owner。

### 6.1 Mini 单图协议与依据

准确模型为 `gpt-5.4-mini-2026-03-17`，默认 OpenAI 官方直连 `/v1/chat/completions`。
首个候选协议固定 `service_tier=default`、非 stream、`n=1`、`store=false`、`reasoning_effort=none`，
一个 system 与一个 user 消息（text part + 单个 JPEG/PNG base64 image_url part），显式 `detail=high`。
不带 tools/tool_choice、预测输出、音频、视频或其他输出模态。只用 `max_completion_tokens`，
等于冻结的更小业务输出界，不与旧 `max_tokens` 并存；原 Guard 核实际 wire，而不只信 SDK 参数。

2026-11-03 只读核验的官方依据：

- [Mini 模型页](https://developers.openai.com/api/docs/models/gpt-5.4-mini)：准确快照、图像输入/文本输出、Chat Completions、reasoning none 和标准 token 价格。
- [图像输入/patch 计量](https://developers.openai.com/api/docs/guides/images-vision#model-sizing-behavior)：Mini high 的 2048 像素边长/2500 patch 处理界、32×32 patch 和 1.2 倍率；最终计数可能差一个 token。low 在此型号不是更小固定费用。
- [Chat Create](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create)：typed 图片输入、max_completion_tokens 覆盖可见及 reasoning 输出，原 P/C/T 和 cached usage。
- [官方价格](https://developers.openai.com/api/docs/pricing)：Mini 标准同步、无地域附加方案的输入/缓存/输出为 USD 0.75/0.075/4.50 每百万 token。执行仍读平台已发布的准确适用价目/FX，不硬编码默认价格。

仓库锁定 Eino OpenAI v0.1.13 已有 multipart、MaxCompletionTokens 与 ReasoningEffort，
复用原组件作本 operation 窄接线，无需新 SDK/provider client；现 NewComponent 使用 MaxTokens 仍须修正该用途并验证。
GPT/Gemini/Qwen 支持目标不取消，但不得将 Mini 的图片界应用于其他准确模型/代理，缺其自身依据仍关闭。
这些是协议依据，不是当前账户/凭据可用性或模型判断质量证据。

### 6.2 同一 Quote、上界与实际用量

单图 typed 描述包含准确 selection/source或approved 身份、版本、实际 SHA256/MIME/字节/宽高、
`detail=high`、计量策略 `openai-mini-image-high-v1`、配置类别摘要及原 ModelProfile/prompt/schema。
全部进入原 input hash；服务端将真正发送的内存 bytes 与该哈希交叉核验，不信浏览器或模型自报值。
图片二进制不进入持久化 envelope/文本 Prompt/Tool JSON/日志/checkpoint。

- 只接受非动画 JPEG/PNG；单图 ≤768KiB，各边 ≤8192、像素乘积 ≤16,777,216，与原 reader/业务更小界取 min。
  先核字节/DecodeConfig/尺寸乘积再解码，拒绝损坏、伪 MIME/多帧；不裁剪、压缩或重编码。
  最终总 wire（包含 base64 和 JSON）≤min(原 route MaximumInputBytes,1MiB)，实际可能更早拒绝，不能据原图字节数直接准入。
- `Bimage=ceil(2500×1.2)+1=3001` 是含官方一 token 误差的保守推导，不是实际扣费量。
  `Btext` 沿原 UTF-8 字节保守计数和1024 framing余量，排除 base64/data URL 字符；
  包含实际 text/绑定/schema。`Bprompt=Btext+3001` 必须不大于原 profile 最大 prompt。
- `Boutput` 取原 profile、Agent limits/模式和本用途允许的更小界，并由准确 max_completion_tokens 强制。
  完整 Bprompt+Boutput、CostMicros、费用确认或调用次数不能容纳则发送前拒绝，不放大上限/截断输入。
- R 沿原官网价/冻结 FX 公式，按全部 Bprompt 非缓存价与 Boutput 输出价保守计算，最终一次 ceil 微点。
  缓存折扣只按完整实际 usage 结算；原币渠道成本只约束原预算，不额外收费。
- 实际 P 已含图片输入，沿原 mapper 计算 `(P-K)×输入价+K×缓存价+C×输出价`，最终一次 floor；
  不另加一次图片费用。原记录的 maximum prompt/completion 与报价一致；缺缓存/矛盾/越界保留原 R/UNKNOWN，
  不用尺寸估计、base64 长度或模型自报计数补 usage；原 R−A、debt-first、月限和 RecoverDue 不变。

原 Guard 必须验证真实 wire 的模型/唯一图片 part/hash/detail/输出界/固定协议字段与 quote 一致，
最后 fresh 授权、配置/价格/报价资格及重读内容均成立后，才走原唯一 provider handoff。
只有原 guard 已知 no-send 终态且资源 release 已成功才能返回已知未发送；其他情况继续原 UNKNOWN，不重发/猜测释放。

### 6.3 确认与原准备回执

原 reader 先读取并冻结实际图片，再由原 Prepare 事务持久化有限的版本化快照/报价；此时无模型外发。
页面显示准确图片/版本、实际模型、费用上界（微点及精确显示点数）和分析范围，用户显式确认模型接收图片。
Start 携原准备 ID/digest、原请求 key 和已显示内容的确认摘要；服务端核原 actor/org/member、
selection/报价与确认摘要一致，不从客户端单个勾选或当前页面状态推断同意。
换图、配置或报价须新显式意图，不能改旧准备/自动换 key。准备或运行 ACK 丢失先查原回执，
已提交结果不被今日来源/价格覆盖，历史 GET 不下载、报价或执行。
原 ConfigurationSnapshotRef、Run/checkpoint、Invocation 当前52列及 official_tariff_json/billable_usage_json
继续原 owner；不建另一张计费表或调用许可。本用途增加的准确 operation/输入版本资格在同一原链路校验。

## 7. 只读结果与消费

候选 `image-analysis-result-v1`：准确 selection/content hash、实际 AI invocation 与模型/配置版本、
已分析覆盖/未知、最多八条 observations 和八条非执行性 suggestions（每项理由有界）。
每条观察只引用本次图片；主体可见性/构图/背景/清晰度采用 possible_issue / no_obvious_issue / unknown，
不使用“合格/侵权成立/商品属性已证实”等权限或事实状态，不提供虚构总体质量分。
字数/数组/模型输出受原 64KiB ModelOutput、profile 更小上界及 Run 2MiB 限制；strict closed schema。

模型视觉判断是带不确定性的建议，不是 Catalog/source/批准事实。
固定 validator 只能证明结构、准确绑定、执行/用量/结果引用和字段范围，不能证明视觉判断真实正确。
原 Run 的结果保留有界模型原始 JSON 字符串及其 typed 只读视图；fixed validator 重解码并核两者相等，
将原始字符串 SHA256 与同一组织/成员/原 Run 的 native Invocation OutputHash 核对。
JSON 空白不改变意见，但不能通过重新序列化伪造原响应 hash；不保存图片 bytes/base64/凭据。
不能用 ToolRef/CallID 冒充真实模型调用，也不能把模型自报 image hash 当服务端证据。
现 `image-advice-v1` 及 `product.image.analyze@v1` 保持纯事实语义，不原位扩入视觉付费结果。

建议只允许核对/换图/到原图片操作入口，不含命令参数、任意执行 URL、生成任务、批准令牌或商品 patch。
结果页显示原图片/版本/本次覆盖、模型意见、费用与实际终态；已知失败不能变为 OUTCOME_UNKNOWN 或成功标题。
若复用 human_review_required，必须同时核 StopReason/固定校验与实际结果，不以该 phase 单独宣称分析成功。
关闭/恢复页面和历史 GET 不重新下载原图、不重新报价或调用模型；权限重验后读取原 recorded 证据。

## 8. 失败、原键与历史

| 边界 | 候选处理 |
| --- | --- |
| 准备未提交 / ACK 丢失 | 同 key 在原 owner 查询/幂等重试；异图片/版本意图冲突，不自动换 key。 |
| 准备已提交、尚未 claim | 原回执可读；已停用/撤权/报价失效只拒绝新执行，不伪称模型已执行。 |
| 发送前图片内容/权限/route 漂移 | 原 guard 拒绝；仅有现协议完整 no-send 终态证据时按已知未发送处理。 |
| claim/预留 ACK 不明 | 原 ledger owner 处理 UNKNOWN，不以“只是分析”释放预留或再派发。 |
| provider handoff 后断线/无完整usage | 原 invocation/预留/UNKNOWN 保留，不换模型/另起调用，不把缺失usage写零。 |
| 已知响应但结构/绑定校验失败 | 展示失败并保留真实用量；本用途不自动追加模型 repair。 |
| 结果已提交但响应丢失/页面重开 | 原 key 只读原结果/失败/未知，不刷新输入/规则或重调模型；读故障不改写原 durable 结果。 |
| 商品/库存当前head已变或图片不可再读取 | 历史按原绑定和现读权限披露；不重解释为新图。新分析需新显式意图，不能接续原未知调用。 |

分析定义新版本只执行自己的准确 snapshot/input/output，旧分析历史按其原格式读取；
首次没有本能力既存运行需要迁移。现 title/materials/生图历史不转换成分析历史，原执行资格不由本候选撤销。
若以后确需改变这些当前产品合同，须具体报告并取得新决定，不能沿用 #1077 的旧 D3 外推。

## 9. 后续必要验证与停止点

仅在本用途适用独立架构复核完成、达到 IMPLEMENTATION_READY 后，依已获继续开发授权，
先扩原测试：图片读取/大小/漂移；Quote/input hash/usage 与唯一 send；
原 Prepare/Run 的跨图片 key 冲突、并发/ACK 未知与历史；固定结果边界；页面不触发生成/保存/批准。
只用合成图片/transport 与任务自有临时 PostgreSQL，沿既有 fixtures，不新建 runner/验收平台或重做全仓审核。
真实视觉准确性、误判/不确定性及用户效果另行验收；技术测试不能替代。

当前 Mini 协议/计量依据已补齐；原 Snapshot/Prepare、固定 image activation/mode 映射、
真实 Tool 预读后单次 Model、准确输入/结果及 native Invocation/资源回归已有仓库实现。
这些组件已用任务临时 PostgreSQL、合成身份/事实/transport 组合验证；生产 reader 装配、
HTTP/BFF 显式确认与原页面/历史结果已接入，当前候选仍待准确 HEAD CI 及最终工程复核。
SharedCatalog 仅在原多模态类别存在完整单图准入定义/官方价目时提供该用途；
没有定义、证据或完整三类资格时保持不可用，不解除 Flare 等独立门禁。
真实账户/凭据准备仍 UNKNOWN。原 PRECHECK_ONLY 已由准确设计 HEAD 的正式 R1 取代，
该范围 IMPLEMENTATION_READY；不把设计准入或旧基础 CI 代替实现、自检和最终工程复核。

预核提出三项 IMPLEMENTATION_TEST：固定预读/单次执行、analysis→原 activation/mode 的固定映射、
原 Snapshot/跨 context key 与内容冻结。§3.1、§6 已明确最小合同，不新增 owner/SDK/账本。
这些义务未满足时阻止本视觉路径交付；无需重做已冻结基础架构或新增通用框架。
实现时复用原 fixtures，须覆盖：

1. source/approved 的准确身份、跨组织/版本拒绝、缺库存≠空、大小/动画/损坏及发送前内容漂移。
2. 3001保守分量、base64不计文本费、完整wire界、准确detail/model/output、唯一send；旧文本无图hash不变。
3. 原完整cached usage一次扣款、缺字段/矛盾/越界UNKNOWN、0实扣及差额释放；不复刻整套账本测试。
4. 原Prepare/Guard/Run的同key并发/ACK找回、异图片/上下文冲突、未确认/撤权拒绝、真实预读audit；
   中断继续原claim/fence，UNKNOWN/invalid/恢复不触发第二次调用或repair。
5. 原页面/BFF的图片/模型/费用确认、未准入原因、原键与只读终态；确定失败与UNKNOWN分开，
   不触发生成、业务保存、批准或新增每页模型配置。

本阶段设计提交/准确 HEAD CI/复核证据维护原 PR，不新增派工或评审人员；
仅在该冻结设计内实施；已批准基础不重复完整评审，修复只复核相关增量。
缺证据 UNKNOWN；运行、真实保存/恢复、浏览器、视觉质量及产品验收 NOT_RUN。
#47 PostgreSQL UNKNOWN、#307 OPEN、#733 咨询 NOT_SENT；保留所有实例、数据和会话，不合并/部署/关单。

### 9.1 当前仓库接缝

- `agentconfig.ImageAnalysisConfiguration` → 原 Prepare JSON/指纹/跨 context 原键；Request 保持可比较。
  与原生图/Skill availability fence 使用同一 mode-first、activation SHARE 锁顺序，无新表或 activation。
- `imageagentapp.AnalysisSources` → 当前 Catalog/ApprovedInventory 和代码装配的原图片 reader；
  准备从实际 bytes 冻结描述，模型前核原 Tool 输出/审计与内容，handoff 重核当前权限/准确事实。
- 原 Eino runtime 的本用途 `SingleCallReadOnly` → 固定 Registry/Gateway 预读 → 一次 Model → 固定 validator；
  不追加模型选 Tool、repair 或 resume，不改变其他用途的原图流程。
- `einomodel.ImageAnalysisModel` → 原 Governed Executor/SDK/Invocation/Resource；当前企业多模态 profile 必须与冻结值一致，
  最后 handoff 重核身份/模式/profile/source；已知 no-send、已结算无效输出和 UNKNOWN 分开。
- `imageagentapp.AnalysisResultValidator` → 原 Snapshot、Run、native Invocation 与只读 typed 结果；
  核准确原始输出 hash 和用量，既不下载新图也不证明视觉判断真实。

上述为同一 Delivery Batch 的组件实现，不是整个用户路径的最终工程复核、可用入口或产品验收。
完整路径形成后复用原唯一 Reviewer 集中最终检查，不逐片重复评审。

### 9.2 原页面和生产装配

- 原 current-application 装配 Catalog 精确版本、ApprovedInventory、原 HTTP 图片读取和 source-media
  不可变对象读取；生成素材/POD 来源沿原 material/artwork owner，不以公共 URL 回退，不要求启动生图 worker。
  source-media 继续从原准入 composition 提供只读字节端口，不新设 producer 或改架构守卫。
- 企业部署准入描述中的有限 `imageAnalysisPolicies` 只限定该用途；当前模型和收费价目仍由
  原 SharedTextReader 的 **multimodal** 类别解析。不会生成另一份 current、账本、价格或逐页模型选择。
- source context 使用原 `acquisition / collection / own_product` 身份。原图片页增加选单图、准备、
  图片/模型/费用上界显式确认和只读结果，图片生成/保存/批准仍走原入口、原权限。
- `images/analysis` 下仅有来源读取、准备、原键确认、原键结果和同来源历史。
  准备从实际字节冻结身份；确认绑定原 preparation/configuration/quote/selection/content/model/最大微点。
  BFF 拒绝客户端 URL、字节、模型或价格；未确认不发送，UNKNOWN 保留原键且不自动重试。
  确认提交后结果读回失败返回 OUTCOME_UNKNOWN；页面仅按严格 code/status 接受初次准备的确定拒绝，
  不把确认后的5xx/冲突/畸形ACK或被拒绝的历史读取作为“未发送”证据，不解除原键未知门禁。
- 历史从原配置快照、Run 和 native Invocation 读取；不重报当前价、不重新下载图、不 claim/resume。
  只在实际成功、固定 validator 有效且原输出/用量绑定一致时展示只读建议。
- 原 Asset Tool 的手写 generated-only Schema 漏掉当前人工批准来源和不可变证明字段；
  复用当前 ApprovedAsset 类型生成严格两种来源形状，并保留 Domain 的原 provenance 校验。
  不伪造 Run/生成身份、不修改批准事实、不放宽 Registry 保留身份字段/大小/权限限制。
- 离线 HTTP 组合覆盖来源及人工批准来源、两步真实预读、单次合成视觉调用、原账本实际微点、
  重复确认不再发送，以及停用/当前模型移除后的原结果与历史读取。UI/BFF 覆盖显式确认、
  上下文/原键绑定、精确点数字符串和响应丢失后只读核实。合成图片不证明视觉质量或真实准备状态。
