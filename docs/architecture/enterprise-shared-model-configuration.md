# 企业统一模型配置与官方价点数折算

状态：**IMPLEMENTATION_READY / FROZEN（限定安全基础）**。Design Basis：Independent Architecture。
Flare 准备/执行、三类默认完整可用及整批用户目标仍未满足，不在本次 Ready 声明内。
Refs #134 / #128 / Draft PR #1078；当前唯一 Writer、原工作区和分支继续，仅复用原唯一 Reviewer。
本阶段沿原 Draft PR 连续开发。原唯一 Reviewer 已完成一次有界架构检查及本文件发布语义消歧增量，
确认官网分项/冻结 FX/微点、原资源事务、统一配置/CAS/reader-validator 的安全基础可实施。
准入证据与实际实现/最终 HEAD/CI 维护在 PR；限定安全基础已完成原独立最终工程检查，真实使用验收未执行。

2026-11-03 用户后续明确“不用咨询，继续开发”，授权原 Writer 在原工作区和 PR 连续实现、
必要离线/隔离回归、自检及 PR 维护；复用原唯一 Reviewer 完成适用架构准入与最终检查。
咨询不发送，也不以取得供应商答复作为开发前置。正式业务路径仍须先取得范围明确的
IMPLEMENTATION_READY；上述限定基础已获准入，未具备执行依据的型号继续不可执行，不能据开发授权降低费用 Must。

2026-11-03 用户进一步明确：**不要做成两套计费**。本设计的计费路径只有一条：
平台发布的厂商官网适用价/FX → 原调用冻结快照及可信用量 → 精确微点 → 原资源预留/结算。
`OfficialModelTariff` 只做该路径内的算费，不拥有余额或扣款事务；原资源账本仍是唯一扣点 owner。
本次文本接线以冻结官网价替换并删除 `ModelPointTariff`，企业不再提交文本输入/输出点数费率，
不保留旧费率 fallback，也不把当前价重算历史或恢复中的原调用。
现有渠道成本金额仅服务原费用预算，不产生第二份用户账单，也不叠加到官网价扣点。
图片旧固定点价、授予/月限/展示及全链微点单位门禁已在同 PR 完成工程切换与独立检查。
这不表示保留实例已切换或完整三类模型可用；Flare 执行仍未完成。
C2-2 消费者在 `4b6c03ca` 完成限定工程复核（证据见 PR），运行和产品验收仍未执行。
Flare 本次只补[最小直连设计及完整费用上界证明义务](openai-flare-image-generation.md)，
三项 token 上界仍 UNKNOWN，本增量未实现、未派架构复核，不包含在已有 Ready/工程批准内。
本 PR 继续 Draft，混量纲写入和旧价回退都不属于允许的交付结果。

## 1. 用户结果与已确认决定

使用者为企业的模型配置管理者和使用 Agent 的企业成员。
目标是一处配置默认渠道及文本、图片生成、多模态三类默认模型，全体成员在业务中共用；
标题、内容、规划、生图、看图分析等入口读取有效配置，按实际模型的官方价格折算 AI 点数。
成员不需要在每个业务页面重复选择模型或维护费率。

产品依据是用户在本会话 2026-11-03 的明确决定：

| 决定 | 已确认内容 | 尚未由此批准的事项 |
| --- | --- | --- |
| U1 默认渠道 | 用户进一步明确“默认 openAI”，默认渠道为 OpenAI 官方直连；企业在统一入口配置三类支持的模型。 | 三类默认型号已在 U6 认可；适用地域/账户方案、凭据归属和实际可用配置尚未冻结；本决定不授权真实配置。 |
| U2 三类模型 | 企业统一配置文本、图片生成、多模态默认模型；所有成员共用。 | 不建立个人默认模型或每业务页面独立的当前模型选择。 |
| U3 供应商范围 | GPT、Gemini、Qwen 三家候选均纳入支持目标，不能擅自缩为只支持一家。 | 三家所有型号、所有协议或所有用途没有因此自动准入。 |
| U4 官方价计费 | 用户进一步明确选择模型厂商官网公开价，不额外加价；按实际使用模型适用的官网价折算 AI 点数，不独立任意填写加价点数费率。 | U6 已认可汇率规则/点数精度；准确适用价目、汇率发布和执行绑定仍需核准；代理实际成本不代替用户计价依据。 |
| U5 阶段 | 用户后续授权原负责人继续开发，不咨询；沿原工作区和 PR，复用原唯一 Reviewer 做必要准入/最终检查，不新增人员。 | 实现须满足适用架构准入；没有合并、部署、关单或真实操作授权。 |
| U6 默认型号/换算/权限 | 用户认可文本及多模态 Mini 准确快照、生图 Flare 准确快照；交易中心中间价由平台按工作日核验发布并报价冻结；1点=百万微点、预留向上/实际最终向下取整；平台维护渠道/官网价/汇率，企业已有 Agent 配置权限者选择三类默认模型。详见 §4.3–4.5。 | 初始设计认可不自动授权实施；后续实施/原 Reviewer 准入沿 U5，仍无真实操作授权。 |

本决定替代“#1078 首条只选一个模型”建议，以及“成员每次分析自行选择模型”的候选。
企业模型选择不改变业务权限、Agent 提示词/输出合同、调用次数或费用硬上限。
同一型号可具有多个能力标签，但看图输入支持不等于生图/改图支持。

依据：[模型与费率管理 V1](model-route-pricing-management.md)、
[文本路由](product-agent-text-provider-neutral.md)、[Google 原生路由](product-agent-google-interactions.md)、
[图片 Agent](product-image-agent.md)、[运行合同](product-agent-runtime-contract.md)、
[点数购买](commercial-retail-pricing.md)、[成员资源](member-resource-allocation.md)、
[领域 owner](project-boundaries.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)。
页面沿[当前 UI/IA Authority](../product/final-ui-ia-authority.md)和原“模型与费率”入口复用；
企业选择和平台维护复用原页面，权限/BFF/组件接线已完成本批工程检查；Figma 实查/更新及产品/视觉验收 NOT_RUN。

## 2. 原调查与当前最小复用

最初只读调查基线：main `c6f53e1a928862e556bbf0a1883c2af03dbedfb7`。下表已按本批实际实现校准，
原调查不是当前源码状态。已实现基础的准确 HEAD/CI/独立工程回执以 PR 的有限安全基础记录为准；
C2-2 的限定准入及实现以 [单图分析设计](product-image-visual-analysis.md) 和 PR 中 `4b6c03ca` 工程证据单独记录；不外推到 Flare。

