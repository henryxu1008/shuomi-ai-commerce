# #1205：已有合成原件窄读取——局部设计候选

状态说明（2026-11-09）：原设计 a9937773e 已通过架构复核并获实施授权；下文 §1–9 保留该冻结设计的提交时说明。单件读取/手工链已实施并完成离线及隔离自检，未部署；§10 是本次调用追踪发现、用户确认补齐的转入设计增量，尚待现有 Codex 的有界准入复核，不能以初始实现宣称全部 Must 满足。滚动证据维护在 PR。
用户已授权当前同一Writer在原工作区提交/推送本设计、创建Ready PR，并核对准确HEAD的CI和现有Codex架构复核。未授权实施、运行配置变更、样本/业务写入、重启、模型/平台调用或合并；不新增Writer/Reviewer。

## 1. 用户结果与结论

在统一实例原账号、原合成项目内，用已经存在的一件合成原件验证“保存经营评估→刷新找回→人工决定引用准确评估”。其他现有功能、账号、数据、会话保留。

**原owner、原件、权限、准确版本/摘要和原键合同可以复用；原有的部署级隔离条件不能原封不动地保留。** 现有fixture在配置层和应用装配层都要求关闭执行、独立research组合；统一实例正在运行Data Services/Product Agent/资料模块。技术上只读原件，不代表其进入新消费者之后仍没有执行风险。

因此没有“只开已有开关、其余边界完全不变”的方案。用户已确认的新范围是：指定合成原件在统一实例仅用于人工评估，不能因此取得执行资格。该范围仍是未实现的设计，架构准入和单独实施/运行授权满足之前不能开放。

## 2. 范围、依据与设计准入

产品决定 PD-1205-EXISTING-SYNTHETIC-MANUAL-READ-2026-11-09：用户已明确认可本局部隔离边界，仅开放既有B000123457的手工评估，禁止进入AI、转入和发布，其他功能/账号/数据/会话保留。该决定允许本设计进入准确HEAD的架构复核，不授权实施或运行操作。

- 依据：现有 `research-commercial-assessment-persistence-v1.md`（评估、人工决定、原键）；`research_fixture.go`（合成原件/权限）；`research_fixture.go:validateResearchFixture` 与 `current_application.go` 的closed-composition准入。
- 当前运行源码：固定候选 `7dec3b955286a288a4778e26e519d855bf3a30f3`。相关owner/fixture/analysis文件已只读核对与调查工作区相同；本设计不改变运行候选。
- Design Basis：若选择在统一实例开放，属于 **Independent Architecture** 的有界来源/执行隔离变更，不能按普通读取bugfix或现有架构完全复用开工。
- 当前产品边界已确认；独立架构复核尚未完成，仍不是IMPLEMENTATION_READY。CI或文档提交不替代架构准入，更不授权实施。
- UI沿现有“智能选品→评估与决定”，不新增顶层菜单、测试控制台或通用导入入口；Figma不决定本次范围。

本次不做：正式插件来源准入、自有商品准入、真实采集/模型/付费、重导入52件、货源核对增量、商品转入/发布、数据迁移、新表/新账本/通用来源框架。

## 3. 已核实的一件原件及候选

只读核对目标库 `product_acquisition`，没有修改记录。

| 绑定 | 冻结值 |
| --- | --- |
| 企业 / 用户 / 成员 | 394513989062623234 / 394513839896395778 / 394513991663091714 |
| 项目 | e83d6399-1e20-4fac-ab34-93b698203602，原“合成验收 #1070”PRODUCT_RESEARCH项目，未归档 |
| 候选 / 当前revision | 6768ffeb-490f-5616-9905-bac1f0530d8e / 2，TO_RESEARCH |
| 当前证据链接 | 5fa57a3f-f91a-51e0-93a2-e017c8aac2e9 |
| Collection条目 / 原revision | 32f9a7c2-4345-5c28-a72e-cb75f3525c9d / 1，未归档 |
| 原集合 | 70cf65db-5bea-59e2-85f4-97e5879d3d4b |
| publication | ae0aeb8c-75c7-51c4-bfdc-b696ea62c62b |
| productKey | research-fixture-v1-ae0aeb8c-75c7-51c4-bfdc-b696ea62c62b |
| producer | research_acceptance_fixture / v1 |
| Catalog原版本 | 1；不把fixture的referenceVersion和Catalog版本混用 |
| 原始内容checksum | sha256:898b11ebf7e631565d56d3f73fbf5ed1739397f64109623b338398a44f8afaf2 |
| 完整envelope hash | 20276049f16dc63b5c1e5957f3a09e980046ea29ad9ee8f1545b5a8bfd1462e1 |
| 完整snapshot hash | 01cf35754a0d5b17b79f21a79878ec3bfb18736b0ce4a88d8cf525342e0216f5 |

