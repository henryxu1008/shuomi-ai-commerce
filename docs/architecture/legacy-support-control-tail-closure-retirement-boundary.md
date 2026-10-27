# #301 旧日志控制、HTTP 无效选项与迁移状态尾部完整退休设计

> **DESIGN_ONLY / SOURCE_NOT_STARTED / AWAITING_USER_CONFIRMATION**。本阶段只改本文、Architecture README、Legacy Register 三份文档。设计基线为实际 main `674d87c05253f77f02eebba20cb154cf9e294191`；#968 已正常合并，其准确 push/main CI40435620130 attempt1 整 run SUCCESS。本批未来实施为 **Reuse Existing Architecture / NOT_READY**，须先完成适用设计复核、用户确认 §1–§7（含 §4.2、§6.1），并在 Issue/PR 记录、读回 Ready；当前没有生产代码实施、合并、部署或关单授权。

## 1. 用户结果与完整范围

使用者是维护当前应用、采集与独立 SHEIN 登录/Store 同步的开发者。这些入口继续使用相同的日志、网络和数据库能力；本批去掉它们附近已无当前消费者的控制接口、无效字段和纯转发方法，避免后续代码误用。没有新页面、操作步骤或试用前置。

| 三组一次收齐 | 本次拟退出 | 当前能力保留 |
| --- | --- | --- |
| 日志控制与扩展 | 自建级别镜像、闲置实例级别 setter/getter、无配置生产者的分级文件 Hook、三个无效/闲置配置字段及专属测试 | Logrus 初始级别、格式、ReportCaller、console/file/轮转/Close、全局初始化锁和懒加载 |
| HTTP 配置 | 从未被实现读取的 SkipTLSVerify 字段及下载器唯一 false 赋值 | 原 http.Client/TLS、超时、重试、取消、响应关闭和日志 |
| 迁移状态门面 | 无生产消费者的 Runner.Status 转发 | Goose provider、Up、原版本表/会话锁/SQLite 锁、幂等和状态断言 |

未来同一主要 PR **18 唯一路径**，不机械拆批：5生产文件（含1整文件删除）、2原测试文件、1原退休检查文件、4当前文档及6历史顶注。精确矩阵见 §5。只退4类型、8函数/方法和6个混合声明字段；不把退休类型内部字段重复计数，不新增生产函数、包、兼容 alias 或替代门面。

**Must**：完整闭包删除、原有效行为与安全断言保持、三组前批 KEEP 准确替代、当前采集及独立 SHEIN 登录/Store 同步保持。**Should**：复用标准库 sync/http、现有 Logrus/Goose 和已有测试。**Out of Scope**：AI/provider、产品/领域/身份/授权/租户/计费合同、新状态/schema/事实 owner/事务/retry/UNKNOWN/recovery、现有日志或配置文件、实例/数据/凭据/进程、workflow/完整 inventory/依赖/保护及其它检查；不增加内部试用前置，#301 保持 OPEN。无新增 Accepted Risk；物理状态 UNKNOWN，不探查。

Threat Model 仅覆盖误删实际日志/网络/迁移能力、丢失全局锁和有效测试、误把无效字段接成 TLS 旁路、放宽退休检查和旧门面回流；不扩大全局安全目标。

## 2. Design Basis、Authority 与旧 KEEP 替代

当前文档阶段 **Design Basis: N/A / Ready**，仅设计说明及索引登记，不改变运行语义。未来源码阶段选择 **Reuse Existing Architecture / NOT_READY**：现有 logging、httpclient、migration owner 和第三方 SDK 完整覆盖本批；没有新的领域公共合同、事实 owner、状态、权限、持久化或副作用边界。三组虽跨目录，均为旧支持尾部退休，保留行为不改，不为目录数量重复全局架构审核。UI/IA/交互不变，Figma N/A。若实施真实需要改变这些边界，停止受影响部分并按 AGENTS.md 升级 Design Basis，不能在本批临时发明机制。

