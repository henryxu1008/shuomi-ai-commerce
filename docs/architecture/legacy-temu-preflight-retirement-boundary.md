# 旧 TEMU 刊登前置查重、原始数据读取与缓存编排退休边界

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **USER_CONFIRMED / SOURCE_RETIRED**。用户已确认 #750 冻结设计 `ca7a6e2ea19314898b024a8d50b31913ce191265` 的 §5、§6、§6.1、§7，授权在同一 PR 实施；准确实现 HEAD、CI 与独立最终检查维护于 PR，合并/真实操作仍未授权。
- 调查与实施开工基线：`main @ 6132a08a028ff6e7a1ad912414ccc67ef3be55ab`（#749 已合并）。原设计 CI/review 是历史设计证据，不替代实现候选验证。
- 当前 **Design Basis: Reuse Existing Architecture；Admission: Ready**，生产修改前已在 #29/#301 记录并读回核验。此前 N/A 只覆盖文档调查；本次不新增公共领域合同、权限/tenant、事实/schema、状态机/事务、恢复或平台副作用 owner。
- 保留当前采集、Store 观察同步、有效平台 API/DTO、规则与共享合同、两种 SHEIN 属性、PriceManager/PriceCalculator、独立 SHEIN 登录 worker 及其 Store 同步。两套及其它 workflow、必需/安全检查、保护和 ceiling 零改动；不操作实例或数据，不增加试用前置。

## 1. 用户结果、范围与非目标

使用者是维护当前内部试用、平台/Store 能力和独立登录的维护者。交付是四个闲置 TEMU Task handler 的源码闭包、实际消费者、持久化及平台副作用归属、删除/有效规则保留和验证矩阵，避免在后续接线中复活旧数字 Task。此清理不阻塞当前用户操作，也不改变当前入口、使用步骤或内部试用前置。

范围仅为 `internal/temu/product/{exists_check_handler,out_goods_sn_check_handler,raw_json_data_handler,cache_handler}.go`，基线 **4 个生产文件共 512 行**。没有专属 DTO 文件或专属测试可随之整删。`product`、`context`、`sku`、`store`、`crawler/fetcher`、`sourceproduct`、`listingadmin` 均不是整包退休范围；其它 SKU/规则文件还有源码消费者。

不增加新刊登、查重服务、缓存策略、跨库事务、恢复/对账协议、公共领域合同、UI、入口、provider 调用或验收工具。复用现有 Go、`skugen`、marketplace publishing owner 和原 guard，不引入依赖。#748 模型诊断与 #692 Store 安装归其它 owner，不接管或重复派工。

不执行命令 main、构造器、平台 Query/Preview/DryRun、采集、缓存写入或 schema 工具；不读取真实配置/凭据、DB/Redis/Profile/browser、历史日志/JSON，不启停/部署实例，不迁移、清理、补写、重放、撤销或删除数据。源码调查不是运行态/在途资产清零证据。

## 2. Design Basis、Authority 与不变量

依据：

- [全新系统基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)：仅 EXTRACT / RETIRE，无 wrapper/fallback、旧 ID 映射、双读双写或第二事实源。
- 已确认的 [CLI](legacy-listing-cli-retirement-boundary.md)、[control-plane](legacy-listing-control-plane-retirement-boundary.md)、[数字执行链](legacy-numeric-task-execution-retirement-boundary.md)、[持久化](legacy-numeric-task-persistence-retirement-boundary.md) 与 [TEMU 保存/提交/结果](legacy-temu-listing-submit-retirement-boundary.md)：旧数字执行入口已经退出。本设计确认后仅接续这些文档保留的四个闲置前置 handler，不推翻其它保留项或历史证据。
- [TEMU publishing owner](../../internal/marketplace/temu/publishing/README.md)：平台确定性规则与 payload shaping 属于既有 `internal/marketplace/temu/publishing`；编码生成复用 `internal/pkg/skugen`。不让旧 Task/context 成为当前规则 owner。
- [独立 worker 身份/持久化](shein-worker-identity-persistence-boundary.md)、[当前 Store](store-center-current-application-v1.md)、[observations](store-center-platform-observations-v1.md)、[准备接线](store-observations-preparation-runtime.md)、[当前 SHEIN](my-supply-chain-shein-v1.md)：身份、当前事实、授权与 retry/UNKNOWN 继续归原 owner。Figma N/A，无 UI/IA 变化。