原件标题已明确“合成”，SourceType是manual_import；RawReference/Trace固定为research_acceptance_fixture，metadata包含原bundle/member/project。该原件模板无图片和平台URL，不重新编造ASIN、URL、抓取时间或parser结果。B000123457只是既有合成记录中的原标识，不改成真实观察。

调查时评估0条、Research命令57条，未执行保存/review/decide。

## 4. 现有能力和实际缺口

| 层次 | 已有能力 | 本候选处理 |
| --- | --- | --- |
| 原件/发布/版本owner | SRC发布/回执、Catalog不可变版本、Collection成员引用 | 原样复用，不复制原件、不改归属、不补发布 |
| 原件证明 | InternalProducer.Verify在现有只读事务中核对完整发布/回执/快照；fixture readSource再核Collection引用 | 抽取这段现行有效读取行为，不能仅按key前缀或标题放行 |
| 现有fixture对象 | 同时有Read、Import、分页导入和GET/POST路由 | 新消费者只拿到单件读取接口，不把整个fixture对象/Import路由交给统一实例 |
| 运行准入 | validateResearchFixture及NewCurrentApplication都拒绝执行模块共存 | 旧模式保持原拒绝；不能绕过外层校验直接调用enableFixture |
| Research读/手工写 | 来源Observer、SupplierMarket、原评估及决定服务 | 仅按允许对象与用途装配，不改变来源引用 |
| Agent消费者 | Research analysis与手工链共享Sources.Observe，当前未按fixtureVersion单独拒绝 | 必须识别并阻止新profile被真实执行消费，不能因一个页面按钮隐藏而宣称隔离成立 |
| 刊登 | 当前upload_plan已拒绝research-fixture productKey | 保留原拒绝；它不能代表所有准备/执行路径已经得到证明 |

## 5. 最小候选：一个已有对象的人工评估读取

### 5.1 原owner与读取接口

仅提议一个内部“读取已存在原件并返回证明”的窄接口。输入绑定当前真实身份、原项目、准确Collection selection；输出为原件现有投影与原完整引用。只允许上表的单件对象，不接任意JSON、URL、其他项目/成员或分页清单。

实现优先复用fixture owner现有readSource/Verify的行为及SRC/Catalog读取能力；不复制一套hash、版本或发布算法。Source端在只读事务中运行，接口不暴露Import/Publish/重新生成；统一实例不注册acceptance-fixture导入路由，也不生成第二份原件。关闭profile不删除或重写原数据。

读取顺序：当前IAM/成员/项目/Collection权限核对→精准匹配固定对象→原SRC回执/Catalog版本/完整envelope与snapshot证明→Collection原条目/版本/摘要一致→返回前当前权限及上下文复核。缺失、归档、撤权、成员漂移或任何不符均拒绝/遮蔽，不取最新版本、不重抓、不重发原件。

原fixture Read目前要求完整三件bundle齐全，并检查原项目active；若未来改成单件读取，必须明确这是调用集合范围缩小，不得删掉该件的任何证明。其他两件原件仍保留，不从新接口开放。

### 5.2 原引用和恢复

返回既有EvidenceRef的原值，保证原链接/摘要不变。现有fixture合同中的jobId是原bundle身份；保持该原值和明确的fixture标识，不新建Data Services job、不把它宣称成真实采集任务，也不把capture/publication填入另一个来源的jobId。

Research继续是候选、评估、决定和命令回执的唯一owner。评估保存与命令/CAS按原同库事务；决定绑定准确assessment ID/revision/inputHash/contextHash，不取latest。响应丢失沿原key查询，不能以新key重做；已提交回执找回规则不因新增profile而改写。完整内容读回仍重新验证原权限及准确来源。

### 5.3 必须明确的新隔离边界

保持旧researchFixture配置及两层closed-composition守卫原样。若用户决定在统一实例开放，需要一个**新、默认关闭的局部准入范围**，明确只绑定上表企业/用户/成员/项目/对象。这是新的配置/来源资格，当前不存在，不能冒称只是启用已有开关。

