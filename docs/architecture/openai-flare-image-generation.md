# Flare 官方直连生图：最小接线与完整费用上界候选

状态：**DESIGN_ONLY / NOT_IMPLEMENTATION_READY**。Design Basis：Independent Architecture。
Refs #134 / #128 / Draft PR #1078。2026-11-03，沿当前唯一 Writer 和原工作区整理；本增量不实现、不派评审。
源码调查基线为 `4b6c03caf49e51e2718ea344968d78ca115f3f53`；既有有限基础和 C2-2 的工程复核不包含本候选。
本文件补齐上界的计费分项、证明义务及最小消费方案；**没有获得完整 token 上界，不能据此启用 Flare**。
2026-11-04 用户认可先补保守上界方案和“完整输入 / 输出上界”两项技术咨询问题；
本次仅更新设计，保留原参数范围、准确型号、预算和一套计费，咨询 **NOT_SENT**，不实现、不派评审。

## 1. 用户结果、权限与范围

企业管理者在现有统一模型设置选择三类默认模型，成员在原商品生图流程生成、检查并人工批准素材。
默认渠道仍为 OpenAI 官方直连，生图准确型号仍为 `gpt-image-2.5-flare-2026-09-08`。
按厂商官网适用价和冻结汇率折算微点，沿原资源账本预留、结算；不新增每张固定点价或独立账本。
产品依据为[企业统一模型与官方价合同](enterprise-shared-model-configuration.md) U1–U6、
用户本次“补齐最小设计及完整费用上界依据，保持型号、原预算和一套计费；暂不实现、不新增评审派工”的明确授权。
UI/IA 继续遵循[当前 Figma Authority](../product/final-ui-ia-authority.md)，复用既有模型设置和生图入口，不新建页面。

范围是原 ImageAgent 的白底、来源图编辑和既有 artwork 生图路径：每个已授权 effect 最多一次同步 Images 请求，`n=1`，非流式、无 partial images。
一条计划可能有多个原 slot/effect；这不把整条计划的调用数改成 1，也不授予额外的 repair、解释或重试调用。
文本、多模态仍沿既有配置消费；不以它们代调用生图工具。三类全部合格才原子发布的门禁不变。
现有调用、原币费用、图片数量、时长、企业余额及成员月限全部保留；本文不设新的生产预算数字。

不做新上传入口、任意 URL、Responses 生图工具、Batch/流式、多输出、自动切换型号/渠道、自动降质量、真实模型试价或供应商咨询。
不迁移保留库、不重解释历史回执、不重放 UNKNOWN、不操作企业配置或账务。运行及产品验收不属于本次设计交付。

## 2. 公开证据与适用边界

以下为 2026-11-03 只读核对的官方公开资料；没有调用模型、上传图片或执行付费 API。

