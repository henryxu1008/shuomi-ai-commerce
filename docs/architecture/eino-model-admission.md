# Eino 多厂商模型按需接入：最小设计候选

状态：**IMPLEMENTATION_READY / FROZEN**（2026-11-04）；用户已授权同一Writer按冻结设计实施；范围内实现与离线/隔离自检已形成候选，IT1–IT6的准确HEAD、证据与自动最终评审状态维护在PR。准确设计HEAD及自动评审证据维护在PR #1093。
Design Basis：**Independent Architecture**。本候选涉及模型用途/输入合同、新 provider 协议和计量准入，不能沿用 #1078/#1089 的 Ready 代替本次准入。
Refs [#134](https://github.com/henryxu1008/shuomi-ai-commerce/issues/134) / [#128](https://github.com/henryxu1008/shuomi-ai-commerce/issues/128)。

用户已认可空业务数据设计，准确设计HEAD已通过CI和Codex自动文档评审；现授权原唯一Writer在原工作区/PR先补离线回归，再最小实现、自检和维护PR。完整候选Ready触发Codex自动评审，不新增人员或重复评审；不合并、部署、关单或执行真实操作。
原工作区继续使用；本地设计分支 codex/issue134-eino-model-admission-design 从实际 main d80d7b1e01c1e91dd47a3c61272f23b8f2b5ce68 起步，保留原分支和全部实例/数据/会话。
工程基线的 Eino v0.9.21、GenAI v1.72.0 与 Go 1.26.0 不因本候选自动升级。
本文D1/D2仅标识本候选的协议与输入设计，不覆盖凭据/Flare等文档中已批准的同名决定。
用户已明确没有历史数据；本轮按全新安装、空业务数据设计，不承接旧协议、旧输入格式或旧业务记录。
既有版本和回归记录仅作工程依据，不构成旧格式读取、迁移、双读或兼容义务。新系统在当前合同内产生的调用仍须幂等、防重复计费并可故障恢复。

## 1. 用户结果、范围和产品依据

使用者是企业的模型配置管理者，以及使用现有 Agent/图片流程的成员。
管理者在原统一入口选择适合业务的模型；标题、内容、资料、研究/选品、诊断、工作台、看图和生图继续读取企业配置，成员不逐页维护模型或费率。
相同协议、相同用途合同的更多准确型号通过可信定义加入，不逐个型号再写 SDK。

已经确认的产品决定：

- 用户要求“支持 Eino 支持的所有模型”，进一步明确“我们项目需要的就引入”。厂商/型号不以 GPT/Gemini/Qwen 三家封顶；不为凑齐框架清单新增没有消费者的功能。
- 企业统一文本、图片生成、多模态三类默认选择，全体成员共用；默认 OpenAI 与已批准 Mini/Flare 准确型号保持。
- 同一企业、同一合格渠道连接可供多个模型引用，配置 Key 仍走原受限维护入口，没有企业网页自助 BYOK。
- 沿模型厂商官网公开参考价、冻结 FX 和原微点账本估算，不额外加价；不恢复精确厂商账单/完整硬上界研究。
- 原调用/Token/费用/微点预算、权限、UNKNOWN、原键和恢复边界保留；“组件存在”不表示具体型号已配置或可执行。
- 没有历史业务数据，不设计旧格式 reader、跨版本续跑、数据迁移或切换过渡期；该决定不授权访问、清理或删除保留环境。

本次范围是现有三类消费者的共同接入合同，以及其必要的 Eino/厂商适配。
当前源码检索未见 Eino Embedding/Rerank 正式消费者；不因此新建向量库、检索或语音/视频业务。以后确有当前消费者再按同一原则引入。
生图继续 ImageAgent/原 provider 端口，不将 ChatModel 多模态输入误作生图支持。
不引入 AgenticModel/原生 server tools、MCP、联网或流式调用，仅因为上游提供了这些接口不足以改变当前业务范围。

适用依据：

- [企业统一配置与参考价](enterprise-shared-model-configuration.md)：三类选择、原 current/CAS/回执、按消费类别准入与一套计费。
- [渠道凭据分离](shared-channel-credentials.md)：Run DB 唯一 owner、企业隔离、轮换、最后门禁与当前合同内的幂等/恢复。
- [Agent 运行合同](product-agent-runtime-contract.md)、[provider-neutral 文本合同](product-agent-text-provider-neutral.md)、[单图分析](product-image-visual-analysis.md)、[图片 Agent](product-image-agent.md)。
- [模型与费率原入口](model-route-pricing-management.md)、[当前 UI/IA Authority](../product/final-ui-ia-authority.md)、[领域边界](project-boundaries.md)与[全新系统基线](../product/greenfield-no-legacy-migration.md)。

UI 沿 Figma tg48P46SSXl6TBy9lZwg63 / 账户中心 31:463 / 模型配置区域
[7658:2087](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=7658-2087)及不可用/调用记录状态，不新增页面或导航。
本轮未实查、修改 Figma 或执行浏览器/视觉验收；已有原型不代替领域权限或执行准入。

## 2. 当前根因与不变量

设计起点的根因是“组件/协议”“型号的输入能力”“业务用途”混在固定判断中。下表为实施前调查与最小修复依据；当前实现及验证证据维护在PR #1093：

| 当前源码 | 实际限制与本次最小处理 |
| --- | --- |
| [NewComponent](../../internal/integration/aicapability/einomodel/component.go) | 已复用 OpenAI/Claude 组件，Google 用当前 Interactions SDK；应复用合格组件及原受控 client，不能为每个型号复制发送器。 |
| [SharedCatalog](../../internal/integration/aicapability/routeconfig/shared_catalog.go) | 统一文字计量目前只接受 OpenAI-compatible；新用途须在原资格出口补协议映射，不只放开 provider 名称。 |
| [ValidRouteShape](../../internal/integration/aicapability/einomodel/route.go) | Google 原生只允许原 title operation。新协议/用途需声明实际消费者与输出合同，不能把原局部资格外推。 |
| [SingleImageInput/Profile](../../internal/aicapability/single_image.go) | 输入带 Mini 专属 detail/计量版本，资格固定 Mini/OpenAI；共同图片身份应与各协议参数分开。 |
| [Definition/Catalog](../../internal/integration/aicapability/routeconfig/catalog.go) | 已有准确型号、scope、连接、manufacturer/channel/价格与预算，无需新建模型表或第二目录 owner。 |

Must：当前 verified identity / effective Organization、准确 model/connection/endpoint/用途绑定、原字段/图片事实与 deterministic validator、固定预算与单次发送、单一账本和原 UNKNOWN owner。
Threat Model 保持跨企业用 Key、错模型/错价格/错图、输入/资源越界、重复发送/扣点及未知结果误处理；不新建通用 IAM/KMS/调度/恢复平台。
没有在本候选接受新的资金或安全风险，也没有为未声明的理论场景增加业务前置。

## 3. 三个层次，沿一个定义 owner

下表是**冻结职责/字段合同**；实施沿现有类型和owner最小补齐，不将分层说明变成新的持久化表、目录服务或另一份运行配置。
继续原受限配置的 Definition/Catalog；不让普通用户填写任意 endpoint、模块路径或执行参数。

| 层次 | 最小字段/职责 | 权威位置 |
| --- | --- | --- |
| 组件/协议描述 | component ID/固定版本、protocol/policy version、允许的 request/response 形状、usage observer 与可接受用途合同 | integration 中有限、代码持有的构造分支；不是可下载插件或新 Registry 服务 |
| 准确模型定义 | 原 reference ID/version、manufacturer/model ID、channel/region/service、connection ID/provider、price ID、scope/预算；新增明确用途合同引用及确有差异时的厂商型号/渠道wire型号绑定 | 原可信 Definition，按现有 scope/ref 保存，不复制 Key/价格/余额 |
| 编译后的用途 profile | 准确用途、输入/输出合同、协议参数、limits、mapper/参考策略与组件描述指纹；复用原 ModelProfile/ImageProfile 与 ProfileVersion | 原 SharedCatalog 资格结果，再进入原 quote/start snapshot/intent |

用途合同采用有限集合：governed text、single-image advice、原 image generate/edit。
型号能力表示“此准确型号有可复用且已声明的用途合同”，不是从名称含 vision/image、厂商名称或 SDK 类型推断出的授权。
category 由用途合同映射，不接收浏览器自由组合的能力标志。

现有 Definition 每个 scope/ref 仍只持有一份相应 profile。用途引用只是它的准入描述，
不再维护一份平行的当前模型能力数据库；需要的输入/输出/协议版本只在此定义中编译并进入原冻结 profile。
同一 model ref 覆盖多个已开放消费者时，manufacturer/model/channel/connection/价格身份必须一致，prompt/schema 与较小业务 limits 按 operation 保留。

ProfileVersion/quote 指纹必须覆盖：准确型号/用途、输入输出合同、协议/mapper、组件固定版本、约束及已解析价格/FX/连接修订。
组件版本由可信构造描述给出，不能由请求或模型输出选择；当前准备与实际执行的组件/协议指纹不匹配时拒绝发送，不默默改用另一种 wire。
它不是新的 current、价格版本或调用账本。

聚合渠道的wire型号可能带厂商前缀，例如Eino [OpenRouter示例](https://github.com/cloudwego/eino-ext/blob/eee37f63ff9548c5eaec48fc157863339f8eda07/components/model/openrouter/README_zh.md)中的google/...。
存在差异时，定义须显式绑定wire型号、厂商计价型号与来源版本；不能自动去前缀、猜测等价型号或按渠道价格冒充厂商价。
原执行tariff仍匹配准确wire绑定，价目来源身份/别名若需字段扩展，只在原payload与校验中版本化，不新建映射表/价格owner。
动态auto或fallback型号若不能确定准确型号/计价身份则不准入，不因SDK接受该字符串就授予执行资格。

## 4. Owner → 注入 → 消费者

| 边界 | 当前 owner / 接线 |
| --- | --- |
| 企业三类选择及平台参考价/FX | aicapability/routeconfig 原 Repository/SharedModelService/PlatformPricesService，原 CAS/原键回执 |
| 可信定义及资格 | integration/aicapability/routeconfig 的 Catalog/SharedCatalog，复用定义读取与资格出口 |
| 构造与实际发送 | integration/aicapability/einomodel 的 NewComponent/GuardedClient/Executor；官方 SDK 只序列化/解码，不获得 retry/预算/业务事实 owner |
| 图片事实/确认/结果 | 原 Catalog/SourceMedia/ApprovedAsset reader → image analysis Prepare/确认/只读结果；ImageAgent 继续原生成/批准路径 |
| 运行装配 | app/runtime/currentapplication/model_route_pricing.go、channel_credentials.go → 原 HTTP application/worker ports |
| 用量、预留与结算 | 原 Invocation / GenerationFact / orgresource；无新钱包、调用状态机或 reconciler |

```text
企业三类选择 + 可信 Definition
  -> 原 SharedCatalog 编译用途资格 / ProfileVersion
  -> 原 AgentConfig snapshot 或 Image intent + Quote
  -> 当前身份/权限/预算 + 原 durable claim
  -> Eino/生图组件 + 原最后门禁 + 单次发送
  -> 原始用量观察 -> 原结算/UNKNOWN -> 当前合同内结果/回执/调用记录
```

模型组件只接当前调用已批准的消息与受控 HTTP client，不直接接 GORM、业务/平台 SDK或控制台用户请求。
普通免费 facts/compute Tool 不因此调用付费模型；C1 和 C2-2 继续原 GovernedModel 分支。

## 5. 企业选择、用途资格与业务许可

1. 保存仍是一个完整的企业三类选择快照，经原 revision/current/CAS/原键回执提交；不建立按成员/按页面选择、第二 current 或 pending 平台。
2. 保存合法引用不等于准许调用。原 Preview/Options 说明准备状态；每个实际执行重新核自身类别与准确 operation/profile。
3. 同一类别已开放且实际读取该默认模型的消费者，必须都有对应技术 profile；不能只验证标题就对其他文本用途报告 Ready。未开放业务或 Eino 的全部可选功能不作前置；Store 等业务权限也不成为纯文本配置的额外条件。
4. 多模态用途只接受单图意见合同，生图只接受生成/编辑合同。模型支持读图不自动获得生成、改图、保存或批准能力。
5. 具备能力也不授予业务权限：原领域 authorizer、source/asset/Target版本、mode、人工采用/批准与提交 guard 保持。任何一个不满足均沿原不可用/拒绝结果，不换型、换渠道或补造事实。
6. scope/ref/用途缺失返回明确未准备好；平台若尚未安装对应 Eino组件，也不得仅因型号在上游目录出现就列作可执行。

同协议/同用途下加入准确型号，通常只是补原可信 Definition/参考价格/版本；
只有实际出现不同 wire、参数语义或计量才新增局部适配。不是允许任意字符串免验证执行，也不为每个型号机械拆任务/PR。
原定义条数（当前128条Definition）及请求/响应/时间等资源上界不自动提高；具体业务确需超过时先说明最小差异，不静默截断候选或建设全量模型目录平台。

## 6. 组件复用与依赖边界

已核上游 [components/model 固定快照](https://github.com/cloudwego/eino-ext/tree/eee37f63ff9548c5eaec48fc157863339f8eda07/components/model)
含 OpenAI、Claude、Gemini、DeepSeek、Qwen、Ark/ArkBot、Qianfan、OpenRouter、Ollama与Agentic系列。
这是可复用组件范围，不是全部型号、所有用途、账号可用性或当前系统交付结果。

| 需要的协议/用途 | 最小复用候选 | 明确限制 |
| --- | --- | --- |
| OpenAI-compatible 文本 | 当前 OpenAI v0.1.13；厂商参数确有差异时复用对应 Eino Qwen/DeepSeek/OpenRouter 等组件 | 不建立多个相同 HTTP/计费客户端；所有组件使用原 guarded client，按准确型号固定受支持参数 |
| Qwen 文本/单图 | [Qwen v0.1.9](https://github.com/cloudwego/eino-ext/tree/0ebab92e14f26088411dbc440a1ebdc904ccd8a1/components/model/qwen)，复用当前同版 ACL v0.1.17 | 静态依赖不要求升级核心；thinking 参数及具体型号图片协议需原 fixture 验证，不能从包名推断全部支持 |
| Gemini 文本/单图 | [Gemini v0.1.36](https://github.com/cloudwego/eino-ext/tree/6fe16d48b736f0de9ffa568cfe0122313c806835/components/model/gemini) + 当前 GenAI SDK v1.72.0 | 使用 GenerateContent，见 D1；版本下限匹配不是实际编译/运行 PASS |
| Claude native 文本 | 当前已引入的 Eino Claude v0.1.26 / 官方 SDK | 当前统一计量仍需原生 usage 映射；不让组件自行决定结算或启用 prompt cache/server tools |
| Flare / Qwen 生图 | 原 ImageAgent + OpenAI/Bailian 生图适配器 | 原批准的参数含 auto、参考估算、结果/批准/恢复不变；不把生图塞进普通 ChatModel |
| 其他项目确需的厂商 | 选择适用 Eino组件/协议与可复用厂商SDK，补对应有限构造分支和准入定义 | 先核 client 注入、认证形态、实际用途与基本计量；不是导入全部包或自动开启所有上游能力 |

不新增厂商认证服务：当前 API Key 连接合同可覆盖的渠道复用 #1089；
若实际所需渠道必须使用当前合同未覆盖的 AK/SK/OAuth/SDK自认证，先明确原凭据 owner 的最小差异并报告，不把多字段秘密塞进一个伪 API Key、依赖环境凭据或增加第二存储。

若项目确需某Eino组件已经支持的图片输出，可复用其协议序列化并适配到原ImageAgent的生成/effect/artifact端口；
须单独满足生图用途、计量与结果恢复合同，不同时走文字和图片两条扣点路径。本候选不因框架具备该能力就宣布新生图型号可用。
聚合组件中的备用Models列表、自动换型等可选项保持未绑定；本系统的单次handoff不被表述为已证明厂商内部处理/收费次数。

部分上游当前版本要求 Eino v0.9.22；已有静态候选不要求先全仓升级。
是否升级、升级哪些模块须由实际组件需求及适用验证决定，不依赖 latest、replace 强压版本或全量依赖整理。
固定组件已在本轮离线构造/编译验证中消费；准确依赖选择与测试结果维护在PR，不为使用ChatModel迁移运行框架到AgenticModel。

### D1 候选：Gemini 使用 GenerateContent

上游组件走 Models.GenerateContent；原 Google title 路径是 Interactions，二者的URL、model绑定位置和usage都不同。
空业务数据下直接定义准确的 GenerateContent protocol/profile：模型工厂复用 Eino Gemini，guard核路径上的准确 model及允许的消息/参数。
不复用 Interactions adapter ID/请求形状，也不放开通用 model/body 判断；不建立旧协议 reader、迁移或兼容分支。

型号协议参数沿可信profile固定：ThinkingMode显式为low、minimal或budget-zero，依据该准确型号支持的参数声明选择；不以统一budget=0冒充所有型号适配，也不向用户页面增加思考/额度选项。原maxOutputTokens覆盖thought，调用/Token/费用上限不变，最终wire核该冻结参数。
GenAI client 注入原 guarded HTTP client，明确 API Key、backend、origin/API version；不读取缺省云身份、不启用非业务所需 server tools、URI上传或缓存接口。
SDK v1.72.0 在 [RetryOptions 为 nil 时只发送一次](https://github.com/googleapis/go-genai/blob/v1.72.0/common.go)；
候选显式锁定 Attempts=1及非流式调用，最后handoff锁/禁止redirect/第二次发送仍归原 guard。
**D1已获用户认可并随准确设计HEAD完成自动文档复核；按本冻结设计实施，不涉及旧数据处置或任何实例切换。**

## 7. 图片身份与用途输入

### D2 候选：单图共同输入与协议参数分离

原 SourceMedia/ApprovedAsset仍拥有真实图片；共同输入只有 selection digest、准确内容hash、MIME、byte length、实际宽高及其原事实绑定。
图片从原合法 reader取得，来源图与批准素材身份保持不同；沿原字节/像素/时间限制、原报价及用户确认。

Mini的 detail=high/专属计量不是所有模型的共同属性：
候选将其归 Mini 的可信用途 profile；Gemini/Qwen使用各自 profile的表示和计量版本。
共同单图输入按当前合同直接定义；不将 Mini专属字段省略后冒充其他厂商输入。
原 ModelProfile/SingleImageInput/quote/start snapshot绑定准确合同与用途profile，复用现有持久化字段与原 owner，不新增数据表、旧格式 reader或迁移。

Eino Gemini已经支持 Base64Data→InlineData，Qwen/同ACL支持Base64Data→data URI；
复用序列化，不允许请求提供任意URL、File ID、新上传或跨来源替图。
最终wire必须核内联图片字节/hash及准确 model/用途；参数覆盖不能替换内容、增加图片数或调用次数。

单图分析继续主体/构图/背景/清晰度疑点的只读输出，不回填属性、判定侵权/合规或执行动作。
**D2作为空业务数据下的当前输入设计候选已获用户认可，不建立新旧格式切换或旧格式回读合同，也不再要求用户决定历史数据处理；准确设计HEAD自动复核已完成；范围内实现已形成候选，最终CI和自动工程复核仍须绑定实施HEAD。**
准确输入、用途、报价和确认仍必须一致；不把未变化的文本、Flare或其他Agent无故升级版本。

## 8. 参考计量与单一账本

平台仍在原价格 owner 维护模型厂商官网公开参考价、适用单位、有效版本、FX与必要估算参数；企业只选模型。
manufacturer 与传输channel分开，代理/聚合渠道价格不替代已确认的厂商计价依据。
预算是本系统按冻结参考配置控制的预算，不伪装成厂商实际账单硬限。
缺官网价的本地/私有模型保持未准入/UNKNOWN；没有本次自行免费或改价的授权。

| 环节 | 最小合同 |
| --- | --- |
| 预留R | 复用原用途/模型输入估算、参考费率/FX、原整数算费与向上取整；不得提高原调用、Token、费用或微点上限 |
| 基础usage观察 | 原 guard在SDK归零/消息标准化前检查 raw字段存在、非负与一致性；各协议明确计数字段及可验证派生规则，映射到原 ObservedUsage/BillableModelUsage |
| 参考文字策略 | 已批准的基础总量策略继续不要求精细缓存/计价时点；Claude等原生输入统计与Google候选/thought字段不能冒用OpenAI字段，未明确映射不得准入 |
| 参考图片/单图 | Flare原D1、生成/编辑与新单图理解的计量用途分开；不能因均含image就套同一mapper，也不新增第二收费路径 |
| 结算A | 原owner按冻结版本和可信基础用量算实际结算量，最终向下取整；成功和已观察的无效输出沿原合同结算，不把内容错误当无调用 |
| 明确零/未知 | 只在当前策略的完整零证明成立时A=0；缺失、矛盾、超原限制或发送不明仍UNKNOWN/保留R/不重发，不从SDK默认0推断 |
| 超R与恢复 | 保留各现owner的处理；Flare原ObservedOverage/同绑定发送屏障不抹去，其他协议不自行补扣、截断A、扣尽R或假定平台承担差额 |

库输出的 TokenUsage用于业务响应/兼容，不单独取得资金事实资格。
mapper仅是厂商协议到原计量类型的有限转换；不建立供应商账本、新结算worker或每家一套点数。
若实际验证发现必须修改原UNKNOWN/资金/恢复合同，停在受影响边界报告，不以新增provider为由顺带引入。

## 9. 企业隔离、冻结绑定与失败恢复

- verified Organization/actor/member及业务资源由原身份/领域授权 owner决定；模型配置或能力标签不授予管理/Store/素材/写权限。
- metadata与secret分权保持：目录、预览、报价及调用记录不读Key；实际执行才经原secret port读取同企业/准确连接修订与endpoint。
- 同一连接多模型共用秘密，模型/用途/协议/参数/预算仍分别绑定。禁止跨企业、默认环境Key、全局fallback或按厂商名借凭据。
- 当前合同内沿原CAS/回执和durable claim进行幂等；相同key不同model/profile/图片/参数冲突。重复请求先找本次原回执，不能用变化后的配置重建同一请求。
- 轮换/停用使失效准备无法通过最后门禁；已通过最后门禁的原一次在途调用可以完成，不增加撤销/重调能力。UNKNOWN与R不因轮换清除。
- SDK构造/wire前拒绝沿原no-send证明路径；仅本地异常、超时或缺响应不能伪造no-send。已发送未明由原Invocation/GenerationFact负责。
- 新系统当前合同内已观察的事实/不可变artifact与结算回执可本地恢复，不因模型/Key/价格移除而重新调用、重定价或释放该次预留。
- 不引入旧业务数据、旧JSON解析、跨协议/跨输入格式续跑或历史兼容分支；上述恢复只处理本系统当前合同内创建的请求。

所有contract→profile→quote→claim→wire均必须同一准确绑定；mapper/组件指纹漂移拒绝不匹配的准备，不能默默切到新的SDK行为。
不改变持久化事实owner、事务边界、原资源ledger或现有恢复责任。

Legacy decision：**EXTRACT | RETIRE**。
复用当前合法输入限制、原guard/权限/claim、已有SDK和原恢复；将Mini/Google局部约束放回对应准确profile，而不是整体删掉安全断言。
不增加旧模型同行凭据、旧Service wrapper、双current/双写/兼容执行或任何保留库升级。
新空系统基线适用；保留实例继续原程序/数据库/会话，本候选不授权任何切换。

## 10. 原页面如何呈现

企业仍在“我的账户 → 企业空间 → 模型与费率”及原三类选择入口操作；
默认渠道、当前三类选择、预览/回执/调用记录继续原owner。平台维护者用原可信定义/价格入口，不增加用户模型发现/任意URL或自助Key表单。

原Options/Preview/ReadCategory返回已声明型号的适用类别与准确准备状态；UI只投影，不自行按名称判断能力。
候选列表说明“可用于文本/单图理解/生图”，对未具备用途、缺连接或缺参考价格给业务语言提示，不能用技术版本或PASS数量假装模型可用。
合法保存的未准备好选择仍显示实际限制，原调用入口按自身资格拒绝；不能在后台替用户换默认模型。

沿原BFF/严格DTO/权限与页面组件补用途投影；没有新页面设计或真实配置保存。
本轮Figma、真实浏览器和产品验收NOT_RUN；如果实现确需新交互，先补原Product Authority，而不是在此追加流程。

## 11. 有界实施落点与验证义务

已获范围内实施授权；以下义务已补离线/隔离回归并形成实现候选。具体RED/GREEN、准确HEAD、必需CI及自动工程复核维护在PR，不将局部通过外推为运行或产品验收。

| 实施落点 | 最小变化 / 证据 |
| --- | --- |
| Definition/Catalog/SharedCatalog、ModelProfile/单图输入 | 用途版本与能力映射、同模型跨操作一致、未知/用途不符明确拒绝；不新建定义表 |
| NewComponent/GuardedClient、协议usage observer | 现成组件注入、准确wire/图片/参数、单次发送、原始计量存在性；复用受控transport fixture |
| currentapplication/原HTTP/worker注入 | 企业选择→准确用途profile→实际consumer；不复制SDK/Key或导入其他owner的在途实现 |
| 原页面DTO/BFF/结果与调用记录 | 准确类别、准备状态、显式确认与只读结果；当前合同内原键不重调、不伪造通过 |

必要义务合并到一个用户结果的主要PR，不为文件数或每个型号机械拆分：

1. **IT1组件复用**：候选依赖编译；受控client收到准确protocol/model/参数，仅一次发送；SDK retry/redirect/未绑定工具/环境Key不能旁路。
2. **IT2能力与用途**：一个协议下多个合格型号按定义接入；错误能力/缺profile/跨操作或同ref不同身份拒绝；未开放消费者不作前置，不误授业务权限。
3. **IT3单图与确认**：真实owner读取合成图片bytes，经同一选择/hash/用途/报价进入wire；错误来源、篡图/变图、不匹配确认与错误模型拒绝；结果只读。
4. **IT4计量与资金**：原raw字段已知/缺失/矛盾/明确零、版本漂移、无效输出与超R沿既有fixture核；只用一个资源owner，不重复预留/扣减或任意改变已产生的UNKNOWN。
5. **IT5原键/恢复/轮换**：当前合同内错误用途不复用回执，重复确认/中断恢复不重复发送或扣点；修改默认或连接后不重发；原一次在途/本地恢复和隔离仍成立。复用#1089未变化证据，仅核本次实际增量，不新增旧数据或旧格式兼容用例。
6. **IT6实际装配**：原owner/Registry/合成模型/任务临时PG的已有组合验证消费了准确新profile；不是只对独立SDK做mock成功。不新建runner、故障注入平台或验证工具。

仅使用原离线/隔离测试与正常必需CI。真实模型质量、实际配置/部署/保存恢复/平台/浏览器/产品不在本批执行范围，保留NOT_RUN/UNKNOWN。
设计阶段复用已完成的准确HEAD自动文档复核；最终稳定候选Ready触发Codex自动评审，不新增Reviewer或重复审核未变化内容。

## 12. 已确认范围与设计准入状态

| 项目 | 状态 |
| --- | --- |
| 按项目需要复用Eino、厂商不限；企业统一配置、共享连接、一套计费 | 用户已明确，不重复要求确认 |
| 全新安装、没有历史业务数据；不承接旧数据/格式/协议 | 用户已明确；无须再次确认历史处置，不授权实例清理 |
| D1：Gemini采用GenerateContent及准确guard/profile | **已认可/完成设计自动复核，实现候选已形成** |
| D2：直接定义共同单图身份，型号协议参数归各用途profile | **已认可/完成设计自动复核，实现候选已形成**；无新旧格式过渡合同 |
| 其他新认证形式或改变资金/状态/恢复合同 | 当前不引入；具体需要出现时先报告，不从本候选自动取得授权 |
| 型号/参数/计量事实 | 由可信定义与对应原用途约束确定；本轮不发布真实定义、费率/FX或更改默认型号 |
| 设计复核 / 准入 | 准确设计HEAD自动Code Review completed/0 finding，IMPLEMENTATION_READY / FROZEN；非Security Review或产品验收 |
| 设计提交、推送、Draft PR与准确HEAD的CI | 本轮已授权；准确HEAD、CI与滚动证据维护在PR，不以CI代替架构准入 |
| 实现、依赖、编译/业务回归 | 现获范围内授权，仅任务离线/隔离自检；实际结果与新准确HEAD在PR维护 |
| 保留环境与产品 | 未操作；运行、真实保存/恢复、浏览器、质量及产品验收NOT_RUN，现场缺证据UNKNOWN |

本轮继续原PR交付范围内实现、必要自检、新准确HEAD CI和自动评审证据；不将工程通过写成真实产品通过。#134/#128保持OPEN，#47 PostgreSQL UNKNOWN、#307 OPEN、#733咨询NOT_SENT。
不迁移保留库、不合并、不部署、不真实启停/配置保存/采集/保存/恢复、不调用真实模型/业务平台、不发邮件短信。
