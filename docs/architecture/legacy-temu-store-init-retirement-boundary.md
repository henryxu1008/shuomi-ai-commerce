# 旧 TEMU 店铺上下文与商品初始化链退休边界

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **USER_CONFIRMED / SOURCE_RETIRED**。用户在当前会话明确“确认设计”，批准冻结设计 `fe3ce75634c487f73bb4c32adf91736fb7943eb9` 的 §5、§6、§6.1、§7；三个目标生产文件及专属缓存/helper 已按矩阵退出。合并和真实操作仍须单独授权。
- 调查与实施开工基线：`main @ 59dc134e3a407a8697a96084d318b411aa4c0441`，已包含 #754 的筛选/AI 前置闭包退休；实施前同步 main 无增量。准确实现 HEAD、CI、独立最终检查与用户决定维护于 PR，不在本文滚动；旧设计 CI/review 仅历史证据。
- 当前 **Design Basis: Reuse Existing Architecture**：用户确认后，生产修改前已在 #29/#301 登记 **Ready**、冻结依据与唯一 Writer，再转入实施；原设计期 N/A 仅覆盖当时的文档/只读映射。
- 保留此前全部约定能力：当前 Store、平台 API/同名 DTO、有效规则/共享合同、两种 SHEIN 属性、PriceManager/PriceCalculator、Reader/Cache/Mapping、AIBatchInput/现有 SKU 映射、当前采集/Store 观察及独立 SHEIN 登录 worker/Store sync。两套及其它 workflow、必需/安全检查、保护/ceiling 不改，不操作实例或数据，不增加试用前置。

## 1. 用户结果、范围与非目标

使用者是维护当前内部试用、采集和 Store 能力的维护者。交付为旧 Task 店铺上下文与商品初始化专属闭包的源码退休，按以下矩阵保留有效行为和事实/副作用 owner，防止数字店铺缓存、旧 Task 默认值或告警被误接为当前 Store/平台规则。当前用户入口不变，不将本维护变成内部试用前置。

范围只有 `internal/temu/store/id_handler.go`（106 行）、`store/info_handler.go`（152 行）、`handlerbase/init_handler.go`（114 行），共 **3 个生产文件 / 372 行**，以及前两个文件内部独占的缓存、统计和主机名 helper。删除前 `internal/temu/store` 恰好只有上述两文件，没有专属测试；其余 `handlerbase` 是混合保留包。

不退休同名 SHEIN/Amazon handler、appenv.GetInstanceID、当前 Store Center、listingadmin Store API/DTO/仓库、TEMU APIClient/Cookie/MallID、平台 Product DTO、混合 context、SKU/runtime/builder、template/category/image/SPU 及有效共享规则。不新增缓存/IAM/Store 身份系统、Product factory、站点/认证/发货策略、provider 调用、runner 或验收工具。

不执行 main、constructor、API/provider/Preview/DryRun 或 schema 工具，不读取真实配置/凭据、DB/Redis/Profile/browser 或历史用户数据；不启停、部署、迁移、补写、重放、清理或删除实例/数据。#759/#692 等其它 owner 不接管。源码无装配消费者不等于运行态、历史请求或 durable assets 已清零。

## 2. Design Basis、Authority 与不变量

依据：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)：只有 EXTRACT / RETIRE；无 legacy wrapper、fallback、旧 ID 映射、双读双写或第二事实源。
- 已确认的 [数字执行链](legacy-numeric-task-execution-retirement-boundary.md)、[数字持久化](legacy-numeric-task-persistence-retirement-boundary.md)、[TEMU save/submit](legacy-temu-listing-submit-retirement-boundary.md)、[前置](legacy-temu-preflight-retirement-boundary.md)、[变体](legacy-temu-variant-retirement-boundary.md)及[筛选/AI 前置](legacy-temu-filter-mapping-retirement-boundary.md)：旧执行入口已退出，本轮只接续明确剩余的初始化阶段，不推翻其它保留项。
- [当前 Store Center](store-center-current-application-v1.md)、[观察同步](store-center-platform-observations-v1.md)、[准备接线](store-observations-preparation-runtime.md)、[独立 worker 身份/持久化](shein-worker-identity-persistence-boundary.md)：当前 Organization Store 与保留数字登录运行态不互相代替事实源；身份、权限、租户、同步/恢复 owner 保持原合同。
- [TEMU publishing](../../internal/marketplace/temu/publishing/README.md)、原 API/DTO、listingadmin Store 合同、共享 ptr、mixed context 和 SKU owner：有效行为保持原实现。Figma N/A，无 UI/IA 修改。

