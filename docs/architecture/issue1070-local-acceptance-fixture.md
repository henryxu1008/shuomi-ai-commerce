# #1070 本地验收项目导入三条合成原件

**Independent Architecture · 第二轮独立评审 · NOT_READY**

## 用户决定与范围

2026-11-06 用户授权沿 #1131 为 `task-processor-unified-20261010` 的指定验收企业 A 研究项目导入上一轮的三条合成原件。合成资料用于走通「Amazon 美国站关键词 → 市场调研 → 同项目智能选品 → 核对 1688 SKU 与费用 → 人工批准 → 待上传」。仅允许 W1 固定资料。W1–W4 属于最小离线实施的验证场景，完整业务验收由用户或指定验证者执行。Figma 仅供理解业务流程。

- Amazon 同批查询 `under sink organizer`，A `B000123456`（参考原件 v3，USD 29.99，可调宽 38–55 cm，免打孔），B `B000123457`（参考原件 v1，USD 24.99，固定宽 42 cm，需打孔），两件单件装。
- 1688 合成货源 `981645030344`（参考原件 v2），SKU `sku-adjustable-white-01`，40–55 cm，验收人工核对 20 件。
- 样本名、参考版本与合成属性固定。Catalog 分配真实的本地发布版本，原夹具 `v3`、`v1`、`v2` 作为合成来源标签保留，不伪造本地版本，也不补造前序发布。最终回执比较实际本地 publication ID 和 Catalog 版本。候选业务 revision 由原 Research 命令推进。
- 范围止于待上传回执与刷新回读。无真实采集、账户额度预留或扣减、计费账本写入、模型调用、采购、刊登、外部提交、线上开关、迁移和通用导入能力。不合并 PR。

## 根因与 owner

- Data Services 的线上 Amazon 关键词查询从 `data_acquisition_jobs` 读取。`data_acquisition_items` 的已保存约束要求真实 reservation，`Claim` 和 `Publish` 均复验。把测试回执写进去会制造账本无法解释的来源。免费 Amazon 插件虽产出收藏，但没有关键词查询批次。1688 免费插件 producer 与现有 Research 的服务端采集准入不同。通用手工/定制导入不能冒充上述来源。
- 合成发布仍由 Product Sourcing/Catalog 持有不可变原件与版本，Collection 持有同批引用，ProjectCenter 持有项目与有效访问，Research 持有候选、证据关系、核对及人工批准，Preparation 持有单件转入。合成输入是固定测试载荷，持久化原件的读取只信 owner 已发布的精确记录。预算与计费没有新增 owner。
- 现行设计见 [Amazon 机会交接](amazon-opportunity-handoff-v1.md)、[市场调研与选品](market-research-selection.md)、[货源核对](research-supplier-review-v1.md)、[样本](../engineering/issue1070-first-trial-walkthrough.md)。旧隔离浏览器夹具的模拟权限与额度回执不移植到验收环境。

## 最小方案

使用一个显式启用的本地验收装配，在已有研究项目中以当前登录人的企业、用户、有效成员和项目为唯一作用域。运行配置固定一个项目、一个作用域和一个三件样本包，不允许客户端提供商品、原件、作用域或外部地址。默认关闭，无通用入口或公开 API。沿现有身份与项目授权核验，需项目写权、Collection 管理权、两种来源既有读取权。实际目标 ID 通过本地已登录会话核实后填入私密配置。入口在部署配置未明确启用、实时采集或 AI 执行处于开放状态、作用域不符时拒绝装配或执行。不能使用测试身份、伪造令牌或扩大数据库 grants。

1. 已有项目内只提供一个明确标为「导入合成验收资料」的动作。请求零业务载荷，路径中的项目须与配置完全一致，服务器从固定代码选择三件。使用 `source_acquisition_runtime` 已有权限，在同一个 Product DB 事务中使用 `sourcing.NewInternalProducer`、`sourcingpersistence.NewTransactionWriter`、现有 Catalog bridge 及 Collection append 原子发布。独立 producer `research_acceptance_fixture/v1` 与保留 `research-fixture-v1-` product key 前缀表示合成，按角色区分 Amazon A/B 与 1688 S1。`amazon_data` 与 `acquisition` 只用于现有 Collection/Preparation 的存储分类，来源身份、标题、批次及原件元数据均明确合成。Amazon 两件复用 `AppendDataPublication`，1688 用一个仅接受已验证合成发布引用的窄入口复用该内部事务追加逻辑，固定合成名称。真实 producer、现有真实采集名称和约束照常。
2. 固定载荷以 `manual_import` 合成原件发布，保留规范化关键词、ASIN/SKU、价格、宽度/安装/包装、原夹具参考版本、bundle ID、项目及有效成员。以经现有映射规则验证的 ProductCandidate 字段和规范化摘要回读，不伪造真实 `captured_evidence`、采集时间或 Amazon CDN 图片。没有图片时使用现有无图展示。合成 Observation 明确带来源标记。调研详情、规格依据与货源核对渲染「合成原件」及参考版本，不返回可点的 Amazon 或 1688 商品外链，也不把合成原件描述为公开商品页面。项目路由提供一个独立「合成样本批次」读模型，返回实际两件同批引用与 `IMPORTED` 状态。其只读 ID 与真实 `amazon/jobs/:id` 区分，前端确认、选品重进和最终复验均重新读同一项目合成批次，既有真实 `amazon/jobs`、外部 API、统计和回收从不消费该记录。
3. Research 在已经授权的原服务调用中显式向 `Sources.Observe`、`ReadSupplierMarket`、`ReadSupplierOffer` 传项目。只有合成键前缀、当前项目、企业/用户/有效成员、允许的样本角色、Collection 批次与原条目、publication ID/本地版本、确切 producer 与摘要都匹配时才从合成来源读。来源读取失败就拒绝，不退回真实采集读法。原 Research 关联、版本、货源核对、费用试算、人工批准和 Preparation 转入都沿当前合同。通用个人 Collection 与待上传允许现有授权成员看见带合成标签的原件和回执。收藏仍归该成员，归档或移动导致验收绑定无法精确复读就要求回原项目，不能跨项目作为 Research 证据。W1 只走现有确认后转入入口，实际提交 `[A]`，回读恰好一件。
4. 合成资料止于待上传。Listing 现有 `prepareOperationTx` 是直接操作及上传计划推进的共同写入点，检查范围内的全部选中来源，包含省略 `sourceIds` 的整批输入，在写操作及启动队列之前拒绝任何保留前缀来源。`UploadPlanInputs.ReadUploadPlanFacts` 提前拒绝用于安排下一步，`SourceSelector.SelectForExecution` 在 worker 与直接上传取原件时再次拒绝。该否决与验收开关无关。目标资料、图片与优化等旁路由下方关闭配置限制。客户端保持标识，禁止进入实际执行操作。以既有普通转入合同保存并读回，保留产品身份和版本，不额外实现通用上传保护框架。