允许的用途只包含：原候选读回、服务端经营预览、save_assessment、review、decide及这些已提交命令/评估的准确读回。为本轮最小验收，建议decide使用明确标记“合成验收”的REJECTED记录，不批准真实测试/转入；其状态选择仍须在后续验收输入中冻结。

**禁止新增的资格：**导入/重新发布、扩展其他对象、Agent prepare/start/resume、模型/付费/资源调用、商品准备/转入及对外提交。不能靠“不点击按钮”实现，也不能仅靠单件不足两件的偶然限制。

现有代码没有独立的manual-only来源用途合同；共享Observe正是需要设计/复核的缺口。后续若实施，必须在真实调用owner的发送/准备前使该用途限制可执行，并证明其他正常项目/来源不受影响；不能另造通用IAM或admission平台。当前没有完成这个证明，因此不能批准“只加Reader然后上线”。

### 5.4 收敛后的最小改动（仍仅为候选）

本轮补充只读追踪后，建议按以下四块实现职责收敛；不是四个新系统，也不拆出新Writer/框架。

| 改动块 | 现有位置及复用点 | 最小新增行为 | 明确不改 |
| --- | --- | --- | --- |
| 单件来源读取 | `app/dataservices/research_fixture.go` 的 `readSource`、`InternalProducer.Verify`、Collection原件读取 | 在原owner内提供仅能核对已存在单件的读取口；返回原引用和证明，不导出Import能力 | SRC/Catalog内容、发布/版本/hash算法、数据库角色和原件数据 |
| 有界装配 | `runtime/currentapplication` 与 `httpapi/current_application.go` | 一个默认未配置的人工合成读取范围，绑定唯一scope/project/candidate/publication/version；按原身份、Product库和回环环境校验，错误/不完整配置拒绝 | 旧`researchFixture`及其两层closed-composition守卫；其他模块启用状态；不注册fixture GET/POST导入或分页入口 |
| 手工消费者接线 | `httpapi/market_research.go` 的Observe、`market_research_supplier.go` 的SupplierMarket，以及原Research评估/决定服务 | 只对准确命中的既有对象采用人工读取口；列表、预览、保存、review/decide和精确回读消费同一原件引用 | 其他51候选的可读性、正式Amazon/Data Services/插件准入、原评估公式/schema/事务/原键恢复 |
| 执行资格拒绝 | `product/research/analysis_read.go` 的精确selection核验及Research Agent现有inspect/prepare/start/resume调用链 | 若一次AI选择包含该人工合成对象，在原领域核验处拒绝，禁止进入准备持久化、任务、模型或资源调用；其他正常来源按原规则 | 不关闭整个项目/企业的AI，不授/撤成员权限，不改计费；不把“只有1件、AI至少2件”当成隔离保证 |

配置仅表达批准的准确对象身份；“允许哪些操作”应是本局部设计固定的人工范围，不提供可由页面/模型修改的通用权限清单或任意范围选择器。未命中/未配置的fixture仍按原路径拒绝，不自动降级为Data Services或插件来源。配置不是授权凭据，所有请求仍须当前IAM和原件权限复核。

旧fixture的全bundle `Read`、`Import`和分页行为不变；新的单件口只抽取/调用该件完整核验，不通过注入旧`enableFixture`绕过其装配守卫。只读事务不是放行来源的理由，准确scope和产品确认仍是前提。

读取器的人工用途不能由请求体自报`manual=true`来决定。应由原Research手工入口和领域调用明确区分，Research AI的inspect及当前/历史selection核验不能获得这项人工读取资格。精确assessment原键回执的历史找回维持既有合同，禁止为了用途拒绝而把已提交命令伪装成未提交。

### 5.5 已有执行拒绝可以复用，新增验证只覆盖真实缺口

代码中已确认三处针对`research-fixture-v1-`的拒绝：
- `listing/preparation/source_selection.go:SelectForExecution` 拒绝fixture；
- `integration/persistence/listing/preparation/operations.go` 在创建后续operation时拒绝fixture；
- `app/listing/upload_plan.go` 在刊登计划事实核对时拒绝fixture。

这些原有拒绝不移除、不改成例外，也不为本次验收另造一套执行系统。它们证明的是对应现有执行入口的拒绝，不自动证明所有UI转入步骤都无写入。因此本人工profile不提供“批准测试/转入”操作资格，范围内命令暂提议只允许明确标记的REJECTED决定；浏览器动作限制不能代替服务端边界。

