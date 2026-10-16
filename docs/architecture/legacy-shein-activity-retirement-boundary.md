# 旧 SHEIN 活动报名、促销与限时折扣编排退休边界

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **DESIGN_ONLY / 待用户确认；没有实施准入**。本候选只提交本设计和索引，不修改生产、测试、guard 或 workflow。
- 调查基线：`main @ 77f5f933baf031034c68bc9dba310bbcdaea0384`。#743 的合并与 push/main CI 归原 PR；不作为本设计候选的直接验证结果。
- 本轮 Design Basis: **N/A**，原因是只读调查与纯文档，不改变业务、领域合同、状态、权限、持久化或外部副作用。未来源码实施拟走 **Reuse Existing Architecture**，须先经用户确认 §5、§6、§6.1、§7，并在 Issue 记录引用、不变量及 Ready；当前不是 IMPLEMENTATION_READY 或 Ready。
- 用户边界：保留营销 API、有效规则、共享合同、两种 SHEIN 属性类型、PriceManager/PriceCalculator、当前采集、Store 观察同步、独立 SHEIN 登录 worker 及其 Store 同步；两套 workflow 不改，不操作实例或数据，不增加内部试用前置。

## 1. 用户结果、Scope 与 Out of Scope

使用者是维护内部试用源码、平台 API/规则与独立登录 worker 的维护者。交付是可逐项确认的服务退出、规则保留和持久化边界，避免复活没有当前装配的活动服务，也避免以“整包无入口”为由删除有效公式、库存与每 SKU 安全规则。本轮没有新增试用阻塞或用户操作前置。

调查覆盖 `internal/shein/activity` 的活动报名、常规/限量促销配置保存与开启、活动快照会话、限时折扣查询/算价/创建，以及专属 Store/Mapping/ProductData reader。仅设计源码退休，不开发新的营销产品、UI/菜单、定时执行器、批量报名、恢复/幂等系统或验证平台。无需新开源依赖，复用原 API、Go、现有规则和测试工具。

不读取真实配置/凭据、数据库、Redis、Profile 或平台状态；不启动命令 main，不执行平台 dry-run、实例启停/部署、数据迁移/清理/补写、活动开关/撤回/重放；不处理其他活跃批次。

## 2. Design Basis、Authority 与不变量

适用依据：