实施采用 **Reuse Existing Architecture**：用户明确确认 §5、§6、§6.1、§7 后，已在 #29/#301 记录冻结依据、用户结果、上述引用、原 owner/调用路径、不变量、验证、唯一 Writer 和 **Ready**，再删除这三个文件/专属 helper 并校准原登记。不新增生产函数/公共合同/入口、事实/schema、身份/权限/tenant、状态机/事务、幂等、retry/UNKNOWN/recovery 或外部副作用 owner；若实际需要改变这些边界，停止对应实现并补适用 Independent Architecture 准入。

Must：本文件保留项、原事实/规则/授权/凭据/恢复及 worker Store sync 不减，两套及其它 workflow/必需和安全检查/保护/ceiling 不改。Should：源码、原 guard、当前登记及直接冲突说明一致。Out of Scope 如 §1；无新增 Accepted Risk。Threat Model 限误删仍消费的 Store/API/context、将数字缓存当当前租户授权、将旧硬编码值抽成当前平台规则、混淆真实持久化/副作用 owner、复活旧入口及误报运行态清零。

发现新消费者或具体当前外部可观察兼容义务，停止对应删除并报告；旧测试、旧文档和同名符号本身不是当前契约依据，不自行增加兼容路径。

## 3. 实际消费者、注入与调用

删除前基线全仓 Go 搜索：三个 TEMU 类型/constructor 只有各自定义和 receiver 内部引用，无外部构造、直接实例化、注册接线，也没有 `task-processor/internal/temu/store` 的 Go import。`go list ./cmd/...` 共 **31 个当前命令**；依赖并集中的旧 `temu/store`、`temu/handlerbase`、`temu/context` 均为 **0**。下表及 §4.1 的退休对象行为是删除前取证，保留实现未变。未运行 main；这是源码/静态图证据，不是实例或产品验收。

| 合同 → 实现 | 注入与调用 | 决策 |
| --- | --- | --- |
| GetStore(int64) → StoreInfoHandler | constructor 接收匿名窄接口；从旧 Task.StoreID 查询，先读 package 全局 sync.Map，miss 调用 GetStore，再将 *StoreRespDTO 放回缓存和 TemuTaskContext.StoreInfo | 退休旧 Task 装配和数字缓存，不新建当前身份 adapter；保留 API/仓库/context 及其它消费者 |
| UpdateStoreId/UpdateStoreStatus 接口 → StoreIDHandler | constructor 保存 storeClient，但 HandleTemu **没有调用两个更新方法**；检查 Task/StoreInfo/APIClient/MallID，ID 不同只告警，返回 nil | “检查和保存”的名称不是持久化证据；退休告警外壳，不删除真实更新 API，也不把该告警升级成当前授权机制 |
| InitDataHandler → api/product.Product | 检查旧 Task/ProductID 后分配 payload、设置固定默认值/time.Now 时间戳、回填 TemuTaskContext.TemuProduct 和日志 | 退休旧 Task payload 预填，不新增 Product factory 或复制旧默认策略；原 DTO/字段及其它 builder 保留 |
| 缓存/统计/helper | storeCacheEntry、storeCache/storeCacheTTL/storeCacheStats、getStoreWithCache/setStoreCache/GetStoreCacheStats 仅在 info_handler.go 内部；GetInstanceID 仅在 id_handler.go 的日志中使用 os.Hostname | 随文件完整退休，没有跨包消费者或独立 durable owner |