## 运行准入和回退

- 当前只读事实为统一实例使用 `719908e69` 后端、`25fa5e336` 前端，Amazon `dataServices.executionEnabled=true`，无 SupplyChain 或官方店铺应用，AIWorkbench、ProductAgent 和图片生成均关闭。图片历史、采集计划与 BrowserCollector 配置存在。上次关闭态预检不适用于当前实例。
- 代码准入与现场开工分开。仅按已核验关闭配置启用本地验收模块。需同时关闭实时 Amazon、SHEIN 试取、AI、图片生成、官方店铺能力和 Listing 工作流，核对没有在途/待启动的受影响任务和隐式执行入口。缺少以上任一证据时仅做隔离开发验证。现行实时 Amazon 已启用，实际切换需先明确有边界的维护窗口与已授权的关闭配置。现有菜单、图片历史和保留卷不作为关闭对象。
- 导入前可以回退到原组合。导入后固定验收数据库不得挂到任何启用官方能力、且没有合成键拒绝的镜像上。常规回退使用已核对包含拒绝规则的候选加关闭配置，先关闭导入开关。原镜像只有在精确比对的离线配置下才可回退，且须重验 Listing 执行路由未装配、官方应用缺席、Amazon/模型关闭。严禁用当前实时开启配置启动回退流程。不能证明时停在停止验收功能，保留样本，不删除数据或恢复整库。所有复验以当时版本与挂载为准。

## 权限、事务、失败

- Bundle ID 由固定包版本与企业、用户、有效成员、项目的组合确定，发布 ID/批次 ID 按 Bundle ID 和固定角色稳定派生。样本 manifest 的固定载荷计算摘要，不接收调用方摘要。只认同一范围、同一内容的已发布回执。重复点击、并发和响应丢失均按同一键重新读取；同键不同载荷、部分已有或引用不符则拒绝，不覆盖、不重新补发或自动修复。原件不能被写成正常采集类型。
- 导入之前和在事务提交前做同一项目及成员权的现有授权检查。该项目与 Product 分库，复用已批准的现时授权检查及原回读边界，不建立跨库提交或 Saga。所有合成读取均检查当前项目、有效成员、收藏、publication ID、本地版本和摘要。撤权或项目归档后拒绝，保留底层历史。验收配置撤销后关闭受控读写，持久合成标记和实际执行拦截继续有效。
- 三件固定，总载荷不超过 64 KiB，导入限现有路由期限内单个 Product 事务，不启动队列、worker、自动重试、provider 或计费客户端。无运行时 DDL、保留库重建、旧兼容和数据迁移。

## 验证与交付

- 隔离验证空、完整、同键重试、同一成员在另一项目、错作用域、撤权、失败回滚与来源污染。检查 Product 仅多出三件合成发布，查询执行表、Money/Resource 和现有计费 intent 都没有新增。验证直接及计划执行调用在外部适配器前拒绝。
- 同一固定候选通过 CI，保留当前统一实例的在线采集菜单和独立 Amazon 页面。用同一数据在指定本地项目经现有入口核对 A 的宽度/安装方式、B 的包装来源。保存同一 1688 SKU/20 件，试算 USD 8.55 与 USD 20.00。人工仅批准 A；确认一次转入请求及回执只含 A，刷新后仍是同一件且原件 ID 与版本相同，B 仍待研究。外部执行、扣费及产品验收由用户决定。
- 当前准入仅允许设计与验证性调查。独立 Reviewer 检查实际 owner、租户授权、事务幂等、来源绑定、执行/计费隔离及与当前实例的组合边界。获得 `IMPLEMENTATION_READY` 后，原 Writer 才实施。