| 证据 | 能支持的结论 | 不能据此推出的结论 |
| --- | --- | --- |
| [准确型号](https://developers.openai.com/api/docs/models/gpt-image-2.5-flare) | 存在已批准 snapshot，接收文字/图片并输出图片；型号身份不替换。 | 每个请求的完整计费 token 最大值。 |
| [官网价目](https://developers.openai.com/api/docs/pricing#image-generation-models) | Standard 同步每百万 token：文字输入 USD 5，图片输入 USD 8，图片输出 USD 30。 | Batch 折扣、代理公开价或输出估价可用作本请求总价。 |
| [生图成本说明](https://developers.openai.com/api/docs/guides/image-generation#cost-and-latency) | 计算器是图片输出估算；需另计文字和参考图输入。直接 Images 不适用缓存优惠。 | 估算是硬上限、旧 GPT Image 输出表可用于 Flare，或 timeout 能撤销已发生费用。 |
| [Generate 参数与响应](https://developers.openai.com/api/reference/resources/images/methods/generate) | 可固定准确 model、n、size、quality 和同步模式；GPT Image 返回 base64。prompt 有字符约束。 | 字符限制已包含模型全部文字计费开销，或发现了可强制的 `max_output_tokens`/总金额参数。 |
| [Edit 参数与响应](https://developers.openai.com/api/reference/resources/images/methods/edit) | 能将已授权参考图片交给编辑请求；需要按准确协议编码。 | API 支持的最大图数等于本产品授权图数，或其他型号的 fidelity 计费公式适用 Flare。 |
| [视觉计量适用范围](https://developers.openai.com/api/docs/guides/images-vision#calculating-costs) | 理解模型的 patch/tile 规则与 GPT Image 生图/编辑计量分开。 | Mini 的图片上界、已退休 GPT Image 1 的输入规则可用于 Flare。 |

生成/编辑 API 仍将 usage 文字说明限定为旧型号，同时 Flare 示例又带 usage，存在文档不一致。
本候选要求可信完整 usage 才结算；示例不是“准确 Flare 在所有成功/失败响应都保证完整 usage”的证据。
上述请求引用中未找到适用于准确 snapshot 的完整输入/输出 token 硬限；这是本次调查范围内的缺证据结论，不宣称厂商永远无法提供。
官方公开费率已确认不等于平台当前已发布合格价目、有效 FX、凭据或企业选择；现场准备继续 UNKNOWN。

## 3. 完整费用上界：公式完整，参数证明仍缺

设 `T` 为实际文字输入 token，`I` 为实际参考图输入 token，`O` 为实际图片输出 token。
本候选仅准入直接 Images 的 Standard 同步、image-only 输出；缓存分项为零，不计不存在的文字输出。
只有协议依据与实际 usage 均符合这一分项集合时才可消费；新计费分项不能被静默忽略。

```text
原币总费用 USD = (5*T + 8*I + 30*O) / 1,000,000
实际 A（微点） = floor(原币总费用 × 冻结 CNY/USD × 100,000,000)
预留 R（微点） = ceil((5*BT + 8*BI + 30*BO) / 1,000,000
                       × 同一冻结 CNY/USD × 100,000,000)
```

`BT/BI/BO` 必须先有独立于期望预算、适用于实际请求的保守上界证明，不能从期望预算反解出来，再称为最大用量。
不要求三份独立的精确 token 公式：适用的完整输入总上界与输出上界，也可以经过保守推导满足本合同，见 §3.1。
以上只是现有 `OfficialModelTariff` 算费规则的展开；正式计算仍复用该整数/大整数实现，不增加另一套公式或中间四舍五入。
1 点 = 1,000,000 微点 = 人民币 0.01 元；只在全部分项加总后预留向上、实际向下取整。
原币费用预算使用同一上界对应的原币金额；它不是附加给用户的第二份账单。

| 费用或约束项 | 所需完整依据 | 当前结论及处理 |
| --- | --- | --- |
| `BT` 文字输入 | 最终 prompt（含模板、受控上下文）及额外开销的分项边界，或 §3.1 完整输入总上界的推导；采用编码算法时明确准确 tokenizer/适用范围。 | UNKNOWN。32,000 字符限制、仓库 prompt bytes 限制都不是已证明的完整 token 上界。 |
| `BI` 参考图输入 | 准确 Flare 的完整参考图计量分项边界，或 §3.1 完整输入总上界的推导，覆盖每张图及额外开销。 | UNKNOWN。图数/字节/像素限制能约束输入资源，不能直接当计费公式。纯文字路径显式无图时才有 `BI=0`，仍需其他两项。 |
| `BO` 图片输出 | 准确 snapshot 在冻结 size、quality、n=1 下的保守最大 token 值或可强制硬限，含全部计费输出。 | UNKNOWN。计算器估值、旧表、样例最大观察值均不足以证明。 |
| partial/主控额外费用 | 可执行 wire 不允许 stream/partial，不调用 Responses 主控或额外解释模型。 | 可由未来 serializer/离线 wire 回归强制排除；本次未实现，不能写成已运行通过。 |
| 实际分项/缓存 | 精确完整字段、加总一致、image-only 输出；直连无缓存优惠。 | 原 raw mapper 可复用；provider 返回缺项或不适用时 UNKNOWN，不猜零。 |
| 官方 tariff / FX | 准确型号、服务/地区/参数域、原币价目版本、来源/有效期和已发布 FX 版本。 | 合同已有；现场实际发布/有效性 UNKNOWN。本次不发布或更新真实配置。 |
| 原 run/企业预算 | 每 effect 的 R/原币上界及全部原额度、调用、数量、时长限制同时满足；并发预留由原 owner 串行化。 | 沿原合同，不提高 ceiling、不以单张估价取代全计划预算。 |

对原 `auto` quality 或 size，证明必须覆盖该值允许的全部模型选择；做不到就保持 NOT_READY。
当前 source-edit 固定 1024×1024、非 VIP quality 为 auto；其他原操作按自身冻结参数核对。
不得为了便于算上界静默改成 medium、缩小原尺寸、减少原必要来源图或提高费用上限。
公开 API 允许更多尺寸/质量不构成扩展当前产品参数域的授权。

资格证据应在原 profile/报价的 `BoundEvidenceVersion` 关联有版本的证明，绑定准确 model、endpoint/协议、操作、参数域和三项上界。
实际请求须处于该域内；所有可执行操作都有完整证据才可开放，不留未核算的 sibling path。
新证据可以是公开的准确型号硬限或能够保守推导的完整规范；不以先真实调用、token-count 外发图片、专项验收平台或厂商书面保证作为新增前置。
只有估算或样本时，上界状态仍 UNKNOWN。provider 账户限额、TPM/IPM、deadline、响应 bytes 限制均不能证明单次已发生费用的最大值。

### 3.1 保守上界的最小可行方案

固定原请求范围 → 取得适用的可追溯保守界 → 映射原 `OfficialModelBilling.UsageBound` →
原公式计算 R 并核原预算 → 未来在原 effect 内执行一次 → 完整可信 usage 计算实际 A、原账本结算。
这里的映射是现有上界合同内的数学推导，不新增余额、price current、扣款 owner 或收费公式。

例如，外部适用依据给出完整输入上界 `U` 与图片输出上界 `V`，且能证明其覆盖准确模型、直接协议与原参数域：

```text
前提：T >= 0，I >= 0，T + I <= U，0 <= O <= V
推导：T <= U，I <= U，O <= V
原分项上界：BT = U，BI = U，BO = V
```

在价格非负、FX 为正的现有合同内，原公式对每个分项单调；因此上述分项界可保守覆盖实际金额。
它可能比使用精确文图分区的上界更宽，但不要求改变原分项 Billing 类型或其逐项超界校验。
没有参考图的原 generation 操作，只有在实际 wire 证明无图且适用依据覆盖该情形时，才可取 `BI=0`。
若有更紧的合格分项边界，也可以沿原合同采用，不把精确性本身升级成新 Must。

较宽的界仅用于预留，不作为实扣用量，不把输入同时按两种价率重复计费。
实际 A 始终使用原 raw mapper 的真实文字、图片输入与图片输出分项，按同一冻结官网 tariff/FX 算一次。
缺完整 usage 保留 UNKNOWN 与原预留；不要求厂商承诺任何故障都返回 usage，也不把缺失当显式 0。

**原预算门禁不变**：已发生 + 其他未决预留 + 本次保守费用不能超过原费用预算；
原币成本使用原预算量纲，微点另核原余额/成员月限及原单次 ceiling，调用、数量和时长也分别核对。
`U/V`、分项界或任何金额若不可表示、缺完整出处、不适用或放不进原预算，就拒绝发起；不放宽预算或截断费用。
没有足够空间时不能把保守预留偷偷改成估算，不以“差额平台承担”替代原预算合同。
本候选不把输入总 token 仅填成图片输入、再把文字上界设为 0 来绕过原逐项核验。

| 工作项 | 能离线证明的部分 | 确实依赖外部的部分 |
| --- | --- | --- |
| 原请求域与 wire | 最终 prompt/模板、来源版本/顺序、bytes/尺寸/图数、size/quality/n 与无重试/partial 的实际编码一致；本次尚未实现。 | 该请求域对应的厂商完整计费边界。 |
| 上界映射 | 在真实且适用的 `U/V` 前提下，推导 BT/BI/BO、字段可表示性、同公式单调性及取整。 | `U/V` 的数值或计算规范、全部开销覆盖和适用版本；本地不能补造这些前提。 |
| 预算与一次结算 | 原币/微点量纲、原额度拒绝、原键并发、R/A、缺项 UNKNOWN；复用现有基础，Flare 增量仍待后续授权。 | 无需新增厂商“所有响应必回 usage”的保证；实际只消费可验证的完整返回。 |

所以当前不必等待三份精确分项公式，但仍必须取得足以推出安全分项界的完整输入/输出依据。
合成测试只能验证这些假设成立时的本地消费者，不能证明厂商实际遵守 `U/V`。
保留原 `auto`、尺寸及全部原操作范围；若外部依据只覆盖更窄范围，先报告不适用，不自动缩小产品范围。
本次不修改代码或新增人工测试；数学说明不是实际 provider 验收或 IMPLEMENTATION_READY。

### 3.2 两项技术咨询草案（NOT_SENT）

共同上下文：准确型号 `gpt-image-2.5-flare-2026-09-08`，OpenAI 官方直接 Images API，Standard 同步，
原 `/v1/images/generations` 和 `/v1/images/edits` 操作，每请求 `n=1`、非流式、无 partial images，
原提示词/参考图限制、尺寸和质量域保持，包含原 `auto`；不通过 Responses 主控调用生图。
本草案仅描述技术参数，不附真实图片、业务资料、凭据或账户数据；未来发送须另获明确授权。

**问题一：完整输入计费上界。**
针对上述准确型号及直接 generation/edit 协议，能否提供可引用、适用于该参数域的完整计费输入 token 保守最大值或推导规范？
可以给文字/图片分项，也可以给总上界 `U`；须覆盖最终 prompt（含模板/上下文）、全部参考图，及厂商预处理/缩放/保真度或其他计费输入开销。
请说明如何从请求可见参数确定该界、适用的版本/限制及超出参数域的处理，并区分它与平均值、估算或 Responses 输入计数；
若不存在这样的公开边界，请明确哪些输入计量部分目前不能据现有规范证明，而非建议先真实调用取样。

**问题二：完整图片输出计费上界。**
针对同一准确型号和协议，原尺寸/质量域、`n=1`、无流式/partial 下，能否提供覆盖全部计费图片输出的保守最大 token 数 `V` 或可推导的硬限？
保留 `auto` 时须覆盖它允许的全部结果，而非要求客户端改为固定 medium；请说明边界对原 generation/edit 是否一致，以及适用版本和参数条件。
请区分该界与官方计算器的输出估算；若存在实际可强制的单请求完整 token/金额限额，也请指出直接 Images 的准确参数与覆盖范围，
不以项目月度消费限额、TPM/IPM、timeout 或响应 bytes 代替单请求边界；若不能给出覆盖原范围的依据，请明确尚缺部分。

两项草案尚未发送、未取得答复；完整输入/输出上界仍 UNKNOWN。仅整理问题不构成型号准入、实现授权或评审派工。

## 4. 最小接线与事实 owner

| 层次 | 复用的当前 owner/接口 | 必要局部补齐（均未实现） |
| --- | --- | --- |
| 企业选择及准入 | `aicapability/routeconfig` 的原 Repository、`SharedCategoryReader`、同库 current/CAS/回执 | 仅为证据完整的准确 Flare 绑定正式 image profile；当前只接受 Bailian image-count 的分支不能改为“支持 OpenAI 即合格”。三类原子准入不变。 |
| 统一算费 | `aicapability.OfficialModelTariff` / `OfficialModelBilling`；`imageagent.ImageBilling` 为同类型别名 | 传完整 `UsageBound` 和证据版本；预留 R 和实际 A 都消费同一冻结 tariff/FX，不设 PointsPerImage。 |
| 原报价、intent 与发起 | `imageagent` 原 generation metadata、quote/intent、run budget 和 slot effect | R 必须与原 Billing 一致；校验已选企业配置、协议、credential/config 摘要及全部预算。保留结构可比较和值语义，不新增 current/permit/账本。 |
| worker 装配 | `app/worker/imageagent.generationProviderFactory` | `resolve` 和 `prepare` 显式增加准确 OpenAI 官方直连分支；当前 Bailian/GRSAI 分支不能兜底接收 Flare。 |
| 请求与来源 | 当前 `ai.RouteBoundImageEditor`、`integration/openai.SourceImageAdapter` 和原 artwork executor | 沿原认证配置/HTTP 能力补有界直连 codec 和 raw observer；准确来源读取、排序/摘要和原使用授权不变。原 source-edit 的型号白名单和 quality 消费也需显式接线。 |
| 实际 usage/结果证据 | `einomodel.ObserveOfficialOpenAIImageUsage`、原 `GenerationSuccess` / slot fact | 原始有界响应在字段存在性仍可区分时映射；使用原观察回调先记录已发生用量、结果摘要及恢复材料，不只返回图片或整数 totals。 |
| 唯一扣点/素材发布 | 原 `orgresource` 预留/结算事务，原 staging/publication/人工批准 | 按 R/A 原键只结算一次；素材保存在原 owner，失败走原恢复入口，不引入另一套 invocation 或付费重试 owner。 |

完整路径为：统一企业 image 选择 → 原 profile/报价与授权 → 原 run/slot effect claim →
准确配置直连一次 → raw usage/结果观察 → 原账本和 staging/publication → 原页面检查、批准和消费。
准备/预览只读，不调用 provider；执行和批准不因选定模型自动发生。配置页面不接受用户自填价格，业务页不另选模型。

原 `SourceImageAdapter.QuoteUsage` 有 scalar `ImageCostMicrosPerOutput` / `CostUpperBoundKnown`，默认值不能当作 Flare 的零费用证明。
最小处理是在原报价路径从同一官方 Billing 的保守原币金额填原费用预算投影，并与外层 metadata/R 核对；
这个预算投影不是用户点价，也不是第二收费来源。原 image-count 或 scalar 假设不得套给 Flare 的实际 usage。

### 4.1 复用 SDK 的实际限制

仓库已锁定 `github.com/meguminnnnnnnnn/go-openai v0.1.2`，不另建通用 provider 框架或全局升级依赖。
只读核对其 `image.go`：usage 使用非指针整数，无法单凭 DTO 区分缺失和显式 0；
edit 方法的实际表单写入也不能仅由 request struct 声明推定，现有实现没有完整传出 model/quality 和多图等本路径所需字段。
因此不能把该旧 edit 方法直接接入后宣称准确型号和计量已满足。
优先复用当前认证配置、标准 HTTP/multipart/JSON 和 route-bound port，在现有 OpenAI integration 内局部序列化、截取有界原始响应；
成熟 SDK 若能提供完全匹配的调用和 raw response，可复用，但不能依赖其整数 usage DTO 或隐式 retry。
只补这个协议的差异，不复制完整 SDK、不包装已退休旧 service、不建立第二个 client manager。

### 4.2 请求冻结与最后准入

原 metadata/quote/intent 必须足以绑定企业/成员、原操作和 slot、准确模型及官方 endpoint、凭据/配置版本、
原商品/素材准确版本与有序来源摘要、最终 prompt/模板版本、size/quality/n/输出格式、上界证据和 tariff/FX 版本。
缺必要绑定先补原合同映射，不建立另一个事实 owner；这些内容变化不得沿原键获得新执行。

- 单次原 effect 只发送 `n=1` 的同步请求；不发 stream/partial，不发旧 `response_format=url`，不使用 Responses image tool。
- 已有来源编辑走 `/v1/images/edits`，既有无参考图 artwork 走 `/v1/images/generations`；按准确 API 编码本地授权 bytes，不新建上传/file-id 生命周期或发送任意来源 URL。
- 返回 base64 经原大小/格式/尺寸/内容检查进入原 artifact owner；不为 Flare 自动启用现有通用 URL 结果下载分支。
- 冻结每条原操作实际 size/quality 等字段；API 默认值、遗漏的 SDK 字段或被忽略的 quality 不能悄悄改变费用域。
- 发出前再次核对当前企业上下文、原资料/图片生成权限、准入和 credential/config/profile 摘要；撤权、变更、缺证据或预算不足就不发。
- 上界覆盖的是最终完整输入。仓库共同资源限额与准确协议限制取交集；不把 Bailian 特有的图数/字节/提示词上限直接宣布为 Flare 计量合同。
- 禁用 SDK/HTTP 自动重试和可重放 POST 的重定向；发送以后不能因 context timeout、网络失败或 Temporal retry 再次调用。

## 5. 一次计费与失败恢复

| 发生位置 | 原 owner 的处理边界 |
| --- | --- |
| 准备、最后准入或序列化失败且能证明没有交给发送端 | 复用原 `provider_not_dispatched` 及原 claim/fence/终结规则；不伪造 provider usage 或成功。 |
| 发出后超时、中断、响应丢失，或无法证明未发出 | `provider_outcome_unknown`，保留原预留和身份；不自动释放/重发、不换 key 绕过，也不以 HTTP 状态码推定零费用。 |
| 完整可信 usage，实际 A 在冻结上界内 | 原 slot/fact、原资源事务记录真实 A；只有可证明应释放的 R−A 才沿原事务释放，不重复扣点。 |
| usage 缺字段、加总矛盾、未知计费维度或实际超界 | 原费用结果保持 UNKNOWN/阻塞，不把 A 截断为 R、不按图数代扣、不猜零；准确 route 停止新准入，按原 owner 保留证据。越界说明原上界证明不成立。 |
| usage 完整但输出不合格 | 不能因为图片不合格而把已发生用量清零；费用事实和输出校验各由原 owner 处理，不附送 repair。 |
| 已观察结果后的 staging/publication/结算 ACK 丢失 | 原键查原事实、原回执和原恢复材料；沿现有 effect recovery/finalization，不重新生图或按当前价重算。 |

自动恢复触发、允许动作和唯一责任沿[原 effect 恢复](image-agent-effect-recovery.md)、
[终结](image-agent-effect-terminalization.md)、[finalization](image-agent-effect-finalization.md)与当前 slot phase 合同。
本文不新增状态机或将未知 provider 结果变成可安全重复。旧调用保持原冻结事实和原版本资格，不跨版本继续或改归属。

## 6. 后续验证与当前停止点

本次仅检查文档差异、引用、计费分项和源码映射；未修改生产路径、未运行模型或业务测试、未派新评审。
未来获得实现授权且达到适用准入后，复用当前测试补以下离线回归，不扩建验收工具：

| 必须覆盖 | 可复用测试位置/证据 | 证据限制 |
| --- | --- | --- |
| 上界/价目/FX 缺失、参数/配置/权限变化均不发送；三类发布不半开放 | 原 shared catalog、quote/intent、generation factory 测试 | 合成通过不能证明真实 provider 的 token 硬限。 |
| 精确 model/quality/size/n、单次 POST、多图顺序、无重试重定向、无旧 response_format；同根因 generation/edit | 原 integration/source adapter、worker 的合成 HTTP transport | 必须看实际 wire，不能只断言 SDK struct。 |
| 三分项合计、完整 0 与缺失区分、非法 usage/超界、ceil R/floor A、并发预算和原键防重复结算 | 原 official usage/pricing、image Billing/ledger/slot 测试 | 固定 fixture 只验证消费者，不能创造上界或 provider 保证。 |
| 已用量但输出坏、结果观察或账本 ACK 丢失的原恢复；相邻白底/来源编辑/artwork | 原 source、slot effect recovery/finalization 测试 | 不执行真实保存/恢复，不把历史回归当当前运行通过。 |

实际阻碍为：**BT/BI/BO 的完整保守依据仍 UNKNOWN**，涉及预算 Must 与核心生图准入；当前不能正式启用该准确型号。
原接口的 direct wire、完整 raw usage、报价预算投影与 worker/source 消费缺口已定位，可在依据齐全、适用架构准入及实现授权后在同一 PR 最小补齐。
本设计的独立架构复核 **NOT_RUN**；既有 C2-2 工程批准不外推。不存在用户必须接受涨价、切型号或放弃预算的默认决定。
没有完整上界时，保持 Flare NOT_READY 和三类发布门禁；本次不以“设计已补文档”标为 IMPLEMENTATION_READY。

Legacy decision: EXTRACT | RETIRE
Reusable behavior: 原配置/CAS/回执、route-bound 来源读取、官网价公式、微点资源事务、单 effect、原恢复和素材检查。
Current owner: 原 aicapability/routeconfig、imageagent、OpenAI integration、orgresource 和素材 owner。
Cutover/deletion condition: 精确直连消费只接当前 owner；旧固定点价、image-count/缺项补零、旧 response_format 和 provider fallback 不进入 Flare 路径，不新增兼容或迁移。

保留现有实例、数据和会话；#47 PostgreSQL UNKNOWN，#307 OPEN，#733 咨询 NOT_SENT。
本增量运行、真实模型/平台、真实保存/恢复、浏览器和产品验收全部 NOT_RUN；缺失现场证据 UNKNOWN。
