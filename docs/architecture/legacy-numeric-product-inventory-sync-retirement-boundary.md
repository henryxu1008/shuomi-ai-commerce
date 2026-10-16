# 旧 SHEIN/TEMU 数字商品与库存同步链退休边界

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **用户已确认 §5、§6、§7，按 Reuse Existing Architecture / Ready 在 #742 实施源码退休**；准确候选、CI 和独立最终检查以 PR 为准。
- 调查基线：`main @ 08d55f930bf286aa857571542817dc5accff7687`；#740 已源码合并，实际 push/main CI 证据保留在原 PR。
- 实施 Design Basis: **Reuse Existing Architecture**。用户确认的冻结设计2cd716969先于正式源码修改，#29/#301已记录Ready；复用原owner与合同，无新身份、权限、持久化、状态或副作用边界。原设计阶段N/A、设计CI/review保留为历史，不替代实施验证。
- 用户边界：保留当前采集、Store 观察同步、有效平台 API/规则、共享合同、独立 SHEIN 登录 worker 及其 Store 同步；不操作运行实例或数据，不增加内部试用前置。

## 1. 用户结果与范围

维护当前内部试用源码、构建和独立登录 worker 的操作者，需要能分清已失去执行入口的旧数字商品/库存同步与仍使用的合同，避免在后续删除时破坏 Store 观察、平台能力或登录后店铺信息写回。本设计交付准确源码闭包和持久化边界；它不阻塞当前试用操作，也不把清理旧代码变成新产品需求。

本候选按矩阵退出 `internal/shein/productsync` 的无注入商品同步实现、`internal/shein/inventory`、`internal/temu/sync` 的旧同步/监控闭包，以及仅服务两平台监控的 InventoryRecord 持久化源码/DTO。保留 productsync 中仍被活动代码使用的两个属性类型和既有营销 client 禁入测试；不按包名、数字字段或“编译过”判断可删。

本轮不重新设计活动报名、刊登/结果写回、核价、平台 client、当前 Product/Store/Submission，也不新建同步执行器、通用接口或恢复平台。没有当前使用要求需要引入新的开源组件；直接复用既有合同、测试和工具。

## 2. Design Basis、Authority 与不变量

拟实施复用以下现有批准依据：

- [全新系统基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)：只做 EXTRACT / RETIRE，不迁移历史数据、映射旧 ID、包装旧 owner、fallback、双读双写或另建事实源。
- [旧 CLI](legacy-listing-cli-retirement-boundary.md)、[control-plane](legacy-listing-control-plane-retirement-boundary.md)、[数字执行链](legacy-numeric-task-execution-retirement-boundary.md)、[数字任务持久化](legacy-numeric-task-persistence-retirement-boundary.md)：原入口/调度/派发/执行和 ImportTask 闭包已退出，不通过本轮恢复。此设计仅在确认后替代它们对本同步闭包的“保留待后续”决定，不重写历史证据。
- [独立 SHEIN worker 身份/持久化](shein-worker-identity-persistence-boundary.md)：账号、Cookie/attempt、Redis/browser/Profile、租户隔离、登录后 Store 同步和查重仍归原 worker/Store owner。
- [当前 Store](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md) 和[准备接线](store-observations-preparation-runtime.md)：当前 Organization/Store、观察记录、官方连接、Temporal 及 UNKNOWN/retry owner 不消费旧数字同步服务；provider 未配置时仍如实不可用。
- [我的供应链 SHEIN](my-supply-chain-shein-v1.md)：当前官方执行、商家绑定和 Submission 副作用责任不转交旧 seller 网页 runtime。无 UI/IA 变更，Figma N/A。

实施不新增/转交事实 owner、表、schema、身份/权限/租户政策、公共当前领域合同、状态机、事务、幂等键、retry/UNKNOWN/recovery 或平台副作用。目标是退出旧实现，不是把它接到当前 Observation/Temporal 或登录 worker。本闭包超过 30 个文件/2,500 行风险提示线；由本矩阵逐项覆盖且完全复用上述既有边界，不通过拆 PR 或搬文件规避。若发现矩阵外的当前必要消费者，或需要改变上述保留边界，停止对应删除、报告证据并升级适用 Independent Architecture，不能在实施中临时发明兼容。

