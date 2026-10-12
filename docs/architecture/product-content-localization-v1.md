# 商品内容完善与本地化智能体 v1

Status: DESIGNING / NOT_READY。执行 Issue [#661](https://github.com/qq550723504/task-processor/issues/661)，父 Roadmap #137。正式生产代码须在必要独立架构检查明确 IMPLEMENTATION_READY 后修改。

## 1. 产品结果与依据

当前企业成员从准确已保存商品选择目标语言与待处理文本字段，得到可读的原值/建议/证据/未解决项，按四种权限采用到同一标准商品版本。首批字段为 title、description、selling_points、seo_keywords；价格、库存、属性、规格、SKU、图片、品牌及source事实保持不变。资料缺失如实报告，不创建来源、销量、利润或官方已适配状态。

用户于2026-10-12批准：#656运行路径完成交接后逐个实现本智能体，再做平台资料、POD、选品、经营诊断；full仅已授权企业业务工具，外部发布/改价/下架、删除和提高额度另授权。模型/支付/SHEIN/SDS后准备。缺配置只阻相关真实执行，NOT_RUN，不伪造可用。

UI依据 [当前Figma authority](../product/final-ui-ia-authority.md)，file tg48P46SSXl6TBy9lZwg63 / 当前非归档31:463。2026-10-12已核对4703:429我的智能体（商品优化的标题/卖点/描述能力）、4703:892能力配置、4722:429执行确认的高保真context及截图。用户本批结果补语言/字段选择及多字段原值/建议，沿原布局和组件；不新建顶层商品中心、独立平台工作台。原型样例数量/启用/模型状态不是运行事实。

权限依赖为已审#657/7058ea07eff673f7018f904881251f9befbbf3b1，已由原owner完成保留实例接入/有界页面核对，仍未合并。原用户新授权的权限/详情纯UI优化由原owner维护#657；本批一个Writer复用附属worktree的新主分支codex/product-content-localization，stacked依赖#657。原#132/#47质量对照历史不被冒充完成，也不恢复为用户新顺序的统一前置。

本次不做：其它四Agent、任意AgentDefinition/Tool/脚本、自定义prompt、官方资料适配/发布、模型/支付/provider开通、数据迁移或清理、新状态机/审批/IAM/账本/runner。代码交付/隔离测试、真实provider、保留实例、用户/生产验收分别记录。

## 2. 当前能力与必须补齐的边界

Catalog.ProductSnapshot已经拥有四字段（internal/product/catalog/snapshot.go），Enrichment的Candidate/FieldChange/Policy是纯计算。Review当前仅允许title，记录/原子Apply及ResolveAppliedSource只证明title-only delta；不能直接令通用Agent改description，或把多字段修改投影成标题批准。

复用当前Review事实与事务，新增明确content policy/typed payload及准确lineage；保持合法现有标题产品路径，不加旧Service wrapper/fallback/双读双写或第二审核owner。四字段用户结果保持一个主要PR，内部切片不机械分PR。

## 3. Owner 与接线

| 合同/实现 | 注入和消费者 | 唯一职责 |
| --- | --- | --- |
| agent的已有Request/ModelInput与agentconfig配置快照 | 原Eino及RunDB内当前Agent PostgreSQL Store | 一个运行/checkpoint/CAS，固定Definition，bounded repair；不是第二任务/重试owner |
| 新registered product.content.agent/v1.0.0与固定content-config-v1参数 | 现企业市场/启用/模板/模式；新内容配置/确认组件 | code-owned注册及受限字段/语言默认值，不授权限，不开放用户Tool清单 |
| internal/product/contentagent的窄消费者ports | app仅注入当前acquisition/collection source、配置、runtime、Review与model ports | 当前保存商品context和真实权限→冻结调用→原run/Review输出；不引入GORM/provider SDK/平台client |
| Product Enrichment / Review content policy、typed补丁 | ProductDB内现Review Store/UoW / Catalog.Publisher及事务内source reader | 证据/字段重验、人工决定/策略采用、唯一原子版本和回执；不与RunDB跨池共用事务 |
| Review真实applied publication lineage | 当前source/Asset/草稿消费者 | 从原SRC版本逐项证明真实补丁，不制造新来源或把content当title-only |
| 现AI Capability governed文本port / Commercial | 现provider-neutral integration与原route/dispatch/settlement ledger | 可信路由、token/cost上界、审计、计量、UNKNOWN原owner |

首条完整操作路径：已保存acquisition/my-data商品→选择内容Agent、字段、目标语言及模板→原配置/模式准备及计划/预算确认→Eino read/compute/propose→Review保存建议→default人工接受/编辑并显式采用，或full原始合法候选策略采用→Catalog新版本/准确回执→原建议页/商品/历史读回。已有真实BusinessTask消费者可投影此运行；不得伪造task ID或进度代替绑定。

配置缺失时仍能读取/配置实际注册能力，生成入口显示真实阻碍；生产绝不使用fixture回复或本地fake model兜底。隔离开发路径可注入受控模型，证据明确注明。

## 4. 请求、配置和模型

目标语言是用户明确数据选择，不从平台/Store/模型猜测。使用现golang.org/x/text/language解析规范BCP47 tag，拒绝und/无效/非规范或超过既有ID边界的值；不自建语言解析器。字段以固定枚举/规范位集合表达，仅四字段、非空、无重复。两者为可比较的typed ContentSelection（零值omitzero），进入Request、ModelInput、configuration/context digest、run fingerprint及Review语义；原title请求不因新增空字段改变历史序列化/摘要。

现agent.Validator.Validate只收到Binding/PolicyVersion，无法区分同商品并发运行的不同语言/字段。将当前port改为传入完整冻结Request及原candidate/history，Eino从当前State直接提供；所有现title消费者按同一port机械适配，title请求只允许ContentSelection零值，content请求必须准确非零选择。不得通过全局变量、当前模板head或binding猜测原运行选择。模型准备、配置快照核实和Review intake也逐项比较冻结selection。

模板Content参数独立于原title/image类型，默认目标语言/字段只作本次可见默认值；执行确认重新冻结实际选择。未识别agent/schema/参数组合fail closed。现ModeSubjectRegisteredAgent、当前definition version、activation/epoch/模板权限、ExpectedModeRevision、ModeSnapshotRef复用，不创建另一份模式事实。

当前agent_configuration.template_revisions只接受title-config-v1/image-config-v1，只有image_parameters载体。因此在同一RunDB配置owner明确添加nullable content_parameters bytea（新内容参数JSON，最大4096字节），以及content-config-v1的精确agent/schema/参数CHECK。旧title/image行必须保持content_parameters IS NULL，新增content行必须agent_id='product.content.agent'、image_parameters IS NULL、目标platform='product'或现有合法platform；不把内容参数塞入image列或知识库ID。新内容执行可以使用product目标，不要求绑定店铺或先配置平台。

新安装schema与显式有界UpdateContentConfigurationSchema分开；更新在installer凭据、原非登录配置owner下的一次DDL事务内核对已知约束/列形状，添加无default nullable列并原子替换已知约束。已有全部行仍按原合同合法，不做数据UPDATE、回填、ID映射或删除；外来/不匹配schema拒绝。重复安装核对已达目标后no-op。正常构造/serving只verify，未更新时内容能力明确unavailable，原标题/图片路径可继续按其原合同运行。运行role无DDL权限；原SELECT/INSERT载体列所需权限按现GrantRuntime精确授予，不改变fact owner。保留实例更新由另获授权的安装owner执行，本Writer仅检查隔离数据库。

新内容model使用同一governed文本执行port与provider-neutral路由。prompt/action schema、Definition、policy version固定；选中字段/语言/原商品/来源/可选已授权企业Knowledge均是data，不是指令或授权。原工具调用仅允许当前code-owned的canonical/source-evidence等只读/纯计算工具。候选只含既有FieldChange；复用严格JSON、大小/deadline/步骤/最多两次repair、实际observed evidence校验。不是新LangGraph/Agents SDK/Temporal运行时。

原source-evidence工具完整来源事实仍由SRC/Catalog授权读取；针对内容需要的文本做有界、版本化模型视图，不能丢字段后假装完整证据。超界返回明确TooLarge/不可用，不默默截断后给采用许可。Knowledge可帮助表达，不能覆盖商品/source事实或官方规则。

## 5. 多字段Review合同

新policy product-content-review-v1；沿现Review proposal/operation ledger和状态pending→accepted/rejected→applied。不增加表/状态机。Record新增optional typed Content：目标语言、准确字段集合、各字段before/after、人工决定历史及content策略proof；现title Record继续原policy/serializer。旧内容缺失不会被当作新授权或自动升级。

字段：
- title/description为规范UTF-8文本，title沿原ValidateTitle；description可换行，拒绝控制字/首尾空白等不可展示输入，按现存储预算有界。
- selling_points/seo_keywords使用严格、规范JSON string array作为既有FieldChange.Value；非空、元素非空/规范/有界，不接收对象、嵌套path、数字或任意JSON。
- 不支持清空/删除字段；新增内容补丁只处理显式选中字段。字段集/locale来自冻结请求，不由模型或Decision扩展。
- source/evidence逐次重新读取，Candidate引用只能是该运行实际成功观察并与原SRC版本绑定的canonical evidence；不采信模型自报置信度/质量为事实或批准。
- 同一候选完整选中集合要么合法，要么显示未解决项、禁止采用。不自动拆分标题先写、描述后写；一次采用对同一clone base应用所有所选字段，其余完整snapshot必须逐字段相等。
- 编辑仅接受该字段集的文本值/array；不允许改语言、范围、证据、版本、scope/metadata。人工行为真实记录；原候选及proof保持不可变。
- 复用当前64KiB record/response/storage预算和未来accept/Apply空间预留；超界创建/编辑/采用在写前拒绝，既有事实不变，不扩大DDL限制或静默裁剪。

保存content建议及每次原回执读回必须验证typed policy/payload的组合，不能让旧title HTTP返回空标题假成功。新的wire schema包含明确content section，原title schema/字段不被松绑。collection/task/read consumers按真实policy展示原值、建议、来源、冻结mode/locale、是否需人工及原采用回执，不制造新报告owner。

## 6. 四权限与采用

| 模式 | 新内容行为 |
| --- | --- |
| default | 生成/保存建议；真实用户确认/编辑后显式采用 |
| read_only | 读取、分析、保存建议；Decision/Apply/策略采用均拒绝，不能改Product/草稿/资产 |
| auto_approve | 仅title-only且通过原程序校验、无unresolved的候选可按策略采用；包含description/卖点/SEO字段保留完整人工审核，不虚构“AI已审批” |
| full_access | 当前有效业务授权及完整合法原始候选可按策略采用；如实标记未经逐项人工审阅，不表示译文/语义正确 |

当前权限不得由模式扩大。read/use/source/review/write、当前企业/member、原activation/定义/冻结配置与当前mode修订，在每个新效果前按#656原路径重验。新effect在mode变更/成员更换/撤权后拒绝；旧请求不升级、不换键重调。发布/改价/下架、删除、增加额度继续另授权。模型quote/计划/预算确认、消费上界/实际route与UNKNOWN不被full绕过。

content策略proof使用新action version product.content.policy-adopt-v1，绑定原run/request/context/source/product publication、Config/Mode refs、成员、locale/字段、准确完整candidate digest及原政策版本。只接受未经人工编辑的原完整候选。复用Review原事务/receipt，不伪造human accept/edit History；实际用户编辑走人工合同。确定性校验是结构/范围/证据/版本校验，不宣称能证明语义/语言正确；沿已批准full未逐项人工审阅风险及默认人工路径，不加paid审批LLM。

## 7. 事务、lineage、失败与恢复

沿ProductDB内Review原UoW（当前buildProductAgentApplication在其productDB/ReviewDB池上组装reviewstore、catalogstore及source reader）：先原operation replay→精确record锁/状态/版本→重新当前授权与原SRC/完整base绑定→纯patch验证→Catalog CAS publication→record及operation receipt同事务提交。RunDB只保存既有Agent运行/配置与模式；不移动Product事实、不跨池伪称原子事务，不新增跨库Saga。操作身份仍org+actor+原Idempotency-Key；同键不同内容/语言/字段/version conflict。主键、published identity及当前CAS保留，不用客户端key替代服务端CommandID。

response丢失/重复/重启读准确原回执；已提交回执只做核实，不再因当前mode head变化生成新效果。仍保留该操作需要的当前读取/source授权和准确归属；新效果需要当前mode/config。未提交故障回滚，不返回成功；UNKNOWN模型/账本/工具跟原owner恢复，不换provider或request。每一步沿原deadline、记录/响应大小及幂等界限，不新增跨库恢复承诺。

ResolveAppliedSource在原Review owner扩展为证明两种合法policy的真实patch lineage：每跳核对同org/actor、record/state/receipt/source evidence/版本，按该policy在完整base上重演准确补丁，并要求完整snapshot等同该published版本。继续有限depth、权限与原SRC原publication核验；假“仅标题变更”、额外数值/属性/图片改动或无真实receipt均拒绝。现title/image/草稿读取新content版本时消费此真实generic publication lineage，不能复制Catalog/SRC事实或伪造title批准。

本批仅§4现配置owner的nullable列和精确CHECK显式DDL；Review/Catalog存储及64KiB预算不改变。不在schema初始化/运行构造器中隐式更新，也不做旧数据回填。共享实例/命名volumes/原配置只由另获授权的运行owner操作，本Writer不访问或改真实业务资料。

## 8. Threat Model / 验证 / 交付

Must：当前登录与tenant/member/资源授权、固定字段/schema/语言绑定、真实已观察来源、准确base/CAS、单一Review/Catalog owner、只读零mutation、真实人工或native策略proof、准确原回执/UNKNOWN、数值等不可编辑事实完整保留。全量语言质量/事实语义并非deterministic truth证明；新范围不添加AI自签许可或改变既有accepted风险。

TDD针对实际变化：四字段/数组及禁用字段、locale/字段指纹，default/readonly/auto/full矩阵，多字段Apply完整clone与当前lineage，跨企业/撤权/更换member/模式修订、同键重放及故障回滚、预算/UNKNOWN不重复dispatch。复用当前owner测试和隔离PostgreSQL，受控模型注入必须标记fixture；不建runner/验收平台，不做与本改动无关故障矩阵。

检查真实完整路径的独立最终差异及必要增量一次；最终准确HEAD CI，不冒充未变代码在新SHA直接重跑。UI复用现Card/Dialog/表单/tokens；正常配置→已保存商品→确认→建议→人工或策略采用→刷新版本和回执。首次交接给运行入口/正常启动命令、数据保存方式、可用/不可用原因；未配置的真实模型、保留实例接入、用户/生产验收分别NOT_RUN，待相应用户/独立验证确认。

Legacy decision: N/A（当前owner扩展；发现已登记旧代码只EXTRACT/RETIRE）。不导入root ListingKit/tenantbridge/compatibility、旧ProductEnrich task/worker或第二标准商品。

## 9. 第一次独立检查处置

2026-10-12，第1轮只读检查绑定17f2adbd0及设计SHA256 2C381DBF2A01D72A5575412BD7E7C02EF70FAF8F486FDB6BF8EE64ED163B5760，结论NOT_READY。

- Finding：content-config-v1缺合法参数载体且设计排除DDL。Product requirement affected：可保存并冻结内容模板。Classification：BLOCKER（核心happy path无法完成，实施层）。Action：§4现配置owner有界nullable参数列/精确组合CHECK及独立显式更新，不新增表或迁移平台；§7同步更正。
- Finding：Review UoW写成RunDB。Product requirement affected：商品版本、Review与原操作回执原子提交。Classification：BLOCKER（核心happy path无法完成，实施层）。Action：§3/§7明确维持ProductDB和事务内Catalog/source，RunDB运行/配置独立。
- Finding：现title-only lineage及wire消费者须适配。Classification：IMPLEMENTATION_TEST（合并层）。Action：§7/§8验证真实混合title/content多跳、每字段证据、完整snapshot和额外字段改动拒绝，不另建owner。
- Finding：64KiB生命周期预算。Classification：IMPLEMENTATION_TEST（合并层）。Action：§5/§8覆盖before/after、原候选、历史/proof及未来Apply空间，临界/超界整事务验证，不扩大上限。

两项BLOCKER修正后仅做第2轮对应设计增量复核；正式代码仍等明确IMPLEMENTATION_READY。