本次实施采用 **Reuse Existing Architecture**：删除未装配的四个旧 handler；先将 §5 两项确定性行为抽取到既有 marketplace owner 的包内私有值函数，编码仍调用原 `skugen.Generate(..., StrategyASINOnly, ...)`。不导出新类型/接口、不新增调用入口；不新增或转交领域/事实/schema、权限/tenant/身份、状态机、事务、幂等、retry/UNKNOWN/recovery 或外部副作用 owner。用户已确认冻结矩阵，生产修改前已在 Issue 写明引用、Ready 与唯一 Writer；原设计检查不作为实施验证或合并授权。

Must：有效 SKU 规则与 §5 保留项不减；当前权限/tenant/凭据、平台/Store/持久化原合同和安全检查不变；两套 workflow 与其它保护/ceiling 零改动；不接管历史数据、在途操作或 UNKNOWN。Should：原 guard 和当前文档与实际源码一致。Out of Scope 如 §1；无新增 Accepted Risk。Threat Model 限误删有效行为、混淆共享 owner、错误接入旧 Task/身份及将源码退出误报为运行态清零。

若实际发现当前消费者/外部可观察兼容义务，或必须改变上述保留边界，停止对应项并报告用户；没有可自动采用的兼容 Exception。新增高风险边界须补适用 Independent Architecture 准入，不在实现中临时发明。

## 3. 实际消费者、注入与调用路径

基线全仓 Go 搜索：四个 TEMU 构造器只见本文件定义，没有调用、直接实例化或注册装配。`go list ./cmd/...` 共 **31 个当前命令**；`go list -deps -f '{{.ImportPath}}' ./cmd/...` 的依赖并集中，旧 `temu/product`、`temu/context`、`temu/store` 命中均为 **0**。未运行命令 main。这是源码/静态编译图证据，不是实例、数据或真实平台验收。

| 合同 → 实现 | 注入、调用与消费者 | 处置 |
| --- | --- | --- |
| ProductImportMappingAPI → ProductExistsCheckHandler | constructor 接收 Mapping API，但无调用；Handle 从 Task 组 Store/Platform/Region/ProductID，再逐个检查 Variations.ASIN | 退休 Task 调用层；共享接口/DTO/仓库不随消费者删除 |
| 平台 CheckSkuSn → OutGoodsSnCheckHandler | constructor 无调用；从 TemuTaskContext 取 APIClient/TemuProduct/StoreInfo/主商品/Variants，临时 NewQueryAPI 并发出请求 | 退休 context/API 编排；保留平台 API/DTO；先抽两项纯行为 |
| ProductReader → RawJsonDataHandlerV2 | constructor 无调用；从 Task 组 FetchRequest，调用 reader.FetchProduct，再 SetAmazonProduct；可选 stats.GetStats 只用于日志 | 退休 Task 请求装配/错误包装；不是直接 Raw JSON writer；Reader/Stats 不删 |
| ProductCache → CacheProductHandler | constructor 无调用；从 AmazonContext 取商品，再从 Task 组 FetchRequest，委托 CacheProduct，失败只警告 | 退休 Task 编排/吞错策略；共享缓存事实与写入能力不转移 |

混合包仍有源码消费者：`temu/sku/{builder,item_builder}.go`、`temu/filter/rule_handler.go` 使用 `temu/product`；多个 SKU/filter 文件使用 `temu/context`。`skugen.Generate` 仍被 TEMU SKU item builder、TEMU SPU builder、SHEIN SPU record 使用。共享 Reader/Cache 仍被 TEMU variant handlers、SHEIN productdata handlers 使用；这些是源码消费者，不据此宣称当前命令已装配全部旧 handler。

上段 TEMU variant consumers 是 #750 调查基线；后续 [#751 已确认变体退休](legacy-temu-variant-retirement-boundary.md) 精确退休三个闲置 variant handler 与孤立 VariantProcessor/专属 helper。共享 Reader/Cache、SHEIN productdata、保留 SKU/规则与原持久化 owner 不随之删除，不据旧基线复活目标。

`internal/shein/publish/exists_check.go` 有同名 ProductExistsCheckHandler，SHEIN publish checker 有另一条 runtime repository 查询路径；都不在此四文件闭包。禁止用全仓类型名或“查重/cache”关键词整删。独立 SHEIN 登录 worker 的原 service/bootstrap/StoreRepository/StoreAPI、账户/Redis/Profile/浏览器、健康/关闭与 Store sync 接线不改。

## 4. 持久化、身份与平台副作用 owner

