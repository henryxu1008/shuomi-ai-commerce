# #301 旧 RabbitMQ、数字 Store 代理运维及宿主配置/辅助尾部完整退休设计

Status: **CONFIRMED / Reuse Existing Architecture / Ready → In Progress**。用户已确认冻结设计 `f59bdf89c410ea8317f8017d7f28e6dcfacb16ee` 的 §1–§7（含 §6.1），授权在 #872 整批实施，包含 fileio；inventory 仅删除 core/logger→pkg/fileio 一条 CURRENT 行。#301/PR 准入已在生产修改前读回，见[用户确认与开工记录](https://github.com/qq550723504/task-processor/pull/872#issuecomment-6317288178)。下列原设计阶段时态保留为历史；实施权限以此状态及 §5.3 当前决定取代，删除/保留边界不变。调查基线 main `9e658aeeb16674c613d39ad37ae14f60574c7d59`（#870 的真实合并提交），[准确 push/main CI39600221958](https://github.com/qq550723504/task-processor/actions/runs/39600221958) SUCCESS，只作基线证据。#301 保持 OPEN，#29 不重开。一个 Writer、一个分支、一个主要 PR 覆盖三组完整闭包，不按包、脚本、配置或 helper 拆小批。准确实现 HEAD、原 CI、最终独立检查和用户决定写 PR。

## 1. 用户结果、范围与权限

使用者是维护当前应用、采集、Store 与独立 SHEIN worker 的维护者。交付结果是旧数字任务的 RabbitMQ 实现、数字 Store 代理改写工具、旧宿主残留配置与闲置辅助包一次退出，当前登录/采集/Store 路径继续使用现有 owner。没有证据表明这些残留正在阻塞核心用户操作；本批是源代码退休，不新增用户入口、平台或内部试用前置。

完整核对三组：

1. `internal/platform/queue/rabbitmq` 及专属 README、旧主链探针和孤立测试配置。
2. `scripts/sync-shein-store-proxies.ps1`、`deployments/kubernetes/shein-proxy-sync`，以及只有这条数字代理链消费的四份 SHEIN Secret 源模板与 prod 装配。
3. `UpdaterConfig` 的 schema/default/build/示例尾部，以及无当前消费者的 `pkg/cache`、`pkg/perf`、`pkg/types`、`pkg/fileio`。fileio 与 inventory 零改的冲突见 §5.3，不因台账登记而赋予它保留价值。

**原设计阶段只授权只读代码/元数据调查、本文、Architecture README/Legacy Register 提案入口和 #301/PR 维护。** 用户现已明确确认整批实施，允许 §5/§6.1 准确源码、测试、配置、部署源和直接文档更新，以及仅一条 CURRENT inventory 校准；其余 inventory、schema、workflow、依赖/锁、ceilings、保护规则均不改。合并、部署、关单另行授权。

不执行旧脚本、应用、业务 CLI、constructor、dry-run、kubectl/kustomize/Helm、SQL 或进程枚举；不读现有私密 binding、代理/登录文件、Secret、凭据、token、配置、日志或报告；不操作既有实例、数据、队列、Profile、Redis、进程或源码外文件。不启动 PG/Redis/RabbitMQ/browser，也不访问真实 provider。原 CI 按既有 workflow 使用原隔离 fixture，不因此取得真实环境操作权限。物理状态/外部人工使用情况 UNKNOWN；运行验收和用户验收 NOT_RUN。

## 2. Design Basis、Authority 与不变量