同名 `internal/shein/store` 与 `internal/amazon/pipeline` handler 有各自消费者/测试；`internal/pkg/appenv.GetInstanceID` 也有独立消费者。不得全仓按 StoreInfoHandler、GetInstanceID 或 storeCache 子串禁止或删同名对象。

`TemuTaskContext.StoreInfo` 仍被 BuildSKUBuildRuntime/SKU/SPU 等消费；TemuProduct/SetTemuProduct 仍被 category、template、image、AI、规则和 SKU 直接消费。BaseTemuHandler/TemuHandler 等仍是包内共享合同，不能因三个 constructor 无消费者就顺带删除 mixed context、其它 builder 或整个 handlerbase。

## 4. 持久化、身份、平台副作用 owner 与有效行为

| 事实/行为 | 实际 owner 与边界 | 本批约束 |
| --- | --- | --- |
| 当前 Organization Store、版本/状态/审计 | internal/storecenter 及当前 authz/StoreQuotaLedger 合同；当前 HTTP/BFF/Console 读写其 owner database | 不以数字缓存/旧 StoreRespDTO 替代；不改当前表、权限、配额、状态/服务或同步 owner |
| 保留数字 Store | listingadmin StoreAPI → gormStoreAPI → GormStoreRepository / listing_store；原 Store DTO、owner/tenant 查询、字段更新和 worker Store adapter 均保留 | 不迁移/删除表、索引/数据，不把无新消费者误写成整个数字 Store 退休 |
| 旧 StoreAPI 的限制 | gormStoreAPI.GetStore 使用 context.Background 和 FindStoreByID；DTO/窄接口不是当前 Organization 请求身份、tenant/owner scope 或 deadline | 保留不等于授权/隔离验收 PASS；不增加当前私有数据消费者，不据本轮重建 IAM 或增加试用前置 |
| 保留旧 SKU/context 的身份限制 | BuildSKUBuildRuntime 在缺少 Task 身份时使用 tenant/store=1；上述31个当前命令不依赖该 context | 保留既存代码不认可此默认值作为当前身份，不新增消费者；本批不改变该行为。若发现具体当前消费/权限或租户 Must 后果，按停止条件报告并补适用准入，不静默复用默认身份 |
| 旧数字缓存 | storeID(int64) → *StoreRespDTO，5分钟TTL，全局内存 map，统计仅 hits/misses；命中不回原仓库 | 无独立当前授权/租户、撤销/版本刷新合同；单一数字 key **本身不证明当前跨租户漏洞**。该链无装配消费者，退休专属缓存，不修成通用缓存、不扫 Redis 或释放锁 |
| 真正的 MallID/Cookie/Store 更新 | 保留的 APIClient.NewAPIClient 会加载 Cookie 并 initializeMallID：GetStore 后可能 UpdateStoreId，或 SetMallID 更新 Cookie/客户端；仓库/CookieManager 的真实效果仍归原 owner | constructor 不是纯计算，设计期不调用。删除 StoreIDHandler 不删除这些路径/API/DTO、凭据/Profile/持久化，也不代表它们满足当前官方 Store 身份准入 |
| payload 初始化 | InitDataHandler 只写内存 TemuProduct，无 DB/platform 提交、事务/outbox/幂等/恢复记录 | 不新建持久化初始化事实，不访问 provider、结算历史或重放请求；时间戳/硬编码组合不成为当前平台默认合同 |

旧缓存与当前身份/owner 的边界不匹配，需要防止误复用；当前无证据说明它正在阻塞用户 happy path。Classification: **NOT_APPLICABLE（当前用户路径/试用阻塞）**；本设计按 RETIRE 退出无消费者闭包，不借此建立新 IAM/cache 或扩大 Threat Model。若刷新发现当前真实 Must 缺陷，先报告具体调用及后果，再决定最小范围。

