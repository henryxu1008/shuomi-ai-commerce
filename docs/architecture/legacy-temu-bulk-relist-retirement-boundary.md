# 旧 TEMU 批量重新上架链退休边界

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **DESIGN_ONLY / 待用户确认**；本轮不实施源码退休，设计评审或 CI 通过不代表生产修改已获准。
- 调查基线：`main @ 86c173d5b8d2ed355260a3a2e34d165b4c84064e`。#744 的合并及对应 push/main CI 归原 PR，不替代本设计候选的验证。
- 当前 Design Basis: **N/A**，理由是只读消费者/调用链调查和文档，不改变产品、领域合同、状态、权限、持久化或平台副作用语义。未来实施拟采用 **Reuse Existing Architecture**，须先由用户确认 §5、§6、§6.1 和 §7，再在 Issue 记录冻结依据与 Ready；不能把本设计自动标为实施准入。
- 用户边界：保留平台 API、有效规则、共享合同、EnrichedSkuInfo/EnrichedSkcInfo、PriceManager/PriceCalculator、当前采集、Store 观察同步、独立 SHEIN 登录 worker 及其 Store 同步。两套 workflow 不改，不操作实例或数据，不增加内部试用前置。

## 1. 用户结果、Scope 与 Out of Scope

使用者为维护当前内部试用、平台能力和独立登录 worker 的开发维护者。交付是可逐项确认的批量重新上架源码闭包、消费者证据、持久化及副作用归属和删除边界，防止误把无当前装配的旧数字 Store 编排重新接入，也防止按“relist”名字误删平台 API 与共享合同。本项不解除或新增内部试用阻塞；当前采集、观察、登录与 Store sync 继续沿原路径使用。

覆盖 `internal/temu/bulkrelist` 的数字 Store 入口、预览、离线商品分页/筛选、SKU 去重与分组、并发/延迟、第一页循环、多 SKU 批量转单 SKU 调用、内存结果与专属 DTO。该包有 **7 个生产 Go 文件、0 个测试，共 1,272 行**。建议整体 RETIRE，不新增 relist 产品、UI/路由、调度器、规则引擎、运行入口或恢复 owner。调查及未来实现复用原 API、Go 和现有 guard/测试，无需新开源依赖或自建平台。

不读取真实配置/凭据、DB、Redis、浏览器/Profile 或平台状态，不启动任何命令 main，不调用旧预览、DryRun、SearchOffline、Relist，不启停/部署实例，不迁移、删除、补写、重放或恢复数据，不处理 #746/#692 等其他活跃范围。源码退出不代表真实平台状态或历史运行事实消失。

## 2. Design Basis、Authority 与不变量

适用依据：

- [全新系统基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)：仅 EXTRACT / RETIRE；不新增旧 ID 映射、兼容壳、wrapper、fallback、双读双写或第二事实源。
- [旧 CLI](legacy-listing-cli-retirement-boundary.md)、[control-plane](legacy-listing-control-plane-retirement-boundary.md)、[数字执行链](legacy-numeric-task-execution-retirement-boundary.md)、[任务持久化](legacy-numeric-task-persistence-retirement-boundary.md)、[商品/库存同步](legacy-numeric-product-inventory-sync-retirement-boundary.md)、[自动核价](legacy-auto-pricing-retirement-boundary.md)、[SHEIN 活动](legacy-shein-activity-retirement-boundary.md)：本设计不重开已经退休的执行链，只处理本次 TEMU 批量 relist 闭包。用户确认后，仅替代先前对 §5 对象保留待后续决定的部分，旧证据不重写。
- [独立 SHEIN worker 身份/持久化](shein-worker-identity-persistence-boundary.md)：账号、Cookie/attempt、Redis/browser/Profile、同 tenant 查重和 Store sync 保持原 owner/格式/权限/恢复边界。
- [当前 Store](store-center-current-application-v1.md)、[Store observations](store-center-platform-observations-v1.md)、[准备接线](store-observations-preparation-runtime.md)、[当前 SHEIN 业务](my-supply-chain-shein-v1.md)：当前 Organization、官方连接、观察、Product/Submission、retry/UNKNOWN 不交给旧 relist 服务。没有 UI/IA 变化，Figma N/A。