- [全新系统基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)：仅 EXTRACT / RETIRE，不建立旧数据迁移、ID 映射、旧 Service 包装、fallback、双读双写或第二事实源。
- [旧 CLI](legacy-listing-cli-retirement-boundary.md)、[control-plane](legacy-listing-control-plane-retirement-boundary.md)、[数字执行链](legacy-numeric-task-execution-retirement-boundary.md)、[任务持久化](legacy-numeric-task-persistence-retirement-boundary.md)、[商品/库存同步](legacy-numeric-product-inventory-sync-retirement-boundary.md)、[自动核价](legacy-auto-pricing-retirement-boundary.md)：旧入口与执行闭包已经退出。用户确认本设计后，仅替代这些批次对 §5 活动编排“保留待后续决定”的部分；历史矩阵及其检查证据不重写。
- [独立 SHEIN worker 身份/持久化](shein-worker-identity-persistence-boundary.md)：登录账号、Cookie/attempt、Redis/browser/Profile、同 tenant 查重及 Store 同步保持原 owner、格式和恢复边界。
- [当前 Store](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[准备接线](store-observations-preparation-runtime.md)、[当前 SHEIN 业务](my-supply-chain-shein-v1.md)：当前 Organization、连接、观察、Submission、UNKNOWN/retry 不交给旧活动服务。没有 UI/IA 变化，Figma N/A。

拟实施只移除无装配的旧编排，将混合服务中的确定性行为解开到**本包已有规则文件/纯函数接缝**。不保留空服务/receiver 壳，不给当前业务新增 activity import，不把这些规则当作已开放营销能力；将来新产品仍须由当前 marketplace owner 单独设计。现有配置/价格/校验规则职责不变，不新设当前事实 owner、公共当前领域合同、状态机、事务、授权/tenant、幂等键或 recovery owner。

基线活动包为 17 个 Go 文件（11 生产、6 测试）；11 个生产文件共 3,017 行，混合文件拆解可能命中生产变更风险提示线。以上既有 Hard-Cut、原 API/规则与独立 worker 边界覆盖本次源码退出，不按行数拆 PR 或搬目录绕过准入。若规则抽离实际要求新当前领域合同、事实/权限/状态/外部副作用边界，停止该部分，先走适用 Independent Architecture；不能用 N/A 或 Reuse 掩盖新架构。

Must：§5/§6 保留能力和有效安全断言不减；没有新活动执行入口或旧 owner 依赖；两套 workflow、必需/安全门禁和保护不改；源码缺席不等于运行态/数据清零。Should：当前依赖图、guard 和源码一致。Out of Scope 如 §1，无新增 Accepted Risk。Threat Model 限误删能力、丢失安全规则和错误接管平台/持久化，不追加全局 hardening 或试用验收要求。

## 3. 实际消费者：合同 → 实现 → 注入 → 入口

基线上执行 `go list -f '{{.ImportPath}}|{{join .Deps ","}}' ./cmd/...`，逐根核对全部 **31 个**命令，没有命令依赖 `internal/shein/activity` 或 `internal/shein/productsync`。未运行命令 main。全仓 Go import、构造器和符号检索结果如下；同包测试和可编译不视作当前产品入口。

| 合同/实现 | 基线实际消费者/注入 | 拟处置与限制 |
| --- | --- | --- |
| ActivityRegistrationService → activityRegistrationServiceImpl / NewActivityRegistrationService | 无包外 import、生产构造或当前命令注入；同包测试构造实现 | RETIRE 服务、Store/Mapping ports、结果包装和所有 IO 编排，不留 alias/stub/转发构造器 |
| RegisterPromotionActivity / RegisterPromotionProducts | 查询候选后构建 SaveConfig，按 REGULAR/LIMITED/BOTH 保存，分页 GetConfigList 找 ID，再 UpdateConfigState 开启 | RETIRE 调用顺序、分页/开关/结果聚合；营销 API 的请求、响应、常量、client 与 wire 测试全部保留 |
| PromotionRegistrationBridge / SessionFactory / Session | 无外部生产消费者；session 缓存 strategy/config/goods，测试验证分块快照和 activityKey | RETIRE 接口、会话、内存快照 owner、key/时间命名和专属 fixture；不创建持久化营销 session |
| CreateTimeLimitedDiscountActivity / createPromotionActivityFromPreparedGoods | 旧 Store/候选读取 → QueryPromotionGoods → CalculateSupplyPrice → 校验 → CreateActivity → 结果检查 | RETIRE IO orchestration；独立的价格/风险/库存/每 SKU 校验及纯请求投影按 §5 EXTRACT |
| RegisterMixedActivity / profit reader | 前者当前直接返回 0,0,nil 并记录未开放；后者读取旧 Mapping/成本后过滤 | RETIRE，不把占位成功当能力，不补齐混合报名或读取回退 |
| ProductDataHelper、旧 mapping/store 转换 | listingadmin ProductDataRepository 按 Store、SHEIN、ShelfStatus 读取 Attributes；Store/Mapping 窄 port 及旧 runtime service 回退；只被本包使用 | RETIRE reader/构造器/JSON 查找及转换；仅保留已测试的按 SKU 提取 Amazon 成本的无 IO 行为 |
| 价格/配置/库存与平台规则 | 本包原测试消费纯函数或通过旧 receiver 间接消费；没有当前运行装配 | 保留规则本体并将有效断言改到无 IO 接缝，不能因旧 service 测试退出而一起丢失 |
| marketing.MarketingAPI → marketing.Client / NewClient | api/marketing 独立实现与测试；activity 是一个闲置 caller；client.go 自身不依赖旧活动包 | 原样保留整个营销 API、全部端点/DTO/常量/例子/测试；不因少一个 caller 收缩平台合同 |
| productsync 两种属性 | EnrichedSkuInfo / EnrichedSkcInfo 被本包 helper、price calculator 和测试使用 | 完整保留 product_sync_types.go 及字段/JSON tags；无 IO SKU 成本规则继续可用，不恢复旧同步服务 |
| 当前采集/观察、PriceManager/PriceCalculator、独立登录与 Store sync | 各自原 owner/接线，不经此活动包；价格查询与 marketplace 公式仍有独立源码合同 | 全部保留，不从本批推导共享仓库、字段、API 或配置可以删除 |

架构问题是平台副作用与规则、旧仓库 reader 及内存会话混在一个服务。SaveConfig 后再分页找 ID 并开配置，以及 CalculateSupplyPrice 后 CreateActivity，都不是跨平台/本地共享事务；内存 result、命名 key 和 session 不能作为 durable receipt 或 exactly-once 依据。源码缺少当前装配也不能证明旧实例/平台事实不存在。处置是退出闲置编排、保留独立规则/API；不包装旧服务或追加新的恢复系统。以上为静态协议边界，未断言真实事故，不作为独立试用的新前置。

## 4. 持久化、事务与外部副作用 owner

| 边界 | 基线职责与 owner | 保留/删除条件 |
| --- | --- | --- |
| listing_store | listingadmin StoreRepository/StoreAPI；旧活动 finder 或 runtime StoreService 只读并投影 StoreInfo | 仅删除 activity caller、回退与投影；原模型/API/tenant/权限、字段、事务及测试完整保留，独立 worker Store 写回/查重不改 |
| listing_product_import_mapping | listingadmin FindLatest / scoped query / API；活动读取平台商品、Store 对应成本等 | 只退出 caller/DTO 转换；保留仓库、模型、ImportTaskID 等字段和 scoped 测试，不导入当前 Product/Submission |
| listing_product_data / Attributes | listingadmin ProductDataRepository 拥有持久化；ProductDataHelper 分页读 Store 商品并解析两种属性 | reader 退出；原表/schema/索引/行/API/owner/tenant/Attributes、属性类型和有效共享测试不动；成本纯函数只处理传入值 |
| OperationStrategy / 配置 | root listingruntime 共享输入合同，含活动模式、折扣、利润率、库存比例与 StoreID | 保留所有共享类型/字段、loader/validator；无 IO 规则只消费原输入，不新建策略表或接管策略存取 |
| SHEIN 报名配置/开关/活动 | 平台拥有 SaveConfig、GetConfigList、UpdateConfigState、CreateActivity 产生的事实；本地返回对象与内存 session 无 durable 写入 | 仅退出调用编排；保留 API，不调用、不关闭/删除平台配置或活动，不猜测结果，也不转交 retry/UNKNOWN/补偿/恢复 |
| 当前业务与独立登录 | 当前 Organization/Store/Observation/Submission；原 worker 的账号 DB、Cookie/attempt/stream/Lua、锁、browser/Profile 及 Store sync | 原 owner/格式/状态/事务/权限/恢复不变；不把旧营销快照映射进当前系统或登录会话 |

失败、响应丢失、重试、重启、取消、并发和恢复不建立新协议。源码退出不表示在途写入完成、失败、可重试或已回滚；最后写入及历史恢复仍归原平台/运行 owner。没有数据库 DDL、迁移、数据清理或真实状态扫描，不能从“不再读取”推出“可删除持久化”。

## 5. 待确认的精确源码删除与保留矩阵

以下均为**拟议**实施范围。路径相对 `internal/shein/activity/`，除明确列出的退出/抽离部分，其余保留。不以整个活动目录缺席为目标。

| 对象 | 待确认处置 | 删除条件 / 明确保留 |
| --- | --- | --- |
| registration.go | RETIRE 完整文件，**先**将下列无 IO 规则移入已有 registration_config.go：ensureLimitedPromotionDropRatesGreater、validateAutoPartakeDiscountsForStrategy、autoPartakePriceModeFromStrategy、autoPartakeActivityTypesFromStrategy、autoPartakeStockRatioFromStrategy、autoPartakeRequiresStockRatio | 服务/impl/constructor、Bridge/Session interfaces、PromotionRegistrationResult、ports、fetch/save/find/open 编排、promotionConfigIDForActivityType、单值 activityType 包装及 Store/Mapping 转换全部退出；不能为保留测试留旧服务壳 |
| mixed.go / profit.go | RETIRE 两个完整文件 | 占位混合报名、旧 pipeline 与 Mapping reader 随旧服务退出，不补齐，不替代成新 owner reader |
| product_data_helper.go | RETIRE 完整文件；仅将 AmazonCostBySKU 的确定性映射抽离为 price_calculator.go 内无 receiver 私有函数 | helper/constructor/repository 字段、分页/JSON/首个成本查找退出；保留 SKU trim/upper、缺失/非正成本过滤的原规则及两种属性合同 |
| price_calculator.go | 仅 RETIRE buildCalculateRequestWithPriceMode 及 reader 专属接线/imports；加入上一行无 IO 成本映射 | 保留 calculateActivityPrice / Discount / Profit / Breakeven、capSHEINActivityPrice、calculateProfitRate、promotionSKUUSSupplyPrice / MaxUSSupplyPrice。80% 降幅上限、分币向上取整、不可盈利/非正值处理、SKU 无 SKC 供货价回退不变 |
| registration_config.go | RETIRE buildActivityConfigs、buildActivityConfigsWithStrategy、buildActivityConfigsByProfit 三个 DB/旧服务 builder；RETIRE buildTimeLimitedDiscountConfig、defaultActivityStartLeadTime、GenerateActivityNameForActivityKey | 保留 minimumPromotionSKUDiscountRate、normalizePromotionSKUCode、firstAvailablePromotionSalePrice；将 buildActivityConfigsFromProvidedProducts、dropRateFromProvidedProduct、calculateEquivalentDropRate、calculateActivityStock 解为无 receiver 函数，并保留本表首行规则。只接传入值与需要的日志参数，不接 repo/API/StoreService/context.Background |
| promotion_create_products.go | RETIRE 三段 createPromotionActivityFromProducts / NewPromotionRegistrationSession / createPromotionActivityFromPreparedGoods、promotionRegistrationSession 及其方法、buildPromotionCreateConfig、clonePromotionOperationStrategy | 保留 applyPromotionCreateConfig、selectedPromotionSKCs、filterPromotionGoodsBySKC、promotionGoodsFromProductSnapshots、enrichPromotionGoodsFromProductSnapshots、configCurrencyOrDefault、promotionSKUPriceInput(s)、normalizedPromotionSKUCode；buildCalculateRequestForPromotionProducts 解为无 receiver 纯投影。promotionFloat64Ptr 只作测试 fixture 时移入原测试，不保留专属生产壳 |
| time_limited.go | RETIRE queryPromotionGoods、calculateSupplyPrice、createTimeLimitedDiscount、queryAllPromotionGoods、buildQueryRequest、checkCreateResult、GenerateActivityName、CreateTimeLimitedDiscountActivity、noAvailablePromotionProductsError | 保留 isTimeLimitedDiscountValid、原 95% 常量、过滤原因上限及两个 append helper；validatePriceRisk、buildCreateActivityRequest 解为无 receiver 的无 IO 规则/纯 DTO 投影，日志仅显式输入。保留每 SKU 严格小于95%的十进制边界、风险/警告、库存/平台上下限、白名单、500商品上限、无效/缺失价格排除及 filter reasons |
| config.go | RETIRE 完整文件、导出的 TimeLimitedDiscountConfig、DefaultTimeLimitedDiscountConfig 和 Validate | 全仓没有包外消费者；不能因旧默认值/StartTime==EndTime 测试保留调度 DTO、日期/命名生命周期或配置 owner。纯规则必要输入按下段剥离，不留原类型的 alias/wrapper/字段复制壳 |
| validation_utils.go | 原样保留 | ValidateDropRate / CalculateDropRateFromDiscount、80% 上限与常量保持，不依赖旧调度 DTO |
| errors.go | 仅保留 ErrProductPriceRisk；RETIRE 四个 ErrInvalidActivity* / ErrInvalidTimeZone 配置错误及 ErrNoAvailableProducts、ErrInsufficientStock、ErrActivityCreationFailed、ErrActivityAlreadyExists 和旧调度器注释 | 配置错误具体为 ErrInvalidActivityName、ErrInvalidActivityTime、ErrInvalidActivityTimeRange、ErrInvalidTimeZone；均随退出的 Validate/生命周期走，不以导出名字推定共享合同；实施前全仓刷新无外部消费者再删 |
| 营销 API、有效平台规则与共享合同 | 原样保留 internal/shein/api/marketing 整包、client/端点/凭据/错误合同/原 wire 测试、当前 marketplace/publishing 规则、listingadmin/root listingruntime 与配置 | 不改 API 签名、请求/响应、开启状态/活动类型常量、Payload 或已有规则；不从一个 caller 退出推导 DTO/Store/Mapping/策略字段可以删除 |
| 用户明确保留能力 | 两种属性及产品同步 client-boundary 测试、所有 PriceManager/PriceCalculator、当前采集/Store observations/独立登录及 Store sync 完整保留 | 不创建 activity 运行入口，不改登录 worker/观察/价格能力的代码、配置、持久化或恢复；两套 workflow 零改动 |

以上 EXTRACT 只剥离有效行为与旧 receiver/IO 的依赖，保留既有纯规则职责，不创建营销应用接口、wrapper、第二套公式或运行接线。不能原样保留旧调度 DTO 再将其称为规则合同：calculateActivityPrice / applyPromotionCreateConfig / 请求投影 / validatePriceRisk 等只消费必要的包内私有值输入或显式参数。私有规则字段仅限 PriceMode、DiscountRate、MinProfitRate、FixedPriceAdjustment、StockLimit、StockPercent、FilterSkcList、AllowRiskProducts、MaxWarningValue；不含 StoreID、时间/名称生成或任何持久化/执行状态，不导出新公共领域合同。只保留实际被上述规则读取的值，不复制旧 DTO 全字段或其 Validate/default 生命周期。

请求元数据沿已有 marketing.CalculateSupplyPriceRequest 和 marketing.CreateActivityRequest 的值输入传递：纯 builder 只填充对应 SKU/价格/库存/过滤结果，不生成日期或活动身份、不读取实例配置、不调用 API。原定价模式、比例、风险参数的样本/有效默认分支继续在规则测试中证明；旧活动时间/时区/name/refTool/库存总配置等默认装配不再作为独立调度产品保留。同期不做公式优化或规范化策略合并；两个已有 SKU normalizer 的等价性不凭猜测顺手统一。已有输入内的站点/供货价选择不是新增 legacy 数据源回退，本轮不得新增仓库/API 双读或改变其规则。

实施前若发现当前外部消费者、durable asset/副作用 owner，或无法按本矩阵脱离旧服务保留有效行为，停止相应项并报告证据；不得 missing skip、新 allowlist、空壳或误删保绿。

## 6. 测试与安全规则的同批处置

旧测试不是架构依据，仍有效的断言也不能随服务整删。逐项处置以下六个原测试文件/fixture，先保全行为后移除接缝，不新建验收 runner。

| 基线测试 | 待确认处置 | 继续验证的行为 |
| --- | --- | --- |
| validation_utils_test.go | 原样保留完整文件 | 降幅上下限、非法折扣及默认值 |
| config_test.go | RETIRE 完整文件，**先**把 TestApplyPromotionCreateConfigKeepsFullStockRatioUnlimited 的业务断言移到原 registration_direct_products_test.go 的无 IO 规则接缝 | 旧 DTO 默认、Validate、StartEqualsEnd、五分钟/十五天计划 builder 的专属测试随调度合同退出，不是当前平台规则的 Authority；保留原营销 API 日期 wire 测试，不能用旧 DTO 维持 fixture |
| activity_stock_validation_test.go | 保留两组测试所有有效 case；仅去掉旧 service 构造，库存计算直接调用无 IO 函数 | 负/零库存、比例和最少一个库存、ActivityConfig 库存/降幅字段断言 |
| price_calculator_test.go | 保留所有纯公式、SKU 价格不回退、Amazon 成本映射测试；退出 TestBuildCalculateRequestWithPriceModeBreakevenUsesPerSKUCost 的 repo/JSON/service fixture | 原每 SKU 成本/无效 SKU 与保本值断言迁到无 IO 成本与现有价格接缝，不把删除仓储 fixture 当删除业务规则许可 |
| registration_direct_products_test.go | **保留文件并收缩**：纯 helper 用例去 receiver，service 端到端用例先把有效业务断言改到本表下述纯接缝，再退出 save/open/create/query/session mocks 与专属 Store stub | 不以整文件删除代替规则评估；保存/开启次数、分页找ID、activityKey命名、分块session、调用来源仓库/实时平台的专属协议断言随旧编排退休 |
| repository_test.go | RETIRE 完整文件 | 仅绑定旧 getStoreInfo / getMapping reader 及其 stub；原 listingadmin tenant/scoped 查询与 worker 查重/Store sync 测试完整保留 |

registration_direct_products_test.go 中必须继续覆盖的规则分组：

- 常规/限量/BOTH、零 minProfit、DISCOUNT/PROFIT/BREAKEVEN、全库存比例，以及限量降幅在舍入后仍大于常规：对应无 IO strategy/config 函数、ensureLimitedPromotionDropRatesGreater、applyPromotionCreateConfig。
- 单/多 SKU、不同/相同价格、最小原价/最高成本、供货价基准、SKU 归一化、完整成本对应和 partial multi-SKU 拒绝：对应 minimumPromotionSKUDiscountRate、dropRateFromProvidedProduct、buildActivityConfigsFromProvidedProducts，不留下 SaveConfig stub 证明已不存在的调用。
- 直接 SKU 价格/成本、retail 与 supply 分离、候选快照优先及缺少实时价格不引入新读取：对应 promotionSKUPriceInputs、promotionGoodsFromProductSnapshots、enrichPromotionGoodsFromProductSnapshots、buildCalculateRequestForPromotionProducts。保留原样本和期望值，去掉已退休的 query 调用次数断言。
- 每 SKU 严格低于 95%（含 16.60/15.77 相等边界、94.999 子分币）、缺失活动价整项排除、sale-attribute SKU 字段、请求利润/保本活动价优先：对应 isTimeLimitedDiscountValid、buildCreateActivityRequest，保留过滤原因和各 SKU 价格断言。

保留营销 API 的 SaveConfig 类型/库存、UpdateConfigState v2 端点、Payload/wire 和已有平台规则测试；不删平台/租户/凭据/确定性断言换绿色。规则剥离只做原行为证明，不扩展新的营销需求、公式或测试矩阵。

## 6.1. Guard、文档与两套 workflow

| 既有对象 | 未来同批实施处置 | 必须保留 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 既有 absence 清单加入 registration.go、mixed.go、profit.go、product_data_helper.go、config.go、config_test.go、repository_test.go；混合文件精确检查退出符号与保留规则/风险错误/属性 | 不禁止整个 activity 包；断言旧 service/constructor/session/TimeLimitedDiscountConfig 及其 default/Validate/专属错误不复活，规则不能通过空壳恢复 |
| tests/import_boundaries_test.go 的 TestSheinRetiredManagementImportsStayBlocked | 删除 activity 下 mixed.go、product_data_helper.go、profit.go、registration.go、registration_config.go、time_limited.go 六个 management allowlist 项 | 后两个仍存在但只剩无 IO 规则，不能因文件保留继续豁免；其它条目/deny 保持，不加新 exemption 或调高 ceiling |
| 架构索引/原设计与 tests/architecture_docs_test.go | 将本设计状态与 §5/§6/本节落地结果同步；若旧文档直接当前保留断言冲突，只更新直接冲突引用/原文档测试 | 历史批次/dated inventory/原 CI 结果不重写；复用原文档测试入口，不额外建治理检查器 |
| docs/refactoring/phase2-runtime-inventory.md | 实际 import 图确认后，只减当前 core/logger → shein/activity 这一行 | logrus 与其余消费者/ceilings/历史清单保留，不将纯规则包登记为整个目录退休 |
| .github/workflows/ci.yml / commercial-readiness.yml | **零改动** | 原必需/安全检查、触发、权限、并发、选择、完整 listingadmin race、current-application / shein-login-worker build、PG/资金/发布检查及保护保持；其它部署 workflow 不改、不执行 |

未来自检限原 retirement/import/命名/文档 guard、受影响规则/API/共享 owner/worker/观察的必要原测试和命令实图/编译。稳定候选跑原适用 PR CI，一次集中独立最终检查；分类器原有 SKIPPED 不记 PASS，设计检查不冒充源码实施或 main CI。

## 7. 确认、实施准入与停止点

1. 本轮只形成设计和 Draft PR，完成文档检查、适用 CI 与一次独立最终文档检查。停在 **设计待用户确认**，不修改生产、业务测试、guard 或 workflow；不把 Draft/绿色 CI 当实施批准。
2. 用户确认 §5、§6、§6.1、本节后，在同一主要 PR/分支继续；先在 #29/#301 当前正文记录 **Reuse Existing Architecture / Ready**、冻结设计提交、原 owner/调用路径/不变量/非目标、唯一 Writer、开工 main，并刷新外部 import、注入与全部命令图。发现新边界按 §2 停止并完成适用独立准入，不临时发明兼容。
3. 先在原 guard 写出旧编排应缺席、有效规则应保留的 RED；按原样本保存有效断言，再做最小 EXTRACT/RETIRE 与 GREEN。切片是同批实现/提交边界，不机械拆 PR。新 finding 先分类并查 siblings；真实 Must 未满足不得合并，不放宽安全检查。
4. 最终候选前同步一次实际 main，仅处理真实组合问题；准确实现 HEAD、CI、独立最终检查和局限写 PR。未变代码证据可明确复用，不作为新 SHA 的直接执行结果；修复后只复核相关增量，不重复同 SHA 全量评审。
5. 实现完成交付 In Review，仍需单独准确 HEAD 合并授权；之后才按 expected-head 正常合并并核对实际 merge SHA 对应 push/main CI。设计确认不是合并/关闭/部署/运行/数据权限，PR CI 不是 main CI。
6. 当前采集、Store observations、营销 API/有效规则/共享合同/两种属性/PriceManager/PriceCalculator、独立 SHEIN 登录及 Store sync 都保留；本设计与清理不增加内部试用前置，不宣称产品验收、生产上线或 #301 已清零。