Must：保留现有有效安全/租户/owner/平台/规则断言；共享类型及 wire 字段不变；无第二同步/恢复 owner；真实实例、表/行/索引、配置/凭据/Profile 和历史事实不操作。Should：现行地图和 guard 如实区分三种“同步”。Out of Scope：真实平台/数据库探测或 dry-run、排空/补偿/重放、运行环境备份/清理、历史导入、新账号系统、其它活跃批次、治理/验收工具及 CI 保护变更。无新增 Accepted Risk；Threat Model 只覆盖误删保留能力、误转交事实/副作用和误把源码退休当运行态安全，不扩展为全局安全重构。

## 3. 实际消费者：合同 → 实现 → 注入 → 入口

基线上执行 `go list -f '{{.ImportPath}}|{{join .Deps ","}}' ./cmd/...` 并逐根核对全部 **31 个**当前命令；未运行任何命令 main。所有命令均无 `shein/productsync`、`shein/inventory`、`temu/sync` 或 `shein/activity` 依赖。全仓 Go import/构造器/方法检索用于进一步区分仅有源码互调与真实注入；测试构造和同包编译不构成产品保留义务。

| 合同/实现 | 基线实际消费者/注入 | 源码处理 |
| --- | --- | --- |
| SHEIN `ProductSyncService` → `productSyncServiceImpl` | `NewProductSyncService` 只有定义和专属测试；无包外生产构造/命令注入。读取平台商品/库存/价格、转换快照并写 listingadmin ProductData | RETIRE 服务、快照、转换/富化/写入实现及专属测试，精确见 §5 |
| SHEIN `InventorySyncService` → 两个构造器 | 无包外 Go import/构造或当前命令注入；内部用 ProductData、mapping、Amazon fetcher、策略与平台写 API | RETIRE inventory 完整基线闭包，不删它引用的共享 API/规则/基础设施 |
| `SyncedInventoryProductFeed` / Query / Record | 只有 inventory 内声明/方法调用和 fake；全仓无生产实现/注入。旧 `listingkit/sheinsync`/store 已退出 | 随 inventory RETIRE；不因字符串 `listingkit_shein_synced_products` 建新 adapter、表或 Observation feed |
| TEMU `ProductSyncService` / `InventorySyncService` / `ServiceFactory` / `NewServiceRuntime` | 全仓无包外 Go import，构造器仅同包工厂或测试；原工厂没有当前装配 consumer | RETIRE sync 完整基线闭包及 runtime adapter，不把 local/API 两路径搬到新 owner |
| `productsync.EnrichedSkuInfo / EnrichedSkcInfo` | `shein/activity/product_data_helper.go` 与 `price_calculator.go` 仍解析 Attributes、读取 mapping/Amazon 成本，原 price_calculator_test 使用两类型 | **原样保留** product_sync_types.go、所有字段/JSON tags/imports 与活动源码/测试。已有源码消费者不等于本轮证明了用户入口或准许新 current consumer |
| listingadmin InventoryRecord 仓库/API | 生产使用只来自上述 SHEIN inventory 与 TEMU sync；其构造器无包外生产调用；另有独占 lifecycle test | 目标服务退出后，RETIRE 三专属生产文件、一专属测试及三 DTO/interface；不改真实表 |
| ProductData / mapping / raw JSON / Store / 规则 | 活动代码、TEMU product/pricing、SHEIN publish/规则和 worker 等仍消费相关合同；当前应用也经 SHEIN draft → publishing → listingadmin/root listingruntime 编译共享合同 | 全部保留，不把包级依赖当某个旧同步入口仍使用的证据，也不顺手删字段/仓库 |
| 当前 Store observations | `app/storeobservations` / `storecenter/observations` → 原 repository、官方端口、Temporal worker → 当前商品/订单观察；31 根图与旧服务无交集 | 原接线/事实/权限/不可用语义保持；不迁入旧 ProductData、Attributes、库存历史或旧任务 |
| 独立登录与 Store 同步 | `cmd/shein-login-worker → app/runtime/sheinloginworker → GormStoreRepository/StoreAPI → sheinlogin/bootstrap → Service`；bootstrap 注入原 managed Store sync/tenant-qualified 查重工厂 | 完整保留；这是登录后外部店铺 ID/状态/备注写回，与本轮商品/库存监控不同 |