### 4.1 初始化行为逐项处置

| 原行为 | 有效保留或退休依据 | 实施限制 |
| --- | --- | --- |
| Task 非空、ProductID/StoreID 非空和类型断言 | 旧 Task 门控退休；当前 Store service/请求准入及 Product 规则保持原 owner，不依赖这个 handler | 不把 Task/context 包装成当前输入，不增加 fallback/默认身份 |
| MallID 存在/一致性告警 | 原 APIClient 的 Cookie/MallID、Store API/DTO/仓库和原安全测试保留；旧 handler 的不一致仅告警，并不建立当前权限判断 | 不把该 logger wrapper 抽成当前认证/授权；不修复或扩展旧生命周期 |
| api.Product/ExtraInfo/GoodsBasicInfo/ServicePromise/ExtensionInfo/Skc、ptr.BoolPtr、空切片分配 | 原 DTO/字段、共享工具及其它 SKU/SPU/category/template/image 消费者保留 | 不因初始化入口退出而删共享模型/字段/Setter；不复制通用分配代码为新工厂 |
| 固定 en/美国站点100、flags/tab/GoodsType、认证模板1和属性1000000001→1000100066、time.Now 时间戳 | 未发现当前 Product/Architecture 合同要求继承这组旧 Task 预填；#749 已确认旧保存 request 固定值不作为当前规则合同 | RETIRE 此组合，不提升为 marketplace 默认规则；其它 builder 自有行为/API 同名字段保持，不借相同字面量扩大删除或修改 |
| ShipmentLimitSecond=2 与“24小时”注释 | sibling SPU 对相同数值注释为2天，category 使用86400/172800/259200；只是源码内部解释不一致，未在本轮验证外部平台单位/要求 | 不将任一旧字面值/注释当平台权威，不为无消费者路径新增修复；退休旧初始化赋值，保留平台 DTO、其它有效规则与原实现 |

本批没有需要新增 EXTRACT 生产代码的当前行为。仍有效的 Store/平台/工具合同已在原 owner 保留；若后续确认必须新增当前规则或公共初始化合同，应先报告相应新要求，不能在本退休 PR 临时发明。

## 5. 精确删除与保留矩阵

以下是**用户确认后已实施的精确矩阵**。

| 对象 | 决策 | 删除/保留条件 |
| --- | --- | --- |
| internal/temu/store/info_handler.go（152行） | RETIRE 全文件 | StoreInfoHandler/constructor、Task GetStore 装配、StoreInfo 回填、数字缓存/TTL/统计全部专属；不删 API/DTO/仓库/context |
| internal/temu/store/id_handler.go（106行） | RETIRE 全文件 | StoreIDHandler/constructor、未调用的 storeClient 字段/匿名接口、告警及专属 GetInstanceID；不删真正的 UpdateStoreId/UpdateStoreStatus 或共享 appenv helper |
| internal/temu/handlerbase/init_handler.go（114行） | RETIRE 全文件 | InitDataHandler/constructor、Task 验证和旧 payload 预填；无专属测试，不新增替代工厂/默认规则 |
| internal/temu/store 包路径 | RETIRE 完整专属包 | 仅上述两Go文件，无外部 import/装配；guard 要保证整个路径的Go源缺席并阻止当前命令重新依赖，不留下空壳或 placeholder |
| listingadmin StoreAPI/StoreRespDTO/StoreIdUpdateReqDTO/StoreStatusUpdateReqDTO、gormStoreAPI/仓库/模型及原查询/权限测试 | 保留原合同与持久化 owner | 仍有 APIClient/SHEIN worker 等消费者；真实表/索引/数据/owner/tenant/状态合同不动 |
| APIClient/StoreRuntime/manager、CookieManager、GetMallID/SetMallID/initializeMallID、API Product/嵌套 DTO/字段、shared ptr | 保留原实现 | 有当前保留的源码消费者；没有被三个待删 handler 独占，不改变副作用/凭据 owner |
| mixed TemuTaskContext.StoreInfo/TemuProduct/SetTemuProduct、SKUBuildRuntime/BuildSKUBuildRuntime、其它 handlerbase/sku/SPU/category/template/image/规则 | 保留全部原边界 | 仍有消费者，不把旧初始化出口删除误当字段/其它行为已无用；不改变本批之外的既存行为或新增当前消费者，保留不证明身份/授权正确（见 §4） |
| SHEIN/Amazon 同名 handler/helper、appenv.GetInstanceID 与此前全部保留能力 | 保留 | 不全仓字符串删除/禁止，独立 SHEIN 登录/Store sync、当前采集/Store观察照原 owner 保留 |