原只读/文档工作：**Design Basis: N/A**。当前实施：**Reuse Existing Architecture / Ready → In Progress**，用户已确认且 Issue/PR 准入已读回。依据：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[Issue 规则](../engineering/issue-driven-development.md)。旧设计只有 EXTRACT/RETIRE，没有兼容类、迁移、旧 ID 映射、fallback、双读写或第二事实源。
- [#836 配置边界](legacy-runtime-config-closure-retirement-boundary.md)、[#839 运维边界](legacy-listing-operations-closure-retirement-boundary.md)、[#867 采集尾部边界](legacy-numeric-acquisition-tail-closure-retirement-boundary.md)、[#870 旧宿主边界](legacy-host-runtime-closure-retirement-boundary.md)。本设计仅拟取代本轮准确目标的 RabbitMQ、数字代理工具/专属 Secret 源、Updater 配置和闲置 helper 保留含义；这些文档的其它当前合同、历史正文/计数/失败不变。
- [#863 独立 SHEIN worker 当前所有权](shein-worker-numeric-persistence-retirement-v1.md)、[当前来源闭环](source-business-closure-v1.md)、[当前身份/组织](auth-and-tenancy.md)、[App 装配](app-assembly-boundaries.md)与[HTTP 装配](httpapi-assembly-boundaries.md)。不改变当前身份、授权、租户、Store、执行 fence、UNKNOWN、持久化或恢复合同。

本批预计退出 45 个专属源/测试/声明/说明文件，若单行 inventory 例外获准则为 47 个，另有混合源和直接文档/原 guard 校准，命中 >30 文件与多子系统提示线。现有架构直接覆盖：RabbitMQ 已无包外 importer/当前命令装配；代理工具仍操作已退出的数字账号/Store 配置，不是当前独立 worker 的适配器；Updater 执行器已退出。没有新增/改变当前领域合同、持久化事实 owner、状态机、权限、事务、retry/UNKNOWN/recovery 或跨系统机制，不为文件数重开全局架构或机械拆 PR。若发现当前合同确需这些实现或需改变上述边界，停止相关正式实现并报告；现有依据不能覆盖时升级 Independent Architecture，不能在实现阶段临时发明 owner。

Must：三组及专属尾部整批处理；当前采集、图像、AI/SKU/spec、价格、平台 API/规则、身份与计费、共享合同、Store 观察/同步、独立 SHEIN 登录/同步不变；必需/安全检查保留；无真实状态操作或新试用前置。Should：复用原 guard、标准库及当前成熟组件，不抽取没有当前消费者的库。Out of Scope：现役共享基础层重构、依赖清理、broker/Secret/旧表/远端属性的实际处理、旧代理迁移/替换、运行发布验收。Accepted Risks：无新增；不把未访问的物理状态写成不存在。UI/IA/Figma N/A。

Threat Model 限于误退当前共享能力、混淆旧数字 Store 与当前组织/UUID Store、把源声明删除扩为真实 Secret/数据库/进程操作，以及移除原 guard 后旧链复活。旧脚本的多步跨边界写入只说明其旧副作用，不据此建设新 Saga、代理分配器或恢复平台。

## 3. 实际消费者与调用链

在准确基线交叉核对 tracked Go imports、`go list -json ./internal/... ./cmd/...` 的 Imports/TestImports/XTestImports、全部 31 个 cmd 加原生身份 CLI `internal/zitadelprovision/cmd` 的实际 `go list -deps`，并搜索 tracked 脚本/声明/文档/guard。没有执行程序入口。

| 目标 | 实际消费者/装配 | 结论 |
| --- | --- | --- |
| platform/queue/rabbitmq：16 生产 +13 测试 | 包外生产/测试 importer 均为 0；32 个当前命令依赖均不包含它。只剩自身测试、provider 上限/消费台账检查及一个正向包边界 guard | 全包旧连接/拓扑/消费/重试/负载监控一起 RETIRE；没有当前队列 owner 要迁入 |
| validate-rabbitmq-flow.ps1 | 旧 `/admin-api/listing/task-management/*`、`/api/v1/management/tasks/*` 和旧消费者 health/stats 探针；唯一直接操作说明已标历史 | 随原主链退出，不替换为新的验收工具 |
| tests/test_config.yaml | 全仓无文件读取引用；只有旧 rabbitmq、result_reporter、load_monitor、node 和数字 crawler 测试数据 | 整份孤立旧 fixture 退出；不是当前通用 config fixture |
| sync-shein-store-proxies.ps1 | 人工入口；读取 `listing_store` 数字 tenant/store，写旧 `platforms.shein.shop_proxy_map` 和 `listing_store.proxy`，重启旧 login deployment | 无当前 Go shop_proxy_map 消费者；不是当前独立登录 worker 的代理配置 owner |
| shein-proxy-sync 两声明 | Kustomization → 同名 SA、两 namespace 各一 Role/RoleBinding、ConfigMap 内联 shell、CronJob；shell 执行同一数字 Store 代理编排 | 完整源闭包退出，不能留下内联第二实现或专属授权声明 |
| shein-listing-secret 四源 + prod 装配 | 当前仓库仅上述脚本/CronJob 读取此名；base 只装配 Namespace，prod 只装配 base + ExternalSecret。其余命中是自身/guard/历史说明 | 最后源码消费者随本批退出后四源和 prod 专属装配同退；共享 Namespace/base 装配保留 |
| UpdaterConfig | 仅 Config 字段、default/helper、NewDefaultConfig、BuildConfig 和 dev/prod/test 三示例自循环；#870 已退唯一旧执行器 | 删除完整配置尾部；其余 core/config 是当前共享能力，不能整包删除 |
| pkg/cache、pkg/perf、pkg/types | 各1生产；types另1专属测试；包外生产/测试 importer 都为0；32命令不依赖 | RETIRE TTL janitor、性能 Tracker、旧 FlexibleID/Time 及专属测试；不推广到其它同名当前值合同 |
| pkg/fileio：1生产+1测试 | 无包外 importer/命令依赖；自身 New 仍导入 core/logger，因此 CURRENT 台账存在1条消费记录 | 应 RETIRE 的孤立旧 task JSON dump；单行台账决策见 §5.3 |

当前保留调用链：

| 合同 | implementation → injection → consumer / owner |
| --- | --- |
| 采集与并发/重试 | 当前来源入口/Processor/adapter → platform/workerpool、原 shared/Retry/HTTP/浏览器能力；当前 Temporal 仍由 platform/temporal 和原 worker 注入。都不引用拟退 AMQP 包 |
| 独立 SHEIN 登录与 Store 同步 | cmd/shein-login-worker → app/runtime/sheinloginworker → sheinlogin/bootstrap、当前组织/Store/SessionTransport/仓库/审计/fence；私密 Binding.ProxyFile 由原 SessionAccount 读取，绑定 organization/store UUID/revision/generation。没有读取旧数字 shop_proxy_map 或 listing_store.proxy |
| Store/身份/平台/计费 | 当前 Store 观察和同步、Auth.js/native CLI、组织授权、平台 API/DTO/纯规则及资金/资源 owner 保持；#692 是独立 schema 设计，不在本批实施 |
| 共享基础层 | core/config/logger、platform config/logging/database/redis/workerpool/temporal/metrics/observability、当前 HTTP/图像/AI/Prompt/缓存/值合同仍归原 owner；未装配但已批准的库/工具不因 orphan 搜索被扫除 |

仅“无 importer”不足以删除通用人工工具/已批准能力。本批同时核对已退旧语义、当前需求/owner 和用户给定范围。`start-shein-local-rabbitmq-portforward.ps1` 是通用 PG/Redis/RabbitMQ 人工诊断工具，保持原源；不调用其进程枚举/终止/启动分支。`middleware-single/rabbitmq.yaml` 是共享 broker 源声明，保留且不 apply；旧数字任务源码退休不授权处理基础设施。

## 4. 持久化、副作用与失败 owner

| 边界 | 源码揭示的旧行为 | 本批处置及保留 owner |
| --- | --- | --- |
| AMQP | Connect 建连接/监控 goroutine；initializer 声明 durable exchanges/queues/bindings；consumer QoS/consume、ack/nack、重投 persistent 消息、DLX、重连与负载并发。tasks.exchange/dlx/delay/result、数字 Store/平台桶及旧 crawler 路由均在包内 | 整包源码 RETIRE；不 connect、声明、清队列或迁移消息，不把原 ACK/retry 语义迁成当前新 owner。真实 broker/messages/未确认消息 UNKNOWN，由原运行 owner 另获授权处理 |
| 数字数据库 | 两代理入口查询 listing_store 的 int tenant/id/proxy，并条件更新对应两列归属，含 enable_auto_listing 筛选 | 删除工具源不执行 SELECT/UPDATE，不迁移/删旧表或数据；当前组织/UUID Store 与独立 worker repo 原样 |
| Secret/远端配置 | 读取 DB Secret、login-config；解码配置/凭据到临时文件，改 shop_proxy_map，create/apply login-config；ExternalSecret 声明远端属性映射 | 专属源码/模板同退；不读/删现有 Secret、SecretStore、远端属性、临时文件或凭据。没有替代 Secret owner |
| 进程/外部副作用 | kubectl exec/psql、rollout restart/status login；Cron 每10分钟、Forbid/backoffLimit1；PowerShell DryRun 在前序读取和写临时文件后才退出 | 不运行任何分支，包括 DryRun；源码退休不证明 Cron/进程已停，不改真实调度或运行部署 |
| Updater 配置 | 旧 URL、enabled、interval、TLS开关、CurrentVersion 的无执行器 schema/default/build | 移除源码尾部，不访问 URL/更新文件/当前版本或替换进程；当前发布器/版本合同不变 |
| 闲置 helper | cache 内存/TTL janitor；perf 内存时间/日志；types JSON/SQL接口零值转换；fileio 可写 taskID 时间戳 JSON 文件和目录 | 无当前消费者要求 EXTRACT；不启动 janitor、不运行旧 fileio 测试分支、不读/删已有日志文件。现有当前缓存/日志/JSON/DB合同不变 |

没有新表、迁移、事务、事实 owner、缓存、状态机、幂等键、自动接管或恢复路径。当前请求失败、响应丢失、并发、撤权、取消、重启、超时/大小上限继续由未修改的当前 owner 处理；独立 worker 的 UNKNOWN 不自动接管。旧代理的 Secret→DB→restart 分步写入及 AMQP 重投只作为待退源码事实，不为其修复或新建跨系统一致性机制。

## 5. RETIRE / KEEP / EXTRACT 完整矩阵

### 5.1 专属源码删除清单（用户确认后，同一 PR）

| 目录/组 | 准确文件 |
| --- | --- |
| internal/platform/queue/rabbitmq：生产16 | client.go、config.go、connection.go、consumer.go、consumer_state.go、error_collector.go、load_monitor.go、load_monitor_config.go、message.go、naming.go、queue_config.go、queue_consumer.go、queue_initializer.go、retry_strategy.go、sliding_window_stats.go、system_metrics_collector.go |
| 同目录专属测试13 | client_test.go、config_test.go、connection_test.go、consumer_test.go、consumer_state_test.go、load_monitor_test.go、load_monitor_collector_test.go、message_test.go、naming_test.go、queue_consumer_test.go、queue_initializer_test.go、retry_strategy_test.go、sliding_window_stats_test.go |
| RabbitMQ 专属尾部3 | internal/platform/queue/README.md（仅声称 queue runtime）、scripts/validate-rabbitmq-flow.ps1、tests/test_config.yaml |
| 数字代理脚本/声明3 | scripts/sync-shein-store-proxies.ps1、deployments/kubernetes/shein-proxy-sync/kustomization.yaml、该目录 shein-proxy-sync.yaml（包括全部7个对象及内联 shell） |
| 最后消费者退出的4个模板 +1装配 | deployments/kubernetes/shein-listing/base/secret.example.yaml；该根下 overlays/prod-store-shards-example/secret.example.yaml、overlays/prod/secret.example.yaml、overlays/prod/external-secret.yaml、overlays/prod/kustomization.yaml |
| 宿主配置 schema1 | internal/core/config/type_updater.go |
| 孤立 helper4 | internal/pkg/cache/cacheutil.go、internal/pkg/perf/perfutil.go、internal/pkg/types/flexible.go、internal/pkg/types/flexible_test.go |

共45个准确专属文件（20生产Go、14专属Go测试及11个其余源/fixture/说明）；不递归删除混有保留对象的目录。不移动到新包、不留空壳/假消费者/兼容适配器。原 AMQP SDK/其它依赖/锁不清理。

混合配置只删：`internal/core/config/config.go` 的 Updater 字段；`defaults.go` 的 setUpdaterDefaults 调用及函数；`loader.go` 的旧默认块；`loader_builder.go` 的旧构造块；`config/config-dev.yaml`、`config-prod.yaml`、`config-test.yaml` 的 updater 子树及准确专属注释。不是删除其它配置值或通用 env/loading/validation。Config.CurrentVersion 等同名当前业务字段不能按字面批量扫除。

### 5.2 保留和不新增抽取

**没有新的 EXTRACT 需求。** 这些旧 helper 没有当前消费者，不为“将来也许有用”新建缓存/Tracker/codec/Task dump/代理分配库；有当前需要的标准库/成熟能力继续使用原 owner。

- 当前采集、来源保存/lineage/读取、图像处理/上传、AI/SKU/spec、PriceManager/PriceCalculator/PriceHandler、两种 SHEIN 属性/MappingInfo、四个来源 JSON 零值槽位和原规范化/hash/读取拒绝合同全部保持。
- 当前身份/组织/计费、UI/token handoff/native CLI、Store观察/同步、独立 SHEIN登录/同步及原 DB/Redis/Profile/Binding.ProxyFile/审计/fence/UNKNOWN 保持，不读私密输入。
- `internal/core/config/type_rabbitmq.go` 实际只含 **RetryConfig/DefaultRetryConfig**，必须保留原样，不能按文件名删；整个有效 core/config/logger、真实 workerpool、通用 Retry/队列合同、Temporal、featureflag/observability、shared/strx、当前缓存/HTTP/JSON/DB值行为原样。
- `deployments/kubernetes/shein-listing/base/namespace.yaml` 与 `base/kustomization.yaml` 原样保留；共享 task-processor Namespace 不归旧代理退休。其余当前 workload、listingkit-workbench、Docker/Compose、middleware-single/rabbitmq.yaml、通用 PG/Redis/RabbitMQ/Temporal/UI port-forward/诊断工具不改。
- 当前 Go SDK/depguard 安全边界、所有 workflow、依赖/锁、ceilings、历史记录/计数、真实资源与既有文件不改。

### 5.3 fileio 单行 inventory 例外的确认与边界

`pkg/fileio` 没有当前调用者，按当前需求应与本批一起 RETIRE；它不是有效共享能力。`phase2-runtime-inventory.md` 的 **CURRENT Legacy consumer register** 仍有且只有 `core/logger → internal/pkg/fileio` 一行。原 `TestPhase2InventoryNamesEveryRemainingLegacyConcreteConsumer` 同时拒绝遗漏和 stale row；删源码、留该行必然违反原检查。台账不是 fileio 保留价值的依据。

用户已[明确批准该单行例外](https://github.com/qq550723504/task-processor/pull/872#issuecomment-6317143198)，并进一步确认本设计整批实施：**只删除这一条 CURRENT 消费者行**，同批删除 `internal/pkg/fileio/fileutil.go`、`fileutil_test.go`，完整专属删除清单为47文件。不改历史/dated行、原始计数、其它当前消费者、ceilings、检查逻辑、workflow 或依赖；原 inventory 准确性检查保持原样。

原严格零改下排除 fileio 的备选已被该明确决定取代，不再作为当前实施方案。不制造假消费、只改 import 绕 guard、移空壳或删除准确性检查；不引入新的架构机制、试用前置或其它当前能力的阻塞。

```text
Legacy decision: RETIRE
Reusable behavior: 无新抽取；当前平台/采集/Store/独立SHEIN与共享规则继续原owner
Current owner: currentapplication/httpapi、当前领域与platform/workerpool/temporal、sheinlogin/Store原owner；不接收旧数字代理/任务事实
Cutover/deletion condition: 用户确认本设计/§6.1及fileio单行例外选择；刷新真实消费者；准确源码与专属测试/声明/文档尾部同退；保留当前合同与原必需安全检查
```

## 6. 必要验证与证据边界

当前设计仅做静态来源/引用/元数据核对、文档一致性和原适用文档检查，不为纯文档制造 RED，不运行业务程序或旧 RabbitMQ/代理工具。历史 main PASS 只作基线，不能冒充新设计/实现 SHA 的直接验证。

未来确认实施后：

1. 在原 retirement guard 增加准确源缺席、全 internal 生产 imports 禁入与旧配置声明/keys 缺席，确认 RED 后再删除/校准至 GREEN；复用原 AST/import decoding/read-failure 设施，不增扫描器/runner/专项验收环境。
2. 复用现有 core/config 测试确认默认/Build/LoadFromBytes 不要求 updater，Retry/平台transport/browser/身份/图像配置仍有效；不建新兼容 reader，不以“忽略旧未知配置”新增产品承诺。
3. 保留当前命令依赖和独立 worker 不编译 HTTP 宿主的 guard；稳定候选运行原必需 CI，31cmd+原生身份CLI实际依赖/编译确认拟退包退出。当前授权/组织隔离/Store/worker/UNKNOWN 原有效证据按代码未变明确引用；不额外重跑无关环境矩阵。
4. 核对45/47专属删除、七个混合配置源及§6.1准确尾部，当前保留输入逐叶不变。workflow/inventory（除明确一行例外时）/依赖/ceilings/保护规则零diff；没有实际读凭据、PG/Redis/broker/browser、SQL、进程或provider操作。
5. 一次完整最终候选独立检查真实diff/保留路径/证据，修复只复核相关增量；开发自检/CI/AI复核均不等于用户实际验收、部署或#301完成。

### 6.1 原检查、文档、inventory 与 workflow 的精确处置

| 对象 | 用户确认后的同批最小校准 | 必须保留 |
| --- | --- | --- |
| RabbitMQ13测试、types1测试；条件fileio1测试 | 随准确旧实现退出，不把未装配实现细节当当前需求 | 当前 workerpool/Temporal/Retry/平台/身份/Store/worker 规则与原安全测试 |
| tests/listingkit_backend_retirement_test.go | 原 closure设施以 TestLegacyQueueProxyAndConfigTailClosureIsRetired 增加本批准确源/包缺席及imports禁入；在 TestNumericAcquisitionAndPlatformTailClosureIsRetired、TestRetiredListingOperationsClosure、TestRetiredTaskAndListingRuntimeConfigClosure 中，将四个旧Secret源/prod装配/代理源正向保留项替换为本批准确缺席；只移除已退模板的旧字段扫描；Config/updater反射/AST/YAML禁入补原位置 | 原 shared Namespace/base装配、当前独立worker镜像/health、Prometheus storage告警/同namespace窄路由、其它工作负载缺席/源码禁入、解析/读取失败必须fail closed；保留listingkit-workbench当前Secret扫描，不能删整个测试或改missing-skip |
| tests/target_architecture_phase2_test.go | 原 TestPlatformQueueRabbitMQDoesNotImportApplicationOrLegacyInfrastructure 改为同目标已退包缺席/生产禁入；原通用索引从存在的internal祖先检查，而非在根已退时假装正向测试成功；同步直接guard索引中该项名称/语义如实际变化 | 原其它SDK/平台/领域边界、import字符串解码、read/parse失败、两个provider ceiling表中的RabbitMQ18等全部冻结；目标消失不放大上限或使其它guard空跑 |
| internal/core/config/config_test.go | 原 TestCurrentConfigDoesNotRequireRetiredTaskInputs 加 Updater 字段与updater keys缺席；原当前配置/Retry测试保持 | 原全部非updater配置/schema/env/loading/validation；type_rabbitmq.go保留原样 |
| 原直接guard索引（若实际名称变化） | Architecture README/Review Checklist/next-technical-priorities仅同步上项原函数引用；未改名则不动索引 | 原coverage/引用顺序/链接和所有无关安全项；不新增治理框架或守卫验证器 |
| 当前说明/先前KEEP | Architecture README、Legacy Register；internal/platform/README.md 删除旧 RabbitMQ queue runtime 归属项并明确已退状态及保留的共享基础设施边界；project-boundaries 对queue/rabbitmq当前owner行与project-wide-refactoring-plan对应queue目标加准确superseded指引；#836/#839/#870设计和code-health仅顶注精确取代本批旧KEEP，#750对应perf保留也只加本批失去最后消费者说明 | 旧历史正文/计数/失败、其它有效共享合同不重写；inventory历史原样；不为兼容旧phrase保留空包 |
| 代理/Secret直接目录说明 | shein-listing/README.md、overlays/prod-store-shards-example/README.md、overlays/prod-store-auto-shard/README.md只校准已退代理/Secret源/prod装配及已退RabbitMQ包表述；登录/Store指引引用#863当前owner，纠正数字账号仓库旧指引 | 共享Namespace/base、其它退休历史与当前独立登录/Store同步；不制造假部署入口、替代Secret或迁移步骤 |
| docs/development/rabbitmq-mainline-validation.md | 保留已DATED EVIDENCE正文与旧失败/步骤，仅顶注旧验证脚本/AMQP包已源退，无当前可运行主链 | 真实验收NOT_RUN；旧接口示例不能当当前操作说明，不引入替代验收器 |
| phase2-runtime-inventory.md | 用户已明确允许**仅删除core/logger→pkg/fileio这一CURRENT行**，配合fileio同批退出；其余零改 | 所有历史/dated行、计数、其它当前行、ceilings/准确性检查/检查逻辑保持；RabbitMQ当前无消费行，无需改provider表 |
| 全部workflow、依赖/锁、保护规则 | **零改**，原准入/必需/安全/商业/数据库检查正常运行；不以删除旧包为由改编译目标/安装依赖/禁用测试 | 原实际main/PR检查，不能把空成功当验证，不添加试用前置 |

本次设计提交只包含本文和两个提案索引。上述源码/测试/配置/其它文档最小校准均尚未应用。原 dated superpowers、phase1-summary/acceptance、test-baseline 等历史证据不扫描改写；直接当前操作引用若刷新发现漏项，仅按同根因准确目标补矩阵，不扩大成全仓文档整理。不得因任何guard失败而先改inventory/workflow/ceilings；先报告准确冲突。

## 7. 确认、交接与停止点

1. 交付本设计与真实消费者/owner证据，一次原GitHub设计检查核对§2–§6.1；同SHA Running不重复触发，不默认增加AI互验线程。只读设计、CI和PR状态均不是实施授权。
2. 用户确认§1–§7（含§6.1），并明确fileio单行inventory选择后，在#301/同一PR记录 Reuse Existing Architecture / Ready、冻结设计HEAD和唯一Writer，再开始正式源码实施。若未允许例外，fileio保持未退休；不能把泛指“确认设计”推定为撤回inventory零改。
3. 同批完成全部授权的准确删除/配置/guard/文档尾部、必要自检、提交/推送与范围内CI/review修复，不机械拆小PR。普通IMPLEMENTATION_TEST通过实现收敛，不新增全局架构轮次或验收平台。
4. 发现当前消费者/外部可观察契约、授权/租户/持久化/状态/恢复边界变化、需要新的owner/兼容/迁移、超出§5的源删除或额外inventory/workflow修改时，停止对应部分并报告具体Must/层级/最小差异。没有当前证据不能以假设扩建或把旧物理状态判不存在。
5. 最终交接准确实现HEAD、原CI、独立最终检查、保留能力和NOT_RUN；等准确HEAD合并授权，正常expected-head合并后核对真实merge SHA对应push/main CI。#301始终OPEN；不自动接真实环境操作、运行验收、关单或下一业务扩展。
