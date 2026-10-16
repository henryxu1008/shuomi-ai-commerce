# 旧数字任务执行链：退休设计与持久化边界

后续校准：本文件记录 #729 当时的删除/保留边界。原保留的 ownership-controller
及专属 auto-shard 配置，随后已按 [#738 已确认范围](legacy-ownership-controller-retirement-boundary.md)
源码退休；旧表/运行态、有效共享合同和独立登录仍保留，历史矩阵不回写为当时已删。

其后 [#740 已确认范围](legacy-numeric-task-persistence-retirement-boundary.md)退役
无消费者的 ImportTask/事件/暂停与超时恢复/调度配置和专属源码/测试；原三项 race
随旧协议退出，两套 race 步骤运行完整保留 listingadmin 包。有效 Store/owner/mapping/规则、
共享配置和真实表/索引/数据/恢复 owner 不变，下方保留矩阵仍描述 #729 当时边界。

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

再后 [#742 已确认同步边界](legacy-numeric-product-inventory-sync-retirement-boundary.md)退出
当时保留的 SHEIN/TEMU 数字商品/库存同步服务、runtime adapter 及专属 InventoryRecord
源码；SHEIN 两种共享属性类型、平台 API/规则、其余有效仓库/合同、当前观察与独立登录的
Store 同步保持。下方 #729 矩阵仍是当时边界，不据此恢复已退出服务或操作真实状态。

- Status: **用户已确认 §5、§6、§6.1、§7，按 Reuse Existing Architecture / Ready 在 #729 实施源码退休**；准确实现候选、CI与独立检查以 PR 为准。
- 实施 Design Basis: **Reuse Existing Architecture**。用户确认的设计1f9c37d98先于正式源码修改；#29/#301已记录Ready。原设计阶段N/A及其文档CI不替代本轮实现准入或验证。
- 调查基线：`main @ db47d15e563c0ced7469ba12d251d8c7be1da4dd`，即 #726 的实际合并提交。
- 用户授权：先设计 `app/task`、`app/taskstatus`、`processor`、旧 Amazon 执行器及分布式爬虫专属接线，设计确认后再实施。保留当前采集、Store 观察同步、有效平台 API、规则、共享合同和独立 SHEIN 登录 worker；不操作运行实例或数据，不增加内部试用前置。

## 1. 用户结果、Authority 与非目标

维护者获得逐项可审查的源码退出范围，避免删除旧数字任务执行器时连带损坏采集、平台规则或独立登录。当前内部试用继续沿原入口使用；这批源码债务不是新的试用阻塞，也不增加新的产品能力或上线承诺。

适用依据为 [全新系统基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[前批控制面退休](legacy-listing-control-plane-retirement-boundary.md) §3–§6、[独立 SHEIN worker 边界](shein-worker-identity-persistence-boundary.md)、[Store 当前应用](store-center-current-application-v1.md) 和 [观察准备接线](store-observations-preparation-runtime.md)。这些依据保留原 Product/Organization/Store/Listing/Submission、采集和 worker owner；不因本设计恢复历史迁移、ID 映射或旧任务续跑。无 UI/IA 改动，Figma N/A。

Must：保留上述当前能力及其安全合同；无 fallback、双读双写、compatibility facade、第二事实源或新的旧链消费者；源码删除不隐式转移在途任务或恢复责任。Should：当前文档、消费者表与检查能准确说明已删和未删范围。Out of Scope：真实运行态探测/处置、补偿或排空、当前产品扩张、平台接入、通用恢复/验证平台，以及共享目录的全面清扫。不扩大 Threat Model，不新增 Accepted Risk，也不将未知旧环境状态写成安全或已验收。

## 2. Design Basis、准入与停止条件

设计1f9c37d98的文档阶段为N/A，现已获用户确认。实施前#29/#301正文已记录确认版本、scope、owner、主要路径、不变量、验证与 **Reuse Existing Architecture / Ready**，同步main db47d15e5；不以文档CI替代实现验证。

本批采用 **Reuse Existing Architecture**：按现有 Hard-Cut 与已确认的当前 owner 合同删除无命令入口的旧执行闭包；保留功能的事实 owner、权限/租户/身份、状态机、schema、事务、retry/UNKNOWN/recovery 和外部副作用保持。预计跨多个目录且超过 30 文件，必须在实施 Issue 说明批准矩阵如何完整覆盖提示线，不为满足文件数机械拆 PR。

若刷新后发现当前运行入口/具体外部可观察合同，或者必须改变保留合同、身份/租户、事实 owner、状态机、持久化、事务或恢复协议，**停止对应正式实现并报告用户**。需要新边界时转 Independent Architecture、适用独立评审与 `IMPLEMENTATION_READY`；不得用删除名义发明 adapter、替代执行器、排空器、补偿或新的兼容例外。历史镜像或运行实例是否仍执行旧代码，不能由本次 Go 图判断。

## 3. 实际 contract → implementation → injection → consumer

基线只读执行 `go list -deps -json ./cmd/...`，解析所有 **31 个命令根** 的实际非测试导入图，并对全仓生产源码核对直接 import 与构造/方法调用；没有执行任何 `main`。

| 包/合同 | 实际编译或源码消费者 | 设计结论 |
| --- | --- | --- |
| `app/task` / `app/taskstatus` / `processor` / 根 `amazon` | 31 个命令消费者均为零；`processor`、`app/crawler/distributed` 仍导入 `app/task`；`app/task`、`processor`、根 Amazon 仍导入 `app/taskstatus` | 不能只删一个包；按下列闭包退出，不搬到当前 app/worker |
| `app/task` 执行 | `NewUnifiedTaskFetcher` 的生产构造仅余定义；待处理任务端口 → claim/status → journal → guard → `TaskSubmitter`/workerpool | 旧数字 Task 生命周期，不是当前 Product/Submission owner |
| 根 `amazon` | `processor.go` 组合 BaseProcessor、SP-API、LLM 与 pipeline；`task_status.go` 回写数字任务状态、旧店铺暂停与日计数。全仓无生产 import 根包或 `NewProcessor` 调用 | 只退根包两个生产文件及专属测试；不删 Amazon 子包 |
| `app/crawler/distributed` | 唯一包外生产 import 为 `crawler/fetcher/distributed_fetcher.go`；client 使用 `app/task` 的 MessageAdapter、路由和 payload | 同时退 client、pending registry、result listener 与专属 adapter/测试，不抽取旧消息协议保活 |
| `crawler/fetcher` | 无命令根，但 SHEIN/TEMU 商品、库存与 pricing 等源码消费 ProductFetcher 接口、读写缓存；不能整包删 | 精确退出 distributed 实现/分支和下述未使用的专属选择/装配，保留 local/remote/shared 部分 |
| distributed 选择/装配 | `contracts.go` 的具体 builder、config/recommended selector 仅定义和自身测试使用；`platformbase/fetcher_mode.go` 的两函数无其它生产调用，但包含 local/remote/auto 选择合同；`temu/sync/inventory_sync_factory.go` 硬编码 DistributedFetcher，其工厂/创建方法无其它生产调用 | 退出专属自动装配；混合 mode helper 仅删 distributed 分支，保留其它选择行为。不默默改为 local/remote 或新 worker |
| `listingadmin` / 根 `listingruntime` | 前者由 current-application、issue36-local-trial-init、shein-login-worker 编译；后者由前两个命令编译 | 整包、原 ImportTask/dispatch/recovery 仓库与有效安全合同保留 |
| Amazon API / 当前能力 | `amazon/api` 由 product-listing-api-schema-migrate 编译；其它 API/规则/模型/pipeline/sourceproduct/source integration 仍有源码消费者，纵使某包无命令根也不能据此删除 | 保留有效平台 API、规则、共享合同及其原测试；不把 schema 工具当发布或爬虫执行器 |

准确旧边为：`crawler/fetcher → app/crawler/distributed → app/task → app/taskstatus`，以及根 `amazon → processor → app/task(/taskstatus)`。当前应用、采集命令、Store 观察、图片 worker 和独立 SHEIN 登录 worker 均不需要这些执行包。

## 4. 持久化 owner、幂等与失败边界

| 边界/事实 | 源码中的 owner 与现状 | 源码退休后的责任 |
| --- | --- | --- |
| 旧数字 ImportTask 与状态 | `app/task` 通过注入的 pending/status 端口读写；`taskstatus.Service` 发送 `listingruntime.TaskStatusUpdate`，transition 设置 ExpectedCurrentStatus。当前已无这些端口的生产装配；保留的 listingadmin repo 是既有表/状态方法 owner | 不将 taskstatus 视为 DB owner，不把旧状态映射到当前任务/Submission。repo、表、列、索引、行与原安全保护保持，不运行 migration/drop/recovery |
| claim journal | `app/task/claim_journal.go`；默认 `filepath.Join(os.TempDir(), "task-processor-claim-journal.json")`，按 task_id 记录，进程内 mutex，JSON 直接 `os.WriteFile` | 删除源码不读、删、改名、搬运或重放任何实际 journal；不把文件自动导入当前系统，不为旧进程指定新恢复 owner |
| claim / startup recovery | 本地 processing 标记 → 远端 expected-status 更新 → journal 记录 → guard/本地 submit。journal 失败尝试回滚原状态；启动以 expected=processing 回收至 pending_retry，失败保留条目 | 原恢复代码随 owner 退休；不转给当前 application、worker、Temporal/Cron 或新进程，不宣称旧在途任务已恢复 |
| 内存队列/去重/cleanup | TaskFetcher、DeduplicationManager、workerpool submitter 及 CleanupService；cleanup 查询状态或按条件尝试恢复并移除本地标记 | 删除专属编排；通用 workerpool/lifecycle、state/计数/锁和仍有消费者的合同不整包删；不停止任何实际 worker |
| RabbitMQ 分布式请求 | distributed client 先校验 route，再注册 pending、发布 TaskEventV2/payload。专属 adapter 以原始 body、DeliveryMode=2、mandatory=false 发布，没有在此建立 publisher-confirm/DB 事务或 broker 去重 | 不 publish/declare/purge/delete/ack/replay 实际消息，不把旧消息交给当前采集；保留通用 RabbitMQ 与仍消费的 TaskEventV2/域合同 |
| 结果队列与 pending | 名称 `crawler.results.node-<UnixNano>`；专属 adapter 实际 durable=false、autoDelete=true、exclusive=false，不能因方法名 DeclareExclusiveQueue 就称其独占。pending/task/result channel 仅进程内，超时取消不撤回已发布请求；坏 JSON Nack(false,false)，结果投递后 Ack | listener/registry 随专属链退休；不自动创建替代结果队列、持久化结果表或重放协议；既有 broker 生命周期不由源码设计证明 |
| 旧 Amazon 执行副作用 | 根 Processor 装配原 SP-API、OpenAI LLM/pipeline、state.MemoryManager 的旧日计数/配额、Store pause 和数字 task 状态；不是一个统一提交 | 删除该无装配执行根，不调用 provider、rollback quota、暂停店铺或修复旧状态；平台子包有效行为及原 owner 保留 |
| 当前业务与独立登录 | 当前 Organization/Product/Store/observations/Listing/Submission 归当前 owner；独立 SHEIN 原 DB、Redis Cookie/attempt/验证码、Profile、锁、tenant-qualified Store 查重与 health/shutdown 归原 worker | 完全保留原身份、授权、持久化与恢复责任；不接收旧数字任务、旧消息或 journal，不增加登录/试用前置 |

### 4.1 现有架构债务的准确解释

远端 expected-status 更新、文件写入、内存队列、broker 与外部平台没有共享事务。状态 CAS 不等于一次完整执行幂等；相同 task_id 或 persistent delivery mode 不证明 broker 去重、发布确认或 exactly-once。响应丢失可能导致状态重试冲突，`taskstatus` 的异步更新/重试使用自身 background context，不代表调用方 deadline 覆盖全部副作用。

journal 的进程内锁不隔离同机不同进程；默认路径共享，写入也没有临时文件原子替换或 fsync 协议。因此它不能作为当前系统的可靠恢复设计模板。批量请求若部分已发布，后项失败后的本地 pending 清理不会撤回前项；重启不能从内存 pending 重建结果。这些是源码可见的旧设计债务，**不是新增已接受风险或本轮修复授权**。采取 RETIRE，不为其建设 Saga/outbox/fencing/新的 UNKNOWN owner，也不声称已处置真实旧运行态。

## 5. 用户已确认的删除矩阵与保留边界

| 源码/工件 | 已确认处置 | 条件与限制 |
| --- | --- | --- |
| `internal/app/task/*.go`（24 文件，含 7 测试） | RETIRE | 精确退出取任务、claim/journal、guard/dispatch/monitor/cleanup、submitter adapter、message/payload/route 与专属去重实现；先消除矩阵内所有包外生产消费者，不换目录保活 |
| `internal/app/taskstatus/*.go`（2 文件） | RETIRE | 消费者仅本批 task/processor/根 Amazon；连同专属 service 测试退出，不删根 listingruntime 的状态 DTO/其它消费者 |
| `internal/processor/*.go`（6 文件） | RETIRE | BaseProcessor、BaseTaskHandler、CrawlerProcessor、VariantTaskSubmitter 及专属测试，无命令根/其它生产消费者；保留通用 workerpool、sourceproduct 与原 crawler/provider |
| `internal/amazon/processor.go`、`task_status.go`、`task_status_test.go` | RETIRE | 根包无其它源码消费者；不递归删除目录。`api/attribute/image/listing/llm/model/pipeline/schema` 等子包全部保持 |
| `internal/app/crawler/distributed/*.go`（10 文件） | RETIRE | client、listener、pending registry、interfaces/types、RabbitMQ adapter 和专属测试全部退出；其唯一 fetcher 消费者同步退出 |
| `internal/crawler/fetcher/distributed_fetcher.go` | RETIRE | 删除专属分布式实现与内部 helpers；不迁移该类或旧消息 payload 到当前采集 |
| `internal/crawler/fetcher/contracts.go` | 精确 RETIRE 专属片段 | 删除 DistributedFetcher 常量、distributed factory case/构造函数；删除无外部生产调用的具体 ProductFetcherBuilder/私有 resolver、CreateFetcherFromConfig、GetRecommendedFetcher。保留 ProductReader/ProductCache/ProductFetcher/Stats、LocalFetcher/RemoteAPIFetcher、factory 的 local/remote 路径、source adapter 与原公开参数/返回合同。对字符串 `distributed` 沿现有 unsupported-type 默认拒绝，不新增 stub、fallback、自动重选或 provider 调用 |
| `internal/temu/sync/inventory_sync_factory.go` | RETIRE | 仅退出无外部生产调用且硬编码旧分布式执行器的工厂；其原 `inventoryServiceFactoryRuntime` 方法集仍由 ServiceRuntime 嵌入，声明原样保留到同包已有 runtime_adapter.go。不改 InventorySyncService、商品/库存 repository/API、定时/规则或当前 Store observations，不重写为 local/remote 工厂 |
| `internal/platformbase/fetcher_mode.go` | 精确 RETIRE distributed 分支 | 只删除 `case "distributed"`；该输入沿原 unsupported-mode 默认错误拒绝。两 helper 的 public 签名与 local/remote/auto 行为、原有效测试、`fetcher_builder.go` 共享接口/func adapter、errors 与其它 platformbase 保持，不整文件/整包清理 |
| listingadmin ImportTask/paused/dispatch 仓库及相关 DTO、root listingruntime、model.Task/status、state 与通用 queue/taskdomain/config | **本批保留** | 仍有有效源码/测试消费者或独立安全合同；不按名称、数字字段或命令不可达推定全部退休，不删除原表/索引/owner/tenant/唯一性/CAS/claim-token/eligible-row guard |
| `crawler/amazon`、integration/crawler/amazon、marketplace/sourceproduct、有效 local/remote fetcher、平台 API/规则/模型/pipeline | **保留** | 新旧 Amazon 包不是同一 owner；本设计不改变 source request、CrawlSource、缓存、平台校验、SP-API 或确定性行为 |
| 现行命令、worker、workflow、Makefile、部署及私密配置 | **保留** | 31 命令无本链依赖，无需删除或更换编译目标。历史 ownership-controller workload/镜像保留；未消费 Config 字段/旧资源债务不顺带清零 |

本矩阵涉及超过 30 文件及两个混合合同文件中的精确片段，另需校准直接相关检查/文档。实际文件数以确认后刷新为准；范围不以数量为目标。若发现本表之外消费者，不以“让测试过”扩大删除。

## 6. 检查、文档与安全断言的处置

本批只在既有测试/guard中补足本闭包，不开发新runner、治理检查器、验收session或验证平台。原设计阶段未改生产/测试/CI；用户确认后按本表实施，workflow与部署保持不变。

| 位置 | 确认后最小处置 | 必须保持 |
| --- | --- | --- |
| `tests/listingkit_backend_retirement_test.go` | 扩展原 absence/真实依赖检查：本矩阵完整文件/包退出，当前命令不重新编译旧链；先 RED 后删，保留当前 roots 的正向存在检查 | 原已退休入口、账号桥/源账号/旧 CLI 不复活，独立 worker 不编译当前 HTTP；不把 consumer 数量零等同运行验收 |
| `tests/import_boundaries_test.go` | 按实际消费者区分 owner 专属实现断言与仍有效安全断言。读取已删 task/status/processor 文件的专属 capability 命名/字段/旧 facade 测试退休或收敛到原 absence guard；混合文件清单只移除对应项 | 留存 root DTO/兼容禁用、全构建目标 import/租户/权限/平台断言；不整体删混合 guard，不新增“文件不存在就忽略”通用逃生口 |
| `tests/integration_provider_logger_contract_test.go` | 从既有直接构造调用者精确基线中只移除 `internal/amazon/processor.go` 的 OpenAI 调用计数 | 其它全部 provider 构造、日志/凭据保护与当前 owner 断言不变 |
| `internal/crawler/fetcher/{contracts,builder,amazon_source_adapter,remote_fetcher}_test.go` | distributed/旧 auto-selection 专属断言随实现退休；保留 local/remote、CrawlSource 注入、取消/deadline、nil 拒绝、缓存隔离及平台规则用例，必要时只改为直接调用保留 factory | 不把原安全失败关闭断言改为成功，不在保留路径增加 fallback；沿原 unsupported 默认分支验证退休 mode 被拒绝 |
| `internal/marketplace/sourceproduct/source_request_boundary_test.go` | `TestFetchersUseVariantFetchRequestAdapter` 是混合源文件表；未来只移除已删除 `distributed_fetcher.go` 的一项，不删除整个测试 | 完整保留对 `product_fetcher.go`、`remote_fetcher.go` 使用 `VariantFetchRequest` 及禁止手工拼接 variant request 的断言；同根因源码文件表盘点覆盖 internal/tests，不以读已删文件失败为由跳过其它有效检查 |
| `internal/platformbase/fetcher_builder_test.go`、本矩阵专属测试 | 保留原 local/remote/auto 选择与 invalid-mode 拒绝；沿现有 invalid-mode 用例覆盖 distributed 被拒绝。其它专属测试逐项确认是否仅验证已退 owner，再退出对应项 | 不按文件名批量删混合测试；共享接口/func adapter/其它平台与授权断言保持，不借退休撤掉仍存生产保护 |
| `.golangci.yml` / 原 depguard 文本守卫 | `app_taskstatus_legacy_management` 的专属 files/deny 随该 owner 一并退出；若原 guard要求此已消失 owner，精确同步该项 | 其它全部 deny、file selection、frozen edge/closure 上限及当前保留路径保护不放宽，不添加豁免；实际允许边只能减少 |
| `architecture-review-checklist.md`、`app-assembly-boundaries.md`、`project-boundaries.md` 的现行测试清单，以及 `external-client-boundary-inventory.md` 与 `tests/architecture_docs_test.go` | 对确实退休的测试名/直接构造消费者同步删除当前清单项，保持当前 guard coverage 与实际测试对应；只校准对应 owner，不清空整个清单或跳过覆盖检查 | 新的/扩展的原退休 absence guard 继续证明源码不复活；历史测试/扫描快照保留，既有当前安全守卫的覆盖要求不取消 |
| 当前消费者/模块地图 | `phase2-runtime-inventory.md` 的机器校验表只删除实际已消失 consumer；`module-target-mapping.md`、`product-sourcing-inventory.md`、repository-structure/直接运行说明按真实变化校准 | 有效保留包/命令与 owner 不改，日期化历史扫描/决策/测试证据保留，不将历史 failure/cancel/未执行验收改为 PASS |

### 6.1 CI、构建与验证范围

两套 workflow `.github/workflows/ci.yml`、`commercial-readiness.yml`、Makefile、触发/权限/并发/保护规则 **本批无需改动**。它们在 #726 已退出旧控制面目标；原 `listingadmin` 三项 claim/rollback/recovery race、PG 权限/public-facts/schema-context、资金、当前 application/login-worker build、前端、Required CI Gate、pinned SHA/pipefail/artifact 等保持。

原设计阶段：文档链接、diff 和既有相关文档 guard；实际依赖图及源码消费者只读调查。无形式 RED，不新增文档测试框架；文档 CI 不当未来源码实施结果。

确认后的实施：原退休 guard RED→GREEN；保留 fetcher/source adapter/platform API/worker 的相关原测试与命令编译/实际 Go 图。稳定候选一次必需 CI、一次集中最终独立检查；对未变化代码明确复用已有证据，只验证新增组合风险。若删除链需要改留存 owner 的边界，先执行 §2 停止规则。真实 browser/provider、队列、journal 恢复、实例/用户/平台/生产验收保持 NOT_RUN。

## 7. 确认内容、实施顺序与交接

用户已确认§5完整闭包、§6/§6.1精确检查处置和本节权限边界，并明确授权在原#729实施。已刷新main、活跃Writer/PR与实际源码/符号消费者，在#29/#301先于正式修改记录批准版本与Reuse Existing Architecture/Ready，由同一Writer在原主要PR连续实施必要修复。

实施顺序为：在既有 guard 记录目标 RED；同批移除 distributed 专属装配/fetcher/client、根 Amazon/processor、task/status 的闭包，保留 factory local/remote/共享合同；只校准随 owner 消失的专属测试/依赖基线和直接冲突文档；完成 GREEN、保留消费者必要回归、必需 CI 与独立最终检查。顺序是提交/验证边界，不是机械拆多个 PR 或新建平台。

在源码交接中明确：旧执行链不再由当前源码构造，保留能力沿原命令与私密配置使用；旧 journal/DB/Redis/消息/配额/镜像/卷/Profile/备份/日志仍归原运行态 owner，本批未检视或处置，不创建新恢复责任方。若用户要求真实旧执行能力或在途任务处理，另定明确当前需求和操作授权，不能把未处理运行态写成已清零。

实现完成转 In Review，用户/其指定验证者确认实际使用效果；合并需准确最终 HEAD 单独授权并核对实际 merge 的 push/main CI。#29/#301 保持 OPEN；设计或 CI 均不签发产品验收。不得运行业务命令、真实数据 dry-run、DB/Redis/queue、Docker/Kubernetes、browser/provider 或 schema initializer；不停止/重启/销毁实例，不读取私密配置/凭据/Profile，不删除真实文件或数据。内部试用不增加前置，生产 NOT_READY。

```text
Legacy decision: RETIRE
Reusable behavior: 有效平台API/规则、ProductFetcher读写缓存合同与local/remote行为、当前采集/观察及独立登录保持原owner，不抽取旧任务协议保活。
Current owner: 当前各Domain/Capability/persistence与独立worker；保留旧仓库和实际运行态各归原owner，无新owner。
Cutover/deletion condition: 用户已确认设计1f9c37d98且实施准入满足，刷新并删除本矩阵零外部消费者闭包；有效共享/安全合同保留，真实资源/数据处置另行授权。
```