Legacy decision: **RETIRE 三个生产文件及其独占 helper/数字缓存**。
Reusable behavior: 有效 Store、平台 DTO/规则和共享工具已归原 owner，无新增 EXTRACT 生产代码。
Current owner: 当前 Store Center、listingadmin 数字 Store、TEMU APIClient/Cookie、原 SKU/平台共享合同、Product/观察/独立 worker 各保持。
Cutover/deletion condition: 用户确认冻结 §5/§6/§6.1/§7、Issue Reuse Existing Architecture / Ready；刷新无新消费者、原 guard 先 RED 后精确删除 GREEN、必要保留回归/登记校准及准确实现 HEAD CI/独立最终检查通过。源码退出不授权运行态或数据退出。

## 6. 验证与交付

设计阶段只提交本文与架构索引；设计结果不代表实施、实例/provider 或用户验收 PASS。用户确认后已扩展原退休 guard，在目标源码仍存在时确认 RED，再按 §5 精确删除后 GREEN；两条对应 import 豁免、原子树扫描与当前登记/说明按 §6.1 校准。准确实现 HEAD 的回归、CI 和独立最终检查证据维护于 PR，不人工重复完整 CI。

用户确认后的最小实施检查：

1. 刷新三个类型/constructor、store import/全部 helper 与 mixed context 消费者及当前命令图；发现新调用或当前兼容义务停止对应删除并报告。
2. 扩展原 TestListingKitTaskBackendSourcesAreRetired：整个 temu/store 路径和 init_handler.go 缺席；仅在 handlerbase 范围用原 AST 能力阻止 InitDataHandler/NewInitDataHandler 复活。保留原全部断言；不得全仓禁止通用 StoreInfoHandler/GetInstanceID/cache 名称。源码仍在时先 RED，再按 §5 精确删除 GREEN。
3. 在原 TestCurrentCommandsDoNotCompileLegacyBusinessBackends 增加退休 temu/store 依赖前缀，原命令集合/其它依赖和全部安全断言保留。补充保留 StoreAPI/DTO/仓库、APIClient/MallID/Cookie、Product DTO/shared ptr、mixed context/SKU、其它 handlerbase 和 SHEIN/appenv 同名对象的适用存在/准确合同断言。
4. 保留包适用编译与原回归：TEMU handlerbase/context/sku/product/API、现有 Store Center/数字 Store adapter/owner/tenant 测试、SHEIN Store/login worker 与既有 marketplace 规则；优先复用原 fixture/安全测试，不复制实现测试。store 包当前无专属测试，删除后不再用已退休包作 go test 目标。两个原命令 current-application/shein-login-worker 只构建，不执行。
5. 稳定实现候选走原必需 CI 和一次准确实现 HEAD 独立最终检查；设计证据不作实施批准。分别记录静态图/本地fixture/编译、PR CI、合并后匹配 main CI、真实产品/provider/实例验收；后者本批 NOT_RUN，不签发生产可用或 #29/#301 全量完成。

### 6.1 原 guard、登记、文档与 workflow 精确处置