Research AI目前的 `analysisSelection` 会调用与手工链相同的`Sources.Observe`，没有按这项新人工范围隔离。这里是必须补的组合缺口：即使拼入一件正常商品满足2件数量，也要在模型/任务/资源调用前拒绝人工合成对象。正常对象不命中该范围时，原流程不得被整体关闭或降权。

对其他Agent/Tool，本次不向它们装配新读取口，不改变其既有来源资格；不能借本任务宣称修复或重审所有原有执行路径。若实际必要调用追踪发现本次新增读取仍能扩大其中某条外部执行资格，应停止并报告那个具体入口及最小补齐，而不是默认添加全局来源治理。

## 6. 保留能力与必须变化的边界

| 要求 | 能否保持 | 具体说明 |
| --- | --- | --- |
| 原owner与原件数据 | 能 | 原SRC/Catalog/Collection，不写原件或第二副本 |
| 原IAM/成员/项目及来源权限 | 能 | 保留原read授权所需的ProductSourcing权限和当前成员复核，不授新角色 |
| 原版本/摘要/请求键/CAS | 能 | 准确值逐项一致，原Research事务与回执机制不变 |
| 原fixture封闭部署规则 | **不能原封不动满足统一实例目标** | 原模式要求执行模块关闭；新profile将合成手工读取与执行模块共存，必须明确批准并证明局部隔离 |
| 现有业务功能不中断 | 目标能保持，尚未实证 | 不关闭Data Services/Agent/资料模块；新增限制只针对指定合成对象，需后续差异验证 |
| 无真实执行副作用 | 当前代码不能仅靠读取保证 | analysis等消费者可复用来源，必须补用途边界并验证；已确认的Preparation/operation/upload拒绝可复用；Research AI仍有共享读取缺口 |
| 零配置/零代码变更即验收 | 不能 | 当前没有独立读取装配入口；本轮不进行这些变更 |

**停止线：**如果必须放宽旧守卫、减少原件证明、修改真实角色/余额，或不能在现有owner局部阻止执行，不采用统一实例方案。保留原件和实例，另报缺口；不能转为“暂时允许，先验收再说”。

## 7. 获准后的最小数据范围（当前不执行）

原件、发布、Catalog、集合、项目、候选与链接均复用，**源端新增写入为0**，不再沿插件入口新造一件样本。

获准后的单条人工链最多：1条research_assessments；3条research_commands（save_assessment、review、decide），原候选按各次CAS推进revision。若正常顺序且无并发，r2→r3→r4→r5；所有以实际回执为准。其他51候选和既有决定不变，不新建货源核对。原完整条件只用明确标记的人工合成假设；未知费用仍未知，不把缺项填零。页面正常访问记录按原机制处理。

上述业务写入尚未授权执行。后续实际写入和更新实例需另行批准准确源码/配置及输入；批准设计不授权操作运行库。

## 8. 后续必要验证与当前未决

只列与新增边界直接相关的既有回归，不新建验收平台：

1. 单件命中和未命中、错误scope/member/project/版本/摘要、末次撤权均失败关闭；读取不产生SRC/Catalog/Collection写入。
2. 原fixture启用条件与Import路由不变；新profile不暴露导入/任意来源，无配置时不装配。
3. 原手工评估/决定事务及原键恢复复用已有证据；只验证新来源装配的组合，不重做未变算法/全仓审核。
4. 指定合成对象被AI准备/起跑/继续或转入/发布使用时，在原owner调用外部provider/任务/账本前被拒绝；正常业务不因该局部规则而关闭。若现有合同不足，保持NOT_READY并报告。
5. 获准更新后才做同一对象保存→刷新同一评估→决定引用→再次刷新关系的浏览器验收，不改变原成员权限制造负例。

已确认的产品决定：允许“现有执行模块正常运行的统一实例，只对这一件既有合成原件开放人工评估/决定，禁止其进入AI、转入、发布”的局部隔离范围。该决定不把原部署级完全关闭执行改写成没有变化，也不降低旧fixture模式守卫。

若坚持原封不动保留封闭部署规则，应继续采用封闭隔离组合，不能在统一实例完成本设计目标；需要另行决定是否更新已保留的独立环境。本候选没有授权或操作该替代方案。

## 9. 本次交付与状态

