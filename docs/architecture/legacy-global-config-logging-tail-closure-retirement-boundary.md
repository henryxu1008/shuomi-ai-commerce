> #301 / #939 **SOURCE_IMPLEMENTED / IN_REVIEW**。用户确认冻结a88b890a7的§1–§7（含§4.1/§6.1），Reuse Existing Architecture / Ready已在[授权6392480965](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6392480965)与[源码前读回6392481887](https://github.com/qq550723504/shuomi-ai-commerce/pull/939#issuecomment-6392481887)完成。原唯一Writer同一PR收齐33工件，原guard实测RED；完整inventory/CURRENT零改，有效加载/弃用日志与原生SHEIN Redis/登录/Store同步保持。仅按§5/§6.1准确退出；GetBackupFiles独占的strings导入随方法失去用途后删除，其余writer body/配置保持。无新生产EXTRACT/owner/兼容层，无workflow/依赖/保护或既有资源操作，不增试用前置，#301 OPEN。以下准确设计a88b890a7完整Git blob逐字保留，旧设计阶段/待确认表述为历史记录；源码HEAD/CI/最终复核另记，合并另授权。

# 旧全局配置与日志便利尾部完整退休闭包

## 1. 用户结果、范围与 Design Basis

状态：**DESIGN_ONLY / AWAITING_USER_CONFIRMATION**。Refs #301；父目标保持 OPEN。用户授权先设计、确认后实施，一个 Delivery Batch / 一个主要 PR 收齐，不按文件、提交或检查拆小批。使用者是维护当前采集、AI/图像与独立 SHEIN 登录/Store 同步的维护者；场景是辨清哪些配置真正进入运行 owner，避免把旧全局配置投影和闲置便利方法当作有效入口。交付是准确 owner、旧 KEEP 替代、删除/保留与原检查矩阵，不新增产品入口或内部试用前置。