当前架构问题是闲置旧商品/库存服务依然携带数字商品写入与平台 mutation，而目录中的纯属性合同又被相邻活动代码引用。解决方式是按符号闭包 RETIRE，保留明确共享合同；不能清空整个 productsync/listingadmin/listingruntime，也不为删目录而改名、包装或迁移事实 owner。

## 4. 持久化、事务与外部副作用 owner

| 边界 | 源码事实/责任 | 本设计的退休/保留限制 |
| --- | --- | --- |
| `listing_product_data` | listingadmin ProductDataRepository；旧同步批次调用 UpsertProductDataBatch，原 repository 自有事务/Store owner 检查；监控更新 Attributes、shelf/平台快照 | 只退出本闭包 caller；模型/仓库/API、tenant/Store/owner/事务与字段全部保留，当前 Product canonical facts 不接管旧行 |
| `listing_product_import_mapping` | listingadmin mapping 仓库；旧商品富化读取平台商品/SKU 与原数字任务关联；活动/刊登等源码仍消费映射 | 仓库/API/owner helper、ImportTaskID 及 root runtime mapping 类型保持，不创建新旧映射、不改唯一索引/行 |
| `listing_inventory_record` | InventoryRecordRepository/GORM API，按 platform/product/region 读最近记录，再按“同一天”条件追加 Amazon 价格/库存历史；该读→写不是同一跨系统事务，不能称 exactly-once | 源码 writer/reader/专属 DTO 已随本闭包在候选退出；真实表/索引/行/历史与最后写入/恢复责任仍归原运行/数据 owner。本轮不探测是否存在，不 DROP/ALTER/清理/补写，不将历史搬到 observations |
| `listing_raw_json_data` | 共享 RawJSONData 仓库/API与 sourceproduct 窄合同；旧监控读取 Amazon 原始证据，相关基础层仍有其它引用 | 原仓库/字段/DTO/采集/缓存/HTTP/browser/fetcher 保留，不以少一个 caller 推导整包删除 |
| 原 synced-product feed | 本闭包只有抽象端口和 fake，没有当前生产 writer/adapter；常量名字不是当前表已被接管的证据 | 连同专属查询/转换退出；旧表/日志/任务的真实存在、备份/恢复均未知且未读取，不接到新 Observation owner |
| 平台及 Amazon 副作用 | 旧服务读取商品/库存/价格、Amazon fetcher；监控可下架/重新上架/改库存并随后写本地快照/历史/Attributes。平台调用与本地 DB 不构成共享事务，局部错误会记录后继续/返回 | 删除旧编排，不转交发送、重试、补偿或 UNKNOWN 责任；有效 SHEIN/TEMU API、transport/client DTO、平台规则和共享定价逻辑保留但不调用 |
| 当前观察与独立登录 | 原 observations 命令/同步/records/heads、Temporal checkpoint/恢复；原 worker 的 DB 账号/Store 行、Redis attempt/Cookie、browser/Profile、查重和 Store ID 写回 | 两条有效链各沿既有 owner，schema/键/状态/租户校验/外部身份/恢复不变；不导入库存历史或让当前组织使用旧账号 |

删除源码不代表真实在途操作已完成、旧实例已停止或历史责任已消失。失败、迟响应、响应丢失、重启、重试、取消、并发和数据恢复不设新协议，不以“没有 main consumer”宣称真实环境可自动清理。源码保留/编译也不代表真实账号、当前观察 provider 或用户验收通过。

共享 `core/config.ScheduledTaskConfig`、平台 ProductSync/InventorySync/Monitor、loader/default/validator/registry 仍有相邻配置消费者，本轮原样保留。运行 manifest、环境变量、部署/YAML、Redis 键/锁/TTL 和配置文件均不改，不借退休服务收缩共享安全门禁。