| 行为 | 原 owner 与边界 | 本轮约束 |
| --- | --- | --- |
| 已刊登查重 | handler → ProductImportMappingAPI.CheckProductExists → GormProductImportMappingRepository.ExistsPublishedProduct；查询 `listing_product_import_mapping`，条件含 store/platform/region/product/deleted 与非空 platform_product_id，原 applyOwnerScope 逻辑仍在 | 只删除该调用层。保留 Mapping 其它读写、Store 派生 owner 的同事务写入、tenant/owner 安全测试和模型；不迁移/重建表或接管 Mapping 事实 |
| SKU 编码平台检查 | QueryAPI.CheckSkuSn → 原平台 `out_sku_sn_batch_check`；传 GoodsID/OutSnList。API 已处理 nil request、网络错误与 Success=false；具体重复项在 Result.FailList | 保留 API、DTO/re-export、transport/认证与失败语义，不调用或新增替代 adapter；平台请求不由提取的纯函数发起 |
| 原始商品读取 | ProductReader 是窄读合同，但实际实现可以是本地 sourceproduct.ProductFetcher 或 remote fetcher，可能读缓存或触发采集/网络；并非保证零外部副作用的 DB 纯读 | 删除 Task 适配不删 Reader、FetcherFactory、本地/remote 实现、SourceFetcher/当前采集，设计期不执行 FetchProduct |
| 商品缓存 | ProductCache → 原 sourceproduct.CacheManager → RawJsonDataClient。共享 CacheManager 决定 freshness/skip/save/重复写处理；若客户端是 listingadmin RawJsonDataAPI，则委托 GormRawJSONDataRepository UpsertRawJSONData / `listing_raw_json_data` | 不改 key/freshness/upsert/重复错误或故障语义，不运行任何写入；不把 handler 当持久化 owner，也不将该缓存写入转给当前 Product/Store/worker |
| Task/TemuTaskContext | 四 handler 仅借用 Task/GetContext、AmazonProduct/Variants、StoreInfo、TemuProduct/APIClient 等现有共享值；SKU 结果函数实际上未向 context 写入结果 | 不新增持久化结果或查重状态；本轮不删混合 context 的字段、方法或接口断言，其它源码还消费这些成员 |

**已存在的架构限制须如实保留边界**：Mapping 旧 DTO 没有 tenant/context，GORM API 使用 context.Background；applyOwnerScope 在无身份时不是强制认证入口。Raw JSON 仓库查询键是 platform/product/region，不带 tenant/store。保留这些共享旧合同不等于宣称它们具备当前 Organization 授权或可直接用于私有商品；本设计不增加消费者，也不把它们接入当前应用或 worker。其当前外部可达性/产品授权未在本轮得到运行验证，不能写成隔离验收 PASS。若发现实际当前私有路径依赖这些旧合同形成 Must 缺陷，报告具体调用与后果，再决定最小修复；不借退休扩大为新 IAM/缓存隔离系统或试用前置。

Reader 使用 Task.GetSourcePlatformOrDefault，而旧 cache handler 使用 Task.Platform；本轮一并退休这两处 Task 装配，不将源/目标平台混用或静默缓存失败策略复制到新路径。来源字段保真、当前采集取消/deadline、共享缓存与 Store/worker 的已有行为继续由原 owner 负责。

四 handler 没有自己的 durable 状态机、事务、outbox、重试恢复记录或幂等键。旧 NONRETRYABLE 字符串、缓存失败继续及临时查询顺序不是当前 retry/UNKNOWN 协议。删除源码不判定任何历史操作成功/失败、不释放预占/锁、不回放或迁移历史缓存/Mapping。

## 5. 精确删除、抽取与保留矩阵

以下路径均相对 `internal/temu/product/`；用户已确认冻结矩阵，**四个原文件已源码退休**，两项有效规则位于既有 marketplace owner，未新增执行入口。

