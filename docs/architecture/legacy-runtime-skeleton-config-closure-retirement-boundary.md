> #301 / #914 **USER_CONFIRMED / SOURCE_IMPLEMENTED**：[旧闲置配置与运行辅助合同完整退休边界](legacy-idle-support-contract-closure-retirement-boundary.md)冻结 §1–§7（含 §6.1），Reuse Existing Architecture / Ready 读回先于源码修改。仅替代 TimeoutConfig/DefaultTimeoutConfig、RateLimitConfig/DefaultRateLimitConfig 及四项专属测试的旧 KEEP；HTTPClientConfig/RetryConfig/default、Config/LoggingConfig、现役Registry路由与platform日志及原安全断言保持。 下方历史正文逐字保留，其它KEEP/计数/证据不改；workflow/inventory及既有资源零改，无新试用前置，独立SHEIN登录与Store同步保持，#301 OPEN。

# #301 旧通用执行骨架与闲置宿主配置完整退休设计

Status: **USER_CONFIRMED / SOURCE_IMPLEMENTED**。用户已确认 §1–§7（含 §6.1），Reuse Existing Architecture / Ready 在 Issue/PR 读回后，同一 Writer 在 #897 整批实施；未授权合并。准确设计/实现 HEAD、原 CI、独立复核和批准写主要 PR，不为滚动证据制造状态提交。接单及前当前段存档见 [#301 / 6344768952](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6344768952)。

调查基线为 main **c30826ad893b0bca9c0380a156d78f0c3757496d**。#893 已按 259ce6d67 正常合并；对应 push/main CI39801396912 attempt2 完整 SUCCESS，首轮临时 PostgreSQL fixture 镜像缺失及一次失败 jobs 重试保留于该 PR。这些是前批证据，不替代本批设计或未来源码验证。

## 1. 用户结果、完整范围与权限

使用者是维护当前 HTTP 应用、采集和独立 worker 的维护者。用户明确选择的源码结果是：当前 HTTP 启动只保留正在使用的模块/路由贡献收集；已经无生产消费者的通用 Task、Workflow、workerpool、Temporal 注册执行分支和闲置配置完整退出。没有证据表明这些残留阻塞当前核心用户操作；不新增产品能力或内部试用前置。

同一 Delivery Batch 覆盖三组及全部专属尾部，不按文件数、目录或提交机械拆 PR：

1. `kernel/module` 中闲置 Task/Workflow/workerpool/Temporal 接口、注册方法、内存索引、专属 helper 和测试。
2. 孤立 `app/runtime/temporal_bundle.go` 及测试，HTTP `runtimeBundle.workerPools` 收集/转换和直接测试替身尾部。
3. `core/config` 中无消费者的 `CacheConfig`、`MonitoringConfig`、旧 `LogConfig` 及各自默认值/专属测试。

保留当前 `kernelmodule.Module`、`Registry/NewRegistry/AddRoutes/Routes` 和现有 HTTP 生产消费者；不搬迁、重命名或新建路由 owner。保留真实采集 workerpool、当前 Temporal/ImageAgent worker、独立 SHEIN 登录与 Store 同步及此前全部当前能力。三组共同结果是退出旧宿主执行骨架及无消费者配置，不引入替代调度器或治理系统。

用户已授权本设计范围内整批实施、必要自检、提交/推送和维护同一 PR。合并、部署、关单另行授权；不读取既有凭据、私密配置/日志/状态、Cookie/Profile/缓存，不执行应用/worker/浏览器/下载/安装/provider/PG/Redis，不操作既有实例、数据、凭据或进程，不清 ignored 文件。调查只读 Git 追踪源码、文档和依赖元数据。

## 2. Design Basis、Authority 与保留不变量

设计阶段 **Design Basis: N/A**（只读调查/三文档，不改生产运行语义）已完成。用户确认后的实施为 **Reuse Existing Architecture**；本设计范围/owner/验证获确认，Issue/PR 的 **Ready / IMPLEMENTATION_READY** 已在正式编码前读回。

适用依据：