**Design Basis: N/A / Ready（当前仅本文、Architecture README、Legacy Register 三文档）。** 未来实施为 **Reuse Existing Architecture / Admission: NOT_READY**：复用当前配置加载/验证、Platform 日志、原生 SHEIN Redis 与实际 HTTP/Retry owner；只退休没有当前消费者的投影/声明/专属尾部，不新增生产 EXTRACT、端口、状态机、schema、身份/授权、事实、事务、预算/计费、恢复或副作用 owner。依据为 [Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[项目边界](project-boundaries.md)、[目标架构](project-target-architecture.md) 与 [Issue 驱动规则](../engineering/issue-driven-development.md)。无 UI/IA，Figma N/A；沿现有 Viper、Logrus、go-redis 等成熟组件，无新抽象或兼容层。

未来完整范围 **33 工件：3 整文件 RETIRE + 5 精确生产混合 + 4 精确测试/原guard混合 + 5 维护工件 + 13 旧依据窄顶注 + 3 当前文档**。预计超过30文件的风险提示线已核对：其中16份为准确文档/历史顶注；其余是两个现有技术 owner 的闲置内部声明/配置投影退出。下文逐一证明当前正式调用、序列化、持久化、权限及副作用边界不变，现有批准架构完整覆盖，不为文件数重复全局 Independent Architecture Review，也不拆分隐藏范围。若发现具体当前契约或必须改变这些边界，停止相应实施并按真实风险重新准入。

实施前须用户确认 §1–§7（含 §4.1/§6.1）并先在 #301/主要 PR 记录、读回 Reuse Existing Architecture / Ready；设计 CI 不授予生产开工。当前不改生产、测试、样例、脚本、旧依据或 inventory。

Out of Scope：workflow、完整 inventory、历史/计数/ceilings/parser/检查逻辑、依赖/锁、保护规则零改；不迁包、tidy/升级、不加 alias/wrapper/fallback、双读/双写/同步、第二事实源或新 runner/验收平台。不读取既有凭据，不操作既有实例、配置、数据、日志、缓存、进程或真实 provider；不执行配置加载/旧脚本/业务命令，不建 PG/Redis/浏览器实例。#29不重开，#301不关闭；合并、部署和运行验收另行授权。

## 2. 调查基线、实际消费者与旧 KEEP 替代

main **692cda1d2a49df920e972d24c544f5c664bbb281**，tree **fbc62817d57c37259646afecb7c1374f977c6d52**。原唯一 Writer 会话01a133ef-757c-73e3-b8f6-76958c642397，在独立 worktree、分支 codex/issue301-global-config-logging-tail-retirement-design 调查；[接单及前批当前段归档6392201343](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6392201343)。开放#931依赖和#692 Store设计不接管，另一个采集/Figma会话不重复派工。全量重新解析3270个受追踪Go文件，含测试/build tags/嵌套debug模块；解码import，核对准确声明、selector、同包引用和脚本/样例/原保活，目标owner无dot/blank import。

| 目标 | 真实引用 | 本批判断与保留 owner |
| --- | --- | --- |
| core/config.HTTPClientConfig、DefaultHTTPClientConfig、BuildHTTPClientConfig | 仅自身声明/default/builder、common_types_test和config_test；没有生产调用 | RETIRE；当前 pkg/httpclient.Config/New/Do/NewSimpleWithTimeout、浏览器下载/IP、SDK/session TLS/代理/timeout/retry保持 |
| core/config.RetryConfig、DefaultRetryConfig | type_rabbitmq.go内声明/default和两测试文件；无生产消费 | RETIRE；ImageAgent与listing结算实际用 shared/resilience.RetryConfig/Retry，全部保持，不按同名跨owner禁入 |
| 根 Config.Logging/Redis/Prompts；LoggingConfig/PromptsConfig | Logging/Redis只有BuildConfig写入，无生产读取；PromptsConfig只有字段/类型，没有生产读取 | RETIRE旧投影；保留根Config其它字段、加载/验证及当前业务 owner；SHEIN自身配置Redis字段/RedisConfig类型、当前ProductImagePrompts不动 |
| buildSplitByLevelConfig → getStringFromMap | 前者仅旧Logging投影调用，后者仅前者调用 | 两者随最后消费者完整退出；其它loader helper、time.Duration数据库连接寿命处理保持 |
| LogManager.GetLoggerWithFields | 只被TestLogManagerWithFields专属自测调用 | RETIRE；原GetLogger、原生Logrus.WithFields、实例level/锁/singleton/formatter/hooks/output保持 |
| rotatingFileWriter.GetBackupFiles | 仅声明，没有同包/包外调用 | RETIRE便利列表方法；实际轮转、备份清理/压缩/写入/关闭和配置能力保持 |
| core/config → platform/logging | config.go实际记录.env加载、配置文件加载、deprecated env警告 | **KEEP完整原调用与CURRENT边**，不能因Config.Logging投影无读取就推断包级无消费者，见§4.1 |

发现的根因是旧投影/门面在真实运行 owner 之外仍被构造或自测保活，维护样例还宣称它们可配置运行；处置是退掉这些精确尾部，保留实际调用，不新建配置/日志体系。包名、文件名或根模块零importer本身不是退休证据。

下列替代只在用户确认/实施Ready后生效；原批准、数字/计数/勾选及完整Git blob历史保留：

| 旧依据（§5.4逐一列文件） | 本批准确替代 | 持续KEEP |
| --- | --- | --- |
| #836运行配置、#872队列配置、#897执行骨架、#914闲置支持 | HTTPClientConfig/RetryConfig/default与旧全局Logging的广泛KEEP；旧Retry原调用者已退休 | 实际Retry、当前根配置/验证、真实workerpool/队列/Temporal、采集/图像/身份/计费 |
| #880 Redis/shared、#885 SDS | 仅根Config.Redis与其Viper投影的继续KEEP | RedisConfig类型、SHEIN独立私有JSON/原生go-redis/安装与执行归属/TTL/fence/UNKNOWN |
| #915宿主/HTTP/图像、#926旧平台API | 通用HTTP配置/default/builder及失去消费者的配置保活 | 当前HTTP/session/SDK、Alibaba1688字段、图像解码/规则和所有实际consumer |
| #893 logger facade、#937闲置支持门面 | 仅两个无消费者日志方法和对应自测；原保活表中本批准确项 | 真实配置加载日志、LogManager/LogConfig、原锁/并发/实例level/hooks/实际文件输出 |
| #917配置门面、Phase2原计划 | 仅三根投影、九core配置声明及丢失最后consumer的私有helper | 当前Viper/加载源/验证/数据库/其余builder与helper，非全量撤销Config |
| #846本地验收 | 仅port-forward帮助输出中的两条旧根Redis env建议 | #357不接管；实际DB/Redis/Temporal转发、参数、默认服务、会话/停止语义及Pester测试原样；不清扫其它帮助内容 |

不顺带退休 integration/crawler/a1688、localagent、dub/geminiimage/openmeter等当前合同或准备能力。原Amazon extractor仍使用goroutine.Processor；timeout/runtimepath仍被实际AI/采集/安装使用。它们和当前Product/SourceIdentity/草稿JSON/版本/hash/读取拒绝都不是本批旧全局配置的替代品，原样KEEP。

## 3. contract → implementation → injection → consumer

- 当前配置链：LoadConfig/LoadConfigFromFile/WithoutValidation/LoadFromBytes → scoped .env/Viper → BuildConfig → 六组有效验证 → 当前采集/worker/应用。只删除三个不再被读取的根投影及专属绑定；Config其它字段、source优先级、dotenv先后/非覆盖、自动环境读取和验证语义保持。
- 当前日志链：appenv → InitGlobalLogger/GetGlobalLogManager/GetGlobalLogger → LogManager；core/config继续通过原GetGlobalLogger记录加载/警告。Config.Logging并未注入这些调用，删除它不移除真实日志或引入新port。DefaultLogConfig仍Console、OutputFile空；可选文件输出、LevelSplitHook/rotating writer及其生命周期保持。
- 当前Redis链：独立SHEIN私有JSON → sheinlogin.Config.Redis（core/config.RedisConfig）→ bootstrap.BuildService → NewRedisStore/原生go-redis。根core/config.Config.Redis不在此链，不能删除类型、私有JSON字段、示例或有效Redis能力。
- 当前Prompt链：worker generation/capabilities → ProductImagePrompts/DefaultProductImagePrompts与版本/hash/AI准入/计量；根Config.Prompts的Dir/HotReload不在此链。不得换Prompt、重算hash、改路由/预算或读取已有覆盖记录。

对import core/config的生产序列化调用已逐项映射：当前runtime/worker私有JSON、schema CLI的配置、EnvelopeSummary和业务DTO各有其原owner，未找到这些旧根投影被作为当前请求/持久化事实保存的路径。这里只证明仓库当前显式源码，不推断已有程序、外部手工调用或环境状态；后者UNKNOWN。若出现具体当前可观察wire/持久化义务，停下报告用户，不自行增加迁移、兼容、双读或回填。

## 4. 持久化、副作用与安全边界

| 边界 | 本批效果 | 保留要求 |
| --- | --- | --- |
| 根配置输入 | 退三字段/两类型、五Redis显式env binding和未接线builder；不存在logging/redis/prompts显式setDefaults可删 | Viper AutomaticEnv/通用未知键处理不改，不新增旧键拒绝/翻译/恢复机制；dotenv/source顺序、所有其它bindings/default/验证保持 |
| 当前SHEIN | 没有字段/JSON/Redis命令变化 | RedisConfig及其5字段/原tag、SHEIN自身Redis、配置fingerprint/严格解码、授权/claim/fence/UNKNOWN、登录与固定Store同步保持 |
| 日志 | 只退两闲置便利方法；旧YAML不再展示无效logging字段 | core/config所有加载/弃用日志及警告计算保持；真实日志输出/路径选择/默认无文件/并发/轮转/压缩/Close原实现与原测试保持；不读写/删除任何实际日志 |
| 维护样例 | dev/test/prod只去logging根，dev另去redis根；无prompts根可删 | 其它键值/注释保持；独立SHEIN JSON示例、Kubernetes共享样例、Chatwoot/middleware Redis等不同owner零改 |
| port-forward | 仅去两条旧Redis建议Write-Host输出，不设置/删除环境变量 | 实际进程创建/等待/停止、参数/默认值/端口/Db/Temporal/Redis转发和其余提示原样，不执行脚本或触碰进程 |

无schema/数据库/Goose/身份/持久化owner变更，无已有行/key/文件的迁移或清理。当前JSON/SourceIdentity四零槽位、规范化/hash/读取拒绝、当前组织/计费等均不改。源码/fixture/CI不等于真实运行或产品验收。

### 4.1 唯一 CURRENT inventory 行的准确边界

前一轮候选“去LoggingConfig后可能删除logging→core/config”不成立，[纠正证据6392246071](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6392246071)已先报告。config.go在当前基线137/508/541/562/569/579/586/596行真实调用logger.GetGlobalLogger；这些调用和loadDotEnvCandidates/logDeprecatedEnvUsage/三LoadConfig入口的运行语义全部KEEP。

| inventory对象 | 本批决定 | 退出条件/处置 |
| --- | --- | --- |
| platform/logging → core/config CURRENT行 | **KEEP原字节；0行校准** | 真实consumer仍存在，退出条件未满足；不为删行做依赖反转、注入、迁包或停止有效日志 |
| 其它3条CURRENT、历史/计数/ceilings、parser/检查 | 全部零改 | 真实原owner继续成立；不改guard排除规则或虚构consumer |

完整docs/refactoring/phase2-runtime-inventory.md不在修改矩阵。未来若另批真实改变配置日志owner，再独立给出证据/准入/授权，不能预借本批“一条边界”当删除许可。

## 5. 完整33工件删除/保留矩阵

### 5.1 三整文件 RETIRE

```text
internal/core/config/common_types.go
internal/core/config/common_types_test.go
internal/core/config/type_rabbitmq.go
```

HTTP类型/default、Retry类型/default与五个专属自测一起退出；不留空包壳，不迁包/复制为新通用helper。core/config根仍存在，不能整包禁入。

### 5.2 五精确生产混合工件

| 文件 | 唯一允许改动 | 必须保持 |
| --- | --- | --- |
| internal/core/config/config.go | 删除根Config.Logging/Redis/Prompts字段、LoggingConfig/PromptsConfig声明及专属注释；删除knownEnvBindings中redis.host/port/password/db/pool_size五项；deprecatedEnvWarnings原说明中的REDIS_*例子仅改为仍有效DB_* | logger import与全部实际加载/警告调用原样；其它字段/绑定、dotenv/查找/优先级/验证与所有其它函数body不变 |
| internal/core/config/loader_builder.go | 删BuildConfig中Redis与Logging两个赋值块，BuildHTTPClientConfig、buildSplitByLevelConfig及专属注释，移除最后退出的logger import | time import仍服务databaseConfigFromViper.ConnectionMaxLifetime，不能删；其余BuildConfig/DB/AI/身份/计费builder和值处理保持 |
| internal/core/config/loader_helpers.go | 仅删getStringFromMap完整声明 | 其它helper/import原字节保持，无替代函数 |
| internal/platform/logging/manager.go | 仅删LogManager.GetLoggerWithFields完整声明/专属注释 | LogManager/LogConfig/默认值/所有其它方法、锁/registry/hooks/output生命周期保持 |
| internal/platform/logging/rotating_writer.go | 仅删rotatingFileWriter.GetBackupFiles完整声明/专属注释 | 写入、轮转、备份清理/压缩/路径/错误/关闭所有其余body与配置保持 |

### 5.3 四测试混合与五维护工件

| 文件 | 准确处置 |
| --- | --- |
| internal/core/config/config_test.go | §6.1仅两个原函数转换；其它全部函数/断言保持；time import只在旧HTTP断言失去最后用途时移除 |
| internal/core/config/config_env_test.go | §6.1仅一专属函数退休、两原环境覆盖测试删旧Redisfixture/断言；所有OpenAI/Browser及其它优先级/隔离/权限/认证断言保持 |
| internal/platform/logging/manager_test.go | 仅删TestLogManagerWithFields；其它每个原测试/benchmark/import保持 |
| tests/listingkit_backend_retirement_test.go | 仅§6.1列出的三个既有函数/表项准确转换，不改其它guard |
| config/config-dev.yaml | 仅删除顶层logging/redis段及其专属段标题/空白；其它键值/注释保持 |
| config/config-test.yaml | 仅删除顶层logging段及专属标题/空白；其它保持 |
| config/config-prod.yaml | 同test；不因文件名操作任何生产配置/实例 |
| tests/logging_config_test.go | 原路径检查转换为三个维护YAML必须缺席logging/redis/prompts根键；保留文件存在、YAML解析失败即失败，不靠空字段/skip形成PASS；详§6.1 |
| scripts/start-listingkit-local-portforward.ps1 | 仅删除现150/151行两个Write-Host的TASK_PROCESSOR_REDIS_HOST/PORT建议输出；其它全部字节/过程不变，Pester原文件零改 |

### 5.4 十三旧依据窄顶注与三当前文档

未来仅给下列旧依据添加准确superseding顶注，指回本文§2/§4.1/§5/§6.1；**完整基线Git blob（含已有顶注）逐字保留为后缀**，历史数字/计数/勾选/批准/失败不改：

```text
docs/architecture/legacy-host-image-helper-closure-retirement-boundary.md
docs/architecture/legacy-idle-support-contract-closure-retirement-boundary.md
docs/architecture/legacy-platform-api-closure-retirement-boundary.md
docs/architecture/legacy-queue-proxy-config-tail-closure-retirement-boundary.md
docs/architecture/legacy-redis-proxy-shared-closure-retirement-boundary.md
docs/architecture/legacy-runtime-config-closure-retirement-boundary.md
docs/architecture/legacy-runtime-skeleton-config-closure-retirement-boundary.md
docs/architecture/legacy-sds-runtime-closure-retirement-boundary.md
docs/architecture/legacy-logger-facade-closure-retirement-boundary.md
docs/architecture/legacy-config-facade-closure-retirement-boundary.md
docs/architecture/legacy-idle-support-facade-closure-retirement-boundary.md
docs/architecture/legacy-local-acceptance-closure-retirement-boundary.md
docs/superpowers/plans/2026-08-30-internal-target-architecture-phase2.md
```

当前三文档为本文、docs/architecture/README.md和docs/refactoring/legacy-register.md；两索引只加当前项，完整原Git blob保留为后缀。其它历史文档、defaults.go、retired_product_runtime.go、type_redis.go、SHEIN私有JSON示例、Pester、部署共享样例/工作流/完整inventory零改。

```text
Legacy decision: RETIRE
Reusable behavior: 无新生产EXTRACT；保留原owner中的当前配置加载/验证/日志/HTTP/Retry/Redis及有效安全断言
Current owner: 原Config/Viper/ValidateConfig；Platform LogManager；shared/resilience；当前HTTP/SDK；SHEIN原生Redis/私有配置
Cutover/deletion condition: 用户确认§1–§7（含§4.1/§6.1）、Ready读回、刷新真实消费者后，同一PR收齐33工件；CURRENT零改，不操作运行资源
```

## 6. 必要验证与验收边界

当前设计阶段只读调查、33工件/路径/相对链接与历史后缀校验，运行原适用文档检查及稳定设计候选原CI；不制造业务RED，不执行生产constructor、配置加载、旧脚本/provider，不建实例或验证平台。

未来先在原guard增加准确缺席/禁入，确认RED，再按矩阵删除至GREEN；保留断言转换与必要回归同批完成。开发自检限配置加载/源优先级/通用env隔离、当前LogManager原行为/并发/无默认文件、原guard/导入/inventory/文档、HTTP/Retry/Identity/JSON及SHEIN/Store现成合成fixture。不得读取既有.env/token/日志或运行外部实例；不使用integration tags或真实provider。只编译当前应用、独立SHEIN登录、ImageAgent worker、1688-local-agent与独立hack/debug原Amazon入口（-mod=readonly、输出任务临时目录，不执行）。沿用原PowerShell AST语法/Pester参数检查，只解析脚本，不启动其转发/进程。稳定候选留给原完整CI，不人工重复CI全套。

最终候选前同步一次实际main，只处理真实交集；未变化代码已有证据可明确复用，不冒充新SHA直接结果。任何真实finding先按当前Must分类并检查sibling paths；涉及新契约/owner/持久化/权限/恢复或表外工件/检查的改变先报告，不把失败转为skip/成功，不因一条理论风险建设新平台。

### 6.1 原检查完整转换矩阵

| 原函数/检查 | 精确转换 | 原样保持 |
| --- | --- | --- |
| TestLegacyRuntimeSkeletonAndIdleConfigClosureIsRetired | 原文件缺席表加§5.1三路径；retiredSupport的core/config追加HTTPClientConfig/DefaultHTTPClientConfig/BuildHTTPClientConfig、RetryConfig/DefaultRetryConfig、LoggingConfig/PromptsConfig、buildSplitByLevelConfig/getStringFromMap九项；logging追加LogManager.GetLoggerWithFields、rotatingFileWriter.GetBackupFiles两项；原struct-field禁入条件仅追加owner=core/config且type=Config的Logging/Redis/Prompts三字段；retained仅删本批五个原core类型/default保活，并新增正向项RedisConfig、RedisConfig.Host/Port/Password/DB/PoolSize、OpenAIConfig、BrowserConfig、PlatformsConfig、Alibaba1688Config（原Config项保持） | 其余历史/禁入/保活表，receiver/声明精确比对、原遍历/解析/错误处理、全部既有ImageInfo/Identity字段/解码注册与当前DB/ptr/日志保护；不按名字全仓禁入 |
| assertLegacyPlatformAPICallClosureRetired | requiredByFile只退出common_types.go/type_rabbitmq.go两行；三整退路径加入原缺席表；其余算法不改 | 原其余平台/SessionTransport/仓库转换/JSON等全部保活、历史禁入与失败关闭；不能missing-skip |
| TestRetiredTaskAndListingRuntimeConfigClosure | 原retiredRoots只加logging/redis/prompts，使三个维护YAML明确禁止旧根键；在原读取/失败关闭模式中准确检查port-forward源文本缺席两项TASK_PROCESSOR_REDIS_HOST/PORT旧建议 | 原ConfigMap/YAML解析、全部其它退休配置、字段/校验/脚本检查不变，不扩大成真实环境扫描 |
| config_test.go / TestCurrentConfigDoesNotRequireRetiredTaskInputs | 原已声明defaults/bindings前缀禁入表仅追加logging./redis./prompts. | 其它全部旧禁入、fixture与当前ValidateConfig断言；不改变Viper AutomaticEnv对任意key的原通用行为 |
| config_test.go / TestLoadFromBytesRetainsPlatformTransportLoginAndRetry | 更名TestLoadFromBytesRetains1688BrowserConfiguration；仅去BuildHTTPClientConfig的Viper设置/timeout/TLS和DefaultRetryConfig断言 | 原LoadFromBytes fixture及1688 Timeout=42/PoolSize=5两有效断言原字节保持；其余函数不动 |
| config_env_test.go / TestNewViper_BindsRedisEnvironmentVariables | 随五旧绑定删除此专属测试 | Database/OpenAI/Browser等仍有效绑定、primary优先/弃用警告/scoped .env/不覆盖/身份安全原检查保持 |
| TestLoadFromBytes_AppliesEnvironmentOverrides、TestLoadFromBytes_AppliesViperManagedEnvironmentOverrides | 仅去旧Redis的Setenv、YAML段、NotNil/Host/Port断言 | 前者OpenAI显式绑定/命名客户端继承断言；后者Browser的AutomaticEnv/headless/poolSize原断言和fixture保持。原通用来源覆盖/隔离仍非空验证，不全删函数 |
| common_types_test.go五测试、manager_test.go / TestLogManagerWithFields | 专属测试随退休声明退出 | 所有其它LogManager并发/懒初始化/级别/文件输出/默认无文件/ParseLevel/Close等原测试保持 |
| tests/logging_config_test.go / TestMaintainedLoggingFilesStayUnderLocalRuntimeRoot | 去专属maintainedLoggingFileConfig，原函数更名TestMaintainedConfigFilesDoNotAdvertiseRetiredGlobalInputs；仍逐一读取三YAML并严格解析map，对logging/redis/prompts每个根键显式报错 | 原文件不存在/解析失败必失败，不能空路径循环/skip。旧路径检查的输入owner已退出，转换为更强的“不再宣称任何旧根输出路径”；不把旧断言写成当前runtime路径约束。真实默认无文件与file-writer安全/生命周期测试完整KEEP，不新增任意本机目录限制 |
| 原SSA/import/inventory/parser/其它安全/CI检查、Pester | 零改 | core/config真实加载日志与CURRENT行保持；所有历史/计数/ceilings和完整workflow保持 |

本表是全部原检查修改边界，不仅检查最新新增guard。禁止为了原旧存在性测试重建类型、增加dummy、改扫描根绕过、降低日志/Redis/授权/源优先级安全断言或取消必需CI。若真实测试暴露遗漏同根引用，先记录具体差异与本矩阵关系；冻结后需要表外检查/工件例外，交用户决定后再应用。

## 7. 交付与停止点

本阶段交三文档到一个主要设计PR，#301当前段记N/A/Ready与准确范围；先归档前一当前段，历史后缀从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起逐字保持，旧勾选不改，正文≤65536字符。设计HEAD、原CI、真实复核与限制留PR；不夹带生产/测试/样例/脚本/inventory提前实施。

设计确认后先记录/读回Reuse Existing Architecture / Ready，再由原唯一Writer在同一PR收齐33工件、原guard RED→GREEN、必要安全保留回归/编译/原CI，不逐小片重复请示或重开PR。完整结果形成后做一次集中独立最终交付检查；修复只复核对应增量，未经用户本轮授权不默认新增AI互验线程。不重复触发同SHA运行中的GitHub评审；启动失败/无正式结果如实记录，不当作具体finding或PASS。当前N/A/未来复用既有架构，不追加全局架构审核或新的试用前置。

实现完成进In Review，待独立验证/用户决定，不签发产品验收。合并另获准确HEAD授权，正常expected-head执行，不绕过保护；随后只以actual merge SHA匹配push/main整run completed/SUCCESS确认，PR CI/其它提交/单Gate不替代。#301持续OPEN，关单另授权。真实登录/Store同步/provider/部署/用户/生产验收均NOT_RUN，既有实例/数据/凭据/进程操作NOT_AUTHORIZED。当前能力、原Amazon/1688浏览器与debug、有效AI/图像/HTTP/Retry/JSON及独立SHEIN登录/Store同步保持。
