# 旧数字任务与刊登专属配置：完整退休闭包设计

Refs [#29](https://github.com/qq550723504/task-processor/issues/29) / [#301](https://github.com/qq550723504/task-processor/issues/301)。

> Status: **IMPLEMENTATION_READY；用户已确认冻结设计 `209b40f54` 并明确授权按§5/§6/§6.1/§7在 #836 整批实施；源码退休已完成，待最终独立检查/合并授权。** Design Basis: **Independent Architecture**。调查基线 main `cb211d5072b6268fc846be346dacc9fa9e19f981`（2026-10-19），包含 #833/#834/#835；主要 PR 记录准确 HEAD/CI/review。一个 Writer、一个完整配置闭包，不按字段、配置组或文件数拆 PR。设计阶段只新增本文和索引；独立设计检查完成、用户确认及 Issue Ready 记录读回之后，才实施以下矩阵。准确实施 HEAD/测试/CI/review 与权限记录在 #836，不把源码完成记作运行或用户验收。

## 1. Product Outcome 与范围

使用者：维护当前应用、采集、图像、Store 观察及独立 SHEIN 登录/Store 同步的操作者。结果：已退休的数字任务、控制面和刊登编排不再保有可加载、默认启用、可校验的空壳配置合同；当前程序不再要求无消费者的 worker/任务配置。不是创建新配置中心、迁移现有配置或重建调度器，也不成为当前内部试用的新前置。

范围以同一根因闭包覆盖：Processor/旧 Worker/ListingControlPlane、旧全局 RabbitMQ 任务配置、SHEIN/TEMU 旧刊登/核价/同步/监控配置、旧 debug/generation usage 开关、两项孤立 schema 转换器与旧平台注册表，以及它们的默认值、加载、env 绑定、校验、示例和专属测试。只删类型不删默认值/加载/测试不是完成。

当前真实 workerpool/queue 基础层、Redis/browser、Amazon/1688 采集、平台 API/规则、当前身份/授权/资金/AI 计量/模型路由、图像、Store/API/schema/观察、独立 SHEIN 登录/查重/Store 同步及此前全部保留能力必须保持。不按 `core`、`listingkit`、`worker`、`RabbitMQ` 名称整包退休。

## 2. Design Basis、Authority 与 Threat Model

**Independent Architecture**：本批改变共享应用配置 schema 和启动校验的明确边界，跨当前命令/配置装配/仓库部署示例，预计超过30个范围文件；需要独立设计准入，不能把它当纯字段清理或拆 PR 规避提示线。无新领域能力、事实 owner、表、权限策略、状态机、事务、恢复或副作用。

Authority：[Greenfield 决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[#726 控制面边界](legacy-listing-control-plane-retirement-boundary.md)及用户本轮完整闭包设计决定。#726 §5 明确保留未消费 ListingControlPlane 配置债务，本设计仅在§5准确对象上接续该债务。#833 已合并及 matching main CI 是前批历史，不是本批实施证据。UI/IA/Figma N/A。

- Must：保留现有采集/workerpool、当前配置与安全/资金/平台合同；一批删完旧配置的完整输入/输出尾部；无真实操作或新试用前置。
- Should：复用现有 Viper、类型/默认装配与原测试；保留未变化有效证据，不新建扫描/验收工具或配置框架。
- Out of Scope：通用配置重构/搬包、Viper 升级、配置迁移/兼容/fallback/双读双写/第二事实源，新运行/发布/用户验收，旧部署目录整体删除，当前 #692 Store schema 工作，任何真实配置/凭据/实例/数据库/provider 操作。
- Accepted Risks：无新增。物理配置、部署进程/队列/数据与凭据状态 UNKNOWN，本批不取证访问；现有运行实例不受源码提案执行。
- Threat Model：只退类型留下输入/默认/校验，混淆两个 WorkerPool/Retry/RabbitMQ owner，误删当前身份/计费/登录配置或误把空壳 canary 当计费保障；不扩大为 IAM/资源/治理或验收平台。

## 3. 实际消费者与配置装配链

静态核对覆盖根模块、cmd、hack/debug、tools 的 tracked Go 及仓库示例。配置文件/测试能构造对象不等于当前业务消费者。所列“无消费者”是该准确基线的源码事实，不是运行审计。

| 对象 | 准确消费者证据 | 设计决定 |
| --- | --- | --- |
| core/config.ProcessorConfig、ListingControlPlaneConfig | 只剩 Config、NewDefaultConfig、BuildConfig/defaults/env 和旧协议测试；原 processor/control-plane 根已退 | 连同完整装配/输入/测试退出，不保留开关壳 |
| core/config.WorkerConfig | 包外唯一生产声明为 app/configadapter.WorkerPool，后者只有自身测试调用 | 两者同退；不删除当前 workerpool |
| 当前采集池 | crawler/amazon/crawler_service.go 直接 DefaultPoolConfig/NewPoolWithConfig，显式并发5、buffer1000；BaseService 持有现 WorkerPool | 生产/测试零改，既有结果隔离/Redis/取消保持；无新转换器 |
| Config.RabbitMQ 与 task/crawler/hybrid、亲和/分桶/恢复/上报配置 | 当前业务没有读取 Config.RabbitMQ；包外只剩无调用的 configadapter.LoadMonitor；加载和 validator 自循环不是运行 owner | 整个旧应用 schema 及转换器退出；platform/queue/rabbitmq 和独立 broker 部署保持 |
| core/config.RetryConfig / DefaultRetryConfig | amazon/api/retry.go、core/errors/helpers.go 实际使用 | 原 owner/声明/默认值原样保留，不能因它位于 type_rabbitmq.go 整删 |
| PlatformConfig 旧任务/调度/核价/同步/监控/配置路径字段 | 只有旧 schema/defaults/BuildPlatformConfig/validator/registry 与测试消费，无当前编排入口 | 完整退休，包含旧 sync→productSync normalize fallback |
| PlatformConfig 的 FetchMode/CookieRedis/LoginService/HTTPClient | platformbase fetcher 解析、SHEIN client/bootstrap/独立 worker、平台传输及 SDS 共用登录类型仍消费 | 完整保留四字段及其原值/加载；SDS/1688 类型和能力原样 |
| 旧 PlatformConfigRegistry/BasePlatformProvider | Register/Get/List/NewBasePlatformProvider 等只有包内自身调用，没有当前注册/查询消费者 | 整文件退出，不补注册器来合理化旧 owner |
| Debug.SavePublishJSON、ListingKit 的 SheinSubmitDebugDumpDir/GenerationUsageLedgerEnabled/TenantIDs | 只剩 schema/defaults/loader/env 与旧断言；无当前请求/运行调用 | 旧 debug 与三个 ListingKit 字段退出；当前 Logging/AI/资金 owner 不变 |
| 当前配置入口 | 文件/bytes→newViper(既有 env/defaults)→loadWithViper 或 WithoutValidation→BuildConfig；ConfigManager.Load/Reload 复用这条链 | 两条入口都收口，不重写 loader；NewDefaultConfig 与 Viper BuildConfig 两套值装配同时清理 |

**发现的架构问题**：旧任务编排退出后，其 schema/defaults 和 Worker/平台/RabbitMQ 校验仍留在当前应用配置中；测试又反向要求默认配置继续存在。旧 generation usage canary 也只检查两条 ConfigMap 值，已经没有生产接线，不能代表当前 AI 计量或 paid plan 保护。设计处理完整空壳，不删除当前资金/AI 的真实检查。

## 4. 配置 owner、持久化与安全边界

- 应用配置 schema、known env、NewDefaultConfig/BuildConfig/ValidateConfig 的 owner 仍为 `internal/core/config`；各当前 runtime 沿原工厂注入。旧组退出后无继任 owner，无替代 struct、no-op setter、字段映射或第二配置源。
- 通用 source/path/codec 为 `internal/platform/config`；ConfigManager、MergeConfigs/MergeMaps、DefaultsApplier 的通用算法保留原样，不因结构字段减少重写反射/JSON/热加载/来源选择。只让它们消费收窄后的同一 Config。
- 当前 workerpool 属于 platform/workerpool + crawler/shared/amazon；通用 RabbitMQ 属于 platform/queue/rabbitmq，自己的 config/consumer/load monitor 与 snake_case fixture 不属于旧全局 Config.RabbitMQ。保留这两个包、tests/test_config.yaml 及 middleware-single/rabbitmq.yaml，不操作 broker/队列/Redis。
- 身份/tenant/owner/admin、RolePolicyDatabase、完整 ListingKit.Zitadel（含拒绝 obsolete username allowlist 的探测/校验）、Workbench、ImageUpload、ImageAgent、商业两个 DB owner、WalletTopUp、AI/模型路由、FeatureFlags、logging/observability 等保持。包括不暴露身份/owner 关闭开关的现有断言；不以清理旧名触碰 #692/schema 准入。
- ListingKit 的旧 generationUsageLedger 配置没有当前计费 owner；删除它和失效 canary 不删除 Ledger、money、资源/Token池、usage settlement/entitlement/provider mutation 等能力或其原检查，不改变任何付费/计量开关。
- 本批不创建/迁移/删除表列行，不访问私密 manifest/.env、Secret 值、数据库、IAM、运行态或 provider。仓库 Secret.example/ExternalSecret 只删除明确失去 reader 的旧字段声明，保留对象名/namespace及其它所有映射；不应用或同步清单，不删除真实 secret/远端属性。
- 不增加新状态、事务/补偿/恢复、授权或外部副作用。所有幂等、UNKNOWN、取消、deadline、撤权和隔离仍由原当前 owner 保证；不因无关理论风险增设 gate。

## 5. 完整删除 / 保留矩阵

### 5.1 九个整退生产文件

```text
internal/core/config/type_listing_control_plane.go
internal/core/config/type_processor.go
internal/core/config/type_worker.go
internal/core/config/validator_worker.go
internal/core/config/loader_rabbitmq.go
internal/core/config/validator_rabbitmq.go
internal/core/config/platform_config_registry.go
internal/app/configadapter/workerpool.go
internal/app/configadapter/queue.go
```

### 5.2 十二个 mixed 生产文件，准确声明与唯一允许变化

| 文件 | 退休对象 / 修改 | 保留边界 |
| --- | --- | --- |
| core/config/config.go | Config.Processor/Worker/RabbitMQ/Debug/ListingControlPlane；DebugConfig；knownEnvBindings 中 rabbitmq.*、listingControlPlane.*、worker.*、platforms.temu.enabled/platforms.shein.enabled 与 ListingKit 三旧字段的 primary/deprecated 绑定 | newViper/共享 env 与 .env 来源优先级、其它绑定、rejectRetiredProductRuntimeConfig 调用及文件/bytes验证入口原样；不读真实 env |
| core/config/type_listingkit.go | 只退 SheinSubmitDebugDumpDir、GenerationUsageLedgerEnabled、GenerationUsageLedgerTenantIDs | RolePolicyDatabase、PlatformAdminUsers/Roles、ImageUpload、Zitadel 全部完整保留 |
| core/config/type_rabbitmq.go | 除 RetryConfig/DefaultRetryConfig 外所有旧类型、NodeRole三常量和 NodeConfig 的五个角色/亲和方法 | RetryConfig/DefaultRetryConfig 与 time import 原样；无新生产抽取或重命名 |
| core/config/type_platform.go | PlatformConfig 的 Enabled、SchedulerEnabled、AutoPricing、ListingPricing、ProductSync、InventorySync、ActivityRegistration、SyncProduct、Monitor、ConfigPaths 十字段；AutoPricingConfig/ListingPricingConfig/ScheduledTaskConfig/SyncProductConfig/MonitorConfig/PlatformConfigPaths 六类型 | PlatformConfig 四保留字段完整；PlatformsConfig、EffectiveSheinCookieRedis/EffectiveSDSAuthRedis、SDS、LoginService/SDS别名、Alibaba1688 类型全保留 |
| core/config/loader.go | NewDefaultConfig 中五根对象与两平台的十旧字段；LoadConfigWithFallback 的旧 RabbitMQ 日志块 | 其它默认值、LoadFromBytes、路径检查/其它日志及 Amazon 专用验证入口不变 |
| core/config/loader_builder.go | BuildConfig 中五根对象、ListingKit三字段及 cfg.RabbitMQ=BuildRabbitMQConfig | 所有其它组和数据库/身份/feature flag/HTTP/OpenAI/日志/SMS等 helper 原样；不修其它历史缺陷 |
| core/config/loader_platform.go | BuildPlatformConfig 仅移十旧字段及 normalizePlatformConfig 调用；整退 normalizePlatformConfig | 原四字段加载、BuildLoginServiceConfig、BuildSDSAuthBootstrapConfig 不变；不创建旧 sync fallback |
| core/config/defaults.go | setDefaults 的旧键和两调用；整退 setListingControlPlaneDefaults/setRabbitMQDefaults 与仅其使用的 firstNonEmptyString/firstPositiveInt/firstPositiveDuration；setTemuDefaults/setSheinDefaults 只去旧十字段路径 | 所有其它 defaults/helpers 原值保持，当前登录/Redis/HTTP/图像/采集等完整保留 |
| core/config/loader_helpers.go | 退仅旧组消费的 getDuration/getInt64Slice/getIntSlice/getIntFromMap，清 unused import | getStringSlice/readLineList/uniqueStringList/getStringFromMap 全部原样，仍服务当前身份/proxy/logging |
| core/config/validator.go | ValidateConfig 构造实参去 Processor/Worker/RabbitMQ；moduleFromField/rank 仅退这三旧诊断模块 | 其它调用、分组排序/格式化、错误聚合保留；当前身份/Workbench校验不减 |
| core/config/validator_validator.go | Validator 三字段/构造参数/赋值、ValidateWorkerConfig与ValidateRabbitMQConfig调用退出 | NewValidator 仅机械缩参；OpenAI/Browser/Amazon/Platforms验证和错误处理原样，包外无构造调用 |
| core/config/validator_platform.go | 退 ValidatePlatformsConfig 中两项旧平台任务调用及整个 ValidatePlatformConfig，清 unused imports | Alibaba1688.Enabled/Timeout/PoolSize 原验证逐字保留；SHEIN登录和平台API各自实际校验不动 |

### 5.3 配置输入与仓库示例矩阵

精准键集合：根 `processor`、`worker`、`listingControlPlane`、旧根 `rabbitmq`；`debug.save_publish_json`；ListingKit 上述三字段；`platforms.temu/shein` 的十旧字段（嵌套组整体）。包括 knownEnvBindings 中这些键的既有主名/旧别名及仓库中相应 env 表达；不匹配其它 owner 的同名字符串。

| 准确资源 | 允许删除 | 必须保留 |
| --- | --- | --- |
| config/config-dev.yaml、config-prod.yaml、config-test.yaml | 上述已出现的精确键/专属注释；无其它业务键变化 | logging/tracing、browser/Amazon/1688、完整登录/Redis/DB/身份/图像/计费/其余字段 |
| deployments/kubernetes/amazon-crawler-api/base/configmap-lite.yaml、configmap-heavy.yaml | data内嵌 YAML 的旧 rabbitmq 节点，及 platforms.temu.enabled/schedulerEnabled、platforms.shein.enabled/schedulerEnabled 四旧键；若两平台节点因此变空则移除空节点 | platforms.alibaba1688 全部原字段、当前采集的其余配置、metadata/data键及Deployment引用；不删其它平台的 enabled |
| deployments/kubernetes/shein-listing/base/configmap.yaml、temu-listing/base/configmap.yaml | data内嵌 YAML 的旧根/平台键 | 有效采集/HTTP/登录/Redis等；不整删ConfigMap/overlay或目录 |
| deployments/kubernetes/listingkit-workbench/base/configmap.yaml | 仅两个 TASK_PROCESSOR_LISTINGKIT_GENERATION_USAGE_LEDGER_* 和紧邻专属 PAY-042 legacy tenant1038 注释 | 其它全部值，尤其ImageAgent admission、身份/登录/browser/图像持久化与canonical owner |
| 下方七个 Secret.example/ExternalSecret | 仅 TASK_PROCESSOR_RABBITMQ_URL 的 stringData项或对应 secretKey/remoteRef完整条目 | metadata/secretStoreRef/其它全部字段，不删真实密钥或外部存储 |
| shein-listing/overlays/prod-store-shards-example/README.md | 只校准旧 RabbitMQ env/分片消费配置说明为源码已退，不保留伪可运行命令 | 独立登录/保留配置说明与历史证据，不运行操作 |

七个准确 secret 资源（均在 deployments/kubernetes 下）：listingkit-workbench/base/secret.example.yaml；shein-listing/base/secret.example.yaml；shein-listing/overlays/prod/secret.example.yaml；shein-listing/overlays/prod-store-shards-example/secret.example.yaml；shein-listing/overlays/prod/external-secret.yaml；temu-listing/base/secret.example.yaml；temu-listing/overlays/prod/external-secret.yaml。

**不新增旧键兼容或一套拒绝框架**：Viper 原 unknown-key/AutomaticEnv 策略保持，退掉的是 typed schema、显式绑定、默认、读取、行为与有效配置承诺；未知输入被原解析器忽略不表示旧开关仍受支持。现有身份 obsolete-username / ProductImage/ProductEnrich 拒绝规则原样保留，不扩大到新迁移 gate，不要求清理私密配置后才能试用。发现具体当前外部可观察配置义务时按§7停下报告，不能擅自加 fallback。

### 5.4 明确保持零改

全部当前 domain/runtime/采集/workerpool/平台queue基础层、core/errors、amazon/api、core/config 的 common_types/merge/defaults_applier/manager/retired_product_runtime/其它schema和validator、configadapter/database/redis、listingadmin、listingruntime、publishing、shein/content/generationtopics/authorizedbrand、独立sheinlogin/managed/bootstrap/runtime及其有效测试保持。真实 broker部署、tests/test_config.yaml（独立通用queue fixture）、config/config-amazon-crawler-api.yaml保持。**两套及全部 workflow、phase2-runtime-inventory 当前/历史计数、依赖/.golangci/ceilings/保护零改。** 无 core/logger/OpenAI CURRENT consumer 行退出。

Legacy decision: **RETIRE**。Reusable behavior：当前 workerpool/queue、Retry/HTTP/Cost纯规则、通用配置算法和各当前owner行为原地保留；无新增生产 EXTRACT。Current owner：原应用配置与各当前能力；旧任务/调度/usage配置无继任。Cutover/deletion condition：准确设计独立结论+用户确认+Issue IMPLEMENTATION_READY/Ready及实施授权后，一批完成§5/§6.1、原守卫RED→GREEN、原CI和最终独立检查；不运行或迁移任何环境。

## 6. 同一主要 PR 的实施与必要验证

先扩展原 scoped retirement guard，捕获旧 schema/入口/示例仍存在的 RED；再一次完成闭包删除和原有效测试转接。只做必要局部测试/编译，完整原 CI 留稳定候选；不加新测试框架、配置检查平台、验收runner或常驻Reviewer。

### 6.1 原测试与必需检查准确处置

| 原位置 | 退出/转接 | 必须保留 |
| --- | --- | --- |
| core/config/processor_config_test.go、platform_legacy_sync_test.go；configadapter/workerpool_test.go、queue_test.go | 四文件整退：只证明旧配置/转换/旧sync兼容协议 | 真实 platform workerpool/queue 和 crawler 原测试全部不动 |
| core/config/config_test.go | 退七个独占测试：TestRabbitMQConfigDefaults、TestListingControlPlaneConfigDefaults、TestListingKitGenerationUsageLedgerDefaultsDisabled、TestConfigBuildIncludesDebugConfig、TestLoadFromBytesIncludesListingControlPlaneConfig、TestLoadFromBytesAppliesListingControlPlaneEnvOverrides、TestLoadFromBytesAppliesRabbitMQDefaultsWhenSectionMissing；IncludesListingKit只退三旧字段fixture/assert；ConfigValidation只去旧组fixture | 全部Browser/Amazon/1688/fetchMode、有效ListingKit ImageUpload、当前有效/无效配置断言保持，不删整文件 |
| core/config/config_env_test.go | 仅整退两项 watchdog env测试；Primary/Legacy绑定测试只退RabbitMQ片段；ListingKit env/file装配只退三旧字段；三个LoadFromBytes/ConfigManager混合测试去RabbitMQ/Worker/平台Enabled fixture与断言 | 用原Browser/OpenAI/Redis字段保留覆盖/优先级/Manager一致性；全部商业DB owner、tracing/FeatureFlags、SDS/SHEIN登录/Redis、身份/owner关闭开关拒绝、Zitadel/SMS/.env来源保护保持 |
| core/config/validator_dependencies_test.go | 退 TestValidatePlatformConfig_SchedulerNeedsTask、TestValidateRabbitMQConfig_EnabledRequiresCoreFields；CatchesDependencyErrors只去Worker/RabbitMQ/旧平台fixture | OpenAI client完整性和Amazon凭据拒绝、聚合拒绝当前无效配置保持 |
| core/config/validator_format_test.go | GroupsByModule的旧RabbitMQ样例换原Browser错误；UsesGroupedFormat用当前无效OpenAI/Browser字段替换Worker | 分组顺序、hint、General/plain error、错误返回和非Management断言保持 |
| tests/commercial_readiness_workflow_test.go | **只替换** TestListingKitGenerationUsageLedgerCanaryIsRestrictedToBillingTenant1038 为精确旧两env键及旧canary注释缺席检查（可准确更名）；不再要求无消费者配置强制开启 | 其余所有schema、credential/API分域、readiness、当前计费/原发布与安全检查完整不变；不重建tenant1038映射 |
| tests/listingkit_backend_retirement_test.go | 增强§5整退文件/声明/字段/绑定/示例缺席及保留owner断言；原auto-shard检查的四路径表仅去已整退loader_rabbitmq/validator_rabbitmq两项，另以准确文件absence守卫承接；对§5.3全部列出YAML/内嵌YAML逐项断言退休键集合缺席，明确覆盖两个Amazon ConfigMap的四平台旧键，同时对照Alibaba1688与所有当前叶值保持 | 其余auto-shard禁复活、所有旧闭包absence、当前命令/worker/Store/字段/平台/安全断言不变；不skip/泛化ENOENT |

新增测试只允许在原 core/config 混合测试和原 retirement guard 内补有意义断言：新schema/known defaults/bindings无旧组，当前有效配置无需Worker/旧队列就可验证；两条load入口/原Manager与同名真实WorkerPool/Retry/平台配置的保留行为明确。复用原 viper、bytes、t.TempDir 和stub；不新增验证工具/生产配置探测，不在设计阶段运行这些实施检查。

必要开发验证：core/config + app/configadapter 与 platform/config、platform/workerpool、platform/queue/rabbitmq、crawler/shared/amazon、platformbase、amazon/api/core/errors、SHEIN client/login/bootstrap/managed/worker、当前ImageAgent配置与相关原检查；当前命令依赖guard与 current-application / shein-login-worker / image-agent-temporal-worker / product-listing-api-schema-migrate 仅编译。保留受影响配置源当前叶值的对照，使用仓库示例/内存fixture，**不读取真实配置/密钥**。

未来文档只窄校准本文、README、legacy-register/module-target-mapping 与 #726 中准确旧配置保留表述；历史矩阵/计数/回执不重写，更多静态runbook只有明确误导到本轮已退键时窄说明，不扩成运维改造。设计CI/独立复核、未来实现CI、运行/provider/用户/生产验收分别记录。

## 7. 准入与停止点

1. 当前只有设计/索引；准确HEAD适用独立设计检查后交用户确认。未确认及未满足 Independent Architecture / IMPLEMENTATION_READY / Issue Ready 前不修改生产、测试、样例配置或运行构造/应用/schema；确认设计不自动授权合并或真实操作。
2. 用户确认并明确按§5/§6/§6.1/§7实施后，先记录并读回Ready，再在同一主要PR整批实现。单Writer/原独占worktree；不因文件数、键数拆PR或临时增加PM/架构师/子代理。同SHA检查运行时不重复触发；正常架构检查最多两轮。
3. 新证据证明保留能力仍读目标配置、当前外部配置合同必须兼容，或删除需要改变当前身份/资金/持久化/恢复/副作用时，停止对应变更，给具体消费者/后果并分类；不删活跃合同、不补新reader、不自建兼容。独立设计冻结后仅真正BLOCKER重开，非Blocker按实现测试/Backlog收敛。
4. 实施稳定后一次最终独立交付检查；修复只复核相关增量。准确HEAD合并与matching main CI、部署/真实操作、用户验收分别授权/核验。不用设计或PR CI冒充运行/用户验收。
5. 保留当前采集、Store观察、独立SHEIN登录及Store同步和此前能力，不宣称 #29/#301、数字Store身份或所有legacy源码清零；无新试用前置。Runtime/provider/user/production acceptance **NOT_RUN**。
