# 市场调研与商业选品 V1：独立架构提案

日期：2026-11-05。Design Basis：**Independent Architecture（设计阶段）**。Admission：**NOT_READY**。产品与交互见[规划提案](../product/research-commercial-assessment-v1.md)。

## 1. Product Outcome、Authority 与范围

用户在会话 `01a1a9a0-fe9d-76d2-82eb-e0aeadb8073f` 明确授权“可以，你来先规划设计”。交付一个可评审的方案：SHEIN 美国研究方向 → 系统获准取数 → 重点商品/需求/评价依据 → 1688 可验证货源 → 带来源经营条件的贡献利润/保本测算 → 有依据的建议与人工决定 → 原商品准备。

当前授权只覆盖文档、只读调查、独立设计检查、提交/推送及关联 Draft PR。没有生产代码、DDL、运行配置、注册/购买、真实数据/模型请求、部署、合并或关单权限。

适用依据：[父 #1070](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1070)、[#1074 原范围](https://github.com/henryxu1008/shuomi-ai-commerce/issues/1074)、[当前领域 owner](project-boundaries.md)、[全新系统基线](../product/greenfield-no-legacy-migration.md)、[Product/Figma Authority](../product/final-ui-ia-authority.md)、[项目中心](ai-workbench-project-center.md)、[来源流程](source-business-flow.md)和[当前商品对比](product-selection-comparison-agent.md)。

Product/Figma Authority：既有 `31:463` 页面及产品提案 §3 的准确节点已只读核实存在、可见、非归档；新增 UI 仍为文字候选，未更新 Figma。#1075 中用户已确认的智能市场/我的智能体入口见其[固定版本 Authority](https://github.com/henryxu1008/shuomi-ai-commerce/blob/8d3b19df15eaf94c1030729006557496e0663f2a/docs/product/final-ui-ia-authority.md)。本设计不从 main 缺失的文档推导已有能力。

Must（本提案的待批准实施范围）：来源/商品/市场/观察版本可追溯；自动取数路径不以人工找齐资料替代；org+actor 私有与原成员权限；事实/假设/未知区分；精确货源/规格/数量人工确认；确定性金额计算；不重复付费/发布/转入；AI 建议与人工决定分离；明确硬性要求不能被利润或总分抵消。

Should：复用完整业务页面、明暗主题与 390px；无模型仍能读取事实、测算与人工研究。Out of Scope：全市场规模/销量预测、通用评分/检索/采集平台、新 IAM/钱包/调度器、浏览器桥扩展、自动同款/采购/支付/刊登、周期监测、团队共享、历史迁移。Accepted Risks：无新增风险接受；来源缺失/挑战、输入不足是诚实未完成，不构成降低 Must 的授权。

## 2. 当前事实与依赖

设计工作区基线为实际 main `05e5b14f7f0d3aa80cdccd8dc222d01916c019db`。本轮开始时 #1075 为 Draft/Open，HEAD `8d3b19df15eaf94c1030729006557496e0663f2a`；其研究实现未进入 main。下述“#1075 已有”只指该准确候选源码，不代表 main、保留实例或真实来源已交付。本轮不 merge/cherry-pick 其生产代码，不修改原 Writer 的分支/实例。

| 当前能力 | 可复用证据 | 本次真正差距 |
| --- | --- | --- |
| 私有研究项目 | `internal/aiworkbench/projectcenter`、当前项目合同 | Goal 是通用文本；需本范围结构化研究查询，不把 AI 自由文本当执行参数 |
| Amazon 数据采集 | `internal/product/dataacquisition/query.go`、`execution.go`，原 Temporal runtime | 查询/证据、ASIN、固定 PriceFen 和 consumer 均是 Amazon 语义；付费异步供应商不能直接塞入 Discover/Fetch 重试 |
| 来源发布/商品/批次 | `internal/product/sourcing`、Catalog、Collection 及 Product UoW | 新来源准入、市场/原商品 ID、变体、价格口径及原始观察需贯通，不只增加一个 HTTP adapter |
| 1688 详情与报价 | 当前 acquisition adapter、`sourcing/source_quotes.go`、`collection/source_quotes.go` | 有详情保存与价格/MOQ 投影，不证明已有自动搜索/以图搜或货源匹配；缺币种/规格时不能生成可用成本 |
| 私有候选/决定 | #1075 的 `internal/product/research` 及其准确证据合同 | Site/ASIN、研究持久化、前端 Zod 均需按新来源定稿；新成本/匹配事实由同一 owner 扩展 |
| AI 证据建议 | #1075 的 research analysis、Agent/Eino/配置/Resource | 冻结 v1 的 2–5 样本/固定工具不是联网调研或商业测算；不可偷加工具/调用/输入语义 |
| 转入准备 | 原 Collection/Preparation 的准确子集与回执 | 新来源的允许类型、schema、事务内检查、消费者需成对验证；不是链接跳转即完成 |

目前没有已核实的完整商业评估计算 owner 或成本评估合同；不能把现有报价展示当利润模型。正式实施依赖 #1075 合入或由原 owner 交付明确可消费的同一合同；不得另写一套研究系统。

## 3. 来源可得性与费用证据

以下为 2026-11-05 公开文档/公开 schema 核实；没有登录、元数据业务 API、样本商品请求或计费执行。供应商声明不是本项目实测或使用许可。

| 来源 | 已证实内容 | 未证实内容与阶段影响 |
| --- | --- | --- |
| ZenRows SHEIN | [美国站指南](https://www.zenrows.com/scrape/shein)列搜索/类目/详情 URL、美国地区参数及列表商品 ID/价格/币种/来源；[Fetch](https://docs.zenrows.com/fetch/api-reference)与 [Extract](https://docs.zenrows.com/extract/endpoints)有正式接口说明 | 当前账户对 `us.shein.com` 各页面类型就绪、分页、规格/评论字段、实际质量、保存/展示/AI 用途许可未验证；作为首选调查候选，未选购/启用 |
| Bright Data SHEIN 实时 | [公开字段目录](https://docs.brightdata.com/scrapers-full.json)的 `gd_mkv55s1f23yjzsl4ix` 只有 collect_by_url / discover_by_category_url；详情 schema 有价格/变体等，类目 schema 主要为 URL | 没有该目录的 SHEIN 关键词方法；price 类型结构、变体计费/上限、创建响应丢失核实及账户适用性未知；不推断类目方法支持搜索 URL |
| Bright Data 预采集 | [产品页](https://brightdata.com/products/datasets/shein)说明预采集与按需新采、刷新选项 | 预采集可能为数天至数月前；与实时 Scraper 分开，不作为当前小范围研究首选 |
| 美国需求趋势 | [Google 官方 API](https://developers.google.com/search/apis/trends)仍需 alpha 资格；[DataForSEO Explore](https://docs.dataforseo.com/v3/keywords_data/google_trends/explore/live/)可表达关键词/地区/期间及 missing | 服务商、账号/许可及执行未确认。归一化兴趣不是绝对搜索量或 SHEIN 销量，原比较范围不可丢失 |
| 1688 自动搜索 | 仓库已有详情路径；[Bright Data 1688 页](https://brightdata.com/products/web-scraper/one-six-eight-eight)描述 Scraper Studio 生成方案 | 公共完整预制目录未找到 1688 对应条目；Alibaba.com 条目不是 1688。官方关键词/图片接口、授权、字段与报价尚未核实；这是自动货源路径的真实缺项，不能以通用网页服务承诺已解决 |

公开参考费用：ZenRows 当前月付 B1 为 $19/45,000 credits，Fetch 每成功页 1/5/10/25 credits，Extract 当前不另加；404/410 也可能计费，见[计价](https://docs.zenrows.com/first-steps/pricing)。拟 3 列表＋5 详情最多 8 次、按 25 credits 预留为 200 credits；按 B1 折算约 $0.084 使用量，实际订阅最低支出仍是 $19/月，且未计趋势/货源/AI/税费。

Bright Data SHEIN [实时 PAYG](https://brightdata.com/products/web-scraper/shein)为 $1.50/千交付记录；发现/详情/变体计数须单独核实，不拿页面数直接换算商品数。预采集页起价 $250/10 万条，不套到实时 API。DataForSEO [Google Trends 当前价](https://dataforseo.com/pricing/keywords-data/google-trends)为 Standard $0.0027/task、Live $0.011/task（每任务至多 5 词）；充值/账户条件未核实，非本项目预算。

趋势拟优先评估 Live 以适合交互，最终由来源/预算决定；不自动切换服务商，不自动改为人工 CSV。评价仅保存合法可用的必要正文/评分/时间/商品绑定与覆盖范围，去除不必要的作者身份、照片等个人资料；来源未支持时保留 UNKNOWN，不宣称有痛点结论。具体 retention/删除与历史遮蔽沿适用许可，不能承诺永久保留。

## 4. Domain / Fact Owner 与调用路径

| 事实/职责 | 唯一 owner 与本范围拟增量 |
| --- | --- |
| 项目身份/目标/归档 | projectcenter；不复制项目元数据，不因关联授予来源权限 |
| 冻结执行查询、来源任务/远端 attempt/成本意图 | 现有 product/dataacquisition owner；补本范围 SHEIN、趋势和货源的明确任务类型/准入，不建第二任务平台。Research 只持有 queryRef/任务回执，不保存另一份可变执行查询 |
| 商品证据、发布与版本 | sourcing → Catalog → Collection 原 UoW；供应商 DTO 只在 integration；原商品 ID、provider ID 与市场分别表达 |
| 趋势/评价原始观察 | 拟由 dataacquisition 的窄类型证据持有，Product DB 内受限读写；趋势不伪装为 SourceEnvelope 商品、不进商品转入。Research 仅保存可授权的精确引用 |
| 货源匹配、条件版本、确定性评估/决定 | 拟扩展 #1075 的 product/research owner；只保存匹配关系、商家条件、被确认的输入证据及测算回执，不复制商品/趋势/模型输出 |
| 平台规则、实际店铺与正式报价 | 原 Marketplace/Store owner；通过窄只读 port 取得已存在且适用的信息。未有费用 reader 时显式未知，不虚构一个已存在费用引擎 |
| 数据/AI 费用与执行 | 原 Resource/Commercial/Agent owner；供应商原始 charge locator 属原采集任务。商业测算不写钱账本、不扣购买款 |
| 建议、历史、准备转入 | 原 Agent run、Research 人工决定、Preparation 回执分别保持 owner；不新建报告平台/同步器 |

建议调用链：Console/BFF → live identity/project scope → 数据服务命令 → 原任务/Temporal → 获准来源 adapter → 原发布或类型化观察 → Research 精确 reader → 货源匹配/经营条件 → 无副作用计算 → 可选 Agent 建议 → 人工决定 → 原 Preparation。`cmd/current-application` / currentapplication 只验证安装及注入依赖；业务规则不放到 composition root。

本批跨来源任务、Product 事务与 Agent/Resource 边界，不试图建立跨库 ACID。每个 owner 本地事实和命令回执同事务；跨 owner 消费原 receipt/ref。任务执行基础设施复用 Temporal；若原回执不足以完成具体交接，再在同一 owner 设计最小 outbox，当前不预建通用 Saga/outbox 平台。

## 5. 准确身份与输入模型

以下是设计语义，不是已发布 API/schema。

- `ResearchQuery`：projectRef、sourcePlatform、market、keyword 或 admitted categoryRef、字段集、页面/结果上限、sourcePolicyVersion；实际查询由 dataacquisition 服务端规范化并冻结，不接受任意可抓 URL。项目目标仍归 projectcenter；界面未提交查询草稿是 scoped 表单输入，不是第二份收费查询事实。Research 通过 queryRef/原任务回执关联执行，不在原键恢复时重建查询。
- `EvidenceRef`：scope + sourceKind + owner-issued locator + nativeProductID/market（商品适用）+ observationVersion + original digest + queryRef；原取得时间、原更新时间、我方读取时间分别表达，不补造原时间。
- `SupplierMatch`：candidateRef、supplier Collection/Catalog/variant exact refs、相同/不同/未知规格清单、数量与计价单位、提出原因、人工确认者/时间/revision。建议、确认、否决分开；图片或文本相似不证明同款。
- `CostInputVersion`：销售国家/平台/履约模式、商品/货源规格与采购量、各金额/费率/单位/适用条件、来源 ref 或商家假设、validAt/asOf、明确 UNKNOWN/NOT_APPLICABLE。不存在平台默认佣金、税率、退货率或隐含汇率。
- `Assessment`：绑定精确候选、确认匹配、输入版本、formulaVersion、资料 digest、风险依据与结果、当前适用状态、原命令回执；重算不覆盖历史，同键换输入冲突。

价格不能仅有数值：币种、计量单位、单件/套装、阶梯数量、优惠及税费口径均需明确。采购数量未达到 MOQ/所用价格阶梯时拒绝该报价的测算；不把少量测试套用大货价。货源可订数量不是自有库存。美国站 URL、供应商所在国、币种、销售市场不是可互相推导的事实。

Research 保存的条件版本是本次评估的业务输入快照，来源事实仍由原 owner 授权。旧输入用于原键/历史恢复，不刷新成当前来源；新观察或假设产生新版本。来源撤权后模型解释及可推导受保护内容整份遮蔽，不能用旧计算数字反向泄露。

## 6. 确定性单位经济模型

金额接口使用十进制字符串，最终金额为币种最小单位整数；中间运算用标准 `math/big` 的精确有理数或合格既有定点实现，不引入 float64 金额计算/新金额框架。先校验币种小数位、单位、费率范围、数量和日期，最后按明确版本舍入；源报价的 float/格式不直接成为财务输入。

定义销售尝试的有限经济情景 s：正常完成、退款/退货等。每个情景有明确概率或假设区间；合计须为 1。未知概率不默认为 0。买家代收代缴税与商家收入分离，费用按适用计费基底计算。

`R_s` 为商家承担优惠及该情景退款后的商品收入；`C_s` 为实际消耗采购/包装成本；`L_s` 为发出/退回/履约成本；`F_s` 为扣除确实退还部分后的平台/支付费用；`T_s` 为商家实际承担税费；`A_s` 为获客分摊；`X_s` 为汇兑额外成本。单位、币种和换算方向全部冻结。

商品毛利（本产品口径）＝Σ p_s × (R_s − C_s)。贡献利润＝Σ p_s × (R_s − C_s − L_s − F_s − T_s − A_s − X_s)，尚未扣企业固定费用。退款已反映在 R_s，不能又把整笔退款额当“退货损失”扣一次；退运费、未退佣金、损耗等才另列。采购成本是否可回收须有明确依据；不会自动把退货视为可再次销售。

区间应由完整且一致的经营情景计算；不得把互斥的所有最好/最坏输入乱拼。显示基准和保守情景、适用假设与改变售价/运费/退货条件的敏感性。缺少关键输入时 `INSUFFICIENT_INPUT`，允许展示已知条件下的部分成本但不输出完整利润/测试结论。

保本售价是使期望贡献利润 ≥ 0 的最低合法价格格点。V1 对已验证的固定＋比例费用可精确求解；分段报价只在现有规则能明确求值并证明适用时处理。非线性/不连续且无适用求解合同的费用输出 `UNSUPPORTED_COST_RULE`，不猜一个保本数。对 `aP−b≥0` 必须结合 a 的符号及合法价格域求可行集，区分无可行价格、全域可行和存在最低可行格点；例如 `0P−0` 全域可行，不能仅凭边际收入≤0 宣称无保本价。税费/广告/汇率缺项不借此清零。

示例仅为未来确定性测试输入，不是 SHEIN 费率或真实经营数据：销售币 USD、售价 30；采购包装 8、发出物流 6、获客 3；正常情景 95%，费用为售价的 10%；退款情景 5%，商品收入 0、采购不可回收、发出物流 6＋退运 2、平台费全部退还、获客 3；税费和额外汇兑成本明确假设不适用。

正常贡献＝10，退款贡献＝−19，期望贡献＝8.55，商品毛利＝20.50。此情景贡献函数 `0.855P − 17.10`，保本价为 20.00；19.99 仍亏损。若遗漏退款情景会误报为 10；若再扣 5%×售价则重复扣减。现有钱包/实际账务完全不受计算影响。

## 7. 状态、事务、幂等与恢复

下面是拟定行为边界；源任务各有自身状态，UI 不再存一个总“已研究”状态机。

| 操作/边界 | 正常结果 | 丢响应/并发/失败的归属 |
| --- | --- | --- |
| 保存匹配/经营条件 | 沿原 Project/来源 port fresh 核实 active/访问；自己的 CAS revision＋规范输入 digest＋command receipt 同 Research 本地事务 | 原键 lookup，异载荷冲突；Research 不保存第二执行查询，历史不因刷新换输入 |
| 发起外部取数 | 原任务先保存 scope/operation/query/budget/deadline 与 attempt 意图，再调用精确来源 | 本地幂等不等于远端幂等。结果不明保存 UNKNOWN；只有该 provider 明确支持原请求核实/幂等时才自动重试创建，不能凭 HTTP 方法或 Temporal 重试推定免费 |
| 轮询/下载 | 绑定原 provider job/snapshot ID、账号及请求 hash，限制次数/字节/期限 | 只重读已确认原任务；没有远端 ID 的未知创建不新建。取消不证明外部任务未完成/未收费 |
| 发布商品/关联项目 | 原来源发布/商品/批次在既有 Product UoW；Research 关联用原发布 ref、独立回执 | 发布成功而关联失败只重试关联；不得重采/重发布。部分结果按真实状态展示 |
| 保存评估 | 先通过既有 Project/来源窄 port fresh 核实权限/active并冻结输入，计算后在提交准入检查点再次核实 Project；Research 本地事务只 CAS 自己的候选/匹配/条件版本并提交结果、hash 和 receipt | 原键恢复只读已冻结结果；并发改条件则冲突，不重算后覆盖。Project 归档与 Research 提交不共享原子锁；检查顺序/并发沿既有公开合同在 K2 定稿，不宣称跨库 ACID。禁止跨 owner 原始 SQL |
| 生成 AI 建议 | 独立 prepare/start，绑定本次评估/证据 digest、原模型配置及预算 | 原 Agent UNKNOWN/receipt/finalizer 负责；不换 key、模型、资料重新计费 |
| 人工决定/转入 | 沿 Research CAS 与原 Preparation 的准确子集/版本/幂等 | AI 建议不写人工状态。转入未知回 Preparation 原键核实，不因 UI 返回失败重发 |

建议的评估只读投影：`NEEDS_MATCH_CONFIRMATION`、`INSUFFICIENT_INPUT`、`COMPUTABLE`、`STALE`、`SOURCE_UNAVAILABLE`；由当前引用/输入事实派生，不新建后台状态推进器。货源确认的人工事实不得从文本相似率生成。

本轮风险检查仅核对选定市场/模式/商品实际适用的现有硬性规则或明确证据，结果为 clear/blocked/unknown，附具体条件和来源。已证实硬性限制禁止本次商业建议给出“可测试”；未知如实列出。人工“批准测试”仍不是合规批准。不得擅自给所有原商品准备流程新增“商业评估必须通过/利润必须为正”的全局前置，改变该产品语义需用户另定。

## 8. 权限、经济准入与资源上限

身份及组织来自原 verified identity/live membership，project 为 org+actor 私有；管理员无权读取他人项目。每次新披露/执行/历史打开核实项目、全部来源、匹配/成本 refs 及 Agent run 的当前权限；切企业清除选择并拒绝迟到响应。供应商 Key/账号只由原受控凭据供给，不向模型/浏览器/日志披露。

来源 URL 由已准入查询构造或精确 owner locator 解析；限制 HTTPS、目标域/重定向及响应字节，沿已有出站/SSRF 合同。不得让模型提供任意 URL、工具、租户或付费上限。来源文本、评价、目标和反馈均是不可信数据，不改变 system 指令。

采集、趋势、货源、AI 分别冻结预算、供应商价格版本、最大请求/结果/变体数量和最长执行期。免费额度是账户事实，缺失时 UNKNOWN；不因“免费试用”跳过用户授权。本地系统服务价与供应商成本分别表达；当前 Amazon 固定单行价/consumer 不适用于新增来源。

现有 Resource owner 仍独占预留/结算；新 consumer 若需要由其合同明确单位和调用完成证明。尚未定义新 consumer 和预算前，只能设计，不能开池调用或伪造 0 成本。旧每项 ¥0.30/20 点预算不自动授予新来源/新模型消费者；未知费用/用量不释放预留。

拟定本范围限额（待产品/来源合同确认，未改生产）：2–5 个评估候选，单候选最多 5 条货源建议、最多 8 个经济情景；单次取数例算 8 个页面请求；输入/输出沿已有 32KiB 受限披露的可复用边界。完整合法身份/必要缺项装不下时拒绝，不先无界读取。具体查询行数、评论数、供应商请求 deadline、全链路费用、存储期限须在最终合同中准确列明，当前不以假定默认值标 Ready。

## 9. AI 输入、输出与现有消费者

AI 只读取准确候选事实、趋势/评价证据、人工确认的匹配、成本条件及已计算结果，输出观察摘要、支持证据、经营假设、未决项和有界建议。数字由确定性模块渲染；服务端校验 schema、每候选 evidence IDs 和范围，但不宣称引用有效即商业判断正确。

#1075 冻结 `product.research.agent@v1.0.0` 的工具和两调用协议不默改。商业评估需要版本明确的 operation/输入/输出消费者，准确 ID/version/能力映射在准入时固定，仍复用原 Agent/Eino/配置/Run/Resource，不另建 runtime。没有消费者、预算和UI映射批准时不注册空壳智能体。自动取数由受限任务执行，不给 LLM 自主扩搜/购买权限。

结果归原 run，Research 只留 locator。新建议不覆盖人工决定、不修改商品、不触发采购/付款/刊登；用户必须明确确认下一步。能力未开放/缺模型时仍可按权限查看事实、确定性测算和人工研究。

## 10. 实施顺序、验证与准入缺项

一个用户结果、一个主要开发分支/PR、一个 Writer 连续交付。当前只有设计文档，不为形式制造生产测试或新 runner。

建议实施提交边界：①确认依赖与中立来源/观察合同；②获准商品发现/详情/信号取数及原发布；③同项目候选与 1688 货源确认；④经营条件/确定性测算；⑤有界建议/人工决定/原准备贯通。边界是必要实现验证顺序，不代表其中的真实来源已获准调用，也不为每片另开 PR。

必要现有设施验证：源身份/市场不混淆；同键异载荷/并发 CAS；MOQ/阶梯价/单位币种及未知不补零；退款无重复扣减、上面的 8.55/20.00 示例及不可求解费用；未知远端创建不重复收费；发布后关联失败只重做关联；撤权后不披露冻结评估/AI；AI 假引用和越权动作拒绝；原子集转入原键恢复。实现时先写目标行为 RED 再最小 GREEN；此处只是未来验证设计，未执行。

真实用户路径由空项目开始，只填方向及自身经营条件，系统取数/匹配、确定性测算、人工决定到准备。用户或指定独立验证者确认实际效果；fixture/CI/AI 评审均不能代签。

| 未决项 | 需要的具体答案 | 责任及影响层级 |
| --- | --- | --- |
| K1 产品范围/首试类目 | 用户评审本提案，确定首试问题、必需来源/字段及商业判断深度 | 产品决定；影响最终合同/交付范围 |
| K2 依赖 | #1075 原 owner 给出实际可消费版本及合入/组合依据；不从候选推断 main 已有 | 研究 owner；影响正式组合实现 |
| K3 1688 搜索与评价/详情 | 准确接口/方法、权限、字段、样本覆盖及原结果核实方式 | 数据来源 owner；影响自动货源/痛点路径，不能人工资料替代 Must |
| K4 来源许可/预算 | 美国覆盖、资料使用/保存期限、价格/账户费用与各类上限；未知创建如何原单核实 | 用户与来源 owner；影响最终执行合同/真实开放 |
| K5 费用与计算口径 | 实际销售/履约模式、报价规则、来源、缺项及公式/舍入版本 | Product/Marketplace owner；影响确定性合同及完整商业评估 |
| K6 UI 与独立设计检查 | 文字流程评审后形成适用 Figma 增量，完成本轮高风险边界检查 | Product/Figma/独立 Reviewer；影响 IMPLEMENTATION_READY |

上述未决项是设计交付中的明确结果，不伪装为已完成，也不要求为了评审文档购买或真实调用。正式实现保持 Backlog / NOT_READY。Legacy decision：N/A；全新安装，复用当前 owner，不迁移/映射/兼容/双读双写。不使用本提案更新保留实例、旧库、凭据或 CI/保护规则。