依据：[Greenfield 产品决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[目标架构](project-target-architecture.md)、[项目边界](project-boundaries.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)和用户本轮设计范围。旧内部公开方法/测试本身不是当前产品要求；没有实际消费者的门面 RETIRE，有效行为沿当前 owner 保留。标准 sync.Locker、Logrus 原生级别和 Goose provider 已存在，不重复造轮子。

| 历史依据 | 本次准确替代 | 仍然有效 |
| --- | --- | --- |
| #893 logger façade | platform/logging 所有公开方法/测试全保留的宽泛含义，仅替代 §5 指定级别镜像/控制、Hook 和无效字段 | 显式日志注入、原生 Logrus、初始化/获取/实际 writer；业务层旧 logger 导入禁令 |
| #934 helper/decorator | LevelFileConfig、level_split_hook.go、级别镜像及所有 manager 测试必须保活 | 当前 LogConfig/LogManager/全局入口、有效日志和其它已退休对象永久禁令 |
| #937 support façade | LogManager.SetLevel/GetLevel 及专属读锁测试的 KEEP | singleton 构造一次、完整发布、真实锁、文件生命周期；当前 HTTP Client/Config/New/Do |
| #939 global config/logging | 原根 logging 退出后仍保留全部 LogConfig 字段/Hook 的宽泛含义 | 根 logging/redis/prompts 继续退休；core/config 有效加载日志与 appenv 注入仍用当前 owner |
| #915 host/image helper | HTTP 当前 Config 全字段保留的宽泛含义，仅替代 SkipTLSVerify | TLS/HTTP 默认行为、四个有效配置字段、图像解码、采集和当前入口 |
| Module Target Mapping | 旧日志控制/HTTP 无效选项/Status 的宽泛支持层保活含义 | 其它目标与历史、实际库和领域 owner 不变 |

仅以上准确边界被替代。未来实施在六份历史依据加当前顶注，后缀保留起始 main 的完整原 Git blob、批准/失败/数字/勾选；不改写历史为 PASS。当前设计阶段不改这六文件，也不预写退休完成。

## 3. 实际消费者与当前 owner

以基线 **3174 tracked Go** 做标准 Go AST 只读核对：包含原测试、未选择编译源和 hack/debug；解码 import、核对别名/局部遮蔽、选择器与函数值，再检查同包调用、混合字段、构造和原检查。另查 tracked YAML/JSON/TOML/TS/脚本，未发现三个日志配置键或该 HTTP 字段的配置生产者；没有读取未跟踪配置或现有凭据。静态消费者结论不是既有实例的运行验收。

| owner | 当前入口/注入/消费者 | 待退对象的真实使用 |
| --- | --- | --- |
| platform/logging | pkg/appenv.SetupLoggerWithLevel 构造 LogConfig → InitGlobalLogger → LogManager；appenv/core/config 获取原 logger。外部生产 import 只有 pkg/appenv/apputil.go、core/config/config.go；另有 appenv 原测试 | SetLevel/GetLevel 只有同包测试；newLogManager 只有原公开构造与专属锁测试；SplitByLevel 无配置生产者，Hook 只在其分支使用；EnableCaller/CallerSkip 无有效读取；LogManager.formatter 只赋值 |
| pkg/httpclient | shared/browser/chrome_downloader.go 使用 Config/Client/New；ip_location.go 使用 NewSimpleWithTimeout | SkipTLSVerify 从未读入 http.Client/Transport，唯一赋值是 ChromeDownloader 构造中的 false |
| platform/database/migration | knowledge-schema-init、accountallocation/accountprofile/productlisting/sourceaccountregistry/storecenter 六生产入口使用 New/NewWithVersionTable，另有两个包外测试 import | Runner.Status 只有同包 runner_test 的两个调用，生产只用 Up；测试能直接使用已有私有 provider，无需新接缝 |

**发现的架构问题**：日志级别由 Logrus 持有，同时又被旧 LogManager 镜像；GetRawLogger 可直接访问 SDK，镜像没有当前业务消费者，也可能与原生值分离。解决方式是删去镜像和闲置控制门面，保留原 Logrus 初始配置与实际调用，不再造 setter/getter/另一份级别状态。HTTP 字段只宣称能力而从未生效，本批删除该空选项，不能顺势实现跳过 TLS 验证。迁移状态没有生产调用，无需让 Runner 再包装 Goose。

**不是闲置锁**：logManagerRegistry.get/init 仍用锁保证懒初始化一次及完整发布，必须保留。只将 locker 字段和 newLogManagerRegistry 参数的自建 rwLocker 类型改为标准 `sync.Locker`；原 `&sync.RWMutex{}`、Lock/Unlock、顺序、factory 和全局实例不变。不能因级别镜像锁退出而删除全局锁。

活跃 PR #966/#956/#692 的源码与未来18路径无交集，仅 Architecture README 有索引交集；不接管其 TLS、Store-browser 或文档 owner。最终候选再核对实际 main，一次处理真实组合风险，不持续 merge-main 或重复派工。

## 4. 有效行为、持久化与副作用边界

| 能力 | 保留 owner 与不变量 | 本批边界 |
| --- | --- | --- |
| 日志 | 原 Logrus + logging.LogManager/rotatingFileWriter；初始 level、format、ReportCaller、stdout、显式 OutputFile、压缩/保留/关闭顺序 | 原 GetLogger/GetRawLogger/Close、parseLogLevel/createFormatter/createOutputs、真实全局 registry 和所有 writer 逻辑保持；仅 §5 指定尾部退出 |
| HTTP | 原 http.Client、下载器与 IP 定位；Timeout/MaxRetries/RetryDelay/EnableLogging、context 取消、错误/响应关闭 | New/Client.Do/NewSimpleWithTimeout 逐字保持，仅 Config 删无效字段、下载器删 false 项；没有 TLS transport 变化 |
| 迁移 | 原 Goose provider/各 schema owner；安全版本表、Postgres session lock、SQLite 完整 Up 进程锁、幂等和独立 history | New/NewWithVersionTable/validVersionTableName/newRunner/Up、Runner 两字段和全局锁/正则保持；不改或执行任何迁移 |
| 产品与采集 | 当前 AI/JSON/hash/绑定版本/有效规则、采集、PoolManager deadline/取消、图像、Store 与独立 SHEIN 登录/同步 | 所有入口、身份/授权/组织隔离、UNKNOWN owner、Store 同步及当前 session/client/Redis/DB 不变 |

### 4.2 JSON、配置与物理持久化

LogConfig 的 `enable_caller`、`caller_skip`、`split_by_level` 是内部旧配置槽位，当前没有生产 YAML/JSON 读写链或已注册管理 API；前两项未被使用，第三项没有当前 producer。未来删除这三个字段/tag、CallerSkip 默认项和专属 Hook 分支，不保留 alias、旧 key 映射、兼容解码/迁移。保留9字段及其原类型/tag/顺序/默认值：Level、Format、OutputFile、MaxSize、MaxBackups、MaxAge、Compress、Console、ReportCaller。

LogManager 保留 logger/outputs/config/fileWriter/mutex 五字段；删 levelState/formatter，局部 formatter 及 logger.SetFormatter 保持。级别归现有 Logrus，去掉镜像并不撤销原 logger 本身。httpclient.Config 保留四有效字段原类型/顺序，无 JSON/tag 或持久化 owner 变化。没有草稿/Product/SourceIdentity 或平台 JSON/hash/版本变化。

日志文件仍归原旋转 writer；数据库版本记录仍归原 Goose/各 schema owner。已有物理文件、配置、表/记录是否存在为 UNKNOWN，本批不读取、修改、迁移、删除、Close、启动或重启它们。设计与源码退休不构成对现有实例的停用或清理；按全新安装和当前模型执行，没有旧数据兼容义务的推定。

## 5. 删除/保留矩阵与18路径

### 5.1 生产闭包：4类型、8函数/方法、6混合字段

| 文件 | RETIRE / 准确改接 | KEEP |
| --- | --- | --- |
| internal/platform/logging/manager.go | 类型 rwLocker、logLevelState；logLevelState.locking、newLogManager、LogManager.SetLevel/GetLevel；LogManager.levelState/formatter；LogConfig.EnableCaller/CallerSkip/SplitByLevel；CallerSkip 默认项、mirror/formatter 字段赋值及分级 Hook 构造分支。把原 private 构造有效正文直接放入已有 NewLogManager；registry locker/构造参数仅改 sync.Locker | 其余5字段/9配置字段；原 imports、解析/格式/输出/Close 全文；原日志初始级别与 ReportCaller；registry 的 manager/factory/global、锁方法/顺序及公开全局入口 |
| internal/platform/logging/level_split_hook.go | 整文件；LevelSplitHook、LevelFileConfig；NewLevelSplitHook、LevelSplitHook.Levels/Fire | 不新增替代 Hook/工厂/配置 owner；独立 rotating_writer.go 原样 |
| internal/pkg/httpclient/httpclient.go | 仅 Config.SkipTLSVerify 一字段 | 其它字段、imports、所有函数/注释和实际 HTTP 行为保持 |
| internal/crawler/shared/browser/chrome_downloader.go | 仅 Config literal 的 SkipTLSVerify:false 一项 | 下载器构造、默认 timeout、日志注入和下载流程原样 |
| internal/platform/database/migration/runner.go | 仅 Runner.Status 声明及专属分隔空白 | imports、其它所有声明/正文、两字段、锁/正则/ProviderOptions 原样 |

### 5.2 原测试与检查尾部

| 文件 | 准确允许变化 | 保留 |
| --- | --- | --- |
| internal/platform/logging/manager_test.go | 删除 TestLogManagerSetLevel、TestZeroValueLogManagerGetLevel、TestLogManagerFallbackLockSupportsConcurrentSetAndGetLevel、TestLogManagerSetAndGetLevelWaitForManagerLock 四专属测试；probeRWLocker 只删 readLockAttempt/readLockPermit 两字段、两初始化项及 RLock/RUnlock 两方法。TestNewLogManager 与 TestLogManagerRegistryGetWaitsForInitToPublishCompleteManager 各一处 GetLevel 改 `GetRawLogger().GetLevel().String()`，原 info/error 断言不变 | probe 的 sync.RWMutex、Lock/Unlock 及原通道协议；registry 构造一次测试全文、完整发布测试其余流程；文件/默认不落盘/格式/caller/级别解析/全局标记/default 与 benchmarks 全文，不用 SDK 内部测试替代四旧测试 |
| internal/platform/database/migration/runner_test.go | TestRunnerUpIsIdempotentAndStatusReflectsAppliedMigration 两处 runner.Status 改 runner.provider.Status | 原函数名称、错误/Pending/Applied/两次 Up/result/runcount 断言，所有其它函数/imports/fixtures 全文；直接用已有 provider，不新增生产导出/测试接缝 |
| tests/listingkit_backend_retirement_test.go | 仅 §6.1 原 retained 表三项转换，新增一个完整闭包 guard | 其它原函数/helper/import/fixture/算法/失败关闭和全部永久禁令保持 |

### 5.3 当前文档与历史尾部

4当前文档：本文；docs/architecture/README.md；docs/refactoring/legacy-register.md；docs/architecture/project-boundaries.md。当前设计只改前三份；未来实施在本批当前段准确更新状态，project-boundaries 只加当前 owner 顶注，原完整后缀保持。

6历史文件：未来只加窄顶注，明确 §2 的替代与保留，完整原 Git blob 作为后缀，不改历史数字/勾选/批准/失败：

```text
docs/architecture/legacy-logger-facade-closure-retirement-boundary.md
docs/architecture/legacy-idle-adapter-helper-closure-retirement-boundary.md
docs/architecture/legacy-idle-support-facade-closure-retirement-boundary.md
docs/architecture/legacy-global-config-logging-tail-closure-retirement-boundary.md
docs/architecture/legacy-host-image-helper-closure-retirement-boundary.md
docs/refactoring/module-target-mapping.md
```

8源码/测试/检查路径 +4当前文档 +6历史＝18。logging/README.md 无退休示例，零改；appenv/core/config/rotating_writer、当前采集/图像/AI/SHEIN/Store/schema 和其它文档零改。不因同名 Status/SetLevel/Config 等扩大范围。

```text
Legacy decision: RETIRE (selectively retain existing behavior in place)
Reusable behavior: 原公开 NewLogManager 承接原 private 构造有效正文；标准 sync.Locker、Logrus 原生级别、Goose 原 provider 及现有有效断言
Current owner: platform/logging + Logrus；pkg/httpclient + http.Client；platform/database/migration + Goose
Cutover/deletion condition: 用户确认本设计与 Ready 读回后，同一批完成全部调用/字段/测试/检查/文档尾部；无 alias、转发、fallback 或第二事实源
```

## 6. 实施与验证

当前纯设计：核对三文档范围、相对链接、完整矩阵/§1–§7/§4.2/§6.1、索引和登记原完整后缀、git diff --check、适用原文档检查；不制造业务 RED，不执行日志/网络/数据库函数或编译入口。准确设计 HEAD 的原 PR CI、设计复核与未执行边界在 PR 分别记录。

未来确认后，先记录、读回 Reuse Existing Architecture / Ready；在原退休设施按 §6.1 加 guard 并确认旧代码真实 RED，再按 §5 一次切换/删除至 GREEN。有效行为不新增规则，无形式化新测试框架；原并发、生命周期和迁移断言保留。必要自检是准确退休/文档/导入检查、logging 包及 appenv 原测试、migration 原 SQLite fixture 测试、受影响 browser 无浏览器 mock 测试，及当前应用/SHEIN 登录 worker/ImageAgent Temporal worker 原相关回归与依赖编译。原 cmd/current-application、cmd/shein-login-worker、cmd/image-agent-temporal-worker 仅编译到任务临时目录，不执行。无需 PostgreSQL/Redis/browser 或真实请求，不新建 runner/验收平台；不操作现有文件/资源。

最终候选前核对实际 main 增量，只处理真实组合风险。准确最终 HEAD 的完整原 CI、最终交付复核分别留证；未来只有用户授权最终准确 HEAD 后才正常合并，之后确认实际 merge SHA 的 event=push/head_branch=main 完整 run completed/SUCCESS，不用 PR CI/单 gate 替代，不改保护。

### 6.1 原检查、测试、文档与 inventory 精确矩阵

| 工件/原函数 | 允许转换 | 保持 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go / TestLegacyRuntimeSkeletonAndIdleConfigClosureIsRetired | 仅从 logging retained 表删 LevelFileConfig、LogManager.SetLevel、LogManager.GetLevel 三项；它们转入新 guard 的精确退休断言 | 原 retained 其它所有项、retiredSupport/文件缺席/字段/遍历/解析/receiver 比对/错误处理及全部其它逻辑原样；不删整个 logging 保活 |
| 同文件新增 TestLegacyLoggingHTTPMigrationSupportClosureIsRetired | 复用现有 trackedFiles/AST 设施，要求唯一旧 Hook 文件缺席；按 owner/receiver 准确禁止 §5.1 的4类型/8函数及6混合字段，检查旧构造/Hook 接线与默认项退出；正向要求9 LogConfig/5 LogManager/4 HTTP 字段原形、std sync.Locker 和原 registry/current 入口/输出/Goose Up 锁、当前下载器与有效行为声明存在 | 不全仓禁通用同名，不靠空目录/dummy/missing-skip；解析/读取错误失败关闭。正向声明不当作等价证明，原有效测试与未变源码共同留证 |
| manager_test.go 的 §5.2 四专属测试、probe 与两观察点 | 仅按 §5.2 删除、改接 | 两个 registry 并发测试全部有效锁/发布/构造次数断言及其它原测试保持，不放宽 deadline/等待或并发断言 |
| runner_test.go / TestRunnerUpIsIdempotentAndStatusReflectsAppliedMigration | 仅两处调用改已有 provider.Status | 所有原断言、SQLite 串行/独立 history/安全表名/错误测试和 fixtures 原样；不能只保留成功 Up 而删状态断言 |
| 文档 | 仅 §5.3 当前段/六顶注，完整原后缀保留 | 不预写产品验收/运行 PASS，不改历史批准/数字/勾选 |
| 完整 inventory 与其它原检查 | **零改**。当前 appenv/core/config 的 logging 消费、原 HTTP 与 migration imports 仍在，无 CURRENT 校准需要 | CURRENT/历史/计数/ceilings/算法、Code Health/lint/必需与安全检查、其它原函数/fixture、workflow/依赖/保护原样 |

若真实 CI 暴露本表遗漏，先查同根 sibling、报告具体根因与准确最小补丁，再取得用户决定；不能把未列项默认为可调，也不能缺文件跳过或放宽门禁。本设计冻结不授权给真实 TLS/迁移/日志能力追加需求。

## 7. 复核、交接与停止点

1. 只完成本批三文档与准确设计 HEAD 证据；未来18路径实施保持 NOT_READY，设计复核通过也不代替用户实施决定。不把此前 #968 的授权或 CI 当本批证据。
2. 用户确认 §1–§7（含 §4.2、§6.1）后，再在同一 Issue/PR 记录、读回 Ready，由单 Writer 连续完成18路径。若发现真实授权/领域/持久化/恢复/副作用边界变化，按 AGENTS.md 停止受影响部分；finding 先分类，只有命中当前 Must 的真实 BLOCKER 重开设计，非 Blocker 不无限审核。
3. 设计复核与源码最终复核分开：按原 GitHub 机制核对真实 diff、实际消费者和有效证据；形成完整源码后一次集中最终交付检查，修复只复核增量。自动复核启动失败至多重试一次；**未经用户本轮授权不默认新增 AI 互验线程**，不复用旧批子代理授权，不为本批追加全局架构或运行验收平台。
4. PR 交出准确 HEAD、当前能力/原检查保留、完整原 CI/复核和 NOT_RUN 限制。实现者自检、CI、另一 AI review 都不是用户产品验收/可上线证明；原入口与使用方式不变，本轮不运行它们，不宣称真实物理环境退休完成。
5. 实施、最终准确 HEAD 合并、部署和 #301 关单分别授权。#301 保持 OPEN，仅更新当前段，既定 #726 marker 起完整历史/勾选/数字逐字保留；不改 workflow/inventory，不操作既有实例、数据、凭据、配置/日志文件或进程，不增加内部试用前置，保留独立 SHEIN 登录与 Store 同步。
