# 两个中心 V1：研究证据分析合同增量

Design Basis: **Independent Architecture**。状态：**DESIGN_REVIEW / NOT_IMPLEMENTATION_READY**。执行 #1072/#1073，父目标 #1070，沿同一 Draft PR #1075。源码核查基线 `366df3516865163636a68852d7d85cf68c530c48`，应用代码仍与已审 d8b85805e 相同。

用户于 2026-11-02 明确确认沿用[两个中心 V1 产品设计](../product/market-research-selection-centers-v1.md)：数据服务 → 市场调研中心、供应市场 → 智能选品中心，共用项目，汇总限明确选中2–5项。本轮继续设计及准入，不修改生产代码、不实际调用模型或平台。产品确认、Architecture Review、正式实施授权及真实试用分别记录；本文不代签。

## 1. Outcome、Authority 与边界

本人研究项目成员在研究视图核对样本，切到选品视图，对同一2–5个准确候选显式运行有来源的分析，查看观察摘要、差异、缺项和建议，再独立作人工决定，沿原批次确认转入待适配。模型不可用时证据、只读汇总、事实比较和人工决定仍可用。

Product/Figma Authority：[产品与完整节点索引](../product/market-research-selection-centers-v1.md)、[当前 Authority](../product/final-ui-ia-authority.md)。Figma 文件 `tg48P46SSXl6TBy9lZwg63`、非归档页 `31:463`；研究详情 [12114:1646](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=12114-1646)、评审 [12114:1746](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=12114-1746)、用量确认 [12114:2011](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=12114-2011)、人工确认 [12114:2021](https://www.figma.com/design/tg48P46SSXl6TBy9lZwg63?node-id=12114-2021)。明暗和关键390px同版节点随产品确认；原节点名称中的“候选”仅保留追溯。

Must：同一私有项目及准确观察；有界披露、原版本/时间/parser/checksum/样本范围可追溯；当前授权先于披露或付费执行；显式配置及最大用量确认；同键异载荷冲突；只从冻结证据生成建议；建议与人工决定分离；UNKNOWN由原owner核实，不能重复扣费或换键重发。无成本/需求证据不能产生利润/销量事实，缺项不能填零。

Out of Scope：新采集通道、Browser桥、评分/评论补采、差评工具、全市场统计/趋势/销量、费用/利润、商业权重、合规自动核查、1688匹配、监测、报告平台、自动审批/采购/上架、新IAM/队列/恢复器/额度体系。原后续目标继续保留。

Threat Model 沿首批冻结合同：跨org/actor访问、项目或来源撤权后披露、错误版本/旧观察、提示注入、同键异载荷、并发覆盖与未知付费响应、错误整批转入。没有新增 Accepted Risk；不为本增量引入理想化通用防护平台。

## 2. 当前差距及最小复用

现有 `product.selection.agent@v1.0.0` 的 Binding 仅含 Collection/Catalog引用，inspect/proposal模型准入固定为商品事实比较；没有研究项目、候选 revision、选定link及观察checksum身份。不能仅在旧 prompt 拼研究JSON，或将旧对比改名为市场证据分析。

沿现有 product-selection/shop-diagnosis 消费者模式，新增一个有明确版本的窄消费者，复用同一 Agent/Eino、配置快照、治理文本执行器、Resource/计量、run持久化和HTTP/BFF形状。不新增持久化AI结果表或第二执行服务，不抽取通用Agent框架。

| 事实/职责 | 唯一 owner 与本增量 |
| --- | --- |
| 项目名称/目标/类型/归档/当前访问 | projectcenter，沿当前 Service；不复制项目表 |
| 商品与原观察 | Catalog/Collection/dataacquisition/SRC；仍由原 provenance reader核实，研究角色无它们的直接SQL权限 |
| 候选关系、选定观察、人工状态及历史 | product/research；新增有界 `InspectSelection` / `AuthorizeAnalysisHistory` 读能力，不写模型状态 |
| 分析输入投影/引用校验/建议校验 | `internal/product/research/analysis`，无业务持久化；消费当前research和既有selection Inspector的合格事实 |
| 模型run/原输入观察/建议/历史/恢复 | 原agent及配置/持久化owner；复用原Recent历史发现并限定context/project，不新建列表owner、结果副本或“已分析”布尔值 |
| 准入、模型路由、预算/预留/结算 | 原agentconfig/aicapability/Resource；只增加明确operation/definition登记，不借旧配置自动启用 |
| 转入 | 原Collection/Preparation，原非空子集、版本核对及原键恢复不变 |

Legacy decision：N/A；无迁移、旧ID映射、compatibility adapter、fallback、双读双写。现行其他消费者的合同和有效运行不改写。

## 3. 精确研究 Binding 与披露

新增 context kind `market-research-selection-v1`，`ContextID` 为原项目UUID，`TargetPlatform=product`。复用现有固定5槽 `SelectionProductRef` 表达准确商品引用，并增加仅此context允许的 comparable `ResearchSelectionRef`：

- `Candidates [5]`，每槽包含原 CandidateID、规范正 CandidateRevision、SelectedLinkID、64位小写 EvidenceDigest；与同索引Selection一一对应；
- `InspectionDigest`，SHA-256，绑定下面规范化有界披露的完整字节；
- 单商品字段及 StoreDiagnosis 必须为空；2–5个唯一候选与商品引用按ItemID升序排列，零槽只允许尾部；非研究context必须ResearchSelection全零。新增字段使用 `omitzero`，不改变其他当前消费者的请求含义或保存事实。

浏览器只提交项目ID、候选ID/revision、选定link、evidence digest和原Collection选择身份；不能提交事实值、query/parser/hash、org/actor/member或模型结果充当权威。服务器fresh identity得到scope后，经当前Project Service和research读方法逐项核对关系/版本/观察，拒绝任一不符，不静默丢项或换成最新版本。最多5次点读，不经分页List猜全量。

`Inspection v1` 包含当前projectID、规范引用、原site/ASIN、capturedAt/parser/checksum、原查询模式/关键词或类目/样本上限（存在时）、原事实与missing/warnings。当前项目目标不隐式发送；仅发送用户确认的 `GoalSummary`（原上限512 UTF-8 bytes）。不发送Cookie、Profile、密钥、原HTML、任意source metadata、原始provider URL或远程图片。来源链接只在当前授权UI中由原owner呈现，不交给模型执行。

复用当前selection投影的合格Catalog字段；观察元数据和观察价格只从research/SRC原观察读取，不以当前Catalog head补证。当前事实矛盾或无法绑定则拒绝该分析，不建立同步/对账器。公开商品文字、query、goal、feedback和tool结果都是数据，不能改变system、工具清单、权限或预算。

限制沿既有合格比较合同：2–5项，每候选最多32个事实，每值≤512 UTF-8 bytes，missing/warnings各最多32项，完整披露≤32KiB。元数据单字段有界（checksum≤256、parser≤128、时间≤64、查询文本≤512 bytes），原精确身份不省略。超限先做字节准入，按既有公平策略省略完整非身份事实并标记missing；不得截断成新事实、先无界展开再过滤；身份与必要缺项仍装不下则在模型/预留前拒绝。

每个factID绑定contract版本、候选、准确EvidenceDigest、字段路径与完整规范值hash。摘要、缺项和候选顺序均确定化后计算InspectionDigest。prepare确认的是同一份有界披露；start、tool和每次新model dispatch重新核实当前引用与完整digest，任何变化停止，要求新的明确准备，不能沿原付费key补新证据。

不新增context快照表或报告副本。prepare仅在原配置快照保存Request/Binding/digest；run中的原inspect Observation保存完整有界披露。首次工具读取必须与prepare digest完全相同；原持久化Observation只能按原key恢复，不重新hydrate、抓取或以当前资料替换。

32KiB披露上限不替代原配置快照的 **8192-byte 完整payload** 上限。新consumer沿原Prepare对实际完整Snapshot编码执行此门禁：Request/两组固定槽、goal、冻结profile、mode及scope/config身份和元数据全部计入。只保存digest及精确身份，不把32KiB披露塞进Snapshot；不截断身份、不扩大共享schema门槛。超限在原owner局部事务的持久化提交及模型/预留前拒绝，prepare有关写入随原事务回滚。2/5项、长合法身份、512-byte goal、边界profile和原键重放必须实际验证；未跑不宣称最大合法输入已通过。

## 4. 样本汇总与建议输出

确定性汇总与模型可用性无关，仅限当前明确选中2–5项、每项一份观察；数量不是批次/项目总量或市场覆盖率。只对已知同币种、同计价口径的有限非负观察价格求min/max，展示“可比数/所选数”；计价口径无法确认则该项不可比，不推断每件采购成本。不同币种分组、缺价未知，属性仅比较原有可比字段。与UI中已批准示例的价格范围对应；数据不满足条件时显示不可判断。

新proposal kind `market-research-selection-v1`。结构为：有界overview；0或1个选中候选的建议引用；1–5个候选assessment（CandidateID、`TEST_CANDIDATE | RESEARCH_MORE | NOT_RECOMMENDED` 建议码、有界advice、1–6个本候选的准确factID）；overview跨候选引用显式给出，最多12个已披露factID。不得借其他候选或外部资料的引用；同候选重复/未知字段/未知建议码/伪造factID拒绝。

原值、数字、币种、来源、样本与missing由确定性披露渲染；不把模型生成数值写入事实区。结构化输出没有销量、利润、总体市场分数或采购/上架动作字段。建议文字明确标AI建议、需人工核实，不能隐藏服务端missing。校验器证明引用身份、结构及范围，不证明商业判断质量；不能把AI评审/validator通过当用户验收。

模型只允许 `tool / propose / interrupt`，固定只读inspect工具；拒绝Product/Listing mutation、任意tool、Confidence、上下文引用或其他domain proposal。不把建议码写入research状态；只有用户在原流程填理由并提交当前revision/digest，才执行人工决定。

## 5. Agent、经济准入与调用链

精确登记：Agent `product.research.agent@v1.0.0`、只读工具 `product.research.inspect@v1.0.0`、policy/prompt/action/config schema各为 `market-research-…-v1`，文本operation `product.research.decision`。原 `product.selection.agent` 和原事实比较入口继续原合同，新能力在研究评审视图显式呈现。不要扩旧definition版本含义或借其他Agent模板/默认配置。

新definition沿现有agentconfig catalog/activation/snapshot/guard/四种执行模式注册；V1仍无named template、KnowledgeBase或技能扩展。新operation沿现有aicapability operation allowlist、RouteProfile校验、quote/point-fact/invocation settlement登记，复用原文本用量及资金owner；不新建钱包、资金来源或后台扣费循环。没有本org显式route policy/activation/profile时是未配置/未启用，不借selection/title profile或任何自动fallback。

沿现有两调用模型及精确工具准入：第一调用只做既有决策/inspect请求，工具返回已绑定披露后才可生成建议；propose必须已经消费本次准确inspect。模型修复、token/cost/runtime及工具预算沿原实现，不能增加用户批准的Limits。profile的两调用最大token/费用必须装入冻结Limits，预算确认显示原最大用量/币种；`read_only`也不能省略经济准入或把付费调用伪装成免费查看。四种模式均无采集、人工决定、采购/上架副作用。

| contract → implementation | injection → consumer |
| --- | --- |
| research精确所选关系/来源读port → product/research方法 | 原当前Project/Sources/Repository注入 → analysis Inspector，无跨owner SQL或授权scope传入 |
| 既有商品事实Inspector → 准确Catalog/Collection投影 | 原Collection Service注入 → 研究analysis的有界披露和UI样本汇总 |
| 新binding/inspection/proposal校验 → research/analysis | 原AgentTextModel窄constructor + Eino定义/tool/model-admission/domain-validator → 原run Store/Guard |
| 显式ResearchTextPolicies及definition/operation登记 | currentapplication原ProductAgent/RunDB、配置、Resource与Research模块依赖注入 → 新HTTP模块；serving不DDL或自动激活 |
| 原配置prepare/run/lookup/resume → native HTTP/BFF | 研究评审UI显示披露与最大用量并显式开始；跨scope、重复字段、错误body/route/key原规范继续拒绝 |

新路由沿已有research项目前缀：`POST .../projects/:project_id/analysis/inspect`、`POST .../analysis/prepare`、`GET .../analysis/preparations/:request_key`、`POST .../analysis/runs`、`GET .../analysis/runs/:request_key`、`POST .../analysis/runs/:request_key/resume`，另有下面的回执列表GET。具体parameter名沿当前Gin已注册前缀一致，不能引入wildcard冲突。ContextID由路径projectID确定，request key为规范UUID；start要求单个Idempotency-Key和准确ConfigurationSnapshotRef；GET无body/任意query，分页列表仅允许一个合法cursor。

HTTP/BFF沿verified identity及fresh组织成员权限。inspect需原collection-read与project-read；prepare/start/read/resume沿原Agent-use/local-agent权限并核对当前member，执行再检查原mode/config。不能凭项目管理员或浏览器给出的projectID访问他人资料。模型/quote/新披露前逐项核对当前成员、项目active和全部当前引用；归档只读历史，不能新执行。

## 6. 持久化、历史与 UNKNOWN

不新增schema、状态机、跨库事务或research角色权限。Project/Research/Product在读边界核实，唯一新写入是原AgentConfig prepare快照及原run/observations/proposal；这些仍在原各自事务/Claim/CAS中完成。研究候选和人工决定不与模型完成双写，无需跨库补偿。

prepare响应丢失通过原配置owner新增的精确按key读取恢复：scope+definition/version+context kind/projectID+request key定位并验证原持久化Snapshot，核对当前member和项目读权限。原Binding含受保护的商品/来源身份，披露前必须逐项执行下文历史授权；不要求当前人工revision等于旧revision。来源失效时仅返回不含原来源身份的“可能已存在”状态，不假装不存在。可披露时只返回原配置ref、原绑定与确认用量，不执行模型；客户端与持久化pending原意图逐项匹配。

POST prepare先查原key，以原持久化Snapshot核对原始规范意图：原definition/member/project、候选/选定link/证据及Collection身份、goal、期望模式身份。不能重新resolve当前profile或重新hydrate当前Binding来重建原fingerprint；该fingerprint中的server profile/digest以原Snapshot为准。相同原意图返回可授权的原回执，异意图冲突；当前profile/config/来源变化不能让原回执被当不存在，不能借新key“恢复”。新的start仍完整检查当前资料和配置。原owner现有原子Snapshot/唯一键继续承接，没有第二准备表。

分析历史只存原run owner，发现复用原 `agentconfig.Store.Recent(size=10)` 的Snapshot→run读取、member预过滤、授权、身份验证及created_at/id分页。在同一owner为本consumer增加固定context kind/projectID过滤，必须同时应用于主SQL及anchor查询，并在分页前生效；不给浏览器选择其他owner或context。原实现会读取有界Snapshot并Lookup/解码原run payload，**不宣称内部header-only**，不新增并行列表owner或指针表。只对外返回当前获准的有界metadata，不返回证据或建议；raw phase不是分析成功判定，也不提供项目总量。

项目当前授权在历史查询前后核实，原Recent的authorize回调采用下文历史语义，当前member仍由原Snapshot核实。打开一个结果继续沿原LookupContext核实frozen配置/member及全部来源，再返回受保护正文。列表过滤/隐藏/缺项不能证明原run不存在；原key Lookup和operationMayExist恢复路径独立保留，不从列表推导换key或重新执行。

新执行与历史读取要区分：新执行必须匹配当前CandidateRevision/SelectedLink/EvidenceDigest/InspectionDigest；人工决定导致revision变化后，历史仍可在当前授权下读取原结果。历史不要求当前候选revision等于旧revision，不重新计算当前摘要冒充原输入；原link须仍属于同一候选且未解除、原来源仍可读。换选观察但保留旧link时可读旧分析，并标注历史依据；解除/撤权/来源不可读时整份受保护的跨候选摘要和建议不可展示，不从副本或其余候选补回。

历史受保护内容不可读不等于run不存在。原键查找先在fresh scope和project下核实存在性；如run存在但来源不可读，返回来源不可访问并保留 `operationMayExist=true`，不能伪装404让UI清空意图、换key或重跑。只读核实不重新调用模型或恢复采集；原run过期fence/finalizer继续归原owner。

| 边界/失败 | 结果与唯一恢复入口 |
| --- | --- |
| inspect失败、超字节、引用或授权变化 | 无AI快照/run/预留；回到当前资料修正，不自动扩大样本 |
| prepare提交响应丢失 | 同原request key查询原配置快照/receipt，验证同payload；无run前不能假定已执行模型 |
| 配置快照成功但start前失败 | 仅原快照已保存；使用该ref/key明确start，无跨owner回滚 |
| 并发start/首个run Claim响应丢失 | 原唯一键/Claim只允许一个执行者；核实原run，不能新建key或再次reserve |
| 模型/usage/reservation结果未知 | 原governed executor/Resource和run恢复负责，保留UNKNOWN，不能直接再次请求模型或自行退款 |
| 第一调用后来源/配置变化 | 拒绝新工具披露/第二调用；已发生用量由原owner结算，研究不退费或改写历史 |
| 完成后HTTP响应丢失、刷新或重启 | 从原run读取原Observation/proposal及原key，当前授权复核；无重采/重新计量 |
| 并发人工决定或换观察 | research原CAS决定结果；模型不写入人工状态。新运行需新明确prepare，旧run不变 |
| 取消/归档/退出企业 | 不增加调度器；新动作实时拒绝或停止，已发生调用/未知用量沿原owner收口 |

## 7. 实施顺序、验证与停止点

本合同评审只覆盖新增研究binding/披露/模型准入/历史与现有经济组合，不重审原冻结研究、selection、Resource或Preparation架构。适用独立Architecture Review最多两轮；finding先按当前Must及AGENTS分类，仅真实BLOCKER重开设计。

正式实施获准后，同一Writer/主要PR连续交付：两个中心原页面/只读研究投影 → 确定性选样与汇总 → 新AI消费者精确登记及有界披露 → 原治理运行和历史读取 → UI显式确认与原决定/转入消费。普通页面如完全复用现有边界按轻量Reuse路径，不为每页另审或拆PR。

必要RED→GREEN复用现有设施：binding错配/同键异载荷；2–5精确选样、伪造/撤权/归档/跨成员；32KiB披露及8192-byte完整Snapshot字节准入/原键重放；缺价/混币种/口径未知；prepare已保存后profile/来源变化仍原键核实且不泄露；prepare→dispatch来源/config变化；未inspect先propose与假引用；两调用/token/cost gates；并发Claim/响应丢失/usage UNKNOWN不重派；人工决定后可读原历史、换观察不刷新旧run、解除/撤权不泄露；Recent旧member与跨项目anchor拒绝、仅对外metadata、列表缺项不改变原键恢复；无模型仍可人工操作；准确子集转入既有证据复用。新增operation/definition各个实际登记与消费者要成对验证，不以guard扫描本身代业务证据。

执行当前改动风险直接相关的开发自检和既有必需CI；不开发验收runner或专项故障平台。最终交付复核完整用户路径及真实diff，修复仅复核对应增量。真实平台/模型/付费、部署、合并、Issue关闭和产品验收保持独立授权；没有真实执行保留NOT_RUN。

### Architecture Review 记录

原独立Reviewer R1绑定 HEAD `5985f9e2a59026098dd6cc83246dfc7944a001e6` / 设计blob `1af1b114ae7c5fb47cf88d9611e02e7de76d9624`。结论：无Architecture BLOCKER；以下三项归 **IMPLEMENTATION_TEST**，最小澄清已统一写入正文，待增量确认后冻结，不重复全局架构评审：

1. 原prepare key恢复必须不依赖当前profile，披露原Binding前做全部来源历史授权；受影响Must为撤权不披露、丢响应可恢复且不重复付费。实施验证保存后profile/来源变化、原意图重放与operationMayExist。
2. 历史发现复用原Recent/member及原run owner，主查询和anchor均先限当前project/context；受影响Must为本人/当前成员隔离、无第二事实、未知结果不重派。实际内部读取有界payload，仅对外返回metadata。
3. 完整Snapshot仍受原8192-byte门禁；受影响Must为合法2–5精确输入保存及超限先于模型/预留拒绝。实施验证长合法身份/goal/profile、边界与原键重放，不扩共享限制或裁掉身份。

以上属于当前Must的必要实现测试，未满足时仍阻止相应切片合并；不是新产品要求、风险接受或验收平台授权。当前尚未签发新增生产路径的实施准入，正式代码授权、真实平台/模型/付费及产品验收仍各自未发生。
