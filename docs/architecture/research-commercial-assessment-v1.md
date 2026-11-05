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
| 1688 匿名发现、详情与报价 | `sourcing/acquisition_plan.go`、`acquisition/a1688/browser/discovery.go`、`sourcing/source_quotes.go`、`collection/source_quotes.go` | 已有关键词/页面发现代码，读取一张匿名可见页的商品链接，再走原详情链；不证明真实成功率、完整分页、第三方 API 或货源匹配已开放。现有金额/MOQ 投影不能代替规格绑定的完整采购条件 |
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
| 1688 自动搜索/详情 | 现有匿名发现代码及下方 §3.1 的专用 API 公开文档；[Bright Data 1688 页](https://brightdata.com/products/web-scraper/one-six-eight-eight)仍只是生成式方案线索 | 公共预制目录的 Alibaba.com 条目不是 1688。专用 API 已有具体调查候选；规格/阶梯/MOQ 语义、许可、实际成功率与费用仍须按选定版本核实，尚未执行真实验证 |

公开参考费用：ZenRows 当前月付 B1 为 $19/45,000 credits，Fetch 每成功页 1/5/10/25 credits，Extract 当前不另加；404/410 也可能计费，见[计价](https://docs.zenrows.com/first-steps/pricing)。拟 3 列表＋5 详情最多 8 次、按 25 credits 预留为 200 credits；按 B1 折算约 $0.084 使用量，实际订阅最低支出仍是 $19/月，且未计趋势/货源/AI/税费。

Bright Data SHEIN [实时 PAYG](https://brightdata.com/products/web-scraper/shein)为 $1.50/千交付记录；发现/详情/变体计数须单独核实，不拿页面数直接换算商品数。预采集页起价 $250/10 万条，不套到实时 API。DataForSEO [Google Trends 当前价](https://dataforseo.com/pricing/keywords-data/google-trends)为 Standard $0.0027/task、Live $0.011/task（每任务至多 5 词）；充值/账户条件未核实，非本项目预算。

趋势拟优先评估 Live 以适合交互，最终由来源/预算决定；不自动切换服务商，不自动改为人工 CSV。评价仅保存合法可用的必要正文/评分/时间/商品绑定与覆盖范围，去除不必要的作者身份、照片等个人资料；来源未支持时保留 UNKNOWN，不宣称有痛点结论。具体 retention/删除与历史遮蔽沿适用许可，不能承诺永久保留。

### 3.1 1688 关键词与详情来源专项核实（2026-11-05）

本次追加授权是公开资料核实与接入建议。读取供应商文档、静态文档资产与公开报价，不登录、不获取账号 key、不点测试/演示、不请求商品/搜索/账户业务 API。文档中的响应均为供应商示例，不是本项目样本。没有安装或运行网上的 Skill/MCP/爬虫。

先纠正现状映射：本仓库已经有 `Canonical1688Discovery(DiscoveryKeyword, ...)`、原 Sourcing acquisition plan 和匿名浏览器 `Discover`，不能再写成“没有自动关键词搜索”。它只提取当前页面可见的规范商品链接，有限额、截断及挑战处理；真实覆盖/稳定性本轮 NOT_RUN。原详情与报价投影可复用，但规格、混批、交易场景和完整数量条件没有因此自动成为利润计算合同。现有代码不是采用新付费服务的授权。

| 路线 | 关键词与商品详情的公开证据 | 规格、阶梯价、MOQ 的证据及限制 | 时效/费用与本轮结论 |
| --- | --- | --- | --- |
| 1688 官方开放平台 | [官方商品文档入口](https://open.1688.com/api/apidocdetail.htm?aopApiCategory=product_new)和[跨境解决方案](https://open.1688.com/solution/solutionDetail.htm?solutionKey=1668600046917)存在；本轮 HTTP 读取只能核对入口/动态文档壳 | 未取得足以确认本应用资格、跨店关键词覆盖和字段合同的公开正文；不把第三方文章的接口名当官方已授权方法 | 当前准入、授权范围、额度、价格、刷新保证均 UNKNOWN；不能据“官方”推定免费或全站可搜。保留直连调查，不作为已经具备的接入前置 |
| API.SH 专用 1688 API | [搜索](https://api.sh/docs/1688-search)：`GET /1688/v1/search`，关键词、分页、排序和价格范围；[详情](https://api.sh/docs/1688-product)：`GET /1688/v1/product`，按 offer ID 或链接。均为 HTTPS、请求头 key | 文档列币种、计价单位、商品 MOQ、数量价格项、SKU 属性/价格/库存及供应商。**同一示例的两个价格项起量都为 2，价格却为 4.50/5.99，且对应不同包装 SKU 价格**；准确关系追查见 §3.2 | 保留候选检索/规格展示的技术评估选择，尚未选购/启用；不能由此认定数量阶梯或混批 MOQ 可计算，自动采购成本准入暂缓 |
| Onebound / 万邦 | [关键词](https://open.onebound.cn/help/api/1688.item_search.html)：`item_search` 有关键词、类目、分页、排序、价格及起订过滤；[详情](https://open.onebound.cn/help/api/1688.item_get.html)：`item_get` 按原商品 ID | 示例有 `skus.sku`、规格名、SKU/spec ID、库存、`priceRange` 和 `min_num`；但商品展示价与 SKU 价不同，`unit`/混批条件可空。可取得这些字段不代表报价已绑定所选规格 | 默认缓存，可传 `cache=no`；TTL/成功率/缺字段保证未证实。公开 A 类计价可参考，1688 两方法的实际等级/起充/非缓存收费需确认；作为第二候选 |
| TMAPI | [关键词](https://tmapi.top/docs/ali/search/search-items-by-keyword)：排序、MOQ/价格过滤，每页最多 20；[按 ID 详情](https://tmapi.top/docs/ali/item-detail/get-item-detail-by-id)/[按 URL 详情](https://tmapi.top/docs/ali/item-detail/get-item-detail-by-url)：列 SKU、价格、库存，可选代发价格场景 | 主站公开响应区本轮为 Loading，未据此核实阶梯/MOQ 的准确响应结构。其 [Postman V2](https://www.postman.com/tmapitop/tmapi-e-commerce-apis/documentation/r69mzsh/1688-api) 与主站版本/鉴权命名有差异，不能拼成同一合同 | 单次/套餐/起充、缓存期限和源数据时间未知；公开示例仍用 HTTP，实际 HTTPS base URL 未验证。补齐准确版本、TLS、字段与报价后再比较，不引用其他卖家的价格 |

TMAPI 文档更新日分别为搜索 2025-12-20、ID 详情 2025-04-18、URL 详情 2026-10-23；它们只说明文档时间。万邦详情示例带 2024-01-06 缓存日期且商品修改时间为空，也只是旧示例，既不能证明现货，也不能反推其当前业务数据全部过期。**文档日期、供应商抓取时间、商家改价时间和本系统收到时间分别处理。**

#### 时效、失败与费用

[API.SH 通用文档](https://api.sh/docs#caching)说明成功结果缓存 24 小时，同参数命中仍按次收费；`fresh=1` 跳过供应商缓存，响应有 `cached` / `fetched_at`。抓取时间不是商家承诺的报价有效期；不把网页价格当锁价订单。其 [SLA](https://api.sh/legal/sla)不覆盖源站拒绝/改版，PAYG 也没有月费 SLA，不能把平台可用率写成 1688 成功率。

[API.SH 端点页](https://api.sh/1688-api)当前搜索/详情均 1 credit/次；[价目](https://api.sh/pricing)为 PAYG $2/千 credits、最低充值 $15 得 7,500 credits；月费从 $50/50,000 credits 起，另有每月 1,000 免费 credits 的账号声明。缓存命中与已执行但找不到商品的 404 计费；拒绝执行/供应商自身失败不计费。价格不含适用税费，账号可用性/免费资格仍未验证，不以免费额度当获准预算。公开价格可能调整，执行前固定版本与上限。

万邦 [2026 年 10 月公开价格 PDF](https://www.onebound.cn/daigou/api-list.pdf)列 A 类标准搜索/详情按量 ¥0.022/次；包月 ¥1,500、每天 5,000 次、1 年起租，按其标价最低合同量为 ¥18,000/年，未含增值税。B 类 ¥0.05/次、C 类 ¥0.09/次不能套在标准方法上；同名接口按平台分级须确认。包月不能据月单价视为可随时取消的小额试用；按量最低充值、失败/空结果/非缓存是否扣费仍 UNKNOWN。PDF 中官方 OAuth/店铺订购/代接费用属于另一服务路径，不当作 1688 关键词访问的官方统一价格。

有界费用例算（未来试验建议，**本轮没有授权或执行**）：同一研究最多 5 个已冻结关键词页、25 个去重详情，共 30 次；在 API.SH 当前 1 credit/次下为 30 credits，PAYG 使用量折合 $0.06，实际首次付费支出仍至少 $15 加适用税费；月费摊销 $0.03 不是最低付款额。万邦两方法均确认 A 类时相同 30 次为 ¥0.66，首次充值额未知。刷新、运费、图片搜索、评论、SHEIN、趋势和 AI 不含在内，需另有明确上限；结果不足不自动扩页、重试或换供应商。

#### 可落地的接入建议与字段准入

建议先把 **API.SH 搜索＋详情**保留为候选检索/规格展示的可撤销技术评估选择，万邦为比较候选，TMAPI 待补合同；§3.2 追查后不再把这项选择理解为自动采购成本准入。这不是正式供应商决定。两个明确 HTTPS 方法、可见费用/缓存时间/按请求取回结果有利于定义有界路径，但不证明价格规则完整，也不宣称其质量、权利或商业稳定性已优于其他服务。正式来源选择仍须确认使用/保存/展示/AI 用途许可与账号条款。

未来实现顺序沿本提案 owner 和已有完整流程，不建通用抓取平台：冻结关键词/分页/费用范围 → 搜索得到原商品 ID/来源链接 → 去重后只取有界详情 → 经已准入 adapter 保存原来源证据 → 人工核对规格与采购条件 → 把确定可用的单价输入现有商业评估提案。匿名发现仍走原 Sourcing plan；付费来源如何组合原入口及 §4 的 dataacquisition/Resource owner 是最终合同项，不能把付费 HTTP 调用直接塞进匿名发现或已有 Amazon 重试接口。

| 进入成本评估前所需信息 | 处理原则 |
| --- | --- |
| 原商品/店铺、SKU 或 spec ID、具体属性、来源版本 | 保留原身份并绑定选择；不能用标题相同/图片相近认同款，不能把 SKU ID 用浮点数保存 |
| 单价、币种、计价单位、采购场景 | 批发/代发/促销区别表达；缺币种/单位/适用场景不从站点或最低价猜测；金额按确定性十进制解析 |
| MOQ、整批/单 SKU 起量、混批条件、增量/包装倍数 | 搜索过滤值不是可执行采购条件；不能把商品 MOQ 自动复制为每个 SKU 的 MOQ；供应商未给的规则保持未知 |
| 阶梯的起量、适用规格、单价与条件 | 只在同规格/同单位/同场景且区间明确时选适用价格；同起量异价、缺少绑定、重叠冲突和混批不明均不自动选最低价 |
| 来源抓取/缓存时间、商家改价时间、收取时间 | 已知时间分别记录，商家未给则未知；浏览可展示缓存年龄，确认经营条件时拟显式刷新所选少量货源并重新核对差异，未经批准不后台刷新或扣费 |
| 重量/体积、国内运费、跨境物流与税费 | 商品详情/中国境内运费不能充当到美国的完整到岸成本；缺项沿 K5 保持未知 |

公开 API.SH 示例只证明有相应形状，§3.2 已把价差对应到袋装/盒装 SKU，但区间生成原因、数量适用与混批规则仍须靠准确解释/可复核样本收敛；万邦也需核对商品最低价与 SKU 价的关系。上述规则是现有“未知不补零、准确版本/规格绑定”的来源细化，不是新 schema、计算实现或额外验收平台。

费用恢复仍沿原采集/Resource 合同：[API.SH 恢复说明](https://api.sh/docs#fetch-a-past-response-again)允许凭已获得的请求 ID 在 30 天内免费重取原响应；它不证明客户端超时且未拿到 ID 时可按本地键找到唯一请求。该情况保留 UNKNOWN/预留并按原单核实，不自动再发 `fresh=1`、不重建任务、不承诺远端幂等。缓存命中也不是免费恢复。客户端仅服务端持 key、验证 TLS、限制超时/响应与并发，前端不持密钥；万邦 URL key/secret 需脱敏，不能照抄关闭证书校验的文档示例；TMAPI 未核实 HTTPS 前不传凭据。

API.SH [隐私说明](https://api.sh/legal/privacy)表示会保存请求参数/响应并可能归档用于丰富其公开数据库，可申请退出。来源搜索只发送获准关键词/商品 ID，不上传企业内部利润、完整研究目标、客户或其他私有经营资料；若业务要求不可被其归档，先取得适用退出/保留条款再开放，不默认接受该风险。其 [使用条款](https://api.sh/legal/terms)也没有保证源数据准确或自动授予原平台资料使用权。

本轮结果：**关键词/详情专用接口及参考费用已取得公开证据；稳定性、真实字段完整度、规格/阶梯/MOQ 语义、有效报价、适用许可/账号费与超时核实仍 UNKNOWN。** 后续若另获有界调用许可，可用同一小类目核对若干规格/阶梯/混批样本及记录真实费用，不购买年度套餐；该建议不是当前执行授权。本轮不新增测试 runner，不将 K3/K4 标 PASS，正式实现继续 NOT_READY。

### 3.2 API.SH 规格、价格与 MOQ 关系追查（2026-11-05）

追加核实仅使用[详情公开文档](https://api.sh/docs/1688-product)、[公开 OpenAPI 静态文件](https://api.sh/openapi.json)、[变更记录](https://api.sh/changelog)及[官方 SDK 说明](https://api.sh/docs/sdks)链接的 npm `@api.sh/sdk@0.2.0` 公开类型声明。没有购买、登录、请求商品/搜索/账户 API、执行 Playground 或联系服务商；SDK 只解读发布包内文本，没有安装或执行。OpenAPI 核实副本为 21,997,616 bytes、版本 1.0.0，SHA-256 `b0cb482dfe3434b39ba15b4e5215742378d3e2f57d84b5116bbd831eb38b71c4`；该版本号不是细分字段合同已冻结的证明。

**已观察事实：**同一文档商品的币种为 CNY、单位为“个”、商品级 MOQ 为 2；价格区间列两项（起量 2、价 4.50）和（起量 2、价 5.99），两项都只有价格/起量，没有 SKU/spec 引用。示例 SKU 对照如下，价格/库存都只是供应商静态示例，不是当前有效报价。

| 示例 SKU ID | 包装 | 适用机型 | 展示单价 / 折扣价 |
| --- | --- | --- | --- |
| 6220283719097 | 袋装 | iP15proMAX | 4.50 / 4.50 CNY |
| 6220283719112 | 盒装 | iP15proMAX | 5.99 / 5.99 CNY |

**原因解释的证据层级：**两个金额与两种包装 SKU 的价格逐一一致，支持“不同包装规格造成价差”的解释；相同数量门槛本来不要求不同 SKU 同价。但这仍是基于静态样例的推断，不能据此确认供应商内部究竟把 SKU 价差合并为商品区间、归一化过程丢失规格维度，还是样例本身错误。公开资料没有给出原始 1688 字段到此区间的映射，不能把推断写成已确认根因。尤其不能按数组顺序配对 SKU，或将最低价用于所有规格。

继续追查得到的限制：

- OpenAPI 的 200 响应 schema 只有开放 object（`additionalProperties: true`），没有定义 SKU 与区间的关联、MOQ 的作用范围或混批/倍数规则；文档示例的两条 SKU 也没有这些数量条件。缺少承诺不等于每个真实响应必然缺字段，但不足以建立计算合同。
- 商品专有入参只列原商品 ID 或 URL，未列 SKU、采购数量或交易场景的报价参数；不能从现有公开合同得到“指定规格＋数量的已核实可执行报价”。这不否认返回完整阶梯后可以本地计算，而是当前没有完整规则证据。
- HTML 中“show full response”只是展开同一截短示例；静态示例及 OpenAPI 都保留“其余列表省略”，没有 `raw`。它们是同一供应商示例的重复呈现，不是多个独立商品样本，也不是商家原始数据。转到 Playground 会执行真实请求，本轮未做。
- 官方 SDK 的商品方法返回通用 `ApishResult`，其类型为 `Record<string, any>` 加调用元数据，未补充上述价格关系；读取 MIT 客户端不等于取得服务端采集/归一化实现。公开 changelog 只找到 2026-10-29 新增两个 1688 端点，未找到阶梯/MOQ 映射说明或针对此问题的修复记录，不能用 SDK/变更页替价格语义背书。

据此给出本次用途判定，**不改变“未知不补零、货源/规格人工确认”的 Must，也不把人工试算当自动货源路径的替代交付**：

| 用途 | 判定及约束 |
| --- | --- |
| 候选检索、规格对照、展示来源参考价 | 可保留为技术评估候选；本项目真实覆盖/成功率仍 NOT_RUN |
| 选定 SKU 的条件成本试算 | 只有人工明确确认同规格、单位、采购场景和数量适用价后，才可作为有版本的参考/经营假设输入；未核实数量条件时不得标“满足 MOQ”或“有效采购成本”，不能自动套量价折扣 |
| 自动阶梯采购成本、跨 SKU 混批 MOQ 判断 | 当前公开证据不足，暂不准入。不能按两项相同起量取最便宜一项、猜测数量段或将商品 MOQ=2 当作每个 SKU 买 2 件/任意混买 2 件 |
| 完整利润结论 | 价格条件未核实就保持条件性/未知；供应商规格参考价不能补齐到美国的物流、税费、退货或其他费用 |

错价影响的纯算术示例：若给盒装错套袋装参考价 4.50，而盒装示例价为 5.99，会低估 1.49 CNY/件；按 100 件演示差额为 149 CNY。**这不是 100 件的有效采购报价，也没有证明该数量可按展示单价成交。**

后续需取得的最小补证已经明确，当前没有对外发送询问：①区间是 SKU 价差还是数量阶梯，原字段映射及与 SKU/spec 的关联方式，同起量异价为何产生；②商品 MOQ、单 SKU MOQ、混批范围、增量/包装倍数各自的含义，缺失如何表达；③指定 SKU 的普通/折扣价格在何种采购数量与交易场景生效，给出未截短、含原始字段的解释样例。供应商可先提供现成静态样例与书面定义，无须为取得定义先购买或执行真实调用。

结论：**规格价差有强对应证据，区间生成根因与完整数量规则仍 UNKNOWN；API.SH 暂只保留检索/规格资料评估候选，不能宣布其已具备自动成本测算条件。** 这是既有来源准入缺项的具体化，不是新运行合同、schema、计算实现、额外验收平台或新增风险接受；K3/K4 和正式实现继续 NOT_READY。

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
| K3 1688 搜索与评价/详情 | §3.1 已核实专用关键词/详情文档；仍需选定版本的规格/阶梯/MOQ 语义、实际样本覆盖、评价原文与原结果核实方式 | 数据来源 owner；公开证据不是运行 PASS，不能人工资料替代 Must |
| K4 来源许可/预算 | 美国覆盖、资料使用/保存期限、价格/账户费用与各类上限；未知创建如何原单核实 | 用户与来源 owner；影响最终执行合同/真实开放 |
| K5 费用与计算口径 | 实际销售/履约模式、报价规则、来源、缺项及公式/舍入版本 | Product/Marketplace owner；影响确定性合同及完整商业评估 |
| K6 UI 与独立设计检查 | 文字流程评审后形成适用 Figma 增量，完成本轮高风险边界检查 | Product/Figma/独立 Reviewer；影响 IMPLEMENTATION_READY |

上述未决项是设计交付中的明确结果，不伪装为已完成，也不要求为了评审文档购买或真实调用。正式实现保持 Backlog / NOT_READY。Legacy decision：N/A；全新安装，复用当前 owner，不迁移/映射/兼容/双读双写。不使用本提案更新保留实例、旧库、凭据或 CI/保护规则。