- [Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)：只 EXTRACT/RETIRE，无兼容、旧状态迁移、fallback、双读/双写或第二事实源。
- [Project Target Architecture](project-target-architecture.md) 的 `kernel/module.Registry` 仅限 **startup contribution collection**，**not the Tool Runtime**；[Project Boundaries](project-boundaries.md) 的 HTTP 注册归 owning HTTP API、app 装配，以及 [Phase2 inventory](../refactoring/phase2-runtime-inventory.md) 的精确消费者登记。现有 `Module/Registry` 路由合同继续有效，未把整个 kernel 包判为 legacy。
- [旧宿主退休边界](legacy-host-runtime-closure-retirement-boundary.md)、[旧任务/刊登配置闭包](legacy-runtime-config-closure-retirement-boundary.md)、[日志门面退休边界](legacy-logger-facade-closure-retirement-boundary.md)：保留有效基础层/workerpool/配置，不因旧名称删除仍有消费者的能力。
- [独立 SHEIN 持久化边界](shein-worker-numeric-persistence-retirement-v1.md) 与当前采集、图像、Store 合同：原 owner、授权/组织隔离、持久化、fence/UNKNOWN 和副作用保留。

本批复用既有 app/kernel/Platform 边界，只去掉无生产注册/调用的分支和无消费者类型；不改当前路由合同/列表/顺序/启用条件、HTTP 授权/租户/认证、schema/事实 owner、状态机、事务、资源/计费、retry/UNKNOWN/recovery 或 provider 副作用。不新增高风险边界；UI/IA/Figma 增量 N/A。若必须改变上述边界，停止相关正式实现并升级 Independent Architecture；不靠机械拆批规避准入。

Must：完整删除 §5 的闲置分支/尾部，保留当前路由与所有当前能力；§6.1 仅删除一条 CURRENT 行，历史/计数/ceilings/accuracy/parser/其它检查及全部 workflow/依赖/锁/保护不改；没有真实资源操作或新试用前置。Should：复用现有 AST/decoded-import guard、Logrus/Viper/Temporal 和标准库，无新 EXTRACT 算法/共享框架。Out of Scope：整包删除 kernel/core/config、全仓配置拆分、真实 worker 重构、通用启动/关闭机制、运行验收/资源清理。Accepted Risks：无新增；真实运行/用户验收 NOT_RUN，物理状态 UNKNOWN。

## 3. 实际消费者及当前调用路径

### 3.1 当前路由是有效合同

`current-application → app/runtime/currentapplication → app/httpapi` 的现有工厂构造 `[]kernelmodule.Module`；每个 owning HTTP API 在 `Register(*Registry)` 贡献 `httproute.Descriptor`。HTTP bundle 按原顺序跳过 nil/disabled 模块、传播注册错误、读取 `Routes()`，再通过现有 authorization/server 装配提供请求。`Routes()` 的防御复制、`/health` 和当前业务路由/授权仍由原实现负责。

`Module{Name, Enabled(*config.Config), Register(*Registry)}` 及 `Registry/NewRegistry/AddRoutes/Routes` 保留，不重新发明 RouteRegistry/Module 协议，也不把 domain HTTP 包改依赖 app/httpapi。当前大量路由消费者无需改签名或 import；commerce Tool Registry、BusinessTask 和真正的 Temporal 工作流不是这套旧 TaskHandler。

### 3.2 已确认的闲置闭包

| 分支 | 追踪源码中的实际消费者 | 决定 |
| --- | --- | --- |
| `TaskHandler`、`RegisterTaskHandler` | 仅 kernel 自身实现及 `registry_test.go` 的 fake/旧注册测试；没有生产注册调用 | RETIRE 接口、map、方法、专属测试/helper |
| `WorkflowHandler/WorkflowRegistry`、`RegisterWorkflowHandler` | 同上；没有当前 Workflow/Tool 的生产注册调用 | RETIRE 名称索引、注册协议及专属测试；不替换当前 Temporal/commerce owner |
| `NamedWorkerPool/AddWorkerPool/WorkerPools` | 注册只在 kernel 和 HTTP 的旧测试；HTTP bundle 只收集空表，`pools()` 无生产调用 | RETIRE 注册、索引、HTTP 字段/转换和独占 fake；真实 `platform/workerpool` KEEP |
| `TemporalWorkerStarter/NamedTemporalWorker/AddTemporalWorker/TemporalWorkers` | 仅 kernel、孤立 Temporal bundle 和这些文件的测试 | RETIRE 整条模块注册/启动链；不接管现有 worker 的启动/关闭/recovery |
| `TemporalRuntimeBundle/BuildTemporalRuntimeBundleFromModules/Start` | 工厂/类型无生产入口调用，只有同文件及 `temporal_bundle_test.go` | 两文件 RETIRE，无替代 wrapper/自动启动路径 |
| 三项旧配置及默认值 | `CacheConfig/MonitoringConfig` 无生产消费者；`DefaultCacheConfig` 仅原测试，旧 `DefaultLogConfig` 仅 core/config 原测试 | RETIRE 精确声明和两个专属测试，不改配置加载/键或实际输出策略 |