| 工件 | 本次改动 | 保留边界 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 原 guard 增加专属包/文件/准确符号缺席及 §6 保留断言；原命令依赖检查增加 temu/store 前缀 | 不禁 handlerbase 整包；当前 worker/采集/平台/安全/并发检查继续保留，不新建通用工具 |
| tests/import_boundaries_test.go 的 TestTEMUProductStoreAndSchedulerRetiredManagementImportsStayBlocked | 精确删 id_handler.go/info_handler.go 两条对应豁免，保留 build_spu_handler.go/price_handler.go/spu_builder.go 三条。loadGoFileIndex 在根不存在时报错；改用稳定 temu 父目录索引，按**原 product/store 两棵子树**精确过滤，仍检查这两棵子树的全部Go源 | 不删整个 store 扫描覆盖，不因路径消失加 os.IsNotExist/skip 或放宽通用 helper；复活 store 仍受原 management deny 检查。其它扫描/roots/deny/ceiling 不动 |
| phase2-runtime-inventory.md 当前 consumer register | 包确实删除后只移除 core/logger → internal/temu/store 一行，添加本批校准说明 | handlerbase 的 core/logger 仍由 base_handler 消费，该行和其它 consumer 行保留；不改 dated 清单/closure ceilings |
| Legacy register、本文/架构索引、code-health-decisions.md 的旧 temu/store retention claim | 用户确认实施后校准 source-retired；只对旧 store 行明确已被本次确认边界取代，保留原 dated 报告/1 declaration 数量及其它决策 | 旧 configured processor 已退出，其历史保留表述不构成当前装配证据；不改全表或把其它 legacy 包顺带判为退休 |
| [external-client-boundary-inventory.md](external-client-boundary-inventory.md) 当前 TEMU 热点/guard 说明 | 用户确认实施后，只校准其将 temu/store 描述为当前 management 热点/现存包的表述：注明该包 source-retired，原 guard 改从稳定父目录索引、仍按 product/store 两子树检查；保留仍有效 product 边界及其它有效扫描说明 | 这是当前清单，不能只更新历史 ledger 而遗漏；明确 dated 的记录保留。全仓 sibling 刷新还命中旧退休文档的基线/当批保留矩阵和 checklist 中未改名的 guard，它们是历史范围/原 gate，不重写或据此恢复 store，不扩清理其它债务 |
| provider constructor/logger/LLM 登记、其它 import 豁免与 gate | 零改动 | 目标没有新建 provider constructor，不据 logger 调用退出删其它登记或改扫描范围 |
| ci.yml / commercial-readiness.yml / 其它 workflow、.golangci.yml、保护规则 | **零改动** | 原分类/触发/权限/并发、race/编译、安全检查/Required CI Gate、保护与ceiling保持；原SKIPPED不记PASS |

## 7. 确认与实施停止点

1. #760 先以 Draft PR 交付设计、准确设计 HEAD CI 和独立检查；#29/#301 保持 OPEN，原正文历史保留。旧设计证据不替代实施结果。
2. **用户已明确确认冻结 §5、§6、§6.1、§7**，在同一分支/PR 继续；生产修改前登记 Reuse Existing Architecture / Ready、冻结依据/确认、唯一 Writer 与 §2 不变量。设计确认授权上述有界实施，不代替准确最终 HEAD 的合并授权。
3. 获准实施时同步一次实际 main，最终候选前再次核对，只处理真实组合风险。新消费者/具体兼容义务，或合同/身份/权限/持久化/状态/副作用边界变化，停止相应项并报告，不自行扩删或补 wrapper。
4. 原 guard RED→GREEN、精确删除/登记校准、保留回归、准确实现 HEAD CI/最终检查后交付 In Review；不以反复审核未变代码扩大流程，也不把源码退出当真实运行态验收。
5. 合并仍需用户准确最终 HEAD 单独授权、正常 expected-head/保护，无管理员绕过。设计/实施都不授权 Issue 关闭、部署、真实 provider/实例/凭据/数据操作，不增加内部试用前置。