| 当前能力 | 复用 | 本批状态及剩余接缝 |
| --- | --- | --- |
| `aicapability/routeconfig` 与配置 revision/current | 合格 binding、去密投影、确定性校验、CAS 发布及原键回执 | 已改为企业三类共享选择，按实际消费者校验；缺任一类别资格仍不发布 current。 |
| `OrganizationRouteResolver` / `ModelProfile` / 原 AgentConfig snapshot | 精确 route/凭据/价格/输入绑定，最后发送门禁 | 六类文本已消费同一类别当前配置，prompt/schema/业务模式仍由各 Agent 提供；新增视觉用途尚待接通。 |
| `imageagent` factory / intent / worker | 原生图执行、结果、预留、批准及 UNKNOWN | 原 `points/image` 配置和独立 current 已退出正式接线；Flare 官方直连执行及完整上界仍缺。 |
| `OfficialModelTariff` / `orgresource` | 原调用账本、成员月限、企业余额、预留/结算和幂等 | 独立点价已退休；完整官网用量/冻结 FX→微点→原账本已完成本批工程检查，无第二收费路径。 |

具体限制：当前正式文本计价 mapper 仅准入 OpenAI-compatible / openai-text-v1；
Google/Claude native 缺对应计价 mapper，不能由旧标题协议支持推定可执行。
已有文本准入不外推到多模态；统一页面不能代替受影响消费者的实际接线。
现免费事实 Tool 不增加付费模型，图片生成不包装为只读分析。
当前六类文本消费者为标题、内容、平台资料、选品、店铺诊断和工作台规划。
开放 PR #1075 的 `product.research.agent` 另有在途负责人；本轮只登记未来消费者映射，
不接手其分支、实例或实现，不把其在途代码写成已在本调查 main 中接通。

## 3. 统一配置的含义与 owner 候选

### 3.1 一处维护、同一事实、各业务消费

“渠道”表示实际传输服务及其受控 endpoint/协议/地域/凭据绑定；
“模型”表示通过该渠道调用的准确型号与适用能力。两者一起标识可选项，不只保存展示名称。
默认渠道是明确的配置，不是发送失败后的自动切换规则。
默认渠道已由用户确定为 **OpenAI 官方直连**；三类默认准确型号已另由 U6 明确认可，见 §4.3。
GPT/Gemini/Qwen 三家支持目标保持，企业选择 Gemini/Qwen 时应绑定其各自已准入渠道，
不能因默认 OpenAI 而取消其支持目标或把其他渠道模型静默发往默认渠道。
新安装若缺默认渠道/凭据/价格/有效模型，显示未准备好，不能用代码常量冒充可用默认。
平台提供默认渠道也须由原凭据/授权 owner 明确形成当前企业的可执行绑定，
不因“默认”借用另一企业、成员或进程全局 Key；共享凭据的新授权合同不能由本候选推定。

候选企业选择包括企业身份、默认渠道引用、text/image-generation/multimodal 三类准确 binding、
配置 revision 及各类别选择摘要。平台价目/汇率单独发布，准备时与企业选择组合成有效执行快照；
企业选择不复制价格或固定一份永不更新的汇率。对应 payload/schema 已实现；未准入类别不会因字段存在自动可执行。
三类可引用同一具备相应能力的型号；选择另一渠道上的型号必须将实际渠道一并显式绑定，
可选范围沿已确认的三家支持目标和实际合格绑定，不暗中追加企业 BYOK 或任意渠道编辑功能。

候选统一当前选择/价格事实继续归现有 `internal/aicapability/routeconfig`，
落原 ProductAgent/Run DB 的配置存储；复用原 immutable revision/current、CAS、回执行为。
现 ImageAgent DB 继续拥有图片凭据、intent/结果；不迁移凭据、运行或图片事实。
新消费者通过同一个配置读取 port 取得已发布选择，不以 ImageAgent DB 再维护第二份当前模型/价格。
该存储归位和图片 API/worker 的只读接线已完成本批工程检查；图片执行资格仍须逐型号满足。

一个企业配置版本在该配置 owner 的本地事务发布，跨库不做双写。
每次调用只冻结所需类别的有效内容摘要、准确 binding 及价格/汇率版本；
其他类别的无关变更不强制使本次报价失效。摘要及原键输入绑定已在当前共享配置合同实现。

### 3.2 contract → implementation → injection → consumer

| 边界 | 候选接线 |
| --- | --- |
| 可信渠道/模型/官方价维护 | 原受限 provision/配置管理 owner 生成有来源和版本的合格去密 binding；不由业务页面提交任意 endpoint/Key/价格。 |
| 企业默认选择 | 原配置 Service/Repository 的校验、预览、expected revision CAS 发布及回执；选择权限见 §7。 |
| 文本 Agent | 原 application 注入统一 reader → 原 Prepare snapshot → 原 ModelProfile/执行器；各用途保留自己的 prompt/schema/模式。 |
| 图片生成/编辑 | 原 application/worker 注入同一 reader → 原 GenerationIntent/quote → 原 factory/provider；不修改批准/资产 owner。 |
| 看图分析 | #1078 沿原受治理 Model 补图片输入/Quote/usage → 同一类别配置 → 原 Run 的 typed 只读结果。 |
| 资源结算 | 原 Invocation/图片 effect 事实 → 原资源 owner；转换规则和精细数量进入原预留/结算，不建立第二余额。 |

企业三类选择仍一次发布，且必须覆盖全部已开放消费者的协议、权限、计量和预算要求。
预览/选项可投影用户已确认的准确默认身份及缺失原因；Flare 缺保守费用上界时明确 NOT_READY，
不是已发布 current 或可执行默认。Publish 拒绝，不写 revision/current，不新增 pending/部分发布状态。
选择必须引用平台可信目录 binding，不能提交任意型号、endpoint 或凭据。
公式、schema、CAS、统一 reader/validator 可用合成合格/不合格 binding 离线实现和验证，
但这些基础完成不等于真实 Flare 已准入或三类默认完整可用。
任何消费者执行前仍须 fresh 检查完整协议、权限、价格/FX、计量和预算依据；
选择未准备好、不能覆盖当前用途或 reader 不可用时拒绝，不借旧 per-operation/image current，
不按用途偷偷选另一型号，也不降低原预算、权限或输出 Must。
C2-2 消费者已完成限定工程接线；是否能执行仍取决于有效配置、权限、输入和原预算准入，不因 Mini 支持图片输入而自动开放。

### 3.3 选择、价目、汇率与执行快照的字段合同