| 对象 | 决策 | 删除/保留条件 |
| --- | --- | --- |
| exists_check_handler.go（109 行） | RETIRE 全文件 | 去掉 Task 查重循环、Mapping 请求装配与日志/错误包装；共享 Mapping 全部保留。100 变体上限只在旧 Task handler 中，未发现其为当前产品/平台合同的依据，不迁入当前规则；其它既有 SKU/规格/平台限制不改 |
| out_goods_sn_check_handler.go（227 行） | EXTRACT 两项有效纯行为后 RETIRE 全文件 | 编码列表 shaping 与重复拒绝归既有 marketplace；Task/context/API 请求及日志编排不迁移，不用 wrapper 维持旧 Handler |
| raw_json_data_handler.go（94 行） | RETIRE 全文件 | 退出 Task→FetchRequest、ProductNotFoundError→旧 NONRETRYABLE 和上下文回填/Shutdown；原 Reader/SourceFetcher/Stats/错误类型保留 |
| cache_handler.go（82 行） | RETIRE 全文件 | 退出 Task→FetchRequest、空商品跳过与警告吞错调用层；缓存共享实现和事实原 owner 保留 |
| 私有编码列表函数（sku_sn_validation_policy.go） | EXTRACT 到 existing marketplace/temu/publishing | 输入只为主 ASIN、按原顺序的 variant ASIN 值列表、prefix/suffix；输出独立 []string。调用原 skugen ASINOnly；仅跳过精确空字符串，主项先、变体后、按最终编码首次出现稳定去重；不 trim、排序、改变策略或修改输入，无 Task/model/context/平台请求 |
| 私有重复项判定函数及私有值快照（同文件） | EXTRACT 到同一 owner | 输入仅 OutSkuSn/UsedGoodsID/UsedSkuID/FailReason 值；nil/空列表通过，任意失败项都拒绝并保留可诊断项。即使失败项字段为空也不能忽略；不持久化、不查询、不修复编码、不新增当前执行入口。旧日志/错误字面不是公共合同 |
| internal/pkg/skugen、QueryAPI.CheckSkuSn、SkuSnCheckRequest/Response、OutSkuSnItem、OutGoodsSnCheckResult、OutSkuSnFailItem 及 api.go 原 re-export | 保留原 owner/行为/格式 | 不因旧 caller 退出删除平台能力、DTO 或其它 SKU 生成策略；也不将 handler 整体移入 marketplace |
| ProductReader/ProductCache/ProductFetcherStats、FetcherFactory/local/remote、sourceproduct 的 FetchRequest/SourceRequest/RawJsonDataClient/CacheManager、DataParser、Raw JSON API/仓库/模型 | 保留原合同及源码消费者 | 不删除/改名/重建共享层，不新增兼容或 current-application 调用；有效采集、来源字段保真、缓存规则和测试保留 |
| Mapping API/DTO/仓库、Store owner helper/事务、安全测试；其余 TEMU product/context/SKU/filter/store；SHEIN 同名 handler 与 productdata | 保留，不属删除闭包 | 不删任何表/索引/数据或共享 context 成员；原独立 worker 及 Store sync 继续原接线 |
| 其它用户保留项 | 保留 | EnrichedSkuInfo/EnrichedSkcInfo、PriceManager/PriceCalculator、有效平台 API/营销 API/规则、共享合同、当前采集、Store 观察同步及独立 SHEIN 登录/Store sync 完整保留 |

Legacy decision: **RETIRE 四个闲置 Task handler；EXTRACT 两项确定性 SKU 行为**。
Reusable behavior: 原编码生成/稳定去重与平台返回重复项的拒绝判定；已存在的共享采集、缓存、Mapping、平台及 SKU 能力保持原 owner。
Current owner: 纯平台 shaping/validation 为 marketplace/temu/publishing，编码生成为 pkg/skugen；事实/采集/缓存/平台请求/Store/worker 仍各归原 owner。
Cutover/deletion condition: 用户确认冻结矩阵、Issue 实施 Ready 已满足；刷新无新消费者，规则 TDD 和原 absence guard 先 RED，再抽取/精确删除并 GREEN。源码删除不授权真实资源/数据删除或说明历史运行态已退出。

## 6. 验证与交付

原设计阶段仅提交本文与架构索引，其 diff/相对链接、静态映射、原 PR CI 与一次独立设计检查保留为设计证据。本次已获实施授权，按下列有界测试与删除矩阵验证；不新增检查器、runner、矩阵平台或真实验收流程。

本次范围内必要实施检查：

1. 两项纯规则先写能捕获行为丢失的测试并确认 RED：主/变体同编码、跨变体重复、空 ASIN、顺序/prefix/suffix、空失败列表、非空失败/空字段失败、多项诊断、输入不变；再最小实现 GREEN。不调用原构造器/平台/采集/缓存，不复制或依赖旧 Task。
2. 在原退休 guard 加四路径及限定旧 product 范围的 handler 符号检查，先确认存在而 RED，再删除 GREEN；同 guard 保留原平台/共享 token，并补 CheckSkuSn/DTO/re-export、共享 Reader/Cache、两私有函数的原位存在检查。不能全仓禁止 SHEIN 同名符号。
3. 仅移除 §6.1 一条消失 allowlist，保留其它 deny/roots/ceiling；刷新 31 命令实际依赖，并对保留 TEMU product/context/SKU/filter/API、共享 fetcher/sourceproduct、独立 worker 及当前命令做适用编译/必要已有回归。不重复运行完整 CI 已覆盖的相同检查。
4. 复用原 skugen、marketplace、sourceproduct/fetcher 及 listingadmin Mapping owner/Store 事务/tenant 安全测试；scope 内编译/规则/guard 必须通过。完整必需 CI 留给稳定候选；准确实现 HEAD 一次独立最终检查，不拿设计 CI 或旧 SHA 结果替代。
5. 如实区分本地静态映射、开发测试、PR CI、合并后匹配 main CI、实例/真实平台及用户验收；后两项本批 NOT_RUN，生产可用性不在此设计中签发。

