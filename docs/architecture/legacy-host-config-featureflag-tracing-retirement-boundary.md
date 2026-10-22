# #301 旧宿主配置管理、功能开关与 tracing 完整退休设计

Status: **PROPOSED / DESIGN_ONLY**。用户已同意先设计三组完整闭包，尚未确认本文或授权源码实施。当前 **Design Basis: N/A**（只读调查、本文、Architecture README/Legacy Register 提案入口及 Issue/PR 维护）；未来实施拟 **Reuse Existing Architecture / NOT_READY**，待 §7 明确确认。调查基线 main `1f388af81c12c58b126460cd0d84538c84a50063`；准确 CI/复核结果留主要 PR，不把基线或设计检查冒充未来实施/合并后结果。[接单与权限](https://github.com/qq550723504/task-processor/issues/301#issuecomment-6330967250)先于文档修改。#301 OPEN，#29 不重开；一个 Writer、一个分支、一个主要 PR，三组不机械拆批。

## 1. 用户结果、范围与权限

使用者是维护当前应用、配置、图像、采集、Store 与独立 SHEIN worker 的维护者。预期源码结果是退出失去生产调用的 ConfigManager/热更新 Source、OpenFeature 自动迁移开关、OTLP HTTP tracing 及完整专属尾部，当前应用继续使用现有配置和能力 owner。没有证据表明这些残留正在阻塞核心用户操作，不新增业务入口、替代配置/开关/追踪系统或内部试用前置。

当前仅授权设计：生产 Go、测试、YAML、workflow、inventory、依赖/锁、ceilings 和保护均零改。确认后拟同批处置 §5 的 14 个完整文件、4 个混合配置生产源、3 个样例及 §6.1 原测试/直接文档尾部；不以删除文件数替代完整消费者判断。

不执行旧 constructor、ConfigManager.Load/Reload/Watch、Source.Read/Watch、SaveJSON/SaveYAML、OpenFeature/OTLP exporter、业务 CLI、应用或 dry-run；不读有效运行配置、现有凭据、私密 binding/Profile/Secret/token、进程环境、日志或既有报告，不操作既有实例、数据、凭据、进程或服务。不启动 PG/Redis/browser/provider，不使用 Docker/Compose/kubectl/SQL。配置调查只读受 Git 追踪的源声明/样例键结构；`go list` 只加载依赖元数据。物理状态/外部人工使用 UNKNOWN，运行/用户验收 NOT_RUN。

## 2. Design Basis、Authority 与不变量

当前 **N/A** 只覆盖设计交付。未来拟 **Reuse Existing Architecture**，直接复用：

- [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)和[Issue 驱动规则](../engineering/issue-driven-development.md)。仅 EXTRACT/RETIRE，无兼容类、旧配置 reader/alias/fallback、迁移或第二事实源。
- 当前 `core/config` 的 Viper → LoadFromBytes/LoadConfigFromFile → BuildConfig/Validate 和 `app/configadapter`/各当前入口；现有环境覆盖、scoped .env 来源、认证/owner 关闭开关拒绝与 schema 准入合同不变。
- [#836 专属配置闭包](legacy-runtime-config-closure-retirement-boundary.md)、[#870 旧宿主闭包](legacy-host-runtime-closure-retirement-boundary.md)、[#874 水印闭包](legacy-watermark-config-closure-retirement-boundary.md)的有效保留边界，以及[独立 SHEIN 当前身份/持久化](shein-worker-numeric-persistence-retirement-v1.md)。没有把仍使用的配置、日志、图像或 worker 一并退休。

本批只退出闲置技术实现和专属声明，不新增/改变产品能力、公共领域合同、事实/schema owner、状态机、事务、授权/组织隔离、计费或恢复协议。原配置加载 owner 保持，不把算法搬到另一目录、不重接旧宿主。预计实施涉及约 30 个以上源/样例/测试/直接文档；这是风险提示而非拆 PR 理由。现有上述架构完整覆盖本次单纯退出，没有新增高风险边界；若刷新证明边界必须变化，按 §7 停止对应部分并升级准入。UI/IA/Figma N/A。

Must：三组完整退出，保留当前能力和有效安全/配置合同，原必要/安全检查保持，无真实状态操作和新试用前置。Should：复用原 guard、标准库/Viper 和已在使用的成熟组件。Out of Scope：新增热更新、功能开关、遥测、配置严格化、依赖清理、通用验证工具、真实环境验收。Accepted Risks：无新增，不把未访问的真实状态判为空。

**精确取代条件**：只有用户确认本设计后，才取代 #836 对本次 Manager/merge/DefaultsApplier 与 flags/tracing 的暂留要求、#870 对这个 observability 包存在性 guard 的保留要求，以及 Phase 2 对本次三个技术包和旧 HTTP 纵向接线的要求。`project-boundaries.md` §3.6 和 platform README 的当前列举按 §6.1 校准；原 Phase 2 计划、旧冻结设计、inventory 的 dated 段/计数保留为历史，不构成复活或新建替代能力的依据。其它当前架构和 SDK 边界不变。

## 3. contract → implementation → injection → consumer

在基线全仓 `go list` 的 Imports/TestImports/XTestImports、受追踪源码引用和实际入口依赖上核对：

| 对象 | 实际接线/消费者 | 判断与保留边界 |
| --- | --- | --- |
| core/config.ConfigManager | NewConfigManager 只有 config_env_test 与 validator_workbench_test 两个调用；生产 managerImpl 只在自身工厂构造，无当前 app/cmd 注册 | 闲置 Load/Reload/Watch/Validate/GetCurrent 包装，同批 RETIRE；不删同名但有效的 Amazon browser.ConfigManager |
| DefaultsApplier / MergeConfigs / MergeMaps | 前者只有声明；MergeMaps 仅被自身递归和 MergeConfigs 调用，后者无外部调用 | 没有当前消费者，不为理论复用再抽取；NewDefaultConfig、Viper 默认/合并与当前输入规范化仍保留 |
| platform/config | 唯一包外生产 importer 是 core/config/manager.go；Source/file/memory/loader/codec/path 其它调用仅包内自测 | Manager 与完整包同退；当前文件/bytes 加载并未通过这个包 |
| 三个当前入口 | current-application、shein-login-worker、image-agent-temporal-worker 的实际 go list -deps 均包含 platform/config，不包含 featureflag/observability | 是闲置 Manager 的 import 造成编译依赖，不证明当前在 watch/read/write；实施后三个入口应均无三包依赖 |
| platform/featureflag | OpenFeature isolated API + InMemoryProvider；无包外生产 importer/构造或 Bool 消费者 | Config.FeatureFlags 仍在 default/build/env/样例/自测，但无现役执行 owner；不把旧开关重新接到当前 schema 初始化 |
| platform/observability | 独立 TraceRuntime/NewTraceRuntime/WrapHTTPHandler/Shutdown；无包外生产 importer/构造或 handler 接线 | Config.Observability.Tracing 仍被解析和自测，但未贯通当前 HTTP/worker；现役 logging/health/metrics/其它 SDK 语义保留 |
| 当前配置入口 | LoadConfig/LoadConfigFromFile/LoadFromBytes、BuildConfig、Validate 与当前 manifest/worker 装配均有真实调用 | 保留原算法/优先级/错误行为；不因 Manager 缺席误退整个 core/config |

旧字段只有 `core/config` 类型、default/build/env、自测和三个受追踪 YAML 使用。唯一旧业务标识为 `featureFlags.flags.product-listing-runtime-auto-migrate`；没有以其当前值授权 schema 或迁移的生产调用。不能从样例/默认 true 推导发生过建表，也不能把删开关等同修改当前 schema owner。旧测试、导出声明、包名和 dated 计划都不是当前产品需求的替代证据。

## 4. 配置、持久化与副作用 owner

| 边界 | 闲置源码中的行为/owner | 删除及保留条件 |
| --- | --- | --- |
| Manager/默认/合并 | mutex 下保存 Config/Source；解析、校验、回调/Reload；反射默认值和 JSON/map 合并修改进程内传入对象 | 同退，无 durable 业务 owner；当前 Viper/BuildConfig/Validate 原样，不添加替代 Manager/兼容接口 |
| Source/codec/path | 读配置文件、fsnotify watcher/goroutine；SaveJSON/YAML 可 MkdirAll/WriteFile；路径辅助读取 executable 元数据 | 完整旧 API 退出，不运行、不删除/改写任何真实文件或停止 watcher；当前有效配置文件加载与路径选择继续原 owner |
| Feature flag | 每实例 isolated OpenFeature API、本地 boolean provider/map 与 shutdown | 无已核实当前持久化/外部 flag 服务 owner；删 wrapper/专属字段，不搬 flags、不新增 provider/开关权威 |
| Tracing | 启用时构造 OTLP gRPC exporter、私有 tracer provider、batch export/flush/shutdown 和 HTTP W3C trace context | 删除闲置封装，不连接 endpoint、flush、改全局 provider 或操作 collector；现有日志/指标/health、其它当前 telemetry 和所有 SDK/锁文件保持 |
| 当前业务状态 | Organization/身份/计费、schema 初始化、采集/资产/Store、SHEIN 登录与可选 execution fence 有各自当前 owner | 无表、迁移、ID 映射、outbox、恢复或清理动作；UNKNOWN 不自动接管，全部保持 |

本批不建立事务、幂等、retry/UNKNOWN/recovery 或新的资源所有权协议。外部是否仍有旧配置、文件、watcher、collector 或旧二进制为 UNKNOWN，源码退休不表示物理状态已退出，也不授权探测。若发现具体当前保留消费者或外部可观察义务，停止对应实施并报告，不自行加 adapter/fallback。

## 5. RETIRE / KEEP / EXTRACT 完整矩阵

### 5.1 同批 RETIRE 的 14 个完整工件

10 个生产 Go 共 751 行，另 3 个专属测试和 1 个包 README；按准确文件而非混合祖先目录操作：

```text
internal/core/config/manager.go
internal/core/config/defaults_applier.go
internal/core/config/merge.go
internal/platform/config/codec.go
internal/platform/config/loader.go
internal/platform/config/path.go
internal/platform/config/source.go
internal/platform/config/source_test.go
internal/platform/config/README.md
internal/platform/featureflag/runtime.go
internal/platform/featureflag/runtime_test.go
internal/platform/observability/config.go
internal/platform/observability/tracing.go
internal/platform/observability/tracing_test.go
```

### 5.2 混合配置源的精确删除

| 对象 | 仅删除 |
| --- | --- |
| core/config/config.go | Config.FeatureFlags、Config.Observability；FeatureFlagsConfig、ObservabilityConfig、TracingConfig 三个专属类型；下列五项 knownEnvBindings |
| core/config/loader.go | NewDefaultConfig 的 FeatureFlags/Observability 两个 initializer；函数及其余默认保持 |
| core/config/defaults.go | 对上述五项 key 的 SetDefault；其余当前默认保持 |
| core/config/loader_builder.go | BuildConfig 的两个 initializer 和 buildFeatureFlagsConfig；其余 build/规则/owner 校验保持 |
| config/config-dev.yaml、config-test.yaml、config-prod.yaml | 三者均为普通根 YAML；仅 featureFlags 和 observability 两个子树及专属说明。后续 logging 和所有其它键/注释保持，不处理有效运行配置 |

五项 key/env 的完整边界：

```text
featureFlags.flags.product-listing-runtime-auto-migrate -> TASK_PROCESSOR_API_RUNTIME_AUTOMIGRATE
observability.tracing.enabled -> TASK_PROCESSOR_OBSERVABILITY_TRACING_ENABLED
observability.tracing.serviceName -> TASK_PROCESSOR_OBSERVABILITY_TRACING_SERVICE_NAME
observability.tracing.endpoint -> TASK_PROCESSOR_OBSERVABILITY_TRACING_ENDPOINT
observability.tracing.insecure -> TASK_PROCESSOR_OBSERVABILITY_TRACING_INSECURE
```

不新增旧配置 reader、别名、忽略告警包装、全局 unknown-key 拒绝或第二默认值来源；只让原入口消费收窄后的同一 Config。

### 5.3 KEEP 与无新 EXTRACT

- 当前 Viper/LoadFromBytes/LoadConfigFromFile/WithoutValidation/BuildConfig/NewDefaultConfig/Validate、scoped .env、环境优先级、认证/owner 关闭拒绝、当前 schema 准入与显式初始化保护原样；Manager 专属 nil/Watch 合同不被提升为新全局配置需求。
- 当前 logging/health/metrics、HTTP/Temporal/workerpool、数据库/Redis、AI/prompt/预算/资产、图像与百炼参数、平台 API/有效规则/共享合同（含两种 SHEIN 属性与 PriceManager/PriceCalculator）、采集、Store 与独立 SHEIN 登录/同步及 UNKNOWN/fence 保持。
- core/logger、appenv、downloader、imagex、jsonx、goroutine、recovery、timeout、runtimepath 及其它仍有消费者的共享层；Amazon browser.ConfigManager 和其它同名 Source/Config/Tracing/Flag 语义不按名字删除。
- 全部成熟 SDK/依赖/锁保持，包括 Viper/fsnotify/OpenFeature/OTel；不做 tidy/升级、替代实现或新抽取库。

```text
Legacy decision: RETIRE（提案，未实施）
Reusable behavior: 无新 EXTRACT；有效配置/校验/环境和现役能力已有当前 owner
Current owner: 当前 core/config + app/configadapter/各正式入口，原 logging/业务/schema/worker owner
Cutover/deletion condition: 用户确认 §1–§7（含 §6.1）、刷新无保留消费者后，准确 14 工件/混合声明/专属尾部一起退出；无物理状态处置
```

## 6. 必要验证与证据边界

设计阶段只做源码/键结构/引用/实际依赖元数据与原适用文档检查；不为纯文档制造 RED，不运行旧包测试/constructor、watcher 或 exporter。原基线、设计 CI/复核不能冒充未来实现、合并后 main CI、运行或用户验收。

确认后复用原 retirement/AST/import decoder/YAML/go list 设施：准确缺席断言先 RED，再退出至 GREEN；读取/解析失败拒绝，不 missing-skip。保留 Browser/OpenAI/Redis 等当前 env/file/bytes 优先级、Workbench/ZITADEL 失败和有效 schema/权限测试；现有 t.TempDir/合成 fixture 即可，不新增 PG/Redis/collector、故障矩阵、runner 或验证工具。稳定候选走原 CI，未变化证据明确复用，仅核对新增组合风险。

### 6.1 原检查、当前文档、inventory 与 workflow 的处置

| 对象 | 确认后同批最小校准 | 必须保留 |
| --- | --- | --- |
| 三个包内专属测试 | 随完整旧 owner 退；不运行 fsnotify/OTLP 测试来替旧 owner 证明存在必要 | 其它当前配置/日志/隔离/权限/平台合同测试 |
| core/config/config_env_test.go | 退 3 个 TracingConfig 专属用例、6 个 FeatureFlags 专属用例；退 TestConfigManagerLoad_UsesSameEnvironmentOverrideBehavior 和唯一 stubConfigSource/Read/Watch/Name 及失去最后消费者的 context/logrus imports | 原 LoadFromBytes_AppliesEnvironmentOverrides / AppliesViperManagedEnvironmentOverrides 已覆盖当前覆盖规则；其它 Primary/Legacy aliases、商业 owner、Browser/OpenAI/Redis/SDS、身份、scoped .env、失败拒绝与字段装配全部保持 |
| core/config/validator_workbench_test.go | 退 TestConfigManagerValidateIncludesWorkbenchZitadelConfig 和专属 logrus import | 原 TestConfigValidateWithErrorIncludesWorkbenchZitadelConfig 与其它直接 Workbench/ZITADEL 缺项/URL/client-secret 拒绝不变；不为旧 Manager 的 nil 包装新建现役需求 |
| core/config/config_test.go | 在原退休配置检查加 FeatureFlags/Observability 字段及专属 default/env key 缺席 | 其余当前 defaults/build/校验与已退休输入拒绝不变 |
| tests/listingkit_backend_retirement_test.go | 用现有设施加一个三组合完整缺席/禁入 guard：14准确工件、三旧平台根与 core/config 专属声明，tracked Go decoded imports、三个 YAML 根和三个当前命令真实依赖退出；精确到 owner，避免同名误杀 | 其它所有退休 guard、当前 SHEIN/平台/共享保留检查及读取/解析失败政策；不建扫描器或 dummy package |
| tests/target_architecture_phase2_test.go | TestPlatformObservabilityDoesNotImportCoreOrApp 仅把此旧 owner 的必须存在/文件扫描目标升级为缺席/禁入保护，可复用原设施，不以错误时 return/skip 求绿 | 其余 Phase 2、跨平台、SDK ownership、parser/alias/dot import 与完整边界检查原样 |
| Architecture README / Legacy Register / 本文 | 设计只加 PROPOSED 入口；确认后将本矩阵转为确认/源退并给准确依据，新 guard 只有实施后才登记覆盖 | 其它 owner、引用、历史计数/失败不改，不把全仓/core/config 宣称清零 |
| internal/platform/README.md / project-boundaries.md §3.6 | 当前仍有三类 current-owner 列举；实施时只退 config/featureflags/observability 对应项并加窄注：当前配置继续原 core/config 入口，无替代 runtime。历史 Phase 2 段原样，由确认设计精确 supersede，不新建 plural featureflags alias | 所有其它 runtime/integration owner、SDK/领域边界与真实当前能力；不重写旧 Phase 2 计划或 #836/#870 的历史正文 |
| phase2-runtime-inventory.md | **完整文件零改**。本次没有三旧平台包的 CURRENT consumer 行；core/config 其它 logging/OpenAI 依赖仍有效。旧通用配置/observability 映射属 dated，确认注记仅取代准确闭包含义 | CURRENT、历史/计数/ceilings、准确性检查逻辑全部原样；test-baseline.txt 等 dated 证据也不改 |
| 全部 workflow / 依赖与锁 / 保护 | **零改**；没有本闭包的独立 workflow 目标需要替换，原 CI 正常执行 | 触发/分类/权限/并发/必需与安全检查/Required CI Gate、所有 ceilings 和 SDK 约束不变 |

若实际检查与冻结矩阵冲突，先给准确失败与最小处置；不先改 inventory/workflow/ceilings 求绿，不据此扩建配置/观测系统。当前只提交本文和两个提案索引，上述源码/测试/YAML/当前 owner 文档校准尚未应用。

## 7. 设计确认、交接与停止点

1. 以准确设计 HEAD 交付真实消费者/owner 与 §1–§6.1 矩阵，一次原 GitHub 设计检查只审本闭包；同 SHA Running 不重复触发，不默认新增 AI 互验线程。N/A 只覆盖本设计交付，不授权生产实施。
2. 用户明确确认 §1–§7（含 §6.1）后，在 #301/同一主要 PR 记录 **Reuse Existing Architecture / Ready**、冻结设计 HEAD 与唯一 Writer，再修改正式源码；本次确认前保持 NOT_READY。普通已有边界不重复全局架构审核。
3. 三组及专属配置/原检查/当前文档尾部同批实施，不按包、SDK、文件数或证据机械拆 PR；必要开发自检、提交/推送和范围内 CI/review 修复留同一 PR。设计复核不代替完整实现后独立最终检查；修复只复核真实增量。
4. 发现真实保留消费者、当前可观察义务、需要新 owner/兼容/迁移、权限/租户/事实/状态/恢复/副作用变化，或额外源码/inventory/workflow 删除时，停止对应部分，报告当前 Must、阻塞层级与最小差异。只有真实 BLOCKER 才重开冻结设计，不以推测扩架构或试用前置。
5. 完整交接准确实现 HEAD、原 CI、独立最终检查、保留能力与 NOT_RUN，等待用户准确 HEAD 合并授权。正常 expected-head 合并后核对真实 merge SHA 的 push/main 整run；#301始终 OPEN，部署/真实环境操作、运行/用户验收、关单和下一批业务扩展均不自动开始。