下表是设计字段，不是已注册的 Go 类型、数据库表或 HTTP 路由。
继续由原 `aicapability/routeconfig` 维护发布事实，在同一个原配置 DB 扩 revision/current 的
有界 payload/scope：平台维护的价目/汇率与企业选择分开，不建立新的控制服务或数据库。

| 事实 | 必需内容 | 写入及作用域 |
| --- | --- | --- |
| 企业三类选择 | verified organization、revision/parent、默认渠道引用、三类 binding ID/准确版本、各类别选择摘要、actor/time、原 key/指纹 | 现企业 Agent 配置权限者；一个企业发布快照，本地事务 CAS，不能改价或凭据。 |
| 可信 binding | provider/channel/model/snapshot、协议/endpoint digest、凭据 owner/非秘密引用及准确版本、支持类别/用途、adapter/usage 版本、token/bytes/deadline 硬上限 | 原受限 provision/准入维护者；实际凭据仍在其原 owner，不进入选择/价目 payload。 |
| 官网价目 | 准确厂商/model/地域/服务与处理档、收费分项/单位/币种/严格十进制费率、来源/核验时间、有效区间、不可变 price version/digest | 原平台维护者；服务端确定适用价目，企业不能提交自己的价目值。 |
| 汇率 | 外币、原公告单位/日期/来源、归一 CNY/1外币的值、生效/到期、不可变 FX version/digest | 原平台维护者；人民币明确使用单位换算，不读取任意企业汇率。 |
| 有效执行快照 | org/actor/member、类别/operation/原业务模式、选择摘要、精确 binding/凭据版本、价目/FX/数量单位版本、prompt/schema/输入摘要、各硬限制 | 原 Prepare/quote owner；只读组合前三类事实，再沿原 snapshot/intent 持久化。 |

同一价目 key 的已发布有效区间不重叠，服务端按准备时间选择唯一有效项；
没有、过期、冲突或无法映射收费分项时明确不可执行，不靠企业旧快照兜底。
平台调价/更新 FX 后，新准备自动读新有效版本，企业不必重选相同型号。
报价有效期不越过已知官方调价或 FX 到期边界；发布变更与时间生效都由最后发送门禁核对。
历史按原快照展示，不使用今日价目重算。平台公共价目不包含其他企业身份、模型选择或凭据。

各类别摘要只含会改变该类别选择/准入的值，不含 UI 标签、发布时间和其他类别选择。
有效执行摘要另外绑定准确价目/FX/数量单位及该 Agent 的 prompt/schema/限制；
同型号文本与多模态执行分别保留类别、operation、输入和原业务权限，不能互换报价。

### 3.4 发布、读取、撤权与幂等

1. 企业发布只接受三类合格 binding 引用和 expected revision；org/actor 从 verified identity 取得。
   三类及已开放消费者中任何一项未满足即拒绝，不写 revision/current，也不保存部分选择；
   未就绪的预览不产生发布回执，不得从请求体接收就绪状态。成功发布仍不代替最终发送门禁。
   原平台维护 API 独立检查平台维护及发布权限，企业选择 API 检查 `workbench.agent.configure`；
   不能因同一页面而合并成可写价目/凭据的企业接口。读者只得到当前企业可用的去密投影。
2. 校验读取原凭据 owner 的去密元数据和当前准入描述，核全体已开放消费者、类别和原硬限制。
   预览保留不可用原因；发布和执行分别拒绝未满足条件的 binding，不以发布快照代替执行许可。
   图片凭据与配置跨库读取不能承诺原子；发布提交前后若凭据轮换，已发布选择可立即变成
   NEEDS_CONFIGURATION，回执仍真实存在，执行再次 fresh 核验并拒绝失效绑定。
3. 原配置 DB 中先按 org/actor/key 找回已存在回执并比较原请求指纹，再锁 current、核 expected revision，
   插入不可变 revision 并 CAS current；无跨库写、不持有配置事务进行 provider HTTP。
   唯一冲突回滚后读原回执；COMMIT ACK 不明只查询原键，不能换键或从浏览器推定成功。
4. 新准备通过同一 reader 取得企业类别选择与当前有效价目/FX；各消费者不缓存第二份当前值，
   也不让 ImageAgent DB 的旧 current/points-per-image 成为另一选择或计价来源。
   reader 故障拒绝新执行，原已记录结果/账单的读回不因此重新调用模型。
5. 原 Prepare/Start 重放先查原键/原回执。后台调价不会把相同用户意图自动变成另一份报价，
   原准备可读但过期时不能发送；改变图片/模式/用户选择等请求意图仍沿原冲突合同。
   最后发送比较原冻结类别/准入/价目/FX、fresh 业务授权与实际凭据，漂移要求新显式准备，
   不在同一次 Start 中静默换模、改价或追加 provider 请求。
6. 已通过最后门禁的原一次发送按原快照完成/结算；配置发布、企业撤权和取消页面不伪造远端取消。
   新请求/下一步骤继续执行原 fresh 门禁。相同价目/FX 内容重新发布的新版本仍有不同身份，
   不通过 A→B→A 复活旧报价。UNKNOWN 找回只读原事实，不借新模型/凭据重发。

### 3.5 最小消费改动与新安装

文本原 resolver 从业务用途 map 改为“operation→所需类别”的固定映射，再组合该 Agent 的
prompt/schema 与其更小限制；工作台规划等原独立命令、配置快照、run/claim owner 保留。
图片 worker 增加原配置 owner 的受限只读 reader，经原 factory/intent 使用准确 binding；
不移动其生成/批准事实或将两库 current 做同步。多模态按 #1078 原受治理 Model 执行，免费 Tool 保持。
三类 consumer 的启动验证检查相同配置 schema/数量单位与所需 reader 权限；只验证、不 seed 或变更数据库。
后续安装只为获准新空库安装所需版本/读取角色，不自动升级保留库、创建实际默认密钥或发布真实配置。

## 4. 官方价格与点数换算候选

### 4.1 价格证据与原币种

每条价目冻结实际服务渠道、厂商/准确模型、地域、服务类型、收费单位、币种、
输入/输出/思考/缓存等适用项或图片数量/尺寸/质量档，官方来源 URL、生效时间和失效条件。
平台维护者核验并发布；本候选不建设自动抓价、模型发现、汇率抓取或供应商账单对账平台。
页面展示官方价格与折算规则，消费点数不再是可独立随意编辑的加价字段。

