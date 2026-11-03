# 商品图片单图视觉分析：C2-2 最小设计候选

状态：**DESIGN_CANDIDATE / NOT_IMPLEMENTATION_READY**。Design Basis：Independent Architecture。
Refs #134 / #128；当前唯一负责人、原独立工作区继续，无新增 Writer / Reviewer / 派工。
用户已明确[认可 D1–D3 并授权 design-only Draft PR](https://github.com/henryxu1008/shuomi-ai-commerce/issues/134#issuecomment-6592979123)：
同一负责人在原工作区提交/推送候选与直接引用，创建 Draft PR 并核准确 HEAD CI。
本文件不是生产可消费合同；具体模型/图像计量/费率依据仍 UNKNOWN，独立架构复核 NOT_RUN。
产品范围认可不表示 IMPLEMENTATION_READY，没有实现、评审派工或真实调用授权。

2026-11-03 用户进一步确定[企业统一模型配置与官方价折算](enterprise-shared-model-configuration.md)：
先有默认渠道，企业在一处统一配置文本/图片生成/多模态默认模型，所有成员共用，
各业务读取已配置模型。GPT/Gemini/Qwen 均保留支持目标，按实际模型官方价折算 AI 点数。
这替代“先仅支持一家”及“成员每次分析另选模型”的建议；本用途读取企业多模态默认配置。
公开型号/价目信息已有本会话只读调查；Mini 文本/多模态、Flare 生图准确快照及汇率/微点规则
已获后续产品设计认可，实际渠道绑定、usage/预算和数量事务合同仍未架构冻结。
共享配置候选同为 NOT_IMPLEMENTATION_READY；D1–D3 的单图、一次调用及只读边界不变。

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

调查基线：main `c6f53e1a928862e556bbf0a1883c2af03dbedfb7`。
旧 SDK/历史设计的视觉能力描述不作为当前正式链路已可用的证据。
当前 C2-1 / C3-2 的 Ready、CI、旧版本决定不自动覆盖本新增带图模型与计量合同。

### 1.1 已确认的产品决定

| 决定 | 推荐候选 | 当前状态 |
| --- | --- | --- |
| D1：分析目的 | 主体可见性、构图/裁切、背景和明显清晰度疑点，输出只读意见；不做精确 OCR、商品材质/规格回填、侵权判定或平台合规保证。 | 已产品认可。 |
| D2：图片范围 | 一次一张，选自当前商品准确版本的已有来源图或已批准素材，身份分开；不接受任意 URL、新上传或多图比较。 | 已产品认可；不把来源图伪装成批准资产，也不要求先批准才能分析。 |
| D3：调用/费用 | 用户显式确认获准模型接收本次图片及费用上界；一次视觉调用直接产生说明和建议，不追加第二次解释/自动重试，不提高原总调用/费用上限。 | 已产品认可；不授权本轮真实调用，模型、费率、图像计量依据缺失仍不可执行。 |
| 企业共享配置 | 本次分析使用企业在统一入口选择的多模态默认模型；成员确认实际模型和费用，不另存个人/页面模型配置。 | 已产品决定；共享选择、官方价折算及精度合同仍是待复核候选。 |

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

当前模型/价格配置按[企业共享候选](enterprise-shared-model-configuration.md)读取多模态类别，
不能直接按原标题 operation 或生图 points/image 推定。模型/渠道变化须在原新准备及最后发送核验；
旧请求不静默换模，已提交结果仍按原事实读取。该依赖同时覆盖官方价、汇率及细点数单位，
不在本用途创建独立可编辑点数费率或第二配置源。

当前 `QuoteText` 只绑定文本。新窄图片输入/Quote 必须将实际图像摘要/媒体参数、模型/配置、
prompt/output schema、完整输入及计量策略纳入同一原 input hash；不能将 base64 文本长度当视觉 token 定价。
具体模型/协议尚未选择，本候选不因 SDK 支持图片 part 就声称该路由已准入。

每个开放 route 必须明确：受控 endpoint/凭据引用、图像输入支持及大小/像素限制、
最坏图片+文本输入用量上界、真实 usage 完整性、价格/产品点数费率版本和返回结果关联。
输入/输出 token、图片单位或其他收费项如何合并，必须由该 route 的实际依据确定。
缺上界、usage 映射、模型/费率或配置时保持 DEPENDENCY_MISSING / NOT_READY；缺用量不按零结算。
本轮没有读取保留实例配置/凭据、查模型目录、核实时价格或发出任何模型/平台调用；现场准备 UNKNOWN。

候选预算：一次显式请求最多一次视觉 handoff，直接返回说明/建议；不再调用模型改写解释、
不做模型 repair/自动重试/切换供应商。固定 validator 失败保留该次实际用量和失败，不追加收费修复。
不提高原总调用/费用/预留上限；报价超过现额度/预算/用户确认上界即拒绝。
实际 quote/claim/预留/guarded send/usage/结算、原键与 UNKNOWN 继续原 ledger/资源 owner，
没有第二个 AI 许可、计费账本或可重放的 provider owner。

## 7. 只读结果与消费

候选 `image-analysis-result-v1`：准确 selection/content hash、实际 AI invocation 与模型/配置版本、
已分析覆盖/未知、最多八条 observations 和八条非执行性 suggestions（每项理由有界）。
每条观察只引用本次图片；主体可见性/构图/背景/清晰度采用 possible_issue / no_obvious_issue / unknown，
不使用“合格/侵权成立/商品属性已证实”等权限或事实状态，不提供虚构总体质量分。
字数/数组/模型输出受原 64KiB ModelOutput、profile 更小上界及 Run 2MiB 限制；strict closed schema。

模型视觉判断是带不确定性的建议，不是 Catalog/source/批准事实。
固定 validator 只能证明结构、准确绑定、执行/用量/结果引用和字段范围，不能证明视觉判断真实正确。
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

仅在产品决定、provider/计量依据及适用独立架构复核完成、达到 IMPLEMENTATION_READY，
并另获实现授权后，扩原测试：图片读取/大小/漂移；Quote/input hash/usage 与唯一 send；
原 Prepare/Run 的跨图片 key 冲突、并发/ACK 未知与历史；固定结果边界；页面不触发生成/保存/批准。
只用合成图片/transport 与任务自有临时 PostgreSQL，沿既有 fixtures，不新建 runner/验收平台或重做全仓审核。
真实视觉准确性、误判/不确定性及用户效果另行验收；技术测试不能替代。

当前缺失：具体 route/图像计量/费率依据；准确执行定义/输入结果版本和固定预读接线的独立复核。
这些阻止正式实施/真实开放，不能用旧架构Ready/SDK方法/CI替代，也不是要求先建设通用平台。
本阶段只交付设计候选及必要直接引用，提交/推送、Draft PR 和准确 HEAD CI 证据维护在 PR；
不为滚动证据制造仓库提交，没有实现、评审派工、人工业务测试或真实操作。
缺证据 UNKNOWN；运行、真实保存/恢复、浏览器、视觉质量及产品验收 NOT_RUN。
#47 PostgreSQL UNKNOWN、#307 OPEN、#733 咨询 NOT_SENT；保留所有实例、数据和会话，不合并/部署/关单。