调查已搜索全部 Git 追踪 Go（含测试、build-tag、独立 debug）的上述符号/调用，以及文档引用。`NamedWorkerPool` 的唯一包外生产类型尾部在 HTTP bundle，Temporal 类型尾部在孤立 bundle；没有另一个现役装配入口。历史 2026-05-29 framework plans/specs 中的旧注册 API 是 DATED EVIDENCE，不能恢复当前调用或扩大权限。

### 3.3 同名能力必须区分

旧 `core/config.LogConfig/DefaultLogConfig` 不等于当前 `core/config.LoggingConfig` 或 `platform/logging.LogConfig/DefaultLogConfig`；后两者和全部当前日志行为保留。`CacheConfig` 不是删除浏览器缓存或当前 cache owner 的依据，`MonitoringConfig` 的未使用预留字段也不改变现有 health/风险判断。`TimeoutConfig/HTTPClientConfig/RateLimitConfig/RetryConfig` 及默认值保留。SDS 测试 fake 与 workerpool fake 共用文件，不能整文件删除。

## 4. 配置、持久化与副作用 owner

| 边界 | 当前 owner / 本批变化 | 保留条件 |
| --- | --- | --- |
| Registry routes | 当前 app/owning HTTP API 的启动贡献列表；无持久化事实 | 原路由顺序、复制、模块启用、nil/错误处理及授权装配不变 |
| taskHandlers/workflowNames、pool/Temporal 名称索引 | 旧 Registry 内存容器，无生产注册消费者 | 只删除闲置源码，不访问或改写现有进程内存；不生成新的执行/恢复 owner |
| 旧 Temporal bundle 启停 | 无生产调用的闭包；不是持久化 Temporal 事实 owner | 不取消或触碰任何真实 workflow/queue/worker；真正 worker 的启动、失败关闭、重试/UNKNOWN 全部 KEEP |
| 三项旧配置类型 | 未进入当前 Config 字段、loader/default/env/validator 的有效加载路径 | 只删声明/专属测试；Viper/私密文件/环境选择、defaults、当前键名、数据库/Redis/身份/计费配置零改 |
| 当前 SHEIN/Store/采集/图像 | 既有各 domain/runtime/persistence/worker owner | 登录、Store 同步、fence/UNKNOWN、数据/凭据/Profile/外部平台副作用均不变 |

没有表、迁移、持久化字段、外部 wire DTO、事务或幂等身份变化；无新迁移/补偿/恢复协议。源码缺席不证明物理状态已清理，也不授权真实环境动作。

## 5. 完整删除、保留与抽取矩阵

无新 EXTRACT：有效路由、当前配置和真实 worker 已在当前 owner，本批只 KEEP；旧分支 RETIRE，不迁移到新包或包装保活。

