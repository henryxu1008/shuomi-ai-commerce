# 百炼 DeepSeek 共享文字模型准入：全新系统设计

Design Basis: **Independent Architecture**。原离线范围的准入及证据保留；**价格精度要求已由2026-11-04用户纠偏替代**，按下节收敛原公共合同，配置/实例/真实调用仍未授权。
执行：[#1073](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1073)，主要PR [#1075](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1075)，公共模型依赖 [#1078](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1078)。本文件只提出原公共owner的有界增量，不是生产实现、配置载荷或实例操作授权。

## 当前产品纠偏：参考价与估算服务业务流程（2026-11-04）

用户明确指出两个聊天过度追求价格准确性，并决定“模型的价格不用那么准的，官方的价格也是经常会变”。当前阶段使用平台维护的参考价和费用估算即可；停止继续研究或咨询逐次官方适用价、精确峰闲计价时点及供应商可证明的完整最坏费用。以下正文中由这些精度目标派生的阻塞与验收前置自本决定起撤出当前阶段，原资料和测试只保留其既有证据，不记为新目标PASS。

当前优先结果是研究与选品业务路径。参考价记录所用配置版本，估算如实标识；缓存细分字段/时点证据不足不再单独阻塞该业务目标。保留原共享模型和唯一账本、租户/权限、配置预算/点数上限、真实调用状态、原键/幂等及重复扣费保护；缺失数据不能伪装成供应商精确事实。预算仍每项¥0.30/20点，两项各一次¥0.60/40点，沿原更小调用/Token/byte/期限边界。

本纠偏已通知#1078原公共owner；由其在原公共合同中收敛最小价格/估算语义，#1075消费同一合同，不并行发明按次或固定扣点、不新增另一收费实现。此次文档同步不自动改变已有代码的TIME_BASIS_NOT_ADMITTED等门禁；相关实现须按收敛后的现有合同调整。原精度前置不再是要求用户等待供应商答复的理由，不以新一轮重型设计/咨询替代交付。

不改凭据、不发布配置、不更新/清空/重建实例、不执行真实采集/AI/付费、不合并/关单；全新系统、不迁移/兼容旧数据的边界保持。下方与本节冲突的严格精度要求不再具有当前产品优先级，旧评审不能恢复已撤回要求。

## 1. 产品决定、范围与验收边界

用户先同意准备DeepSeek准入和保留实例的数据演进设计，随后明确纠正：**“不用保留旧的，这是新设计的系统，原来的应该可以不要的”。** 最新决定撤回旧配置/历史/余额/预留/在途执行的承接要求，按[全新系统基线](../product/greenfield-no-legacy-migration.md)和当前模型合同推进。本任务不设计旧数据迁移、旧ID/回执转换、旧状态续跑、兼容adapter、双读写或第二账本；已有调查保留其时点，不作为新空安装的实施前置。

用户随后明确授权：先与#1078原公共模型owner确认DeepSeek准入接口，达到实施准入后沿原工作区/PR #1075继续离线实现、合成回归和CI。原owner已接受可比较窗口值、规范UTC字符串和零值JSON省略，并明确由#1075原唯一Writer在原公共owner中有界补齐本文件的增量，保留已批准多模态逻辑。其指定依赖4b6c03caf49e51e2718ea344968d78ca115f3f53的必需CI已成功；实施准入与真实资格严格分开。现有 `task-processor-unified-20261010` 不清空、不更新、不重建；不改凭据、不发布配置、不执行真实采集/AI/付费，不合并/关单。未来若沿用该实例名称实施新空安装，须先明确实际安装目标和旧资源处置清单并单独取得操作授权。“旧数据无需承接”不等于立即删除卷或数据库。

用户结果：Local Account Center有权成员从智能市场启用原研究/选品操作，复用同企业共享text模型；原工作页核资料、权限和预算后发起有界分析，结果仍需人工决定。预算每项渠道成本≤¥0.30、消费≤20点，两项各一次总≤¥0.60/40点；每项最多两次模型调用，每调用≤32000输入/4000输出Token，原更小字节、步骤、运行期限和费用边界继续收紧。

Must：准确原型号、组织级凭据、厂商原价、渠道成本分离、完整计费用量、时段适用可证明；一份共享选择和一条原Invocation→资源结算链；没有可信证据不派发/不把缺失当零。原同键/事务、权限/租户、最后发送检查和UNKNOWN责任保持。

Out of Scope：旧数据演进/读取兼容、真实安装/删除/切换、密钥复制或发现、模型自动发现、自动抓价/节假日/调度平台、第二预算或收费owner、视觉/搜索/显式缓存创建、模型质量验收、自动采购/刊登。Accepted Risks：未新增风险接受；缺计量或不确定价目不降为可调用。

本设计没有新页面、菜单、IA或交互步骤。现有产品入口沿[当前UI Authority](../product/final-ui-ia-authority.md)和[两个业务智能体设计](../product/market-research-selection-centers-v1.md)，Figma Product Authority继承当前文件所引用依据；不对Figma作新变更。未开放原因用原设置页的业务文案说明，不引入新的业务操作流程。

## 2. 依据与问题定位

公共依据是[企业共享模型冻结合同](enterprise-shared-model-configuration.md)，稳定依赖为原owner明确指定的4b6c03caf49e51e2718ea344968d78ca115f3f53（含原341278d5有限安全基础和本轮C2-2）。已直接核对其CI41662214683和Required Gate126015144739成功；公共owner已结束该批写入。仅在本工作区组合该依赖，保留其多模态逻辑，不修改其工作区，不据此推定完整三类或真实DeepSeek已开放。

已核实的现有记录只作新设计的元数据候选：LAC组织级 `local-content-route-830-20261020`，原型号 `deepseek-v4.1-flash`、DASHSCOPE/OpenAI兼容、固定非思考策略、45秒凭据超时、北京workspace端点格式及准确credentialVersion/endpoint摘要。Key存在不证明远端有效。新空目标不能借旧per-purpose current取得执行权；若该目标没有当前owner合法供给的本企业凭据，保持未配置，不由本任务复制Key或导入旧记录。

| 实际接缝 | 当前实现 | 本设计的最小处理 |
| --- | --- | --- |
| 厂商来源 | `officialPriceHost`仅openai/google/qwen/cfets，没有deepseek | 原价目owner增加准确deepseek官方域名和型号对应依据，保留权限/CAS |
| 计费时段 | 价目已有连续有效期，tariff只有单组rates；百炼和厂商忙闲日历不同 | 原价目owner冻结有界的显式时间窗口集合，原公式做上界/实际选窗，不建周期调度器 |
| 完整usage | 原openai-text-v1要求缓存分项，通用示例不能代签本候选 | 原受控适配补准确DeepSeek响应资格，不更换计费owner或假定缓存为零 |
| 同一凭据复用 | anchors与显式bindings可能产生同(scope,id)重复，而既有内容候选恰有同clientName | 原装配对准确同键作一次受限准入信息补齐；不复制/重命名凭据绕过 |
| 旧53列 | 当前新owner为52列；现场多3个独立点价列、少2个官方价/用量JSON列 | **退出本次处理**。只按新空合同安装，不把旧53列作为升级输入 |

来源：原[价目](../../internal/aicapability/routeconfig/platform_prices.go)、[共享目录](../../internal/integration/aicapability/routeconfig/shared_catalog.go)、[运行装配](../../internal/app/runtime/currentapplication/model_route_pricing.go)、[原计费公式](../../internal/aicapability/official_model_pricing.go)、[原usage mapper](../../internal/integration/aicapability/einomodel/official_usage.go)。

## 3. owner与接线路径

| 事实/行为 | 唯一owner及复用 |
| --- | --- |
| Key及组织级记录 | 原openai credential owner；共享目录只用去密投影，发送前原resolver取得Key |
| 准入描述和BoundEvidence | 原routeconfig Catalog及受控einomodel策略；部署描述不是current或价格 |
| 厂商价目、显式窗口、CNY恒等FX | 原PlatformPricesService/revision/current/CAS；原平台维护/发布权限 |
| 企业三类选择 | 原SharedModelService；三类全部ready才原子发布，不部分发布text |
| 研究/选品输入、运行与恢复 | 原各用途Prepare/Run/Invocation；各自prompt/schema、证据、read_only、成员授权保持 |
| 预留、结算、成员月限、债务 | 原orgresource/原model-invocation资源适配器；无新余额或收费事务 |

contract → 原PlatformPrices/Catalog → SharedTextReader → 原managed resolver/受控SDK → 原两调用业务Run → 原Invocation完整事实 → 原orgresource预留/结算/RecoverDue。

领域代码不导入provider SDK或凭据仓储；原application只装配。原Invocation数据库存官方价和用量JSON；资源数据库只按原owner proof结算。没有跨DB事务或新协调器，源事实先durable，原恢复入口读同一事实、同一finalize key。原后端授权、有效企业和成员作用域决定访问，页面选择/角色名/Key存在均不授予执行权。

## 4. 准确绑定与受控协议

### 4.1 元数据和用途不变量

准入必须同时绑定：org、clientName、准确credentialVersion、endpoint摘要及受控北京workspace格式、DASHSCOPE传输provider、`deepseek`模型厂商、原型号、OpenAI Chat协议、固定非思考策略、完整usage与时段证据、原币成本上界、各用途prompt/schema/字节/Token/deadline。

`ReferenceVersion`来自上述非秘密准入描述的稳定版本；`BoundEvidenceVersion`来自准确协议、计量、时段及上界依据的核准版本。二者由原owner生成/校验，不用credentialVersion、旧内容revision或客户端任意字符串顶替。Key轮换/端点变动/证据或边界变化使原资格失效；新报价重新确认，已派发仍归原invocation。新空目标没有这些证据时不得自动seed描述、价目或企业选择。

同一clientName必须覆盖该企业全部已开放文本scope。原装配保留每个anchor的业务职责；显式Definition若与anchor的(scope,id)相同，只允许核准的provider/credential/endpoint/来源/价目和证据元数据补齐，profile必须通过原prompt/schema/币种相同及所有上限不放大的检查，最终只交给NewCatalog一条定义。输入bindings内同键多次仍INVALID；不同clientName按原候选路径处理。缺任一已开放用途返回OPEN_CONSUMER_MISSING，不删除原内容等用途来规避检查。

### 4.2 请求和响应边界

保留 `dashscope-deepseek-v4.1-flash-nonthinking-v1`：固定 `enable_thinking=false`、纯文本、单结果、非流式、无模型工具/联网搜索/视觉/音频/显式缓存创建。SDK实际wire须证明输出硬上限与原byte/deadline；不因模型公开大上下文而增加原上限。原max_tokens仅在准确官方协议仍支持时使用；改参数须原策略版本和离线wire证据，不能静默换协议或模型。

响应model必须与核准的准确身份一致；仅有官方可追溯别名对应时可列入该策略固定allowlist，不用前缀/模糊匹配或以1017代替原型号。原响应大小、重复字段、计数范围、超时、取消与一次发送保护继续有效。外部超时不证明provider没有执行，绝不自动重发。

完整计费必须保留字段存在性：P=prompt_tokens、O=completion_tokens、T=total_tokens及C=prompt_tokens_details.cached_tokens都存在，非负、T=P+O且0≤C≤P；按P−C、C、O计费。reasoning若存在是O子集，不重复加算；未知、矛盾或额外收费维度不能由SDK默认值抹掉。原openai-text-v1保留；准确DeepSeek envelope校验在原受控适配内完成，不另建mapper服务或收费版本假名。

本策略显式核计usage中的收费字段：音频/图像、搜索/工具次数、显式缓存创建等不在已准入纯文本项内时，不得仅因P/C/O完整就称完整；未知分项保持usage未知。已核准且不收费的元数据可按准确响应合同接收。缺缓存字段/null不当0；已发生调用但输出无效时，完整可信用量仍按原规则结算，输出失败不免除已发生消费。

### 4.3 真实资格与离线证明分开

[百炼Chat参考](https://help.aliyun.com/zh/model-studio/qwen-api-via-openai-chat-completions)和[上下文缓存](https://help.aliyun.com/zh/model-studio/context-cache)证明通用语义；[DeepSeek调用参考](https://help.aliyun.com/zh/model-studio/deepseek-api)中的V4 Pro示例缺缓存分项，不证明本型号总能返回完整C。因此真实绑定资格继续NOT_VERIFIED。

BoundEvidence必须给出准确原型号/地域/策略/非流式协议下：缓存命中及未命中的分项存在性、全部收费维度、准确响应model、可证明的计价时点与字段。来源可为官方准确合同及已有合法去敏证据；合成HTTP只能验证代码如何处理这些形状，不能签发真实provider资格。本阶段不执行模型或读取历史原文；缺真实资格不阻止实现拒绝路径，但阻止Ready=true及真实派发。

## 5. 厂商原价、窗口冻结与预算

### 5.1 原价来源与分离

原价owner新增 `provider=deepseek` 时只接受HTTPS准确官方host：`api-docs.deepseek.com`、`deepseek.com`、`www.deepseek.com`；继续拒绝userinfo、fragment、非官方域名，不以qwen厂商名发布DeepSeek价目。准确型号对应依据将百炼原 `deepseek-v4.1-flash` 对应厂商原版DeepSeek-V4.1-Flash，不自动改1017；channel/region/service必须与受控绑定一致（百炼北京、standard-sync）。

公开证据：[2026-09-10发布及原价图](https://api-docs.deepseek.com/zh-cn/news/news260910/)、[官方更新日志](https://api-docs.deepseek.com/zh-cn/updates/)、[当前厂商价表](https://api-docs.deepseek.com/zh-cn/quick_start/pricing/)、[百炼原型号详情](https://help.aliyun.com/zh/model-studio/deepseek-v4-1-flash)。原版价图已实际查看，1017更新明确价格未变。人民币/百万Token的厂商闲/忙：非缓存输入1/2、缓存0.02/0.04、输出4/8；百炼北京缓存为0.1/0.2，作为独立渠道成本。厂商工作日非中国法定节假日09–12/14–18为忙，百炼每日08–22为忙，均东八区。价格来源日期和真实发布/有效期仍由原平台权限核准，不把公开调查当已发布。

### 5.2 复用显式窗口，不新增周期平台

原价目输入已有EffectiveFrom/Until，同一price ID允许非重叠连续窗口；复用该有界列表表达逐个已核准的UTC区间。原最多32个价格条目的上限不放大；不新增自动日历、调度器或假日抓取。不把百炼的忙闲标记传给厂商收费，也不将忙时价全天用于实际结算。窗口缺失、交叠、来源/型号/渠道/地域/服务不匹配均不准入。

DeepSeek的时段价在原OfficialModelTariff内增加可选的**冻结文字窗口组**，仍序列化到原official_tariff_json，无新表/价格current/余额。该组含原price ID、同一价目revision、各条目版本/原币rates/UTC区间、核准的计价时点策略版本和原来源；所有窗口属同一准确型号、渠道、地域、服务、币种及usage mapper。其版本由原owner对有序条目版本和时点策略生成；不是调用者自填。普通非时段tariff保持原路径，不能默认给未证明的型号开启时段计费。

具体拟议窄合同是 `OfficialModelTariff.TextPriceWindows`（可选）：schemaVersion、原price ID/revision、groupVersion、clockPolicyVersion、boundEvidenceVersion及≤原32条上限的windows；每项只含原entryVersion、from/until及原OfficialModelRates，按UTC开始时间排序，严格非重叠。model/channel/currency从外层固定，不能窗口间改变。外层Rates必须恰为组内逐分项最大值，组版本绑定全部有序条目版本/区间/rates/时点证据，原priceVersion采用该owner生成的组版本。不能接受客户端计算的分组或上界。组只从原已发布PlatformPricesSnapshot派生，未另存一份平台价格事实。

本型号已公布峰/闲两档；同组必须有这两档完整、可核验的原价上界及合法窗口证据。不能仅提交当前闲时窗便将外层Rates/R降到闲时价，缺任何可能适用价档的上界证据即不Ready；也不能捏造高峰日期或借别的型号填满证据。R按完整峰/闲上界，A仍只按本次可信时点落入的唯一合法窗口。按实际序列化输入收紧R的原行为保持，不因本澄清强迫总按32k预留。

原 `BillableModelUsage` 增加可选 `BillingTimeUTC` 和 `BillingTimePolicyVersion`，跟随原billable_usage_json持久化；窗口tariff要求二者实际存在、范围合法且策略版本与组一致，非时段tariff不从它们推定资格。解析准确envelope仍由原einomodel拥有；最终原事实/proof涵盖这两项，SDK裸计数不能自行补时点。以上为公共owner确认前的提议字段名，不建立正式schema或提交生产类型。

公共owner指出现有tariff/Profile参与直接值比较、冻结快照与当前配置核对；不得把窗口slice或pointer直接存入这些值，也不把全仓比较替换为reflect.DeepEqual。拟议 `TextPriceWindows` 为原aicapability owner拥有的可比较值结构：私有元数据、有效计数和 `[32]` 私有窗口数组，所有成员只含可比较的标量/原OfficialModelRates。原构造器及JSON解码先验证、规范排序、归一UTC时点并清零未用数组尾部，复制调用方输入；不保留外部slice、map、pointer或time.Time位置/单调时钟身份。相同有序窗口事实经独立构建或持久化往返必须 `==`。不同原entry/group/evidence版本仍不相等，不以值相似吞掉事实差异。

新增窗口字段拟用 `json:"textPriceWindows,omitzero"`；复用项目Go 1.26的标准encoding/json省略真正零值。该值只在JSON编解码的临时DTO中使用有界slice，自定义JSON方法仍输出上述窗口组object和仅有效windows，不输出固定数组尾部、字符串包装或count内部字段。解码复用原严格JSON边界拒绝重复/未知字段、null、空组、超32项及非法窗口，再交原owner全部证据/上界校验；可比较表示自身不授予价格或模型资格。

`BillingTimeUTC` 拟用规范UTC RFC3339Nano字符串，缺失为空字符串并 `omitempty`；`BillingTimePolicyVersion` 同样以空字符串表达缺失并省略。原envelope解析用临时presence判断，拒绝缺失/null/不合法时点，只有核准的时点来源和策略可以生成该事实；不从时间字符串存在推定资格，也不丢失原协议时间精度。窗口内部UTC区间沿同一标准规范表示，避免同一时点的时区/格式别名造成值身份差异。不适用时这三个新增JSON字段必须全部真实省略，不能用null/空字符串/零时间改变普通非时段provider的当前canonical JSON和proof。实现测试仍须逐字比较真实原非时段编码/输入指纹，不只比较显示值。

2026-11-03受限非正式POC仅证明上述Go值/JSON表示可行：外层profile保持可比较、零窗口/零时点字段与未新增字段前JSON逐字一致、独立排序构建及JSON往返后相等、调用者slice变更不改变已构建值、JSON只有有效窗口且拒绝显式null。POC不是生产实现，不证明真实tariff/proof、严格重复字段解析、计价公式或provider准入；这些仍为后续原owner实现测试。原公共owner已确认本表示，并将下述有界增量交原#1075 Writer在原模块实现；不建立第二owner。

窗口组连同完整profile必须满足原快照/输入hash/序列化byte上限（包括原8192字节约束及用途更小上限）；Catalog和Prepare使用同一原校验，超限不Ready，不靠截断窗口/删证明或提高限制。原32条是平台价目总上限，不是承诺每个操作都能携带32个窗口。

原ReservationMicropoints对整个已冻结组的各收费分项取保守最大单价，再沿原公式ceil一次；ActualMicropoints只用**可证明的provider计价时点**选择组内唯一窗口并floor一次。二者使用同一厂商来源和冻结组，非缓存/缓存仍互斥。派发前已知的冻结组/时点策略版本进入原profile快照、Claim和InputHash；响应后才可得的实际BillingTime及其依据版本只进入原终态billable_usage_json/settlement proof。不得改写已创建Claim/InputHash、按响应重做Reserve或重建调用身份；原modelPointFingerprint仍清除终态BillableUsage，不新增第二份收费事实。

具体时点字段只有在官方证明其决定该API请求的收费时段时才能核准，例如响应created不能仅凭字段名字当计价时点。准确时点来源/转换仍缺证据时，Catalog返回TIME_BASIS_NOT_ADMITTED；成功响应没有可信时点、时点在冻结组外或有两个匹配窗口时保留原预留/UNKNOWN，不用本机当前时间、完成时间或后来的价目猜测。此条件不要求本阶段真实调用。

Prepare核完整运行/每调用deadline有已发布窗口覆盖；每次最后发送仍fresh检查原共享类别、资格、凭据、冻结组版本、权限及剩余覆盖。同一已冻结组内跨忙闲边界不换配置/模型或重新报价，按每条调用的可信时点选窗；组变化/到期或覆盖不足时，未发送的下一调用确定拒绝。已经发生的调用保留原组、原最大预留和原key；不为补齐第二次调用重跑第一次。新窗口发布不回写在途快照或历史账单。

组内单价上界还须包含官方声明的本次派发有效期最大适用原价；无法证明该上界时不派发。渠道成本仍沿原profile单独按北京忙时上界控制，只报告可证明的成本估计，不叠加成用户收费。实际资源结算继续要求0≤A≤R；A>R保留原待核实，不截断/补扣/伪造债务。

### 5.3 原预算不变

最多两调用、每调用32000输入/4000输出，无缓存忙时上界：`2×(32000×2+4000×8)/1000000=¥0.192=19.2点`；闲时实际无缓存算例为9.6点，R仍按完整窗口组上界。百炼北京最坏成本同为¥0.192，低于每项¥0.30；两项各一次最坏算例¥0.384/38.4点低于¥0.60/40点。实际输入被原执行器收紧时沿原规则减少预留，不能要求总按32k预留。CNY继续原cny-identity-v1，不新增FX条目或独立点价。

## 6. 事务、幂等与失败结果

配置发布继续原CAS/immutable revision/原request key，平台价目发布仍是原权限，企业选择权限不能维护价格。三类选择一次事务发布；DeepSeek text合格也不解除Flare或视觉消费者缺项，不部分发布text或造pending current。新空安装不会自动从旧内容current/独立点价推导三类选择。

模型派发、原Invocation claim、资源预留/结算和原RecoverDue不变；窗口组和usage时点只是该owner的定价/计量证明增量。同key不同输入/组版本冲突，响应或ACK丢失只查原key。源终态先durable，原资源事务按原finalize key/proof核实；不跨DB长事务，不换模型、Key、渠道或重新发送恢复。成员月限和月份仍绑定原reservation/StartedAt，窗口跨界不换月、不二次预留同一调用。

| 情况 | 处理与持久化边界 |
| --- | --- |
| 凭据/资格/价格/窗口/权限在发送前失效 | 原最终门禁拒绝未发送调用；按原合法no-send proof释放其预留 |
| 第一次成功、第二次发送前失效 | 保留第一次完整用量/原结算，第二次不发送；Run明确未完成，不重跑第一次 |
| 取消、网络超时、非完整响应 | 原Invocation/预留/UNKNOWN，原key恢复；本地超时不作为no-send证明 |
| usage完整、输出无效 | 记录原usage_observed_failed并按真实用量结算，不发修复调用 |
| 缺C/计价时点、额外收费项、分项矛盾 | 保留原预留，不填零/当全价或换新价重算 |
| 资源提交成功但ACK丢失、重启/并发找回 | 原事务/回执唯一性和RecoverDue，结果只结算一次 |

## 7. 全新安装与最小开发范围

原系统基线继续有效。新目标只沿原schema installer安装当前52列Invocation、原配置/资源ai-micropoint-v1及消费者合同；运行时只Verify不DDL。旧53列既不升级、不改名复制，也不在新代码保留旧三个独立点价字段。原历史/余额/在途事实不输入新系统，不新建历史适配reader或迁移registry。

若未来实际复用同一实例名称，先另行确定命名数据库/目录/卷/配置的处置清单；不得直接对现有卷执行初始化、覆盖或通用prune。新空目标的凭据只能沿原owner合法配置；不读出/复制旧Key、不假定旧组织current存在。当前设计不决定删除哪些资源，也不授权新建实例、备份恢复或清理。

| 原owner | 本轮有界离线实现 |
| --- | --- |
| routeconfig平台价目 | DeepSeek官方来源/准确身份、冻结有界窗口组及稳定版本，沿原CAS/权限/有效期上限 |
| aicapability原计费类型/公式 | 可选窗口组、保守R、可信时点选窗A及完整proof，不增加SQL列或第二账本 |
| einomodel受控适配 | 固定策略wire、准确model、完整计数/额外收费维度及计价时点提取；缺证据保持未准入 |
| SharedCatalog/运行装配 | 全部已开放scope兼容、同(scope,id)受限补齐、窗口/证据/上界资格、原reader注入 |
| 原研究/选品消费者 | 仅消费原profile/快照的新证明，保留两调用、各自prompt/schema/权限/read_only和原key；不另写模型或费率逻辑 |

公共事实和类型继续归原owner。#1078原owner确认本批不实现DeepSeek时间窗口，允许#1075原唯一Writer在4b6c03caf稳定依赖后，有界修改原tariff/usage、平台价目、共享目录、原运行装配和受控mapper及对应回归；不复制公共实现、不改其工作区、不与其并行写入重叠文件。用户已允许接口确认和IMPLEMENTATION_READY之后连续离线实施，无须逐步重复请示；配置/安装/真实运行权限仍分开，接口认可不授予模型Ready。

## 8. 必要验证与准入状态

只复用既有domain、guard/SDK、SQL事务、HTTP及任务专属PostgreSQL测试；不建runner或专项验收平台。正式实现后按TDD捕获目标问题并做下列有界验证：

1. 官方host/model/channel/region/service/CNY恒等FX；重复/交叠/空窗、改价版本和不同忙闲日历；最大列表与原大小上限。
2. 同anchor/id准确一次补齐，输入重复拒绝，prompt/schema/上限/期限不能放大；原已开放scope不能缺失。
3. 固定非思考、非流式无工具的实际wire和原hard limits；model不符、字段缺失/null/重复/负数/越界/额外收费维度拒绝。
4. 同冻结窗口组下R按最大单价、A按可信时点唯一窗口；切边界/组外/缺时点/时点依据不合格不收费，跨两调用不重发或重复结算。
5. 原账本/成员月限/debt-first/原月份、零实扣、并发同key和ACK丢失恢复；缺完整事实保留R。测试使用明确合成价格/时点，不冒称真实百炼资格。
6. 新空owner安装和只Verify启动；旧53列和旧/混合单位目标继续拒绝。没有旧业务数据读取、迁移或自动seed。

R1独立检查绑定f1eb0f02281395e6115825d87fadcb017bb1205a，未发现AGENTS设计BLOCKER，仅DESIGN_REVIEW_PASS。三项已分类为IMPLEMENTATION_TEST/合同澄清：R1-1派发前Claim/InputHash与响应后计价时点分离；R1-2完整峰/闲R上界；R1-3可选字段省略及非时段canonical不变。对应增量300285db0a5b4313cba0450e12dd020ac4fe03cc（设计blob cbf62055a7d7a4a0cd3957afbe2b1a9cdb46b856）已由原同一Reviewer复核PASS，三项在设计层闭合、无新增BLOCKER。正式实现仍须逐项必要测试；未跑测试不写PASS。

可比较值/JSON增量已由原同一Reviewer复核PASS（设计blob 2c1aedf94cf4db4ccd1ded19a8cb8c82719e6461），无新增设计BLOCKER；真实类型比较、非时段canonical/proof逐字一致、严格解析及原byte上限保留IMPLEMENTATION_TEST。原公共owner确认接口与文件分工，指定依赖必需CI成功后，本设计达到IMPLEMENTATION_READY，仅允许上述离线实现/合成回归。准确真实usage/计价时点/上界证据仍NOT_VERIFIED；新空安装/真实配置发布/采集/模型/付费/用户验收NOT_RUN。真实资格缺项始终阻止Catalog Ready和实际开放。历史数据演进方案CANCELLED，不记为PASS。