## 5. 用户已确认的精确源码删除矩阵

矩阵只指调查基线的源码/测试；实施前已刷新消费者，与本矩阵一致。**RETIRE 共 63 个完整文件（51 生产、12 测试）并精确收缩 1 个混合生产文件**，已在本候选按下表退出；不是运行资源/表的删除清单。§6 的既有 guard/文档随同一主要 PR 校准，不机械拆 PR。

| 对象 | 确认处置 | 删除条件 / 明确保留 |
| --- | --- | --- |
| `shein/productsync` 5 生产文件 | RETIRE product_sync.go、product_sync_fetcher.go、product_sync_enricher.go、product_snapshot.go、product_snapshot_repository.go | 无包外服务/快照生产注入；不保留 stub、别名或转发构造器 |
| 同包 4 专属测试 | RETIRE product_sync_repository_test.go、product_sync_save_test.go、product_snapshot_test.go、product_snapshot_repository_test.go | 只验证已退休服务、快照/转换及其 repository seam；原 listingadmin owner/批次安全测试不改 |
| 同包 `product_sync_types.go`、`marketing_client_boundary_test.go` | **保留两完整文件** | 两属性类型仍被 activity 使用，字段/tag/引用不变；营销 client 禁入 guard 已遍历实际 Go 源，保留其全部三项禁止断言及读取失败语义，不变成空包成功 |
| `shein/inventory` 15 生产文件 | RETIRE amazon_fetcher.go、api.go、change_checker.go、config.go、cost_calculator.go、helper.go、monitor.go、price_strategy.go、product_data_conversion.go、product_data_repository.go、record.go、strategy.go、sync.go、synced_products.go、types.go | 无包外 import/生产注入，全部只服务旧监控；其中 api.go 是旧编排方法，**不是** `shein/api` 平台 API |
| 同包专属测试 | RETIRE product_data_repository_test.go，含旧 feed fake/分页/转换 fixture | 无共享 fixture/当前 consumer；若刷新发现被保留测试使用，仅保留所需现有纯测试 helper，不复活服务 |
| `temu/sync` 28 生产文件（下列三行） | RETIRE 完整基线生产闭包 | 无包外 import/生产注入；不删除 `temu/api`、product、pricing、store、SKU、marketplace 平台规则或 shared 配置 |
| TEMU 商品服务/转换部分 | factory.go、runtime_adapter.go、product_sync_service.go、product_sync_types.go、product_snapshot.go、product_snapshot_repository.go、product_converter.go、product_data_builder.go、product_utils.go、mapping.go、sku_details_handler.go、sku_mapping_enricher.go | ServiceFactory/ServiceRuntime 与私有 local/API fallback 一起退出，不新建替代 factory |
| TEMU 库存部分 | inventory_product_data_conversion.go、inventory_product_data_repository.go、inventory_sync_amazon_fetcher.go、inventory_sync_api.go、inventory_sync_change_checker.go、inventory_sync_concurrent.go、inventory_sync_config_getter.go、inventory_sync_cost_calculator.go、inventory_sync_helper.go、inventory_sync_interface.go、inventory_sync_monitor.go、inventory_sync_record.go、inventory_sync_service.go、inventory_sync_strategy.go、inventory_sync_types.go、inventory_sync_updater.go | 只删该旧 service 的并发、更新、记录和 helper；共享 pricing/workerpool/recovery 等不改 |
| 同包 6 专属测试 | RETIRE product_sync_repository_test.go、product_sync_service_save_test.go、product_snapshot_test.go、product_snapshot_repository_test.go、inventory_product_data_repository_test.go、inventory_sync_types_test.go | 旧实现细节/快照 writer/DTO fallback 随协议退出；仍有效的平台、当前观察和共享 owner 测试完整保留 |
| `listingadmin` 3 专属生产文件、1 测试 | RETIRE inventory_record.go、inventory_record_repository.go、gorm_inventory_record_api.go、inventory_record_query_test.go | 包含 ErrInventoryRecordNotFound、InventoryRecord/Repository、GORM row/转换/构造器与 InventoryRecordToRespDTO；全仓其它生产使用在目标同步闭包内，专属 test helper 无外部使用；真实表/索引/行不动 |
| `listingadmin/management_api_types.go` | 只 RETIRE InventoryRecordCreateReqDTO、InventoryRecordRespDTO、InventoryRecordAPI 三声明及专属注释 | 上方 NewProductDataBatchSaveReqDTO、下方 RawJsonDataReqDTO 及其它所有 DTO/interface/helper/import 保持；不按 Inventory/Task 子串批删 |
| ProductData/mapping/raw JSON、Store/owner、root listingruntime、平台 API/规则、activity、pricing/发布和基础层 | **保留全部实现/合同/有效测试** | 未列入本矩阵的共享清理不自动启动；包括 ImportTaskID、属性 mapping 字段、worker provider/client、schema/事务/幂等与 scoped 读写保护 |
| 当前采集 / Store observations / 独立登录 worker 与 Store sync | **原样保留** | 不停止/重启/部署，不接管各 owner，不改源权限或产品可用性；保留 worker 的数字持久化债务仍使 #301 不能宣称全仓清零 |

