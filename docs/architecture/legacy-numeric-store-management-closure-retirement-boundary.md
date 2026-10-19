# 旧数字 Store 管理与刊登配置：完整退休闭包

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

> Status: **设计提案，待独立检查与用户确认；不授权生产实现。** Design Basis: **Independent Architecture**。只交付本设计与索引；适用独立检查、用户确认及 Issue 的 IMPLEMENTATION_READY / Ready 均满足后，才在同一主要 PR 实施。调查基线为 main `e360069e067489c87754f2699e79eb0cde3fbc67`，2026-10-19；#827 已合并，准确 push/main CI 已通过，后续 #829 图片模板变更与本矩阵无交叉。

## 1. 用户结果与一个 Delivery Batch

使用者是维护当前采集、草稿/校验、Store 观察以及独立 SHEIN 登录与 Store 同步的操作者。结果是旧数字 Store 的无入口管理写入、回收站、延期和刊登配置退出源码，使用中的账号读取、查重与同步仍沿原 owner 工作。当前核心操作不等待本批，不增加内部试用前置。

一个主要分支/PR 一次处理七个 mixed 生产文件、准确声明/字段/过滤/映射及专属测试，不再按 CRUD、字段、helper 或测试拆小批。无须新增生产 EXTRACT：实际有用行为已在当前保留 owner 中，原位复用。

本批不把“保留独立 worker”写成“全仓数字身份依赖已清零”。worker 原数字账号 DB/Redis/Profile 与当前 Organization/Store 是不同事实边界；#29/#301 保持开放，不为关单引入迁移或新账号系统。