未来拟复用以上架构、原平台 transport/API 和 Store 合同，只退出无装配的专属调用方，不新增或转交事实/schema owner、公共当前领域合同、身份/权限/tenant、状态机、事务、幂等、retry/UNKNOWN/recovery 或平台 mutation owner。7 文件删除及既有 guard/文档校准有明确边界，不为满足阈值拆 PR 或搬包。若新证据要求改变上述保留边界，停止相应部分并走适用 Independent Architecture，不在实施中临时发明兼容。

Must：§5 保留能力完整；有效平台规则、凭据/权限/租户/owner 和安全检查不减；两套 workflow 与保护零改动；持久化、平台事实及历史恢复不移交。Should：当前源码地图与实际依赖相符。Out of Scope 如 §1；无新增 Accepted Risk。Threat Model 只覆盖误删能力、误把导出符号当现行合同、错误转交持久化/副作用或把源码缺席当运行态清零，不增加全局 hardening 或试用验收要求。

## 3. 实际消费者：合同 → 实现 → 注入 → 入口

以下为调查基线的静态证据，不是运行实例审计：

| 对象 | 实际实现与消费/注入 | 结论 |
| --- | --- | --- |
| BulkRelistEntry / BulkRelistService | 全仓 Go import 无包外 `task-processor/internal/temu/bulkrelist` 引用；两个 New 构造器只在定义及包内调用出现，cmd/app/HTTP/worker 无注入 | 没有当前用户入口或需切换的调用方；不能由导出名字推定外部兼容义务 |
| 全部当前命令 | 执行 `go list ./cmd/...` 得 31 包，`go list -deps -f '{{.ImportPath}}' ./cmd/...` 中 bulkrelist 精确路径及子路径命中 0；未执行命令 main | 当前采集、Store observations、独立登录等入口不编译本包；实现前再次刷新 |
| Entry → client → service | Simple/Custom/Filtered/QuickRelist 以 int64 StoreID、client.StoreRuntime 建客户端；service 构造 inventory API、ProductFilter、ProductProcessor、BatchProcessor、PageLoopProcessor | 整条旧装配退出；不替换成当前 Organization/Store adapter 或新 Submission owner |
| Preview / Batch / PageLoop | Preview 取所有页按 GoodsID 生成摘要；Batch 按 SkuID 去重再按 GoodsID 分组；第一页循环、全页与过滤流均为包内实现 | 分页、预览、选品、循环计数不是当前 Store observations 合同，随调用方退出 |
| ProductFilter / 专属类型 | ShouldSkipProduct、GetSkipReason、MatchesFilter 及所有 7 个 types.go 类型只有包内消费；contains 仅委托 shared/strx.Contains | 没有当前独立规则合同；按 §5 退休专属筛选策略，保留共享 strx 与现有有效平台规则，不造无消费者的公共规则 API |
| inventory / client / API 汇总 | `internal/temu/api/inventory` 和 `api/client` 是依赖目标；api/api.go 导出 InventoryAPI、客户端及请求/响应类型，另导出 product.BulkRelistOptions / BulkRelistSummary | 不把“本 caller 无消费者”误推成其依赖可删；保留平台 API 全包及既有导出合同 |

调查方法为逐文件闭包阅读、全仓 Go import/构造器/类型检索及全部 cmd 依赖图；`tests/import_boundaries_test.go` 的 allowlist/root 字符串和历史文档不是业务消费者。同名 Amazon crawler BatchProcessor、TEMU product/filter 与 API product DTO 是其他包，不能按名字全局删。

### 已识别的架构/协议问题

旧包把数字 Store 凭据装配、选品、平台写入和结果计数混在内存服务中，不能作为当前持久化或恢复 owner：

- `DryRun` / `PrintProductData` 只声明、未被执行代码读取；设置 DryRun 仍会进入 Relist，不能当无副作用试运行。Preview 虽无直接 Relist，客户端构造和认证处理仍可调用 Store 写接口（§4）。
- ShouldSkipProduct 的 SkipLocked 分支在 AllowOperate 为 true 时跳过，GetSkipReason 则在 false 时报告锁定；SkipNeedRectification 只在原因中检查，MatchesFilter 也未执行声明的 HasPunishTags / NeedRectification。日志推测的 PunishTags=1、ShowSubStatus4VO=3001 不构成平台 Authority。
- 多 SKU 请求在无 error 且响应 nil/Result.Result=false 时会再逐 SKU 发送；任一 SKU 成功即归成功组，调用方不保留该成功组的部分失败说明。transport error 分支只返回错误，不走上述单 SKU 分支。内存 success/日志不证明 exactly-once、完整成功或响应丢失后可安全重发。
- Batch 拉页错误/nil 响应会跳过该页并继续，第一页循环以连续 3 轮无成功退出，过滤流边读边写后递增页号；这些局部停止/继续条件不是 durable checkpoint、取消或恢复协议。

