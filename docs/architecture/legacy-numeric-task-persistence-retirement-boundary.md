# 旧数字任务持久化与恢复残留退休设计

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

- Status: **用户已确认冻结设计 49833f924 的 §5、§6、§6.1、§7，按 Reuse Existing Architecture / Ready 在 #740 实施源码退休**。准确实现候选、检查与复核结果以 PR 为准；没有操作真实运行态或数据。
- 调查基线：`main @ 68b4e99a4cb6afd95deea7ed95b9dad6b4e65edd`（#738 已合并）。31 个现行命令根均已用实际 Go import 图核对；源码检索区分生产、专属测试和同包编译，不运行任何命令 main 或 schema 工具。
- 实施 **Design Basis: Reuse Existing Architecture**。用户确认设计及消费者刷新先于正式源码修改，#29/#301 已记录 Ready。原文档阶段 N/A、设计 CI/复核仅为历史，不替代当前实现准入或准确候选验证。

## 1. 用户结果与范围

使用者是当前内部试用维护者。旧刊登 CLI、control-plane 和数字任务执行器已退休，但共享包仍携带旧 ImportTask 写库、派发事件、暂停/超时恢复、调度配置、任务统计及闲置 schema 装配。维护者需要一份可执行的源码删除矩阵，避免旧任务 owner 被误接回当前应用，也避免把有效 Store/平台/独立登录合同随旧实现误删。这项设计不阻塞现有试用、不宣称新增产品能力。

拟 RETIRE 的只是无当前执行消费者的旧任务闭包；当前采集、Store 观察同步、有效平台 API/规则、共享合同、必需检查和独立 SHEIN 登录 worker 均保留。真实表/行/索引、DB/Redis/broker/锁、配置/凭据/Profile、实例/镜像/卷/备份/历史不在删除范围内。源码退出不转移在途任务、历史恢复责任或数据库 ownership。

## 2. Design Basis、Authority 与不变量