### 6.1 原 guard、文档与 workflow 精确处置

| 工件 | 本轮改动 | 保留边界 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 扩展原 TestListingKitTaskBackendSourcesAreRetired 的四路径/旧 product 符号缺席与平台/共享/纯函数保留检查；继续原 TestCurrentCommandsDoNotCompileLegacyBusinessBackends、独立 worker guard | 不删旧安全断言，不给缺文件加 skip；不全仓封禁同名 SHEIN handler 或平台 DTO |
| tests/import_boundaries_test.go | 仅从 TestTEMUProductStoreAndSchedulerRetiredManagementImportsStayBlocked 删 exists_check_handler.go 一条已消失路径 allowlist | 其它 6 条和 product/store roots、deny/graph/ceiling 不变；全仓检索未发现另三路径的硬编码豁免，不臆造更多删除 |
| marketplace/temu/publishing/sku_sn_validation_policy.go 及同包测试、README | 两项私有值规则和 TDD，说明无 Task/API/IO/状态/新入口；调用原 skugen 确定性策略 | 不改其它现有校验、平台 API/DTO、pkg/skugen 生产实现或公共合同 |
| Legacy register、本文、架构索引与直接当前盘点 | 确认后校准四文件 source-retired 与有效规则 owner | product 仍有 logger/openai 等其它 import 消费者，不能删整个消费者行；历史图/证据与 ceiling 不改 |
| ci.yml / commercial-readiness.yml / 其它 workflow、.golangci.yml | **零改动** | 保留全部必需/安全检查、原编译/race 目标、分类/触发/权限/并发、Required CI Gate、保护/ceiling。原 SKIPPED 不记 PASS |

## 7. 确认与实施停止点

1. 原设计以 Draft PR 交付，准确设计 HEAD 的原适用 CI 与独立设计检查通过；证据是历史设计验证，不作为新实现候选 PASS。#29/#301 保持 OPEN。
2. **用户已确认冻结设计 §5、§6、§6.1、§7 并授权同一主要分支/PR 实施**。生产修改前 Issue 已写明冻结设计、Reuse Existing Architecture / Ready、owner/调用链/不变量/非目标、唯一 Writer；不扩为合并、部署或真实操作授权。
3. 开工已刷新实际 main、构造器/实例化/注册与 31 命令图，无新消费者；如出现真实新消费者或必须改变 §2 保留边界，停止对应项并报告，不能直接整包删除或接兼容路径。与本批无关的其它 owner 工作不接管。
4. 按 §6 先有效规则 TDD/absence RED，再 EXTRACT/RETIRE 与必要 GREEN，finding 先分类并检查 sibling。非阻塞完善不扩为新系统；当前 Must 未满足不能合并。
5. 最终候选前同步一次实际 main，只处理真实组合风险；实现 HEAD、CI、独立最终检查和限制维护于 PR。完成进入 In Review；准确 HEAD 合并须另获授权，随后核对实际 merge SHA 对应 push/main CI。
6. 不授权关单、部署/运行、真实平台/实例/数据操作，不宣称用户验收/上线或 #29/#301 全量清零，不增加内部试用前置。所有用户指定保留能力继续归原 owner。

## 8. 源码交付校准

四个旧 handler 缺席；原 guard 在删除前确实捕获四路径/四符号，删除后通过。两个 SKU 函数/私有快照先测试 RED 后最小实现 GREEN，复用原 skugen，不依赖 Task/model/context/API/client/state/IO、不修改输入或修复编码、不新增当前调用入口。仅删除 exists_check_handler 的一条 allowlist，其它六条及 roots/deny/ceiling 保留。

共享 Reader/Cache/Mapping、平台 API/DTO/re-export、混合 context、SKU/其它规则、所有用户指定保留能力与原权限/持久化/副作用 owner 均保留。product 仍有其它 logger/openai 消费者，其当前盘点行和全部 dated 图不删除。两套及其它 workflow、保护和 ceiling 零改动。实现/必要检查、准确 HEAD/CI/独立最终复核维护于 PR；真实平台、实例/数据、部署与用户验收仍 NOT_RUN。