`product_sync_types.go` 的保留只是已有消费者所需的属性合同，不是继续保活已退休同步 service，不成为新的 compatibility 类别或新当前应用入口。此次不为清空目录改它的 owner/路径；未来排空活动等相邻消费者须另有当前需求及有界准入。

## 6. 测试、guard、文档与 CI 的处置

| 既有对象 | 同批实施处置 | 保留约束 |
| --- | --- | --- |
| `tests/listingkit_backend_retirement_test.go` | 原 absence 清单扩展 §5 63 文件/两个完整目录；原混合文件 guard 禁三库存 DTO；补明确两类型/营销边界文件必须存在。原命令依赖检查加入旧 sync 目录，productSync 仅禁退出服务文件/符号，不能禁仍保留的整个包 | 在旧源码仍存在时先得到目标 RED，再最小删除后 GREEN；不因读不到应保留文件跳过，不恢复空壳，不将类型保留误写成整个产品同步仍可用 |
| `TestSheinRetiredManagementImportsStayBlocked` | 删除本批退出 inventory 及两个 productsync service 文件的精确 allowlist 项 | product_sync_types.go、activity 等保留项和原禁止 management client 的断言不变；不扩新增豁免，不顺手清理其它历史目录 |
| `TestTEMUSyncAndPricingRetiredManagementImportsStayBlocked` | 移除退休 sync 的扫描根及全部 sync 专属 allowlist 项（含已缺席的专属 factory/helpers 项）；扫描保留 pricing，原拒绝断言不变 | 在原 retirement guard 明确验证 sync 目录缺席，不能为缺目录增加 continue/skip；保持 pricing 保护，不整删/固定通过该 test |
| `TestRetiredManagementRuntimeNamingHotspotsUseCapabilityNames` | 只删除已退出的 TEMU product_sync_service.go、product_sync_repository_test.go、SHEIN product_sync_repository_test.go 三检查项 | 其它仍消费文件的命名禁止检查原样保留；不扩大原有历史缺席处理，不读不存在文件伪造 PASS |
| 原 TestTemuSyncRuntimeUsesCapabilityNames / TestTemuSyncUsesManagementAPIPortAliases / TestTemuSyncFallbackLogsUseCapabilityNames | RETIRE 三个只覆盖已退出 sync 实现/目录的专属结构、导入与日志命名检查；首轮CI同根因 IMPLEMENTATION_TEST 补列 | 原 retirement guard 明确验证整个 sync 闭包缺席；pricing/API client 等 sibling 的有效命名/禁止导入检查完整保留，不加 missing skip 或恢复 stub |
| phase2-runtime-inventory 当前 Legacy consumer register | 只移除 core/logger → shein/inventory、temu/sync、shein/productsync 三条已不存在的实际边；同批 CI IMPLEMENTATION_TEST | 保留 importer ceilings、其它当前行和全部 DATED EVIDENCE；共享属性类型不再 import logger，不将包保留误记成旧 logger consumer 仍存在 |
| productsync 营销 client guard、SHEIN activity/平台 API/规则、listingadmin owner/Store/批次安全、worker/managed/bootstrap | 保留原测试并跑适用回归；不迁移旧 service fixture 到当前产品、不用固定成功替代约束 | 两属性类型/JSON 不变，租户隔离/查重 fail-closed、健康/关闭、当前 Observation 权限/事实/UNKNOWN 仍按各原合同 |
| 当前地图与批准前批留债说明 | 实施时校准本设计/架构索引、legacy-register、repository-structure/runbook 和前批数字执行设计中关于 InventorySyncService 的现时保留语句，保留其历史证据 | 只更正本闭包当前状态，历史依赖/基线/测试结果不重写成新 PASS，不提供实际 DROP/迁移/停机步骤 |
| `.github/workflows/ci.yml`、commercial-readiness 及全部其它 workflow | **不改**；现有全保留 listingadmin race 仍有大量有效 Store/owner/mapping/规则测试 | 不新增空包命令或零匹配 regex，不改 step/id/pipefail/log/artifact、分类、触发、Required CI Gate、权限/并发/保护规则。原 PostgreSQL/public-facts/schema-context、资金并发、发布策略和当前 runtime build 等全部保持 |