依据：[全新系统基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[旧 control-plane](legacy-listing-control-plane-retirement-boundary.md)、[旧数字执行链](legacy-numeric-task-execution-retirement-boundary.md)、[ownership/controller 配置退休](legacy-ownership-controller-retirement-boundary.md)、[独立 SHEIN worker 边界](shein-worker-identity-persistence-boundary.md)、[当前 Store](store-center-current-application-v1.md) 和 [Store observations](store-observations-preparation-runtime.md)。本设计只替代前批对本闭包的“保留待后续”决定；不重写当时证据或扩大其它范围。无 UI/IA 变化，Figma N/A。

本次 Reuse 路径复用上述已确认的 owner 与 Hard-Cut，不新增表、事实 owner、公共当前领域合同、权限/身份/租户政策、状态机、事务、幂等/retry/UNKNOWN/recovery 或平台副作用。不因文件数或删除量重新设计全局架构：批准矩阵完整覆盖本批提示线；若发现未覆盖的当前消费者或必须改变保留边界，停止对应删除、报告具体证据并按需升级 Independent Architecture，不能临时发明兼容路径。

Must：有效 Store/owner/规则/worker 行为与原安全断言保持；无 fallback、空壳、双读双写、第二事实源、新调度/恢复 owner 或旧任务向新身份映射；源码删除不触及真实状态；三项旧协议检查的处置必须和源码同批生效，保留有效检查与失败阻断。Should：现行文档和 guard 如实区分旧任务表、保留字段与当前观察事实。Out of Scope：实际旧环境探测、排空/补偿/重放/迁移、当前业务扩张、共享包全面清理、通用恢复或验证平台。无新增 Accepted Risk，不扩大 Threat Model，不把未知运行态写成已安全退休。

## 3. 实际消费者：合同 → 实现 → 注入 → 入口

| 合同/源码 | 实际生产消费者与注入证据 | 判断 |
| --- | --- | --- |
| `listingadmin.ImportTaskRepository` → `GormImportTaskRepository`；dispatch event repository；`PausedTaskRecoveryService`；scheduled task repository；Store task statistics repository | 全仓生产源码检索只剩本闭包定义/内部互调；构造器、恢复 service、aggregate prerequisite 在包外无生产注入，原调用方已随 #723/#726/#729 退出。专属测试仍构造它们 | 候选 RETIRE；测试通过或同包被编译不构成当前产品消费者 |
| `listingadmin.AutoMigrateRuntimePrerequisites` | 仅剩定义，无生产或测试调用；仍枚举旧任务模型及其它保留模型 | RETIRE 闲置聚合函数/文件，不删除其它模型或给新启动路径增加迁移 |
| `listingruntime.ImportTask`、`ScheduledTaskConfig`、`TaskStatusUpdate` | 仅剩类型声明，无生产引用；同文件仍有有效 Store/mapping 合同 | 只 RETIRE 三个类型；不整删 `types.go` |
| `listingadmin` / `listingruntime` 两包 | `current-application`、`issue36-local-trial-init` 均在真实传递图中依赖两包；`shein-login-worker` 依赖 listingadmin；其余 28 个命令不依赖这两包 | 包级依赖不是目标类型被使用的证明；两包均必须保留，完整 31 根不执行 main |
| 独立 SHEIN 登录 | runtime 构造 `NewGormStoreRepository` + `NewGormStoreAPI`，注入原 bootstrap/account provider、Store sync 与 tenant-qualified duplicate lookup；未构造 ImportTask/统计/调度仓库 | 保留原 DB/账户、Redis/session/attempt/Cookie/Profile、验证码、健康/关闭与同步，不移交到当前 Organization/官方流程 |
| 当前规则与平台共享能力 | `marketplace/temu/publishing/pricing_policy.go` 消费 PricingRuleRespDTO；`publishing/shein` 消费 sensitive/generation policy/override 仓库；SHEIN/TEMU 仍消费 mapping、product data 与 root runtime Store/规则合同 | 保留各合同、API、规则、仓库、装配与平台行为，不能整体清空 listingadmin/listingruntime |
| 名为 `import_task_owner.go` 的 Store owner helper | `product_data_repository.go` 与 `product_import_mapping_repository.go` 仍调用 `resolveStoreOwnerUserIDForUpdate`，在原事务内以 tenant/store 查 owner 并对 PostgreSQL Store 行加锁 | 文件与有效测试全部保留；不能按文件名 RETIRE，亦不为改名机械搬迁 |
| `ImportTaskID/ImportTaskId` 字段及同名配置 | mapping/product data、SHEIN/TEMU 同步/发布仍消费这些字段；`core/config.ScheduledTaskConfig` 仍被共享平台配置 loader/registry 使用，是不同类型 | 不等于 ImportTask 仓库当前有调用方，也不授权删这些字段、列、mapping、Store pause 合同或共享配置 |

架构问题是闲置任务持久化/恢复代码与有效 Store、owner、规则共处一个 Go 包，使编译依赖和测试误看起来像业务保留义务。解决方式是按符号闭包精确 RETIRE，不重建 control-plane、不包装旧仓库、不把旧任务计数改成当前观察统计。

## 4. 持久化、状态与失败恢复 owner

| 边界 | 原事实与协议 | 源码退休后的边界 |
| --- | --- | --- |
| `listing_product_import_task` | 原 GORM ImportTask repository 拥有 CRUD、target-platform 规范化/active-only 唯一索引、数字 tenant/store/owner、状态与 processing_node；claim 以 id/expected status/deleted 条件写 queued；rollback 还校验 queued/node | 删除源码合同、writer、迁移 helper；不 DROP/ALTER/TRUNCATE/UPDATE 真实对象，不处理历史 duplicate，不转交事实或恢复责任 |
| processing/queued/paused 恢复 | 旧 repository 在 UPDATE 时再次限定状态、截止时间或 tenant/store/platform/reason/stage；processing→pending_retry、queued/匹配 paused→pending。暂停计划还读 Store 和 Redis runtime pause；各步不是跨 DB/Redis/broker 的共同事务 | 不迁移计划/回执、重放、修租约或增加自动接管；原并发测试的作用止于这些旧协议，不能据此宣称当前系统 exactly-once |
| `listing_dispatch_event` | 模型在 import_task.go，writer 在 ImportTask repository，查询在 dispatch_event_repository；不是当前 Submission receipt 或资源/账务流水 | 连同专属源码退休；真实事件/日志/索引/历史归原运行 owner，保留原始事实 |
| `listing_scheduled_task_config` | 原数字 tenant/store/platform/taskType scope 的配置仓库及唯一索引；没有当前 scheduler 注入 | 只删专属模型/读写/查询/配置逻辑；真实配置不迁移到 core/config 或当前 Temporal |
| Store task statistics | 无注入的仓库 join `listing_store` 与旧任务表计数，非当前 Store observations | 退休统计实现和专属 DTO/测试；不伪造 0、不复活旧接口、不改当前 Cockpit/观察统计 |
| 保留 Store/product/mapping/规则及 worker | 原 Store owner、scope/auth/audit/事务 helper、当前 Domain/Capability 和独立 worker 各自持有事实；当前 observations 的独立 schema/Temporal owner 不变 | 不删共享 owner helper/约束，不因字段含 ImportTaskID 而改列或语义；provider 失败/迟响应、取消/重启、Cookie 提交与 Store 同步失败的既有行为不变 |

不存在本轮新幂等键、新状态转换、跨库一致性协议或恢复触发。响应丢失、重试、UNKNOWN、并发和历史任务如何处置仍由原运行 owner 决定；本设计不访问真实环境，不把源码删除当运行态安全证明，也不添加试用验收要求。

## 5. 用户已确认的删除矩阵与明确保留

以下以 `internal/listingadmin/` 为前缀；实施前已刷新与 §3 一致。专属源码已按矩阵退出；真实表、索引、行及运行 owner 不变。

| 文件/符号 | 决定 | 删除条件与保留 |
| --- | --- | --- |
| `import_task.go`、`import_task_model.go`、`import_task_query.go`、`import_task_repository.go` | RETIRE 完整四文件 | 包含旧 DTO/errors/model/dispatch/recovery/writer；不移动到新包保活，不删共享 domain/task/model 的状态/规范化能力 |
| `dispatch_event_repository.go`、`paused_task_recovery.go` | RETIRE 完整两文件 | 专属读模型/计划/执行/Redis reader 接口退出，不移给 worker/app 启动路径 |
| `scheduled_task_config.go`、`scheduled_task_config_model.go`、`scheduled_task_config_query.go`、`scheduled_task_config_repository.go` | RETIRE 完整四文件 | 只涉及旧数字任务配置；`core/config.ScheduledTaskConfig` 不在删除范围 |
| `store_statistics.go`、`runtime_prerequisites.go` | RETIRE 完整两文件 | 前者只有旧任务统计，后者是无消费者的聚合 AutoMigrate；其它 Store/规则/模型与正常初始化保留，不增加自动 schema 操作 |
| `schema_migrate.go` | 精确 RETIRE `ensureNullableImportTaskCategoryID`，以及 `ensureImportTaskActiveUniqueIndex` 起至文件结尾的 ImportTask 专属函数段 | 保留前部通用 owner/audit、text、sensitive-word 类型处理、column/type 解析、`ensureUniqueIndex`；仅去掉因目标删除而闲置的 import，不能顺手删共享 migration helper |
| `internal/listingruntime/types.go` | 只 RETIRE ImportTask / ScheduledTaskConfig / TaskStatusUpdate 三类型 | StoreInfo、StorePauseStatusDetail、StoreService、ProductImportMapping/Upsert（含 ImportTaskID）完整保留；operation_strategy/rules 不改 |
| `import_task_owner.go` 与 `import_task_owner_test.go` | 保留全部实现及三个现有 owner/跨 tenant 测试 | 名字不决定所有权；Store 锁、可信身份、owner 缺失/跨 tenant 拒绝和错误传播保持原样 |
| mapping/product data/raw JSON、management DTO、Store 仓库/API、owner scope/约束、敏感词/价格/主题规则、共享 paging/numeric helper | 保留，不能整包扫除或按 ImportTaskID 子串删除 | 仍有生产消费者；新业务不因此获得复用旧数字 owner 的许可 |
| 现行地图、runbook、legacy-register、前批设计中的后续留债说明 | 校准当前源码状态与本方案链接，旧证据保留 | 不改历史统计为本轮 PASS；实际部署 README 中旧任务表的运维语句须标明历史/不可据此执行，不新提供 DROP/清理脚本 |

## 6. 测试与原三项并发检查处置

| 对象 | 用户已确认的处置 | 保留的约束 |
| --- | --- | --- |
| `import_task_dispatch_test.go`、`import_task_model_test.go`、`import_task_query_test.go`、`dispatch_event_repository_test.go`、`paused_task_recovery_test.go`、`scheduled_task_config_repository_test.go`、`store_statistics_test.go` | RETIRE 七个专属测试文件及只服务它们的 fixture | 原 runConcurrently/seedDispatchTasks 等仅在这批专属测试中使用；实施刷新如发现保留测试使用某 helper，只保留所需纯测试行为于既有 test helper，不引入 runner/framework |
| `schema_migrate_test.go` | RETIRE ImportTask 专属测试/fixture（从 `TestAutoMigrateImportTaskRepositoryMakesCategoryIDNullable` 起的专属段）和闲置 import | 前三项 owner constraint、sensitive-word、通用 PostgreSQL column type 检查全部保留；同段内仍被前部有效断言使用的原 `intPtr` 纯 helper 留在原文件，不恢复旧测试 |
| `owner_scoped_batch_test.go` | 只 RETIRE `TestBatchCreateImportTasksRejectsOwnerlessBatchBeforeInsert` | ProductData 无部分写、事务内 Store owner、mapping lookup/DB 错误与请求身份不可被 payload 覆盖的其余六项完整保留 |
| `owner_scoped_repository_test.go` | 只 RETIRE `TestUpsertScheduledTaskConfigRejectsOwnerlessWrite` | Store/category/规则等十项保留测试及通用 fixture/ownerless 断言不变 |
| 名含 ImportTaskID 的 mapping/product/raw JSON 其它测试 | 保留原有效断言 | 不以子串搜索批删测试、字段、列或数值 fixture |
| `internal/app/runtime/listingkitschemamigrate/runtime_test.go` 的 `openRuntimeSchemaTestDB` | RETIRE 零调用的旧任务建表 fixture；同批 BACKLOG finding 的精确补列，不重开架构 | 保留所有 runtime 测试、商用/Workbench scope 与旧 scope 在 I/O 前拒绝的断言；sqlite/gorm/modernc imports 仍被有效 Workbench SQLite 测试使用，全部保留 |
| 原 `tests/listingkit_backend_retirement_test.go` | 在同一原 guard 扩展目标 absence；将 import_task_repository 从 retained 列表转到 absence；增加混合文件目标符号不复活与保留合同检查 | 先在完整源码仍存在时确认目标 RED，再最小删除后 GREEN；其它必读文件、当前命令实际依赖、旧目录禁入和安全断言保持，不遇缺跳过 |
| 原 `tests/commercial_readiness_workflow_test.go` | 同步 §6.1 的精确命令与旧目标禁止引用断言 | pinned commit、race step/id、pipefail、日志/证据上传、失败分类和其它所有原要求不变；不整删测试或降低门禁 |

三项原检查逐项判断：

| 原测试 | 实际保护的旧行为 | 删除条件 / 现行保护 |
| --- | --- | --- |
| `TestConcurrentClaimForDispatchOnlyOneWorkerWins` | 8 个 writer 对旧任务 expected status→queued 的单赢家 UPDATE | 旧 claim/仓库/模型与全部注入归零后 RETIRE；不迁移旧排队协议，不制造空壳保测试 |
| `TestConcurrentRollbackDispatchOnlyOriginalQueuedClaimIsRestoredOnce` | 原 queued/node token 只允许一次 rollback，7 个 loser 返回 not-found | 同闭包退休后 RETIRE；当前 Submission/平台 UNKNOWN 与恢复 owner 不变，不能把这项旧测试当其覆盖证据 |
| `TestConcurrentRecoveryOnlyUpdatesStillEligibleRowsOnce` | 旧 processing/queued 过期行只被一次更新到 retry/pending | 同闭包退休后 RETIRE；不重新建设 watchdog/reconciler，当前 Store observations/worker 状态和测试保留 |

这三项是有效的历史旧协议安全测试，不代表所有并发安全需求都已取消。它们只在对应生产协议同批物理退出且无保留消费者时退休；保留 Store/owner/权限/幂等和当前执行协议的测试不得被替换成固定成功或新造的“等价”测试。

### 6.1 用户已确认的两套 workflow 精确变更

两处原专属 `-run` 正则均只包含上述三个旧测试。源码退休后不能继续保留它，否则 Go 可因零匹配而成功。同一主要 PR 中 **仅替换 run 命令，保留步骤、id 和失败阻断**：

| 文件 / 步骤 | 确认后的命令 | 保留内容 |
| --- | --- | --- |
| `.github/workflows/ci.yml` / `Run listing concurrency race tests` | `go test -race ./internal/listingadmin -count=1` | 原 step 与 Backend/Required CI Gate、当前两个 runtime build、所有其它检查保持；不保留旧 regex，不加 skip/continue-on-error |
| `.github/workflows/commercial-readiness.yml` / `Run ListingKit race tests` | `go test -race ./internal/listingadmin -count=1 2>&1 \| tee evidence/listingadmin-race.log`（实际 shell 命令的管道不带转义） | `id: race_tests`、`shell: bash`、`set -o pipefail`、原日志路径、outcome 分类、artifact、准确 SHA、触发/权限/并发/保护与其它步骤不变 |

替换后实际运行的是保留包内全部现有测试并启用 Go race detector，覆盖仍使用的 Store/owner/mapping/规则行为。它**不声称等价验证已退休的三套状态转换**，不新增测试平台或新业务需求；不扩大到全仓 `-race`。实施刷新已确认该包仍有上述保留测试，不能让空包/空选择冒充安全检查。源码实施与精确命令替换已获用户同批授权。

原 PostgreSQL 权限/public-facts/schema-context、资金并发、发布 OIDC/RBAC、前端/采集、依赖健康及其它所有必需检查完整保留；不改变 classifier/Required Gate/保护规则或把 SKIPPED 写成 PASS。合并另需最终准确 HEAD 授权，并核对实际 merge 对应 push/main CI。

## 7. 实施准入、验证与停止点

1. 用户已确认 §5、§6、§6.1 和本节，#29/#301 已先于修改记录 Reuse Existing Architecture / Ready、单 Writer/主要 PR、准确 main 和保留 owner；31 命令图、外部构造器/符号及旧表源码消费者刷新与冻结矩阵一致。原 #739/#736/#692 等其它活跃范围和运行 owner 不接管。任何新当前消费者不按旧设计强删，报告差异。
2. 原退休/实际依赖 guard 在旧源码仍存在时先 RED；按矩阵逐项删专属闭包、精确收缩混合 schema/types/测试，保留 owner helper 和 mapping/Store/规则全部有效行为；再 GREEN。原三项旧测试和两处精确命令同批变更，不出现零匹配成功、临时 stub 或空壳恢复服务。
3. 只执行适用原 retirement/文档/workflow guard、保留 listingadmin 与 SHEIN service/bootstrap/worker、必要 current 规则回归、31 命令编译/实际图及 §6.1 保留包 race 检查；不运行命令 main、schema、真实 recovery dry-run 或平台操作。稳定候选再跑完整 PR CI；未变代码证据可明确复用，不自行加验收矩阵/runner。
4. 最终候选前同步一次实际 main，只解决真实组合问题；一次集中独立交付检查，finding 先分类，同根因 sibling 一起处理。若实际发现当前 Must 的 owner/租户/数据/恢复 Blocker，停止对应实现并返回适用设计；非 Blocker 不重开全局架构或追加产品前置。
5. 实现完成进入 In Review，交付准确 HEAD、适用 CI/复核与保留入口/限制。本轮实现、自检、提交/推送和维护原 #740 已获授权；合并、关单、部署或实例/数据操作另行授权。真实运行/用户/平台验收 NOT_RUN，生产 NOT_READY，#29/#301 保持 OPEN。

```text
Legacy decision: RETIRE
Reusable behavior: 当前 Store/owner/规则/mapping/共享解析与原 worker 全部保留，不迁移旧任务协议。
Current owner: 当前各 Domain/Capability/observations 与原独立登录/Store 仓库；历史任务/表/事件/队列/锁归原运行 owner。
Cutover/deletion condition: 用户确认矩阵和 CI 处置、执行 Issue Ready、实际目标消费者归零、保留合同及适用检查满足；只退休源码，不处置真实持久化状态。
```