| 准确工件 | 同批处置 | 明确保留 |
| --- | --- | --- |
| `internal/kernel/module/interfaces.go` | 删 TaskHandler、WorkflowHandler、TemporalWorkerStarter 及仅为其服务的 context import | Module 原三个方法及 config 参数合同 |
| `internal/kernel/module/registry.go` | 删 NamedWorkerPool/NamedTemporalWorker/WorkflowRegistry；Registry 六项闲置字段与 NewRegistry 对应初始化；六个注册/读取方法、WorkflowRegistry.Register、仅被这些分支使用的 isNilInterface 及 fmt/reflect/strings/workerpool imports | Registry.routes、NewRegistry、AddRoutes、Routes 原有行为 |
| `internal/kernel/module/registry_test.go` | 删旧 Task/Workflow/pool/Temporal 注册测试、stubTaskHandler/stubWorkflowHandler/stubWorkerPool/stubTemporalWorkerStarter 及独占 imports | TestRegistryAddRoutesPreservesOrder、TestRegistryRoutesReturnsDefensiveCopy 原断言 |
| `internal/app/runtime/temporal_bundle.go`、`temporal_bundle_test.go` | 两个完整专属文件 RETIRE | 同包 image_agent_temporal_runtime、temporal_client_support 及其有效测试全 KEEP |
| `internal/app/httpapi/runtime_bundle.go` | 删 workerPools 字段、reg.WorkerPools 收集、pools() 及独占 workerpool import | routes/authDependencies、原模块循环和错误、buildServerBundle 完全保留 |
| `internal/app/httpapi/http_module_test.go` | 将 CollectsRoutesAndWorkerPools 收敛为原路由贡献断言，去掉 AddWorkerPool/fake/pool 断言 | health、nil registrar、disabled/nil modules、路由/HTTP 结果等原有效断言 |
| `internal/app/httpapi/legacy_product_retirement_test.go` | 删除仅测试旧 pool 结构的参数和 mutatedPools；原 module 禁入 helper/负例保留；失去意义的 pool 结构断言由 §6.1 原 guard 的声明/字段/import 缺席保护接替 | 当前生产组合非空、旧 Product/Image HTTP 路由 404、system 模块存在、product/product-image 模块 mutation 必须拒绝 |
| `internal/app/httpapi/test_stubs_test.go` | 仅删 stubWorkerPool 及其 context/workerpool imports | 同文件 stubSDSCatalogRouteHandler 及 net/http、Gin 合同不动，不顺带退其它 owner |
| `internal/core/config/common_types.go` | 精确删 CacheConfig/DefaultCacheConfig、LogConfig/DefaultLogConfig、MonitoringConfig/DefaultMonitoringConfig 六声明 | TimeoutConfig/HTTPClientConfig/RateLimitConfig 及默认值原样；其它 Config/LoggingConfig/Platform 日志全保留 |
| `internal/core/config/common_types_test.go` | 仅删 TestDefaultLogConfigDoesNotEnableFileOutput、TestDefaultCacheConfig | Retry/Timeout/HTTPClient/RateLimit 的默认/自定义/零值断言不变；Platform 原日志默认不创建运行文件安全测试不变 |

以上为 11 个准确源码/原测试文件（含删除的两个文件），加 §6.1 原 guard、CURRENT 一行和三文档，不另开机械辅助 PR。若有未列实际消费者、必须改冻结 KEEP/当前合同或新的领域边界，先报告精确差异，不自行扩范围。

```text
Legacy decision: RETIRE
Reusable behavior: 当前 Module/Registry 路由贡献、配置加载和真实 worker 已由现有 owner 提供，保持原行为
Current owner: kernel/module 的现役路由合同 + owning HTTP API/app assembly；Platform/Temporal/current runtimes 与各 domain/persistence
Cutover/deletion condition: 用户确认本设计/Ready读回；闲置分支无生产消费者，整批源码及专属测试尾部退出，原有效断言与必要检查保持；不等于物理状态清理
```

## 6. 必要验证与证据

设计阶段仅文档/链接/当前架构索引/消费者登记适用原检查与准确设计候选原 CI、一次独立设计复核，不把当前源码未退的状态写成 GREEN。

实施获准后：先在原退休 guard 增加下面精确缺席检查，RED 捕获现有骨架/字段/旧配置；再按矩阵整批删除得 GREEN。使用既有 AST/decoded import/追踪文件设施，无新 runner/SSA 平台/共享治理工具；读取、解析错误必须 fail，不 missing-skip，不加 dummy/alias 迎合旧测试。

必要自检只覆盖：原 kernel 路由顺序/防御复制，HTTP health/模块启用/nil/错误和原当前路由/授权回归，保留配置值与原安全测试，原 Phase2 非增长/CURRENT 准确性和原退休检查。用既有 synthetic/fake/httptest，不启动真实 HTTP listener/worker/PG/Redis/provider，不执行旧 bundle 的模拟启停来保活已退协议。三个当前入口按原构建/依赖元数据验证，不运行产物；稳定候选交原完整 CI。无新产品路径或运行验收矩阵。

真实 worker 的正确关闭/恢复测试不能因删除孤立 bundle 的同名测试而删掉；原 ImageAgent runtime、采集 workerpool、SHEIN/Store 的有效测试继续由其原 owner/原 CI 约束。准确 HEAD、直接执行和未变化证据复用分列；运行/用户验收始终 NOT_RUN，不能由 AI 或 CI 自行签发。

### 6.1 原检查、CURRENT inventory、文档与 workflow 精确处置

本矩阵已获用户确认，随 #897 整批实施；设计阶段未提前应用：