这些是静态源码问题，未断言真实事故。处置是退休闲置编排，不修补后重新开放、不新增 outbox/Saga/重试系统，也不将它们作为当前内部试用的新前置。

## 4. 持久化、平台副作用与失败归属

| 边界 | 基线 owner / 真实调用能力 | 本次删除条件及保留边界 |
| --- | --- | --- |
| 本包结果/预览/选项 | RelistAllResult、Results、preview、map/channel/WaitGroup、计数、延迟仅在进程内；7 文件无 repo/DB/Redis/文件写入、schema 或 durable receipt | 随专属包退出，不迁移到当前 Store/Observation/Submission，不创建执行表或接管历史结果；日志不作为业务事实/恢复依据 |
| listing_store | listingadmin.StoreAPI / GormStoreRepository 拥有原 Store 合同；client.NewAPIClient 读取 Store/代理和 Cookie，initializeMallID 可调用 UpdateStoreId；GORM adapter 委托 UpdateStoreID 写原 listing_store | 只删除 bulkrelist 的调用方，原模型/字段/API/仓库、owner/tenant/权限、事务及 worker Store 写回/查重不改；不增加数字 ID→当前 ID 映射 |
| Cookie / 暂停接口 | CookieManager 经 StoreRuntime.GetStoreAPI().GetStoreCookie 读取；共享认证暂停 handler 可调用 SetStorePauseStatus。**基线 GORM adapter 的 Cookie 返回未配置错误，pause 返回 false/nil，并无实际 Redis 写实现** | 不能把接口能力说成当前已启用持久化；不新增 Cookie/pause provider，不补齐 adapter，不改 shared client/auth/reload/retry 语义，不读写真实 Redis/Cookie/暂停状态 |
| 平台读取 | inventory.SearchOffline POST /mms/marigold/sku/v2/search；请求/响应、分页限制及鉴权/transport 归原 API/client | 平台拥有商品/离线上下架事实；保留 API、DTO、wire/错误/凭据合同。只退出专属调用方，不发请求或用预览证明安全 |
| 平台写入 | inventory.Relist POST /mms/marigold/sku/online；GoodsID/SkuIDs 与平台响应经原 client 发送；该 API 根据顶层 Success 返回 error，processor 另检查 Result.Result | TEMU 平台拥有 SKU 上架结果；旧 processor 只编排调用，没有跨 Store DB/平台的共享事务或 durable 幂等凭据。API 原样保留，批量/单 SKU fallback 编排退出；不撤回/下架/补发/补偿任何商品 |
| 当前业务及独立 SHEIN 登录 | 原 Organization/Store/Observation/Submission、账号 DB、attempt/stream/Lua、锁、browser/Profile、Cookie 与 Store sync | 保持原 owner、格式、事务、权限、恢复和运行边界；TEMU caller 退休不转移 SHEIN 身份、Store sync 或当前观察事实 |

构造器、Preview、DryRun 都不作为本次验证入口。失败、响应丢失、重试、重启、取消及并发后的平台事实，由原平台及对应既有运行 owner 负责；本设计不根据内存结果判断已提交/未提交/已回滚，不转交 UNKNOWN、在途操作、last writer 或恢复权限。没有本包持久化表不代表外部业务资产为空；未扫描或操作真实运行态，不能以源码退出授权清理数据、锁、会话、实例或历史平台状态。

## 5. 拟删除与明确保留矩阵（待用户确认）

以下路径相对 `internal/temu/bulkrelist/`。全部为 RETIRE，7 个文件完整退出；本包没有需独立抽取的当前有效规则合同，不留旧导出 DTO、空服务、alias、wrapper 或相同状态字段的私有复制壳。