用户已在本会话 2026-11-03 的单独回复中确认：**模型厂商官网公开价，不额外加价**。
用户扣点依据是准确模型适用的厂商官网价目；代理公开价、账户专属折扣或平台任意点数价
不能替代该依据，也不自动叠加代理附加费。仍须冻结该模型/地域/服务项的准确官方价目版本。
实际渠道成本单独供原成本预算使用，不改变上述用户收费决定；
未知渠道成本/价差继续 UNKNOWN，不写成已知或已接受风险。实际成本预算与用户扣点分别可证明。
缓存/阶梯/促销等是否适用必须由准确模型及实际 usage 证明，不使用赠送额度假定零成本。
无法证明完整计价和发送前预算的模型保持不可选/NOT_READY，并说明具体原因。

本会话前轮只读查阅的候选证据包括：

| 候选 | 可引用的公开依据 | 仍缺的执行依据 |
| --- | --- | --- |
| `gpt-5.4-mini-2026-03-17` | [型号/价格](https://developers.openai.com/api/docs/models/gpt-5.4-mini)、[图片计量](https://developers.openai.com/api/docs/guides/images-vision#calculating-costs) | 实际渠道/地区、组织绑定、各用途兼容、图片 Quote 和完整 usage 验证；看图支持不作生图准入。 |
| `gemini-3.8-flash` | [型号](https://aistudio.google.com/docs/models/gemini-3.8-flash)、[价格](https://aistudio.google.com/docs/pricing#gemini-3.8-flash)、[图片](https://aistudio.google.com/docs/image-understanding) | Interactions 图片预算/usage 精确依据和超出原标题用途的接线；2026-12-31/2027-01-01 价格边界必须处理。 |
| `qwen3.8-flash` | [型号/地区价格](https://help.aliyun.com/zh/model-studio/qwen3-8-flash)、[图像计量](https://help.aliyun.com/zh/model-studio/vision) | 实际地区/部署范围、参数与计量上界、准确绑定及各消费者准入。 |

这些是支持目标和文档证据，不是已配置/可调用/质量通过。生图准确型号/官方单位另行核定，
不能从上述三条视觉输入能力推定；现场准备和真实计费一致性 UNKNOWN。

### 4.2 换算与精度

当前批准的资源购买价是 `1 AI 点 = ¥0.01`，本候选保留该购买关系。
候选：`实际点数 = 官方可计费用量 × 适用官方单价 × 冻结人民币汇率 ÷ ¥0.01/点`；
人民币价目汇率为 1。非 token 收费先按准确官方单位计算金额，不伪装成文本费率。
计算使用严格十进制定点/有理数和检查过的整数，禁止浮点累积或先把小额费用取整到人民币分。

发送前预留保守上界，最终按完整实际 usage 结算并释放差额；预留数量不代表最终收费。
有思考收费必须计入完整生成量；输入模态不能重复加总；缺字段、矛盾或超界不能当零结算。
官方价/汇率/精度版本进入同一 quote/profile/input hash 和原结算事实，不能只写页面文案。

原 `ModelPointTariff.Points` 曾将单次用量向上取整到整数点，现已退休。
例如官方金额 ¥0.002 对应 0.2 点，当前公式不能沿用旧逻辑扣 1 点。
当前新空库合同在原资源 owner 内使用更细定点数量，展示层仍是 AI 点；
`1 点 = 1,000,000 微点`，预留向上/实际最终向下取整，差额边界见 §4.5。
该规则已完成限定基础架构及工程检查；只用于合格新空库，不重解释保留余额，不称“无舍入误差”。
原购买授予、成员月限、预留、结算、退款/释放和展示须使用同一版本的数量单位；
预算比较与溢出保护必须覆盖换算前后，不能只在最终扣款时转换。
不另建余数钱包/分摊账本/周期结算平台来绕过精度设计。

### 4.3 OpenAI 三类默认模型候选（产品设计已认可）

2026-11-03 重新只读核验 OpenAI Docs 的型号、快照、价格和图像指南后，提出下表。
这是面向本项目有界业务的工程候选；没有执行质量、延迟、真实费用或账户可用性验收。
用户在本会话后续明确认可本表准确默认型号及 §4.5 的汇率/精度/权限分工；
该认可只确定产品设计，不得写为已实现、已配置、已复核或 IMPLEMENTATION_READY。

| 企业默认类别 | 推荐准确型号 | 公开能力与依据 | 本项目选择理由 |
| --- | --- | --- | --- |
| 文本 | `gpt-5.4-mini-2026-03-17` | [型号](https://developers.openai.com/api/docs/models/gpt-5.4-mini)支持文本、结构化输出、Chat Completions，支持 `reasoning.effort=none`。 | 有界标题/内容/资料提案等先复用原兼容传输；各用途兼容仍须逐项核验。 |
| 图片生成/编辑 | `gpt-image-2.5-flare-2026-09-08` | [型号](https://developers.openai.com/api/docs/models/gpt-image-2.5-flare)定位日常图片生成，接受文字/图片并输出图片；[Images API](https://developers.openai.com/api/reference/resources/images/methods/generate)列出准确快照。 | 在原 ImageAgent 单 slot/effect 下直接执行，不为一次生图另加主控文本模型。 |
| 多模态理解 | `gpt-5.4-mini-2026-03-17` | 同一型号支持图片输入、文本输出；[图片计量](https://developers.openai.com/api/docs/guides/images-vision#calculating-costs)公开 patch/倍率规则。 | 先覆盖本项目单图理解，三类设置不要求三个不同型号；不扩为音频/视频理解。 |

公开标准同步费率如下，均为 **USD / 1,000,000 对应 token**，读取日期 2026-11-03。
[统一价格页](https://developers.openai.com/api/docs/pricing)的 Standard 与 Batch 必须区分。

| 型号/费用项 | 非缓存输入 | 缓存输入 | 输出 |
| --- | --- | --- | --- |
| GPT-5.4 Mini，文本及已折算的图片输入 | 0.75 | 0.075 | 4.50 |
| GPT Image 2.5 Flare，文字输入 | 5.00 | 本候选直连 Images API 不适用缓存优惠 | 无文字输出费用 |
| GPT Image 2.5 Flare，图片输入/输出 | 8.00 | 本候选直连 Images API 不适用缓存优惠 | 30.00 |

图像输出的 30 USD/百万 token 不是 30 USD/张；也不能把 Batch 的 15 USD 当同步输出单价。
GPT-5.4 Mini 地区数据驻留端点有公开额外费率；最终选择该类端点时按实际适用官网项计价，
这里的通用标准费率不证明已选择某地区账户或数据处理安排。

### 4.4 参数、usage 与原架构接缝

- 文本/单图理解推荐固定 `reasoning.effort=none`，保留原 output schema 与更小业务预算。
  每个 Agent 的原 ModelCalls 上限不变；C2-2 仍只允许一次视觉调用。
  原 component 当前配置 `MaxTokens`，必须核 SDK 实际 wire 参数、准确型号的输出硬上限及参数支持，
  通过原离线 transport/guard 证明，不从 SDK 存在方法推定模型已准入。
- 单图理解推荐显式 `detail=high`：该型号公开上限为 2,500 patches，再乘 1.2，
  图片输入估算约不超过 3,000 token；官方说明最终计数可能有 1 token 差异，报价须留保守余量。
  另外计文字/协议与完整输出上界；实际字节/像素及总 wire 继续受原更小限制。
  `low` 在该型号可有更大 patch 预算，不能套用旧型号“固定少量 token”的价格假定。
- 生图候选使用原 ImageAgent 的单 effect → 一次直接 Images API 调用；
  不走 Responses 的 image generation tool，避免新增主控模型收费和当前不能从响应核验的生图缓存优惠。
  官方[成本说明](https://developers.openai.com/api/docs/guides/image-generation#cost-and-latency)
  明确两种 API 的主控/缓存差异。默认 `n=1`、非流式，无 partial image；整个计划仍可逐 slot 生成多张。
  业务选择的尺寸/质量须在报价前变为明确合法值；不暗改原图片要求或把 `auto` 当固定成本。
- 生图按文字输入、图片输入、图片输出分别计价，从完整 raw usage 验存在性、模态分项及加总，
  输出计数不重复计入其他输出项。官方参考的 usage 说明残留“gpt-image-1 only”字样，
  但同页新型号示例/指南展示 usage；不将该文档不一致当作真实 Flare 必然回完整用量的证明。
  需在原实现层核准确 SDK/schema、缺字段与非成功响应；实际回执不足时保持 UNKNOWN。
- 原 `ObservedUsage`/`InvocationRecord` 已补完整收费分项/存在性和冻结官网价，
  OpenAI raw mapper 保留缓存明细，原 `ModelPointTariff` 已删除。
  C2-2 复用同一 mapper/公式，不按 base64 字符或另加图片输入重复收费；
  缺缓存字段/矛盾/越界依旧 UNKNOWN，不按全价猜测。
- 用户消费折算基于各收费分项的完整定点总金额，不能先分别向上取整到货币 micros 再推导扣点。
  原保守成本上界仍可用于预算；实际消费金额、保守预留和估算成本的字段职责须分清。

C2-2 的发送前图片+文字上界及输入/结果接线已完成限定独立准入与工程检查（PR 中 `4b6c03ca` 证据）；
Flare 的尺寸/质量对应完整输入输出上界与正式执行协议仍缺，不能从已实现 raw mapper 推定。生图指南的交互计算器是估算依据，不单独证明可强制的最坏费用上界；
缺这一依据不能放大原上限或只靠账户预算设置声称单次费用受控。
以上是确需实现/合同验证的接缝，不是要调用真实模型才能开始整理设计。

### 4.5 汇率、微点与权限候选（产品规则已认可）

**汇率候选**：平台维护者按工作日核验并发布一版 USD→CNY，来源建议采用
[中国外汇交易中心人民币汇率中间价](https://www.chinamoney.com.cn/chinese/bkccpr/)。
其页面明确交易中心受人民银行授权每日计算和发布；本轮只核出处，不将页面数据读数当本产品已发布汇率。
记录原始单位、公告日期/来源、归一为“人民币/1美元”的值、生效/到期与服务端版本。
非工作日可明确引用最近公布值，但仍须有有效期；缺失/过期拒绝新报价，不默认 1:1 或任意汇率。
准备时展示并冻结汇率；未发送时相关版本变化须重新准备/确认，已准入在途按原冻结值结算。
不新增自动抓汇率任务，不改动任何保留账户或供应商结算安排。

**精度候选**：保留 `1 AI 点 = ¥0.01`；原资源 owner 内 `1 AI 点 = 1,000,000 微点`，
前台以 AI 点显示，必要时披露至小数后六位。所有原资源金额/月限/预留及账单字段绑定同一数量版本。
预留上界向上取整到微点；实际完整金额只在最终一步向下取整到微点，避免多扣。
每次实际扣量与理论折算差额小于 1 微点，即小于 ¥0.00000001；用户已认可该舍入取舍，
不称完全无误差、不建立跨调用余数事实，也不因此放宽供应商原币种成本上限。
已知精度舍入与缺失 usage 的 UNKNOWN 必须区分，不能用舍入规则把缺失成本记零。

纯数学例子：假设 2,000 非缓存输入、500 输出 token，按 Mini 上表费率为 USD 0.00375；
若仅为举例设汇率 7 CNY/USD，则为 ¥0.02625 / 2.625 AI 点（2,625,000 微点）。
汇率 7 不是当前实际汇率或建议固定值，本轮未发布任何汇率或真实费用。

**权限候选**：平台维护渠道/合格型号/官网价和汇率；企业已有
`workbench.agent.configure` 权限者在合格列表中选择三类默认型号，成员读取并按原业务权限使用。
企业选择不能改变官网价、凭据、成员月限或其他业务写权限；角色到现有权限的授予映射保持，
新选择动作必须检查有效权限，不能以 admin 标签推定许可。本轮未授予、撤销或读取真实用户权限。

### 4.6 原始 usage、收费分项和精确换算

新的计量字段归原 Invocation/图片 effect owner，保留实际来源和 immutable proof；
不从 UI、模型输出正文或新当前配置补造用量。建议原计量事实增加 schema/usage 版本、
已观察的分项及完整性标志、价目/FX/单位引用、理论金额与最终微点量，沿原记录事务保存。
实际终态同时保存足以复算的原冻结费率/FX 数值、来源版本和摘要；这些是本次执行证据，
不是另一份当前价目。恢复不依赖今日价目/汇率或其服务可用，不按今日值重算。
只保存守卫核验后的计量片段/存在性和安全请求引用，不存 Key、图片二进制、隐藏思考或完整原始错误正文。

| 协议/用途 | 正式计价分项和等式 | 缺失/矛盾处理 |
| --- | --- | --- |
| Mini Chat，文本/图片理解 | P=prompt、C=completion、T=P+C；K=已观察 cached prompt，0≤K≤P；非缓存输入=P-K。费用=(P-K)×非缓存输入价+K×缓存价+C×输出价。图片输入已在 P 中，不另加一次。 | 必需总量、缓存明细或适用语义缺失时不能按全价/零缓存代替；保持 UNKNOWN。C 中思考等细分不再次收费；与准确 schema 矛盾或出现未支持收费维度拒绝结算。 |
| Flare 直接 Images API | I=输入总量=文字输入+图片输入；O=输出总量；T=I+O。文字/图片输入分别按价目，Flare 已准入 image-only 输出按图片输出价。若输出分项存在，必须与 O 一致。 | 必需字段/分项缺失、负值、加总不符、未知计价单位或非预期文字输出保持 UNKNOWN。该直接 API 不应用 Responses 生图缓存优惠。 |

字段存在性和整数范围必须在原 raw response 解析时保留；禁止先经默认零值 DTO 后再声称存在。
完整的全零用量也需与缺失区分：仅在已准入协议能证明明确终态失败且所有用量确实报告为零时，
作为“已发送、已观察零用量”处理，不改写成 no-send。该终态证据类型需随原账本新 schema 一并复核；
不改变 HTTP 超时/无 usage 继续 UNKNOWN 的规则。成功图片/文本与零输出矛盾时拒绝该 proof。
新合同不自动把原未知调用重新分类，原 no-send 释放仍须原完整未发送终态证明。

费率统一存严格十进制数及公开分母单位，token 候选为“每百万 token”、价格小数至六位；
USD→CNY 候选至六位，保留官方公告原精度和单位，超过支持精度拒绝发布而不静默舍入。
本组三类默认均按 token 计价，设 rᵢ 为放大 10⁶ 的原币种每百万 token 单价，uᵢ 为各分项用量，
F 为放大 10⁶ 的 CNY/1原币汇率，则 `S=Σ(uᵢ×rᵢ)`，精确原币金额为 `S/10¹²`，
精确微点为 `S×F/10¹⁰`。人民币价目 F=10⁶；不能直接把公告中“每100外币”的数字当 F。
本候选用受长度限制的整数/BigInt 完成加总和除法，最终才 floor 实扣/ceil 预留并核 int64 溢出。
非 token 的其他型号须按其公开单位确定分母，不把“每张”强行代入以上百万 token 公式。

原 `CostFor` 的保守货币 micros 仍可用于预算比较；用户实扣从上述完整分项金额计算，
不能消费该已进位的估算字段。报价和记录同时保存原币金额、冻结 FX 与微点结果，
原币成本硬上限和企业点数/月限分别核验；汇率变化不能扩大原模型调用/费用上限。

### 4.7 生图费用上界：当前仍缺的具体依据

当前最小接线、完整计费分项及证据缺口见 [Flare 设计候选](openai-flare-image-generation.md)。
该候选明确 BT/BI/BO 各自所需证明，复用原公式/账本与预算；设计补文档不等于完整上界已证实或型号已准入。

本轮重新只读查阅的[Images 请求参数](https://developers.openai.com/api/reference/resources/images/methods/generate)
公开了 prompt/n/size/quality 等，但未找到适用于准确 Flare 的显式总 token/金额上限参数。
[成本指南](https://developers.openai.com/api/docs/guides/image-generation#cost-and-latency)
把计算器结果表述为输出费用估算，并明确不含文字/图片输入及部分图像额外费用。
因此目前不能只把计算器值填成 maximumCost，也不能用 SDK timeout/响应大小限制充当已发生费用上限。

准确 profile 需要一份可追溯的 BoundEvidence：model/snapshot/endpoint/协议和参数域，
文字及各来源图输入上界，固定尺寸/质量/n/非流式下的输出上界，推导/官方出处及适用版本。
验证时实际 serializer 必须强制该参数域和较小 bytes/deadline；报价取公开边界的保守值，
缓存输入按非缓存上界预留，再按实际完整分项结算。没有这份依据，Flare 准备明确 NOT_READY，
不静默换型号、不把上界改为“尽力估计”，也不下调用户已确认的预算 Must。

已准备待核问题，未发咨询、未调用 provider：
1. 准确 Flare snapshot 在明确 size/quality/n=1/stream=false 下，输出计费 token 是否有可引用的保守最大值/推导，超界参数如何拒绝？
2. 直接 generation/edit 的文字与图片输入计费上界如何根据实际 bytes/dimensions/图数推导，是否有其他计费维度？
3. 直接 API 在成功、拒绝、输出校验失败时，哪些完整 usage 字段是正式保证，如何区分零用量和缺失？
仅新增公开说明或相应合格依据即可收敛，不先要求真实调用、专项验收工具或供应商书面担保。
不以 provider token-count/上传接口做未确认图片的额外外发，也不先生成后补报价。

## 5. 状态、并发、失败与恢复

| 边界 | 候选不变量 |
| --- | --- |
| 校验/预览/发布企业配置 | 零 provider 调用、零业务运行；相同 key+payload 找回原回执，异载荷冲突；expected revision CAS 处理并发。 |
| 配置提交响应丢失 | 原 key 读回执和实际 current，不换键重复发布，不凭浏览器状态推断成功。 |
| Prepare/报价 | 读取当前企业/类别和真实合格绑定，固定配置摘要、图片/文本输入、官方价/汇率/数量单位及预算。 |
| 未发送时改模型/价格/权限/凭据 | 原最后发送 guard fresh 核验受影响摘要与资格；漂移拒绝，不静默替换模型或重新报价发送。 |
| 已通过最后发送门禁/在途 | 允许原一次调用完成，按原冻结规则结算；发布不是网络取消屏障。 |
| 发送后结果或用量不明 | 保留原 invocation/effect、预留、UNKNOWN 与原恢复 owner；不换模型、渠道或 key 重发。 |
| 已有真实用量但输出不合格 | 原用途固定校验失败，保留并结算已发生用量；C2-2 不增加 repair/解释调用。 |
| 历史/已完成结果/账单 | 读取原事实和原显示单位/版本；不按新官方价或新模型重算，不下载今日图片补旧证据。 |

不以长数据库事务跨 provider HTTP；配置读和凭据读跨 owner 不能宣称原子。
最终门禁按该次实际读到的配置与有效凭据定义准入，沿现有 in-flight 边界处理竞争。
纯配置变化不改变业务事实、审批、发布许可或已有执行定义的版本资格。

### 5.1 微点贯穿原资源 owner

量纲标识 `ai-micropoint-v1`，固定 1 AI 点=10⁶ 微点；正式新空库 schema 和原资源全链读写接线已完成本批工程检查。
只改变 AI 点数的记账精度，DATA_ROW/存储等其他资源和 money 的人民币分单位不变。
新空库安装在 schema/资源元数据中固定单位，并在 quote、owner proof、余额、月限、预留、
消费/释放/债务事件及回执中保留相同单位版本；混合或缺版本拒绝启动/写入，不猜测数值含义。

| 接缝 | 数量及事实责任 |
| --- | --- |
| 购买 | Commercial 仍按已批准 1分/整点出售；原可信订单 item 的整点量在授权授予边界精确乘 10⁶ 一次，写原资源 proof。重复相同 source/key 找回原授予，不再次乘量/发点。 |
| 试用/其他已批准 AI 来源 | 原 source 继续唯一授予 owner；新安装明确业务点数量到微点量的转换及单位。没有新发放/补造余额授权，不从旧记录回填。 |
| 成员月限 | 管理入口原点数语义明确转换成微点；原 LimitVersion、月份和月度 consumed/reserved 均同单位。界面显示精确点数，不能因换单位扩大原限额。 |
| 预留 | 原 Invocation/effect claim 固定原业务、org/member、输入、模型/价目/FX/量纲及上界 proof；原资源事务锁操作、月限/计数与企业 bucket，按上界 ceil 微点预留。 |
| 结算 | 读取原 owner 完整终态分项和冻结规则，按实际 floor 微点；同一原 finalize key/相同 proof 幂等，异 proof 冲突，不信任调用方传入扣点量。 |
| 释放差额/债务 | 差额沿现有 `applyPositiveCredit` 的正向入账与债务抵扣规则；不直接给 Available 加值而绕过原 debt owner。相关债务/抵扣数量也必须同量纲。 |
| 展示/传输 | 内部 int64 微点，前端使用明确单位的整数十进制字符串/服务端点数格式化，避免 JS 浮点和安全整数范围丢失；对外购买数量仍整点，不将微点当购买点数。 |

资源事务保留原 operation→reservation→member-month→bucket 锁/幂等机制；
预留 R 后仅在已观察合法用量时结算 A，必须满足 `0≤A≤R`。
原 reserve R 从 Available 移到 Reserved，并使该成员月 Reserved 增 R；
结算使企业及原成员月 Reserved 减 R、Consumed 增 A，差额 D=R-A 经原正向入账规则分配。
无债务时 Available 增 D；存在债务时先按现有规则抵扣，回执明确净可用入账和抵扣量。
所有参与值与中间加减核溢出/非负；A>R 不截成 R、不补扣、不虚构债务或偷偷释放差额，
保留原预留/待核实并报告违反报价上界。量纲变化不授予扩大余额、月限或原币预算的权限。

结算月份绑定原 reservation/StartedAt 的原月份；跨月完成不记到新月份，重复回读不重复消费。
源 DB 的终态记录与 Resource DB 不建立跨库事务：原源事实先 durable，原 RecoverDue 只读证明，
重试同一资源 finalize 事务；ACK 丢失后读回执，不派发第二模型调用或创建第二资源预留。
有合法 no-send proof 时释放整笔 R；已观察零用量的明确终态按 §4.6 的新 proof 规则审定，
与发送后没有完整结果的 UNKNOWN 分开，缺旧模型/凭据时不替换它们补造证明。

已知金额小于 1 微点、最终 floor 为零不等于免费准入：价目和完整用量仍须存在，
非零上界仍先 ceil 预留，回执保留理论金额与量化差额；缺失 usage 永不走这条规则。
这套单位变更涉及原购买、月限、图片与文本资源消费者的一致新安装，不能仅修改公式或前端小数。

### 5.2 新 schema 的最小增量与安装顺序候选

以下字段已沿原 initializer/schema verifier 实现，并在任务自有临时 PostgreSQL 验证。
只支持原 owner 的新空库初始化；不升级旧表、不连接任何保留库。

| 原 owner | 必要增量 | 明确保留 |
| --- | --- | --- |
| 配置 owner | revision payload 区分企业选择/平台官网价/FX，明确作用域及版本/有效期；current/有效价目索引保持唯一且不跨企业；拒绝未知 schema。 | 原 CAS、key/指纹、immutable history，不为本任务增加队列/协调器或配置双写。 |
| Invocation / 图片 effect | 原输入/上界/终态证明增加计量版本、收费分项/存在性、冻结价目/FX/数量单位与算费证据；原 attempt/claim/result identity 继续唯一。 | 图片/业务事实及原 Run 状态；无第二 provider 调用 owner，不能把共享型号误当共享 invocation。 |
| orgresource | 明确 AI unit schema，bucket/月限/事件/预留/债务/释放/授予及回执按同一微点量纲；幂等 proof 必须绑定单位和源收费事实版本。 | 原资源事务、正向入账/债务抵扣和 RecoverDue；money 仍人民币分，其他资源单位不变。 |
| Commercial 可信授予边界 | 新空库的订单 item 明确整点商品数量；原资源 grant command 明确转换后微点及 unit version，同 source 重放核等值。 | 原报价/订单/支付/授予来源，不新发额度或重放旧购买。 |

启动按原配置/Invocation/资源/图片 owner 各自 Verify 检查版本、量纲和只读权限；
任一新消费者不满足就说明不可用，禁止半新半旧消费者写同一个 AI bucket。
三个当前模型与价格不从旧 per-operation map/旧图片 current 导入，也不自动 seed 企业选择。
保留实例维持原程序/库/会话；后续真实切换、数据演进或删除均需独立明确授权。

## 6. 新安装与原数据边界

全新安装/空业务数据基线继续有效。DDL/初始化/最小角色与连接行为仅在任务自有临时 PostgreSQL 验证；
没有连接保留库或发布真实配置。当前 schema/单位只支持新空库；保留库不迁移、不改量纲、不清理。
旧 current/独立费率路径在新合同消费者中应 RETIRE，不新增双读、双写或 fallback。
已存在运行和账单继续原事实读取，不把它们迁移成新三类配置运行或跨版本续跑。
如果正式实施确需保留实例数据/余额演进，必须先报告具体冲突并取得新的产品决定，不能暗中做迁移。

Legacy decision: **EXTRACT** 已有 revision/CAS、精确资格/guard、账本与资源事务行为；
**RETIRE** 新消费者按用途独立选择当前模型、独立随意点数价和旧量纲消费入口。
Current owner: `aicapability/routeconfig`、原 Model/ImageAgent 执行 owner、`ledger/orgresource`。
Cutover/deletion condition: 获准新安装消费者完整接通并有范围内证据；本轮不执行切换/删除。

## 7. 权限、页面及待决项

候选分工：平台维护者维护可信渠道/型号/官方价/汇率，企业获准配置者仅选择三类默认模型；
普通成员读取适用配置并沿原业务权限使用，不能通过选择型号扩预算、改价格或获得业务写权限。
当前模型维护页仅平台维护者有权，企业 Agent 配置已有 `PermissionWorkbenchAgentConfigure`；
用户已单独认可将已有 Agent 配置权限用于企业三类默认模型选择；
HTTP/服务层动作、角色映射、fresh 撤权及凭据/费率隔离已通过本批独立工程检查；不改真实权限。
org/actor 来自 verified identity；跨企业选择、凭据引用和旧页面提交均拒绝。

| 项目 | 当前状态与剩余依据 | 阻止的阶段 |
| --- | --- | --- |
| 默认身份与支持目标 | U1/U3/U6 保持；共享选择/去密资格已实现。未映射协议不准入，不取消三家支持目标。 | 不合格型号执行；真实账户准备 UNKNOWN。 |
| 汇率与微点 | 冻结 FX、完整收费分项、精确微点、原账本和全链单位已通过有限基础工程检查。 | 无真实价目/FX发布；旧/混量纲库拒绝，不迁移。 |
| 配置权限及页面 | 企业 configure 与平台 admin.write 分工、只读投影及原键恢复已通过工程检查。 | 浏览器/Figma/产品及真实配置验收 NOT_RUN。 |
| C2-2 | [单图设计](product-image-visual-analysis.md)的实际图片输入、Mini 上界/协议和只读消费在 `4b6c03ca` 完成限定工程检查；未变代码复用原证据。 | 不再列为待实现阻碍；现场配置 UNKNOWN，运行/效果/产品验收 NOT_RUN。 |
| Flare | §4.7 的完整发送前保守费用上界与官方直连执行仍缺；不咨询、不以估算冒充最大值。 | 该型号准备/执行及完整三类原子发布；不以部分/pending current 绕过。 |
| 运行与产品 | 工程/合成 transport 证据不证明真实模型质量或现场可用。 | 运行、真实保存/恢复、浏览器、效果及产品验收 NOT_RUN。 |

## 8. 后续验证与本轮交付边界

实现授权已取得；适用独立架构复核明确本轮安全基础 IMPLEMENTATION_READY 后，
只扩现有 routeconfig、Model/AgentConfig、image factory/intent、orgresource 与页面测试：
三类共享读取、用途兼容、跨企业/撤权、CAS/ACK、同 key 异载荷、价格/汇率/单位绑定，
低于一整点/边界/溢出、真实 usage 不完整和无二次派发、历史原单位只读。
跨 owner 接线只用任务自有临时 PostgreSQL/合成 transport；不新建验收平台或重做全仓审核。

本地文档适用检查与 GitHub 准确 HEAD CI 必须分开；旧候选 CI 不覆盖后续设计增量。
本轮统一配置设计范围以用户后续明确产品决定及继续指令为依据，
提交/推送与准确 HEAD CI 沿既有 [design-only Draft PR 授权](https://github.com/henryxu1008/shuomi-ai-commerce/issues/134#issuecomment-6592979123)维护原 PR；
后续实现以用户最新“不用咨询，继续开发”的明确指令及范围内独立准入为依据，
不扩大为新增人员、合并、部署、关单或真实操作权限；未准入的 Flare 执行继续明确不可用。
本候选不覆盖 C2-2 的单图范围/一次调用/只读结果要求，也不宣称 #134/#128 已完成。
缺证据 UNKNOWN；限定基础架构准入及其独立最终工程检查已完成；完整三类用户目标仍未完成；
运行、真实保存/恢复、浏览器、模型效果及产品验收 NOT_RUN。
#47 PostgreSQL UNKNOWN、#307 OPEN、#733 咨询 NOT_SENT；保留实例、数据和会话。


## PR #1078 本轮接线记录（工程实现，非运行或产品验收）

当前实现沿冻结的有限安全基础：原配置 Repository 的企业三类选择/平台官网价与 FX 两类 payload；同一个 ProductAgent/Run DB 的原 revision/current/CAS/回执；原 Resource owner 的微点、授予转换、member limits、图像 R/A 和 debt-first 差额释放。没有新增钱包、余数账本、双 current、迁移或第二收费引擎。

- 官网价 payload 显式冻结地区、同步服务类型、必要图片尺寸/质量及官方来源/有效期；型号/渠道/地区/服务/图片参数不匹配、价格/FX 缺失或过期均不授予执行。
- 稳定价格条目在同一 owner 的下一 revision 内复用原版本；修改/移除再加入产生新版本，不通过回滚复活旧报价。平台发布也保留原 key 优先、CAS 与 uncertain ACK 语义。
- `ValidateShape` 只检查准入描述和原币成本上界；正式 `ModelProfile.Validate` 仍要求完整的当前官方 tariff。部署描述里的历史 tariff 不会被统一 reader 作为收费价目消费。
- 图片 `Points` 是由统一官方公式计算的预留微点缓存，须与冻结 Billing 完全相符；结算/proof/closure 传实际 A，另保留 R。缺完整实际 usage 不结算；A=0 仍可 committed。原工厂/ledger/key/恢复责任保留。
- 新空资源 schema 含明确数量单位；已有缺列/混合单位/旧单位库拒绝且不修复。购买源数量维持整点并在可信入口转换一次；余额、月限、事件和 proof 同量纲。
- 当前页面与 BFF 替换旧独立点价编辑及 per-purpose 发布路由，三类模型一次选择；平台官网价与 FX 仍单独受既有平台维护授权。业务页不另选模型。

Legacy decision: EXTRACT | RETIRE
Reusable behavior: 原配置 CAS/回执、去密凭据投影、单 effect、原资源事务、debt-first 入账、UNKNOWN 和原结果找回。
Current owner: `aicapability/routeconfig` 的原 ProductAgent/Run DB；原 Image 凭据/生成 owner；`orgresource`。
Cutover/deletion condition: 旧输入/输出点价、图片 PointsPerImage、两个 serving current 和业务页 VIP 新选择退出正式接线；不迁移或重解释保留事实。未变化的历史记录及旧证据只按原范围保留。

本轮不把完整三类交付标为 Ready：Flare 完整保守费用上界/直连适配仍缺；C2-2 的生产读取、企业多模态解析、原页面确认及只读历史在 `4b6c03ca` 已通过准确 HEAD CI 与原独立最终工程复核，证据见 PR。此后 Flare 只增加 design-only 候选，不能外推已有工程批准。完整三类原子发布及未准入门禁保持。GPT/Gemini/Qwen 的支持目标未撤回，未准入型号不伪装可用。既有验证只使用合成数据/transport 与任务隔离 PG；本设计增量不重跑测试，现场准备 UNKNOWN，运行/真实保存恢复/浏览器/模型质量/产品验收 NOT_RUN。