仅设计文档及目录索引；原Writer/工作区继续，按用户授权创建Ready PR并由现有Codex做R1架构复核，不新增人员或重复触发评审。未改生产代码、配置、schema、原件/评估/决定、角色或凭据，未重启、部署、调用模型/平台，不关单。运行候选7dec未变。产品隔离范围已确认；架构复核等待准确HEAD结果，实现/新增边界验证及产品验收NOT_RUN。滚动HEAD/CI/评审证据维护在PR，不提交到本合同。

Legacy decision: EXTRACT（当前仍有效的fixture精确读取/证明行为）；没有legacy兼容、迁移、双读/双写或第二事实源。文档中的候选接口与profile均未实现，也不是现有能力声明。


## 10. 商品准备转入拒绝的有界补充

2026-11-09，用户确认在同一 PR 补齐：仅新 manual profile 绑定的 B000123457，拒绝单件及整批中的新转入；原键回执找回、其他商品及旧合成验收流程保持。

### 10.1 实际缺口与分类

现有 `preparation.Service.Transfer` 先找原命令回执，再通过 Collection `SelectBatch` 获得固定选择；持久化 `preparation.Repository.Transfer` 在原同库事务中枚举该选择，仅以 `ListingTransferKind` 准入来源。B000123457 的原 `amazon_data` 类型合法，因此原“我的数据→转入商品准备”接口不消费 Research 的 REJECTED/APPROVED 决定，能够绕开只在 Research 拒绝批准的限制。

Finding classification: BLOCKER（当前 Must 的用途限制不能由页面或下游执行拒绝代替；影响本切片最终交付）。不声称已产生真实错误写入。既有执行和发布守卫仍有效，但不能把它们当作转入拒绝的证据。

### 10.2 最小方案与 owner

- 不改 Collection 原件、来源类型、权限或已发布事实，不给原件加第二份持久化用途标签，不改所有合成资料的通用准入。
- 在商品准备领域 Service 增加一个仅由服务器构造的不可变“人工原件转入排除”绑定：原 organization/actor/member + Collection item ID + publication ID。身份仍由现有当前成员授权确定；请求不能自报或解除此限制。
- 同一配置同时装配 Research 窄读取与商品准备排除。当前正常 Listing、Source Materials、Completion Drafts 及 Acquisition Plans 提供 Preparation 转入入口的组合全部纳入；共享 application 重用同一 owner，独立 Preparation 组合也不得漏装。缺绑定/非法绑定拒绝启动，而不是仅约束某一页面。
- Service 仍优先 `FindTransfer` 按原键/原载荷返回已提交回执。仅对新转入，把冻结的排除绑定封装在原 `TransferCommit` 中；不新增状态机、表或日志事实。
- 持久化 owner 在原 `VisitSelection` 的逐件回调中、加入来源缓冲前核验排除。原 scope 下任一项匹配排除的 item ID 或 publication ID 均拒绝整个新事务，不跳过该项后把部分结果写成成功。显式单件与未提供 itemIds 的整批选择使用同一检查；已创建 Preparation 行及此前缓冲/写入的来源均由原事务回滚。
- 不增加 DB 写权限、重试入口或业务命令。原键不同载荷仍冲突；并发同键按原 advisory lock 和原事务处理；UNKNOWN 仍按原键找回，不盲目新建。
- 关闭新 profile 仅移除本次局部装配，绝不删除数据、改写原回执或恢复旧运行资格。此段不授权实际运行配置变化。

### 10.3 必要实施义务

1. 用既有隔离 PostgreSQL/Collection/Preparation 测试证明：未排除时原测试链仍成立；启用新绑定后，该件的单件转入与包含它的整批转入均返回拒绝且 Preparation/来源行无新增。
2. 合成批次须包含正常项，证明不能部分写入；仅转其他项成功；原 scope 之外的正常组合不受影响。
3. 在启用绑定之前已提交的相同请求，启用后仍可按原键/同载荷找回；不同载荷不冒充回执。原回执读取不触发新转入。
4. 正常应用装配测试覆盖所有当前转入入口的服务连接；保留原 fixture 配置/导入守卫，不把所有 fixture 一律禁用或改变旧合成链。
5. 继续复用下游执行/发布拒绝，不重做全仓验收；真实转入、运行、浏览器及产品验收 NOT_RUN。

本补充只明确用户已经批准的“禁止该对象转入”在原 owner 的最小落点；不接受新风险。当前等待现有 Codex 对本增量的范围内架构准入，尚未实现这一新转入绑定。通过后先以失败回归捕获通用入口缺口，再在原 PR 最小实现。