| 对象 | 拟处置 | 删除条件 / 明确保留 |
| --- | --- | --- |
| bulk_relist_entry.go（150 行） | RETIRE 全文件：Entry/constructor、Simple/Custom/Filtered、Preview、三个 QuickRelist 方法、数字 Store 接线 | 全仓包外 import/构造器刷新仍为空；不以新 HTTP/CLI/worker 接替入口，不改共享 StoreRuntime / StoreAPI |
| bulk_relist_service.go（144 行） | RETIRE 全文件：service/constructor、默认选项、all/filter/batch 装配 | 不留服务壳或为保留 exported 名字包装当前 owner |
| bulk_relist_batch.go（185 行） | RETIRE 全文件：OfflineAPI、BatchProcessor、拉页、SkuID 去重及分批/合并/延迟 | 专属算法没有当前消费者；不抽成通用批处理系统，Amazon crawler 的同名类型不动 |
| bulk_relist_page_loop.go（253 行） | RETIRE 全文件：PageLoopProcessor、第一页/全部页/过滤流、停止计数和结果合并 | 不迁移内存进度，不新增 checkpoint/recovery，不改变平台 SearchOffline |
| bulk_relist_processor.go（282 行） | RETIRE 全文件：ProductProcessor、分组、串并发、Relist 及逐 SKU fallback | 不保留单独可调用副作用 helper，不改变 inventory API 的线上端点或请求/响应 |
| bulk_relist_filter.go（183 行） | RETIRE 全文件：ProductFilter、skip/reason、MatchesFilter 及 contains wrapper | 类目/关键词/价格/库存范围仅为旧 relist 选品条件，未构成当前平台规则或共享合同；锁定/处罚推测不升级为新规则。shared/strx、独立 product/filter、temu/rules 及 marketplace 规则原样保留 |
| types.go（75 行） | RETIRE 全文件：BulkRelistOptions、SkipConditions、ProductFilterOptions、RelistAllResult、RelistDetailResult、OfflineProductPreview、OfflineProductSummary | 均为本包专属、非 durable 结果/选项；**不删 API product/request_types.go 同名 BulkRelistOptions / SkipConditions、BulkRelistSummary / CategoryStats，也不删 api/api.go 既有 re-export** |
| 平台 API / 有效规则 / 共享合同 | 原样保留 internal/temu/api 全树（含 client、inventory、product 和汇总）、temu/rules、temu/filter、temu/product、marketplace 规则、shared/strx、listingadmin/root listingruntime 及配置 | 不因本包退出删端点、字段、错误/认证处理、凭据保护、规则/策略或其它消费者合同；API 同名类型此轮明确不判退休 |
| 用户明确保留能力 | 原样保留两种 SHEIN 属性、全部 PriceManager/PriceCalculator、当前采集、Store 观察同步、独立 SHEIN 登录 worker 和 Store sync | 包括 #744 已保留的 SHEIN 营销纯规则/API 与原有效测试；不改其任何接线、持久化、权限/tenant/恢复边界；两套 workflow 不变 |

Legacy decision: **RETIRE**。Reusable behavior: 已有平台 API、规则及 strx 原 owner 继续保留；无需从本包 EXTRACT。Current owner: 平台 transport/API、listingadmin Store、当前业务/独立登录各守原职责。Cutover/deletion condition: 用户确认矩阵，Issue Ready，准确候选无当前外部消费者，§6/§6.1 原检查继续有效。若发现具体当前兼容义务、durable asset/副作用 owner 或被遗漏的有效规则消费者，停止对应项并报告，不能自行新增兼容、删断言或放宽 allowlist。

## 6. 测试和安全规则处置

本包基线没有 _test.go；不制造旧服务运行测试或通过真实 dry-run 证明退出，也不为纯设计修改测试。实施时先在现有退休 guard 加入目标缺席与保留合同断言，确认它们在旧源码仍在时 **RED**，再删除闭包并 **GREEN**；不用重建旧协议的行为测试。

保留现有平台 API/client、有效规则/定价、listingadmin scoped/tenant/owner、SHEIN 两种属性及营销 wire、当前采集/observations 和独立登录/Store sync 测试。inventory 当前只有 inventory.go，不声称有尚不存在的 inventory 专属 wire 测试；沿用已有检查和必要编译，不借清理新增测试平台或完整故障矩阵。

未来必要开发检查：现有 source-absence/command-dependency/独立 worker 与 TEMU runtime import guards；本次保留 API/client/规则/共享包必要编译及既有相关测试；当前命令（含 current-application、shein-login-worker）必要编译；原 CI 的完整 tests 与 listingadmin race 继续由两套原 workflow 负责，不人工重复一套 CI。静态/编译/fixture、PR CI、main CI、运行验收分别记证据，未执行保持 NOT_RUN。