开发验证限于原 retirement/import/文档 guard、保留 productsync/activity 与有效平台规则/owner/worker/当前观察回归、31 命令实际依赖与必要编译；不执行命令 main、schema/真实 recovery/provider/实例操作。稳定候选再跑适用完整 PR CI，未变代码有效证据明确复用；不新造 runner、验收平台或全仓故障矩阵。原设计阶段只读调查与文档检查的历史证据保留；本轮实施及源码 RED→GREEN 另绑准确候选记录于 PR，不把计划或旧设计 CI 当实现结果。

## 7. 实施准入与停止点

1. 用户已确认冻结设计2cd716969的 §5、§6、本节，授权在原#742实施；#29/#301在任何生产修改前记录 Reuse Existing Architecture / Ready、唯一Writer/主要PR、main基线、保留owner/不变量和准确消费者刷新。开工main仍为08d55f930，31命令图及外部构造/引用与矩阵相符，无其它同范围Writer。设计CI和旧#740授权不替代本轮实现验证或合并许可。
2. 使用同一主要设计/实施 PR 连续完成范围内工作；先在旧源码存在时使原 retirement guard RED，再按矩阵删除、精确收缩混合 DTO/原 guard，GREEN。实施中遇到保留消费者/helper 或 Must 缺陷，先分类并核对 sibling；未满足当前 Must 不能合并。
3. 本闭包只退出源码，没有新持久化/安全/状态/副作用边界，不重复全局架构评审；若真实发现必须改变这些边界，停止对应实施并完成适用 Independent Architecture 准入。不能恢复旧 scheduler/bridge/HTTP、加 adapter/fallback、移交到 worker/当前 Observation 来绕过设计。
4. 最终候选前同步一次实际 main，只处理真实组合问题；稳定完整候选集中做最终独立交付检查，修复只复核相关增量，不重复同 SHA/未变代码的完整评审。设计及实施证据分列，SKIPPED 不写 PASS，真实运行/用户/平台验收保持 NOT_RUN。
5. 本轮矩阵内源码实施、必要隔离自检、commit/push与原PR维护已获授权；实现完成进入In Review，交用户决定，准确HEAD合并仍须另授权。部署/关闭Issue/运行实例/真实数据操作均不在本轮。#29/#301保持OPEN，生产NOT_READY，不增加内部试用前置；#741/#692、当前产品与运行owner不接管。

```text
Legacy decision: RETIRE
Reusable behavior: 原平台 API/规则、SHEIN 属性共享类型、Store/owner/商品/mapping/raw JSON 和独立登录全部保持。
Current owner: 当前各 Domain/Capability/Store observations、原平台共享 owner 与独立登录/Store 仓库；历史事实/恢复责任归原运行 owner。
Cutover/deletion condition: 用户确认准确矩阵、执行 Issue Ready、目标服务/仓库消费者归零、共享合同和有效检查保持；只删除源码，不处置真实状态。
```