## 2. Authority、Design Basis 与产品边界

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[Project boundaries](project-boundaries.md)：只有 EXTRACT / RETIRE，无 Compatibility 类别。
- 已确认的[独立 SHEIN worker 身份/持久化设计](shein-worker-identity-persistence-boundary.md)：保留原 provider、数字账号、隔离/查重、Store 同步、Redis、Profile、会话和关闭责任；不接入当前 Store，不改变该产品决定。
- 已确认的[#824 支撑闭包](legacy-numeric-listing-support-closure-retirement-boundary.md)与[#827 Publishing 闭包](legacy-shein-publishing-runtime-closure-retirement-boundary.md)：此前保留 Store 是为实际消费者；本设计只为 §5 明列的无消费者声明/字段提出更窄退休边界，历史证据及其余保留矩阵不变。
- [当前 Store](store-center-current-application-v1.md)、[平台观察](store-center-platform-observations-v1.md)、[当前 SHEIN 官方执行](my-supply-chain-shein-v1.md)：当前 Organization、Store、凭据/merchant binding、Product/Listing/Submission 保持其各自 owner。
- 用户已授权本整批设计，明确要求设计确认后再实施、保留全部此前有效能力、两套 workflow 不改、不操作实例/数据、不增加试用前置。UI/IA/Figma：N/A，无新入口。

选择 Independent Architecture，是因为本批收缩旧内部 repository/DTO/查询合同及其 GORM 读取投影，并涉及 owner/租户安全测试的处置；不能用“无人调用”代替明确合同矩阵。设计不新增或改变当前身份/授权、事实 owner、状态机、事务、恢复、计费或不可重复副作用。

Must：保留当前采集、草稿/严格 codec/离线校验、有效平台 API/DTO/规则、两种 SHEIN 属性、PriceManager/PriceCalculator、AI/SKU/spec/图像/Prompt、共享合同、官方 Submission Kernel、Store 观察、独立 SHEIN 登录 worker 及其 Store 同步。有效 tenant/owner、共享读取、管理员、有效期、分页、已删除行隔离与失败关闭语义保持。

Should：完整闭包、单 Writer、一个主要 PR，复用原规则/测试与检查。无新增 Accepted Risks。

Out of Scope：新字段/表/索引/初始化或导入入口，旧 ID 映射、兼容包装、fallback、双读/双写、第二事实源；旧数据库/已有列或行、Redis/Profile/凭据/平台/provider/Prompt 记录操作；部署、关父 Issue、worker 重构或接入当前 Store、新恢复/调度/验收系统。所有 workflow、依赖、`.golangci.yml`、ceilings、保护规则不改。

Threat Model：误删活跃 worker/平台合同，按字段同名误删共享规则，以删除旧测试为由削弱 tenant/owner/active-row 隔离，把源码退出误报为数据删除或运行验收。不扩张为新 IAM、凭据、事务或极端故障治理任务。

## 3. 实际消费者：contract → implementation → injection → consumer

| 保留链路 | 实际证据与需要的合同 | 决定 |
| --- | --- | --- |
| `cmd/shein-login-worker` → `app/runtime/sheinloginworker` → `listingadmin.NewGormStoreRepository/NewGormStoreAPI` → `sheinlogin/bootstrap.BuildService` | 原 config.Database handle；原 Repository/StoreAPI 注入，无当前 Store Center 接线 | 组装、DB 生命周期、配置、健康/关闭与所有生产 worker 文件原样保留 |
| `sheinlogin.ListingAdminAccountProvider` → 原 `ListStores` | TenantID、Platform=SHEIN、ReadAccess=true、Page/PageSize；200 分页；tenant/user/roles 缓存键与 5 秒缓存；账号映射读取 ID/TenantID/StoreID/Name/Platform/Username/Password/LoginURL/Proxy | 读合同、有效账号映射与缓存不变。当前路径不消费 §5 的配置字段，亦不按 EnableAutoLogin 过滤；删除该孤立字段不改变实际登录选择 |
| 原 `gormStoreAPI` → `sheinloginmanaged` → `Service` | PageStores 的 tenant/分页/完整行检查；GetStore 的有效期读取；UpdateStoreId、UpdateStoreStatus 与原同步/查重失败关闭 | 保留全部行为、原 context/owner 路径及 #722 修复；不增加后台查询或写回入口 |
| `temu/api/client.StoreRuntime` → 原 `StoreAPI` | GetStore 的 Proxy/StoreID、GetStoreCookie、UpdateStoreId，以及认证过期的 SetStorePauseStatus；原 manager/Cookie/MallID/transport | 整个 TEMU API/client 源码与 StoreRuntime/ClientAPI 原样保留；不删除仍被平台调用的成员，不新增 Cookie/暂停实现 |
| 原 StoreAPI 其余成员及 DTO | GetStore、PageStores、GetStoreCookie、UpdateStoreId、UpdateStoreStatus、DeleteStoreCookie、SetStorePauseStatus、GetStorePauseStatus、GetStorePauseStatusDetail 共九方法；StorePauseStatusRespDTO 等 | 九方法签名、相关 DTO 与原 GORM adapter 的未配置/false/nil 行为保留。保留合同不表示未配置能力已可用，不借本批建设 Redis provider |
| 同包的 SensitiveWord / GenerationTopicPolicy / GenerationTopicOverride | 当前敏感词/授权品牌清理、话题策略读取、owner 拒绝与原 schema/helper | 所有生产代码及有效测试原样保留；不因同包、同字段名或无直接 setter 调用而退休 |
| `listingruntime` → 当前 pricing / SHEIN pricing/activity/authorizedbrand/productsync | StoreInfo、ProductImportMapping、PricingRule、OperationStrategy 的完整值合同 | 三个生产文件全部原样保留，包括历史命名字段；不是本批旧 Store 仓库投影的删除目标 |

实际 tracked Go 调查覆盖根模块与 hack/tools 嵌套模块：七项旧管理方法除声明、接口和专属测试 stub 外无生产调用；§5 配置字段在 `listingadmin` 内仅被旧通用写入、查询或 DTO 映射传递。包外同名字段有独立类型，按类型/owner 保留。未查询数据库、凭据或 live runtime；源码可达性不等于部署/用户可用性证据。

根因是宽 Store 合同把登录所需读取/同步和已经没有入口的刊登管理写入、配置投影混在一起。包仍被 import 不能证明所有方法/字段仍是需求；本批按准确 receiver/type 和调用边收口，不另造 Service、port 或 facade。

## 4. 持久化、状态与副作用 owner

| 边界 | 唯一原 owner / 保留语义 | 本批效果与限制 |
| --- | --- | --- |
| `listing_store` 原账号记录 | 原 listingadmin Repository、数字 tenant/Store 行 ID；不是当前 Organization/StoreID | 只缩减 §5 源码投影。ID/tenant/owner、账号、代理、店铺身份、状态/备注、有效期、audit/timestamps、deleted 等其余字段/tag 不变；无 DROP/ALTER/清表/回填/迁移操作 |
| 原账号 schema 辅助 | `AutoMigrateStoreRepository` 当前仅调用 `ensureOwnerAuditColumns`，不是旧表创建器 | 函数、schema_migrate/owner_constraint 全部原样保留且不执行；不新增 worker 初始化或导入链。隔离测试只按原 SQLite fixture 建任务内表 |
| 原同步写入 | UpdateStoreID / UpdateStoreStatus，原 Find→Update→Read、owner/tenant 与 active-row 条件 | 返回值、错误、字符串/空备注处理和原非跨系统事务边界不变。删除无入口的通用写入不转交另一个 owner，不增加事务/重试/补偿 |
| Redis / attempt / browser / Profile / Cookie | 原 sheinlogin Service/Runtime/RedisStore 与原 worker | 键、TTL、状态机、lease、取消/重启、并发、UNKNOWN/恢复、外部登录及 Store 同步责任全部原样保留；本设计不执行任何动作 |
| 当前 Store / 官方商家事实及执行 | 当前 storecenter / Product / Listing / Submission / integration | 无 diff；不从旧行或登录态补齐当前官方凭据/merchant binding，不改变唯一发送、幂等/claim/fence/UNKNOWN owner |
| 物理数据库/历史资产 | 真实表/列/行、Profile、备份、运行实例的原 owner | 存在性与当前值 UNKNOWN；本批不读取、不处理，也不声称字段或记录已物理消失。源码合同退休与数据生命周期授权分开 |

查询、超时、取消、重复调用与并发沿原实现；不增加请求、载荷、缓存或新的故障边界。失败/响应丢失、重启/取消时的 DB、Redis、Profile、外部服务结果仍由原协议负责，不从本次删除推导出新的原子性或恢复保证。当前权限撤销和缓存期限也不改。

## 5. 完整删除、保留与复用矩阵

### 5.1 七个 mixed 生产文件；无整生产文件删除、无新增抽取

| 准确文件 | 允许的退休/机械收口 | 必须保留 |
| --- | --- | --- |
| `internal/listingadmin/store_repository.go` | 从 GormStoreRepository 删除 CreateStore、UpdateStore、DeleteStore、ListDeletedStores、RestoreStore、PermanentlyDeleteStore、ExtendStoreValidity；只清 unused import；现用 helper 的 deleted=0 实参随 §5.3 收口 | 类型/构造、AutoMigrateStoreRepository、ListStores、GetStore、UpdateStoreID、UpdateStoreStatus、FindStoreByID；原 scope、有效期、分页、返回/错误及 current write 行为 |
| `internal/listingadmin/store.go` | StoreRepository 同七方法；Store/listingStore 各删除附录 A 的 17 字段；StoreQuery 删除附录 B 的 15 字段 | Repository 四方法签名；StorePage；Store/listingStore 其余字段/tag/TableName 原样；StoreQuery 保留 TenantID、OwnerUserID、ReadAccess、Page、PageSize、Platform |
| `internal/listingadmin/store_model.go` | 删除 listingStoreFromStore、applyStoreCreateDefaults；toStore 只去 17 对应赋值 | toStore 其余赋值及 firstNonEmptyStoreValue、原 audit/owner 表达式原样 |
| `internal/listingadmin/store_query.go` | 删除 findStoreRows、parseTenantID；applyStoreQuery 只去 15 字段对应条件/回收站分支，active-row scope 按 §5.3 保留；清 unused import | applyStoreAccessScope 原样；原 ReadAccess/tenant/owner、Platform trim+大小写、分页 owner 不改 |
| `internal/listingadmin/store_access.go` | 删除 deleteStoreAccessRow；takeStoreAccessRow/updateStoreAccessRow 仅按 §5.3 删除孤立 deleted 参数 | storeAccessQuery、storeReadAccessQuery、applyStoreReadOwnerScope 及原错误/RowsAffected 判断、tenant/owner 条件保持 |
| `internal/listingadmin/management_api_types.go` | StoreRespDTO 去附录 A 的 17 字段（Sku 大小写按原类型）；StorePageReqDTO 只去 EnableAutoPrice | StoreAPI 全九方法、PageResult、StoreIdUpdateReqDTO、StoreStatusUpdateReqDTO、StorePauseStatusRespDTO 与其它字段/tag 不变 |
| `internal/listingadmin/gorm_store_api.go` | PageStores 只去 EnableAutoPrice 赋值；StoreToRespDTO 只去 17 对应赋值 | 所有九方法、cookie 未配置错误/暂停等原返回值、GetStore 有效期、StoreToRespDTO 其余字段与 FlexibleTime 转换原样 |

总计：七个旧 repository 方法、五个专属 helper（listingStoreFromStore / applyStoreCreateDefaults / findStoreRows / parseTenantID / deleteStoreAccessRow）退出；五种 scoped struct 合计 **67 字段**退出；两个私有 helper 的孤立 deleted 参数收口。不能用方法前缀误删 UpdateStoreID/UpdateStoreStatus，不能对同名字段全仓替换。

### 5.2 原 owner 与整文件保留边界

- 原 worker/service/bootstrap/managed bridge **生产文件零改**；不改 AccountProvider、账号映射/缓存、原 build/runtime 注入、同步/查重、Profile、Redis 或状态机。
- 整个 `internal/temu/api`、SHEIN/其它平台 API/DTO/规则、当前 Product/Listing/Submission/Store Center/observations 的生产与现用 schema、所有当前应用接线零改。
- `internal/listingruntime/types.go`、`rules.go`、`operation_strategy.go` 全部零改；四个共享值合同含全部字段/方法保留。当前 PriceManager/PriceCalculator、品牌规则与两种 SHEIN 属性不跟随旧 Store 投影退出。
- listingadmin 的 access_scope、owner_constraint、repo_paging、schema_migrate、SensitiveWord、GenerationTopicPolicy/Override 生产代码零改。保留 schema 安全约束、owner/audit 规则与当前词库/话题能力。
- 未列入 §5.1 的生产代码均不改；无新包/接口/依赖、兼容层、原 Service 包装、fallback 或第二数据 owner。

### 5.3 active-row 与安全不变量的最小收口

1. StoreQuery.Deleted 退出后，ListStores 仍先限定 `deleted = 0`。ReadAccess=true 继续按原 tenant + owner-or-shared；否则继续用原 applyOwnedTenantQuery。不能删除过滤、把回收站分支留作默认，或放宽 owner context。
2. takeStoreAccessRow / updateStoreAccessRow 删除 deleted int16 参数；所有保留调用原来均传 0，因此各自 Where 固定为 `id = ? AND deleted = 0`。其它 scope、错误映射、更新字典与 RowsAffected 判断保持，不改变 GetStore/StoreID/Status 的现用行为。
3. applyStoreAccessScope 仍是原 tenant/owner 辅助，不负责 deleted；FindStoreByID 的显式 `id/deleted=0` 条件保持。其独立权限断言保留。
4. 当前品牌/文本规则沿既有 listingruntime/authorizedbrand/submitprep 等 owner 保留；删除 numeric Store 的三项品牌投影及旧映射测试不删除当前授权品牌规则或有效断言。

Legacy decision: **RETIRE**。Reusable behavior：现用账号读/同步、平台/共享规则与安全行为沿原 owner 保留，无新增 EXTRACT。Current owner：原 listingadmin 账号持久化与原独立 SHEIN worker、平台 API，以及当前 Store/Domain 各自 owner。Cutover/deletion condition：设计确认、适用独立检查、Issue Ready 后，原精确 guard RED→同批删除/测试转接 GREEN→原 CI/最终独立检查；无数据或运行切换。

## 6. 实施与验证：仍在同一个主要 PR

后续获明确实施授权及 Ready 后：先扩展原 guard 捕获 §5 目标仍在的 RED，再一次完成七 mixed 文件及 §6.1 测试；仅运行必要开发自检与稳定候选原 CI。设计阶段不执行这些实施测试、不构建/启动 worker、不访问配置/数据/provider。

### 6.1 原测试与检查的准确处置

| 原文件/测试 | 允许变化 | 必须保留的证明 |
| --- | --- | --- |
| `internal/listingadmin/store_model_test.go` | 退休 TestApplyStoreCreateDefaultsFillsRegionDailyLimitTypeAndAuditNames、TestListingStoreToStoreIncludesAuthorizedBrandFields；TestTakeStoreAccessRowRespectsDeletedState 转到当前 GetStore/写入拒绝 deleted 行；TestFindStoreRowsSupportsDeletedListingScope 转到 ListStores 仅 active 且 tenant/owner 隔离 | 不再证明旧创建/品牌投影或回收站可用；仍证明 deleted 不可通过当前读/写绕过，tenant/owner 不混淆 |
| 同文件的 TestApplyStoreQuerySupportsSharedReadAccess / TestApplyStoreQueryMatchesPlatformCaseInsensitively | 调用退休 findStoreRows 的 fixture 转用原 ListStores，按 page.Items 断言；不新造生产 wrapper 或通用 fixture 系统 | 原 own+shared 可读、other-owner 排除、平台大小写匹配及有效分页语义 |
| 同文件其余四测试 | 原样保留：TestListingStoreToStoreUsesAuditFallbacks、TestApplyStoreAccessScopeHonorsTenantAndOwnerWithoutDeletedFilter、TestGormStoreRepositoryGetStorePlatformAdminBypassesOwnerScope、TestGormStoreRepositoryGetStoreAllowsLegacyUnownedStoreRead | 原审计投影、辅助 scope、管理员/原共享读取边界；保留 worker 原协议不使其成为当前 Organization 的来源 |
| `internal/listingadmin/owner_scoped_repository_test.go` | 只退 TestCreateStoreRejectsOwnerlessWrite、TestCreateStorePersistsTrustedInternalOwner，因对应生产创建入口消失 | 两个共享 SQLite/assert helper，GenerationTopicPolicy/Override、SensitiveWord 的拒绝空 owner 与零写断言全部原样；owner_scope/access_scope/schema_migrate 原测试不动 |
| `internal/listingadmin/store_api_adapter_test.go` | 只删 fixture 的 autoLogin 变量/EnableAutoLogin 字段，不删测试或断言 | TestNewGormStoreAPISupportsSheinWorkerStoreOperations 的分页、Cookie 未配置、StoreID/Status/备注写回全部保留 |
| `internal/sheinlogin/accounts_test.go` / `internal/sheinlogin/bootstrap/build_test.go` | 分别只删 stubListingAdminAccountStore / stubStoreRepository 上与退休七方法同名的 dummy methods；其它方法、fixture/测试/断言不变 | 原完整分页、cache 的 tenant/user/roles 隔离、BuildService 无 repository 拒绝及 Service/Close 保留；不删除当前 provider 所需 GetStore/ListStores |
| `tests/listingkit_backend_retirement_test.go` | 在既有退休 guard 中增加按 package/type/receiver 定界的准确 12 声明、67 字段和两个参数/active-row 边界断言；复用现有 Go AST/文件检查，不创建 checker/runner/framework | 当前 Store/API/worker/平台 required tokens、命令依赖、安全与防复活原检查保持；同名共享字段不误报，读取/解析错误仍失败 |

其它测试文件零改；无整测试文件删除。特别保留 `sheinloginmanaged` 查重 tenant/分页/foreign-row/取消/错误失败关闭原测试、所有 worker/runtime/Store 同步测试及 TEMU API 测试。原 owner_scope/access_scope/repo_paging/schema/当前品牌/codec 断言不弱化。

必要验证：七生产/五 mixed 测试/一个原 guard 的准确 diff；原 source guard RED→GREEN；listingadmin、sheinlogin/bootstrap、sheinloginmanaged、app/runtime/sheinloginworker 与 TEMU API 相关原测试；current-application / shein-login-worker 仅编译；原稳定候选 CI。安全转接测试若有业务断言改变，先确认目标 RED，再最小实现；纯声明/fixture 删除不伪造行为 RED。

可随实施窄校准的文档仅为本设计、README、legacy-register、module-target-mapping、上述 worker/#824/#827 设计的直接冲突保留说明；只补准确 §5 的 superseded 注记，不重写历史证据或其它保留范围。**phase2-runtime-inventory 当前行/历史行/计数零改，tests/import_boundaries_test.go、两套及全部 workflow、依赖、.golangci.yml、ceilings/保护零改**；本批没有包/导入边退出。

Runtime/provider/user/production acceptance：**NOT_RUN**；本地测试/CI/源码检查均不替代真实运行或用户验收，不增加手工验收工具/矩阵。

## 7. 确认、准入与停止点

1. 当前仅设计文档/只读调查。取得适用独立设计结论和用户确认后，Issue 明记 Independent Architecture / IMPLEMENTATION_READY / Ready；再由用户明确授权按 §5、§6、§6.1、§7 在本 PR 实施。设计就绪不自动开始编码。
2. 单一 Writer 复用原独占 worktree/主要分支，不新增多 Writer 或机械拆 PR。原 GitHub 设计复核同 HEAD 运行时不重复触发；最多两轮正常架构检查，非 Blocker 收敛为 IMPLEMENTATION_TEST/BACKLOG。
3. 实现发现 §5 之外的真实当前消费者、当前外部可观察契约，或必须改变身份/tenant/事实/事务/状态/恢复/副作用时，停止对应修改，提供当前证据与最小差异；不能静默删除消费者、增加兼容或用测试恢复旧管理 owner。
4. 实现收敛后一次最终独立交付检查；修复只复核相关增量，不重复已确认代码。用户验收、准确最终 HEAD 合并、matching main CI、部署和真实操作分别记录授权。
5. 本批无法关闭 #301 的独立 worker 数字账号债务，不宣称整个 #29/#301 完成；保留当前能力和试用阶段决定，不追加治理项目或新前置。

## 附录 A：三个 Store 投影的 17 个准确字段

以下按原 `internal/listingadmin` 的类型定界。Store 与 listingStore 各退出同一组 17 字段；StoreRespDTO 只把 SKUGenerateStrategy 对应为其原字段名 **SkuGenerateStrategy**，其它 16 名一致。各字段在 toStore / StoreToRespDTO 的直接映射同时退出；不删除任何其它类型的同名字段。

```text
DailyLimit
DailyLimitType
FixedStockCount
SKUGenerateStrategy
Prefix
Suffix
EnableAutoListing
EnableAutoLogin
EnableDraft
EnableAutoPrice
EnableRebargain
EnableBrandAuthorization
AuthorizedBrandCode
AuthorizedBrandName
TemuPriceRejectStrategy
PriceType
DedicatedQueueEnabled
```

对应原 GORM 列名只用于源码审计，**不是物理删除清单**：daily_limit、daily_limit_type、fixed_stock_count、sku_generate_strategy、prefix、suffix、enable_auto_listing、enable_auto_login、enable_draft、enable_auto_price、enable_rebargain、enable_brand_authorization、authorized_brand_code、authorized_brand_name、temu_price_reject_strategy、price_type、dedicated_queue_enabled。

## 附录 B：StoreQuery 的 15 个准确字段

```text
StoreID
Name
Username
ShopType
Region
SKUGenerate
EnableAutoListing
EnableAutoLogin
EnableDraft
EnableAutoPrice
EnableRebargain
PriceType
Status
Expired
Deleted
```

另只退 **StorePageReqDTO.EnableAutoPrice** 及 gormStoreAPI.PageStores 的对应赋值。总数为 Store 17 + listingStore 17 + StoreRespDTO 17 + StoreQuery 15 + StorePageReqDTO 1 = **67**。Store/listingStore 各自原有的 StoreID、Name、Username、ShopType、Region、Status、Expired、Deleted 等不因 Query 同名字段退出而删除。