## 6.1. Guard、文档与两套 workflow 的精确边界

| 既有对象 | 未来实施拟处置 | 保持有效 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 在 TestListingKitTaskBackendSourcesAreRetired 的目录列表增加 internal/temu/bulkrelist；在 TestCurrentCommandsDoNotCompileLegacyBusinessBackends 的 retired prefix 增加精确包路径 | 原有所有缺席、共享合同/有效规则、两个属性、独立 worker、workflow race 和 command 检查继续；不全仓禁用 BulkRelistOptions 等同名符号 |
| 同一现有 guard 的 retained 清单 | 增加 api/api.go、api/client/interfaces.go、client.go、cookie_manager.go、auth_pause_handler.go、api/inventory/inventory.go、api/product/request_types.go 的存在检查；在原测试内核对 InventoryAPI、Relist/SearchOffline、原请求/响应与 API product 同名类型/既有 re-export 仍在 | 按 package/path 区分旧专属类型与平台合同，避免目录删除后误删被依赖者；不新建 runner/专项平台 |
| tests/import_boundaries_test.go | 只从 TestTEMURuntimeAndBridgeRetiredManagementImportsStayBlocked 删除 bulkrelist root 和 bulk_relist_entry.go 那一条 allowlist | 消失 owner 的路径扫描由上述缺席/依赖 guard 承接；保留所有其它 root/allowlist/deny、不新增 missing skip，不改 retained API/client 的 guard |
| docs/refactoring/phase2-runtime-inventory.md | 仅移除当前 Legacy consumer register 中 core/logger → internal/temu/bulkrelist 那一行，添加本批源码退休校准说明 | DATED EVIDENCE 的旧包图、Phase 4 历史列表及所有 closure ceiling 不改，不能把历史清单当新派工 |
| docs/refactoring/legacy-register.md、本文、architecture README | 用户确认后记录该 7 文件闭包 source-retired，引用冻结设计/实施 PR；仅校准本批边界 | 不清空 legacy 全表，不宣称 #301 清零或用户验收；external-client-boundary-inventory 的 dated hotspot 原样保留 |
| .github/workflows/ci.yml / commercial-readiness.yml | **零改动** | 必需/安全、架构、代码健康、ReleaseAuthority、listingadmin race、Required CI Gate 等原规则和原 skipped 语义不降级；其它 workflow、.golangci.yml、保护及 ceiling 也不改 |

本轮实际提交范围仅本文和 README 索引；上表 guard/源码地图变更都是用户确认后的实施范围，不在设计 PR 提前落地。

## 7. 确认、实施与停止点

1. 将本设计以 Draft PR 交付，完成文档链接/索引检查、适用原 CI 和一次准确 HEAD 独立最终检查，记录证据与限制；父 #29/#301 保持 OPEN。设计检查通过仍是 DESIGN_ONLY。
2. **等待用户确认 §5、§6、§6.1 和 §7**。确认前不改生产代码、测试/guard、workflow 或持久化，不把文档 PR 合并视为实施许可。
3. 用户确认后，在同一主要分支/PR 继续；生产修改前更新 Issue 为 Reuse Existing Architecture / Ready，引用冻结设计、当前 owner/不变量/非目标和唯一 Writer。刷新实际 main、包外 import/构造器/31 命令及同名合同；若发生新高风险边界，先停对应部分并补适用准入。
4. 按 §6 在原 guard 先 RED，再按 §5 精确 RETIRE、§6.1 校准并 GREEN。相关 finding 先分类并检查 sibling paths，保留 Must/安全检查；不修补或重新开放旧批量 relist，不新建恢复/验收系统。
5. 最终候选前同步一次实际 main，仅处理真实组合问题；必要检查、准确实现 HEAD、CI、独立最终检查及限制写 PR。实现完成进入 In Review，仍须准确 HEAD 的单独合并授权；合并后才确认实际 merge SHA 对应 push/main CI，不以 PR CI 代替。
6. 本设计、源码退休和任何 CI 结果都不授权关闭 Issue、部署、真实平台/实例/数据操作，不宣称用户验收/生产上线，也不增加内部试用前置。当前采集、Store 观察、平台 API/有效规则/共享合同、两个属性、定价与独立 SHEIN 登录及 Store sync 保持原能力和 owner。