| 对象 | 同批准确处置 | 不得改变 |
| --- | --- | --- |
| `tests/listingkit_backend_retirement_test.go` | 在原设施加一个本闭包退休检查：两个 bundle 文件缺席；kernel owner 的退休顶层类型/接收者方法/Registry 字段缺席；kernel 全追踪 decoded imports 不得再依赖 workerpool；HTTP runtimeBundle.workerPools/pools 和 core/config 六声明缺席。对保留 Module/Registry 路由方法及当前配置 owner 有正向约束 | 不全局禁止 kernel/module、Task/Workflow/LogConfig 等同名当前能力；不禁止真实 workerpool/Temporal 进入当前命令依赖，不动原退休/安全/解析/读取错误逻辑 |
| 原 kernel/HTTP/config tests | 仅 §5 指定旧实现细节测试/fixture 转换；pool 结构 mutation 的退出由上述源码缺席 guard 接替 | 原模块 mutation/路由404/system存在、路由复制/顺序、所有当前授权/隔离/幂等/状态/配置/日志行为仍必须 fail-closed；不是删除整个原回归 |
| `docs/refactoring/phase2-runtime-inventory.md` CURRENT | 仅删下方一条 kernel/module→platform/workerpool 行 | 所有其它文本/行、历史、计数、ceilings、accuracy/parser/排除规则不改；core/config→Platform/OpenAI 等仍有消费者的行保留 |
| 本文、Architecture README、Legacy Register | 当前提案/用户确认状态及准确 supersession，继续引用当前路由 owner | 保留所有原历史正文/行；Project Target Architecture 中现役 Registry 的 startup contribution collection/not the Tool Runtime 合同不改 |
| `.golangci.yml`、Code Health、其它 import/provider/SDK/SSA/Phase2 checks | **零改**，运行适用原检查 | 不加豁免/新允许项、改 ceiling、放松当前消费者准确性或改检查逻辑求绿 |
| 全部 workflow/命令/部署/依赖/锁/保护 | **零改**，原 Backend/Frontend/Required/Security/Admission 等保持 | 不删/跳过检查、不改 build 目标/flag/镜像/tag、强推或管理员绕过 |

唯一删除的 CURRENT 行：

```text
| `task-processor/internal/platform/workerpool` | `task-processor/internal/kernel/module` | runtime-support debt | app retirement / Phase 8 |
```

原 workerpool ceiling 23 等历史数值不变；仅这个实际 import 退出，不意味着全部 workerpool 或 kernel 路由源退休。历史 framework plans/specs、dated inventory 的旧注册说明继续保留，由本设计在准确分支上 supersede，不批量重写旧文档。

## 7. 准入、复核、停止及父目标边界

1. 设计阶段 N/A 三文档、原 CI 和一次独立只读设计复核已完成；用户随后确认具体设计与整批实施。设计阶段没有提前写 guard/生产代码或删除源/CURRENT；设计 CI/评审不替代源码结果。
2. 用户确认 §1–§7（含 §6.1）后，同一 Writer 已在同一主要 PR 记录 Reuse Existing Architecture / Ready 并读回，再整批实施。既有边界足够则不重复全局架构审核；若实际必须改变当前合同/权限/持久化/状态/恢复/副作用边界，停止相应正式实现并升级准入。
3. 实施最终候选只做一次完整真实 diff/调用路径/保留能力/证据检查；已确认未变化部分复用，finding 先分类，修复只复核增量。不得由 Reviewer 增加新的试用资格或验收平台。
4. 原 CI 真失败先读准确日志并查同根因 sibling paths；矩阵外修订/冻结 KEEP 变化给出最小精确提案，不能自行改 workflow、历史/ceilings、检查或默认恢复旧兼容构造。
5. 发现当前外部可观察契约/持久化运行态似乎要求兼容时先报告；没有可执行的 migration/compatibility Exception。不接管其它 owner，不因同名 Task/Workflow/Config 扩大删除。
6. #301 保持 OPEN，#29 不重开；更新前新读 Issue，仅改当前段，63542 字符历史后缀与勾选逐字保留、正文不超过65536。源码完成不代表全父目标/运行验收完成。
7. 合并、实际 merge SHA 的 push/main 确认、部署、关单及真实资源动作分别按用户授权；保留独立 SHEIN 登录/Store 同步，不读既有凭据、不操作实例/数据/进程、不增加内部试用前置。
