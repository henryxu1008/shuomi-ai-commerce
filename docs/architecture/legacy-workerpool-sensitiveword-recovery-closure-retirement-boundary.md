> #301 / #940 **SOURCE_IMPLEMENTED / IN_REVIEW**：用户已确认[完整退休设计](legacy-shein-sensitiveword-closure-retirement-boundary.md) §1–§7/§6.1，准确替代本文件中管理/反馈方法、LastUpdated/Version/更新helper及专属4槽JSON/mutation并发KEEP；旧入口失去生产消费者后整批退出。当前17清理函数、两词表map与原tags/默认字典、mutex读锁/防御复制、当前纯规则/品牌/平台JSON、独立SHEIN登录与Store同步继续原owner。仅补当前顶注，原完整正文/历史/数字/勾选保持；无文件数据处理、迁移或替代owner，workflow/inventory与既有资源不动，不增试用前置，#301 OPEN。

# #301 旧工作池、SHEIN 敏感词文件尾部与恢复辅助层完整退休设计

Status: **USER_CONFIRMED / SOURCE_IMPLEMENTED；待原 CI 与准确源码最终交付复核**。用户已确认 §1–§7（含 §4.1/§4.2/§6.1），Issue/PR 的 Reuse Existing Architecture / Ready 在源码修改前读回；原唯一 Writer 在同一主要 PR 整批实施。准确 HEAD、原 CI、设计与源码复核、合并批准写主要 PR，不为滚动状态制造仓库提交。接单及前当前段存档见 [#301 / 6354620829](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6354620829)。

调查/开工基线 main **71758518b24a72454b75c0b59d03fcf0e6dfc713**，已含 #897；相对其 merge 4c9cfcef3 只多 #899 邀请流程四文件，无本批变化。#897 的实际 push/main CI39815055773 完整 SUCCESS 是前批证据，不替代本批设计或实施检查。

## 1. 用户结果、完整范围与权限

使用者为维护当前清理、采集和独立 worker 的维护者。一个完整源码结果：没有生产消费者的旧数字任务工作池退出；当前敏感词服务只承担已有内存词表/清理行为，去掉无调用的文件入口和保存协程；最后两个使用支路消失后，整退旧 panic/recovery 工具包。不按三个包、文件数或测试机械拆 PR，不新增用户功能或内部试用前置。

当前根因不是缺少替代执行器：`platform/workerpool` 最后一个包外生产依赖已随 #897 退出；SHEIN 的现役工厂明确选择内存服务，却仍创建保存队列和 goroutine。当前 production adapter 只返回清理接口，没有 Close 交接；两个原授权品牌测试也没有 Close。应去掉无用生命周期，不通过补 Close port、全局 scheduler、worker 或文件配置兼容层保活它。

三组同批覆盖：

1. `internal/platform/workerpool` 的六个生产文件及两个专属测试。
2. `internal/shein/content` 的闲置构造器、路径查找/文件读写、保存队列/协程/关闭、专属日志尾部；保留有效清理、内存词表、元数据和原互斥锁。
3. `internal/pkg/recovery/panic_handler.go` 的全部接口/包装/恢复函数；只能在前两组准确消费者退出后删除，不搬包或重建 wrapper。

本轮已按用户确认整批实施；设计阶段未提前修改生产路径/测试。全程不运行旧工厂或触发文件保存；不读既有凭据、私密配置/词库/日志/Cookie/Profile/ignored 状态，不操作既有实例、数据、文件或进程，不执行应用/worker/provider/PG/Redis/浏览器入口。实施已授权；合并、部署、关单另授权，#301 保持 OPEN，#29 不重开。

## 2. Design Basis、Authority 与不变量

设计阶段 **Design Basis: N/A**，仅只读调查和三文档；设计 CI 与一次用户授权独立只读设计复核已完成。实施 **Design Basis: Reuse Existing Architecture**：复用已批准的当前 content 内存清理 owner、现有 `sync.RWMutex`、Logrus 注入和标准库 `time`；原文件分支/工作池无现役消费者，不转交任何新持久化 owner。用户已确认 §1–§7（含 §4.1/§4.2/§6.1），Issue/PR 的 Reuse Existing Architecture / Ready 已在修改源码前读回。

适用依据：

- [Greenfield](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)：只 EXTRACT/RETIRE，无旧状态迁移、兼容、fallback、双读/双写或第二事实源。
- [#833 旧数字内容配置边界](legacy-numeric-content-management-closure-retirement-boundary.md) §3–§5：当前 sanitizer 直接用原 content 内存工厂，默认词/显式自定义词/品牌/字段/语言/SKC 保留，不恢复数字配置仓库。该文 content 全零改与测试 Close 的要求是其批次冻结边界；本设计确认后，仅在下面准确尾部上 supersede，其余规则和历史正文不改。
- [#897 旧执行骨架边界](legacy-runtime-skeleton-config-closure-retirement-boundary.md)：现役 Module/Registry 路由合同继续 KEEP；其中对 Platform 工作池的广泛 KEEP、不全局禁止该依赖的表述，仅在本设计确认的闲置包范围被 supersede。真实采集 browser pool、Temporal/ImageAgent、独立 SHEIN 登录/Store 同步不属于这个包。
- [Project Target Architecture](project-target-architecture.md)、[Project Boundaries](project-boundaries.md) 与 [独立 SHEIN 持久化边界](shein-worker-numeric-persistence-retirement-v1.md)：路由、授权/组织、Store、真实 worker、fence/UNKNOWN 和持久化 owner 不变。

当前 Must：完整闭包；当前默认词库/显式词表、品牌保护、清理/校验返回与 JSON、元数据字段和值、内存互斥/防御复制保持；无文件/协程生命周期或替代服务；workflow/inventory（CURRENT/历史/计数/ceilings/parser/准确性）及其它检查零改；保留全部当前能力和独立登录/Store 同步，无真实资源操作或试用前置。

Should：原 guard/原测试设施、现有锁与标准库解决，不新造框架。Out of Scope：整个 content/采集/Platform logging 退休、词库管理 UI/API/持久化、默认新增禁词/自动策略、配置迁移、通用 panic 或并发治理。Accepted Risks：无新增；运行/用户验收 NOT_RUN，物理文件/进程状态 UNKNOWN。UI/IA/Figma N/A。

本批不新增 schema、持久化事实、领域状态机/事务/补偿/恢复、权限/租户/计费/provider 或外部副作用；内存更新留在同一对象/原互斥锁，不新建领域 owner。若实际必须改变这些边界，停止该部分正式实现并升级 Independent Architecture，不靠拆小批规避准入。

## 3. 实际消费者：contract → implementation → injection → consumer

| 对象 | 全仓 Git 追踪 Go 的实际证据 | 同批决定 |
| --- | --- | --- |
| platform/workerpool | 无包外生产或测试 import；全仓包路径文本仅原检查/历史记录。Pool/Worker/Processor/WorkerJob/JobHandler/QueueStats/指标/配置及旧生命周期测试相互构成孤立包 | 整包 RETIRE；当前采集实际 browser pool、共享浏览器和 Temporal/SHEIN 不动 |
| NewSensitiveWordServiceInMemory(logs) | 唯一包外 production 构造在 `publishing/shein/submit_prep_sensitive_adapter.go`；五个测试构造点为 cleanup_context_test 三处、processor_authorized_brand_test 两处 | 工厂/签名、显式 logs 注入和当前 sanitizer interface KEEP；不新增 factory/Close port |
| NewSensitiveWordService、两个 WithPath 工厂、ReloadConfig | 除自身定义/内部委派外，无生产/测试调用 | 全退，无旧别名、弃用空壳或文件 fallback |
| 文件 load/save/resolve 路径 | 仅旧 WithPath/Reload/保存支路；所谓 InMemoryWithPath 的 load 在路径缺失时仍可 MkdirAll，不能把它当无副作用预检 | 整个路径寻找/读写/默认文件创建退出，不调用它探测已有词库 |
| startSaveWorker/saveQueue/stopSave/wg/Close | 内存工厂也启动；mutation 入队后仅更新内存元数据，生产 adapter 不提供 Close，只有 cleanup_context_test 的三个 defer Close | 删除协程/队列/关闭和仅生命周期断言尾部；保留内存更新，不能留 no-op Close |
| pkg/recovery | 仅 pool.go 的 Recover、worker.go 的 RecoverWithCallback、word.go 保存协程的 Recover；其它导出包装无包外调用 | 两个消费者支路同批退出后整包 RETIRE；不把 Recover 包在同步清理周围保活 |

搜索覆盖根模块/独立 debug/tools、测试及跨 OS/build-tag 的追踪文本/导入和符号，不只看当前 GOOS。外部仓库/真实运行态没有读取；若出现具体当前外部可观察兼容证据，按 §7 报告，不能从旧导出函数/测试推定兼容义务。

## 4. 持久化、副作用和内存 owner

| 边界 | owner / 准确变化 | 保留/失败语义 |
| --- | --- | --- |
| 旧 Pool channel/数字 WorkerJob/指标/启停 | 无当前消费者的进程内旧执行模型，无 durable owner | 只删追踪源码；不停止既有进程、不 drain/取消实际任务，不替换真实队列 |
| 词库 JSON/路径/目录 | 旧 WithPath/load/save 是唯一源码 owner，无当前生产注入 | 无继任持久化 owner；不读取、删除、迁移或清理任何已有文件。新源码没有这条读写/Reload 协议 |
| 当前 StaticWords/DynamicWords | 当前 content 对象内存 + 原 RWMutex | 默认词/语言分类、归一化、去重、清空、授权品牌和全部返回/JSON语义保留，不增词、不新增自动加载 |
| LastUpdated/Version | 当前 SensitiveWordConfig 四字段继续存在；从旧 save 的纯值分支选择性抽取到原 owner | 保持 `time.Time`/`last_updated` 和 string/`version`，初始/更新 Version="1.0"，不增加持久化/哈希/序列号合同 |
| 当前路由/Store/worker | 原 owning HTTP API、domain/runtime/persistence owner | auth/tenant/状态/事务/retry/UNKNOWN/fence、凭据/Profile、平台/模型副作用均不变 |

### 4.1 保留元数据，去掉保存生命周期

在原 config.go 以一个局部 `markConfigUpdatedLocked`（名称以实施时同义局部 helper 为准）承接两项赋值：`LastUpdated=time.Now()`、`Version="1.0"`。调用者已持有原 mutex 写锁；它不再加锁、不 I/O、不入队、不启动 goroutine，不返回伪造保存成功。不是新 service/adapter/事实源。

- 初始化继续用原 createDefaultConfig，默认词、空动态 map 和初始元数据不变。
- addSensitiveWords 仅实际新增词时更新；addWordsByLanguage 仍在原非空输入/有效 map 的更新点触达元数据；ClearDynamicSensitiveWords 保持即使已空也触达；不用改变词库规则让测试通过。
- HandleValidationErrors 保留提词→内存追加→当前 payload 重新清理/返回；将原显式 save 对应的元数据触达放在原 mutex 写锁内，再释放锁后调用 ProcessProductData，避免嵌套读锁/死锁。仅删除文件保存专属错误/日志，不吞掉当前清理错误。
- 原排队后最终更新改为同一内存临界区内同步完成；这一时机变化明确纳入设计。方法完成即可读取已更新元数据，不要求保留后台任务次数、日志时序或两次更新时间戳差值。

### 4.2 同一现有锁保护 map 选择和更新

现有 AddDynamic/AddStatic 将 `s.config.*Words` 在取锁前作为参数读取，ClearDynamic 在写锁内替换 map；不能把这个锁外选择问题原样带入提取后的元数据更新。此为当前内存保护的局部实现修正，不新增同步协议。

公开 Add* 方法在原 mutex 写锁内选择字段/调用原算法；两个私有 mutation helper 可明确改为 Locked 形式并去掉内部重复取锁。字段选择、写入和元数据属于同一现有临界区；公开签名、默认行为及算法不变。Get* 原读锁/防御复制保持，Process/校验/日志不得在持写锁时重入读锁。并发必要测试仅复用原 content 测试，覆盖 Add/Get/Clear 与元数据，无新 runner/故障平台或试用门槛。

## 5. 完整删除、保留与抽取矩阵

| 准确工件 | 已确认整批处置 | 必须保留 |
| --- | --- | --- |
| internal/platform/workerpool/config.go、interfaces.go、metrics.go、pool.go、types.go、worker.go | 六生产文件整退：Pool/Worker/Processor/WorkerPool/WorkerJob/JobHandler、PoolConfig/默认/校验、QueueStats、Metrics/Snapshot、ErrQueueFull/ErrPoolClosed 和全部启停/钩子/队列日志 | 真实采集 browser pool、shared browser、Temporal/ImageAgent/SHEIN worker及其关闭/恢复/资源保护零改 |
| internal/platform/workerpool/pool_lifecycle_test.go、worker_test.go | 两个完整专属测试随旧 owner 退休；不迁移旧数字任务执行协议 | 其它现役 worker/采集/并发/安全测试不动 |
| internal/pkg/recovery/panic_handler.go | 最后三个导入点退出后，全部接口/LoggerPanicHandler/constructor/method/Recover*、SafeExecute*整退；包目录缺席 | 不删/改同名当前业务 recovery、Temporal/Store UNKNOWN owner，不抽取新共享 panic 框架 |
| internal/shein/content/config.go | 退 resolveConfigPath/loadConfig/saveConfig/saveConfigUnlocked/ReloadConfig/saveConfigAsync/Close；退 service 的 configPath/persistConfig/ctx/saveQueue/stopSave/wg 及独占 context/json/os/filepath/jsonx imports；保留两项元数据更新于本 owner 的局部 Locked helper | SensitiveWordConfig 四字段/JSON keys、service.config/mutex/logger、createDefaultConfig/默认词/内存 map 和日志依赖 |
| internal/shein/content/word.go | 退无调用别名 NewSensitiveWordService、两 WithPath 工厂、startSaveWorker；当前内存工厂仅去生命周期初始化；HandleValidationErrors 只转内存元数据触达，原 mutators 去保存并按 §4.2 选 map/更新 | NewSensitiveWordServiceInMemory(logs)、全部词表/查询/清理/校验语义、原 logs 注入；CleanupContext/品牌/字段/语言/SKC 规则不变 |
| internal/shein/content/validator.go | 仅 addWordsByLanguage 及调用方式按 §4.1/§4.2 转为同锁内存 mutation/元数据，移出文件保存尾部 | 提词、正则/语言、分割、去重和 addWordsToConfig 原算法/返回/日志不变 |
| internal/shein/content/utils.go | 退仅文件 loader 使用的 logConfigLoadStats；logSensitiveWordStats 仅删配置文件路径日志 | 所有纯分类/清理/品牌/去重/计数，当前内存词表/最后更新时间日志保留 |
| internal/shein/content/cleanup_context_test.go | 仅退三个 defer service.Close；在原文件加入最小内存/元数据/同锁并发保留断言 | 原三个测试的品牌/普通字段、nil/空上下文、语言/SKC JSON、显式自定义短语且不改默认词的全部断言原样 |

以上 14 个准确源码/原测试工件（含九整退文件），加 §6.1 原 guard 和三文档同一批；不修改 publishing adapter、品牌/纯规则文件或另开辅助 PR。processor_authorized_brand_test 原测试完整零改；不是通过删掉业务断言解决 goroutine。

```text
Legacy decision: RETIRE — 工作池、文件持久化/保存生命周期和 recovery 包
Reusable behavior / EXTRACT: 仅从旧保存纯值分支保留 LastUpdated/Version 更新，仍在原 content owner/原 mutex；现有词库/清理算法 KEEP
Current owner: content 内存清理对象；当前路由、采集/Temporal/SHEIN/Store 各 owner 不变，旧文件/执行模型无继任 owner
Cutover/deletion condition: 设计确认/Ready读回后，先原guard RED，再三组及专属尾部同批退出，原有效断言/CI/最终独立复核通过；不等于物理文件/进程清理
```

## 6. 必要验证与证据

设计阶段只做文档/链接/当前索引与原适用检查、准确设计候选原 CI 和一次只读设计复核。不为纯文档制造业务 RED、不调用旧 WithPath 工厂、不运行工作池测试来维持已拟退休协议。当前源码仍存在是设计基线，不写成已退休 GREEN。

实施获准后，复用原退休 guard 捕获 RED；原 content 测试补必要 RED/保留断言，再按矩阵完整退出至 GREEN。仅现有 synthetic/内存 fixture：默认词不变、显式词/去重/语言/清空、元数据 Version/字段类型和方法完成后时间更新、查询防御复制、Add/Get/Clear 同锁并发；原 nil/品牌/语言/SKC JSON 与 publishing 原字段/品牌/清理路径保持。使用原 race 支持验证受影响的内存并发，不增加新验收平台或完整理论故障矩阵。

必要开发自检为受影响 content/当前 publishing 测试、原退休/Phase2/CURRENT/索引检查及三个当前命令依赖/编译；不运行产物或 provider。现役 browser/Temporal/SHEIN/Store 未变化测试由原 owner/CI继续约束，未变化证据可明确复用；不把孤立工作池的旧启停测试替代真实 worker 的保护。

### 6.1 原检查、测试、inventory、文档与 workflow 精确处置

本矩阵已获用户确认，实施只应用以下准确边界；其它冻结 KEEP 不变：

| 对象 | 实施获准后的准确处置 | 不得改变 |
| --- | --- | --- |
| tests/listingkit_backend_retirement_test.go | 在原 AST/decoded-import/追踪文件设施增加一个本闭包 guard：两退休目录/九文件缺席、全部追踪Go不得再导入两退休包；content准确旧顶层工厂/receiver方法/生命周期字段缺席，production content无保存GoStmt或os/filepath文件入口；正向保护内存工厂、配置四字段/JSON、config/mutex/logger和有效算法owner | 原 #897 路由/当前日志/配置正向保护、#833 adapter显式logs AST调用约束、其它原退休/读取/解析fail逻辑原样，不全局禁用 Recover/Worker/Close 等当前同名能力，不 missing-skip/dummy/新runner |
| content/cleanup_context_test.go | 仅 §5 三个旧 Close defer退出；最小内存词表/元数据/并发断言就在原文件，不导入退休包 | 现有三个测试全部有效业务断言、processor_authorized_brand_test及其它当前发布/清理/授权/JSON测试不删不弱化 |
| phase2-runtime-inventory.md / CURRENT与历史 | **零改**：#897已去掉最后workerpool CURRENT边，本轮没有新增或删除CURRENT行 | 四条现有CURRENT消费者、历史/计数/workerpool ceiling23、accuracy/parser/排除规则全部不改；旧历史路径可继续作为守卫target |
| Phase2/legacy_hard_cut/import/provider/SDK/SSA/CodeHealth/.golangci.yml等其它检查 | **零改**；允许的包被整退不要求删历史ceiling或解除禁入 | 不改构建图/边数/上限/例外/allowlist，不用新检查代替原检查求绿；真实失败先查准确日志 |
| 本文、Architecture README、Legacy Register | 仅当前三文档新增/索引与精确supersession | 原 #833/#897及其它历史正文/矩阵/计数保留；当前路由、logging/实际API合同不变 |
| 全部 workflow/命令/部署/依赖/锁/保护 | **零改**，原检查/目标/条件继续 | 不删/skip门禁，不改镜像/tag/flag/runner，不强推/管理员绕过，不读取凭据或操作实例 |

若原检查实际要求矩阵外编辑、出现遗漏消费者、必须改变冻结 KEEP/当前安全或公共合同，先给出精确提案及原因；不能自行校准 inventory、旧检查或扩大删除。

## 7. 准入、独立复核、停止及父目标边界

1. N/A 设计三文档、适用原检查/准确设计复核已完成并获用户确认；设计 CI/PASS仍不当实施或运行结果。
2. 用户已确认 §1–§7（含 §4.1/§4.2 与 §6.1）；原单 Writer 已在同一主要 PR 记录 Reuse Existing Architecture / Ready 并读回，再整批实施；不机械拆分或增加常驻实现者。需改变当前事实/授权/状态/恢复等边界则停止并升级准入。
3. 设计复核和完整源码最终交付复核分别绑定准确 HEAD；已有未变证据可复用，finding按当前Must/后果/阻塞层先分类，修复只复核对应增量。GitHub启动失败不当成PASS，也不对Running复核重复触发；未经用户授权不默认新增AI互验线程。
4. 若发现具体当前外部可观察文件合同/持久化运行态必须兼容，先报告用户；没有可执行兼容/迁移Exception，不添加临时fallback/双读/读旧JSON/空壳Close。不因旧测试或导出名称恢复休眠owner。
5. 不接管其它现有任务/PR（包括#692），不批量清理共享helper/词库默认值/当前状态、平台或实例。不把源码缺席当物理清理或用户验收。
6. #301保持OPEN，#29不重开；每次更新先新读Issue，仅改当前段，历史后缀自“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起63542字符与勾选逐字保留，正文不超过65536。
7. 实施、合并、准确实际merge的push/main确认、部署/关单各按用户授权；当前能力、独立SHEIN登录/Store同步及fence/UNKNOWN保留，不操作既有实例/数据/凭据/进程，不改workflow/inventory，不增加试用前置。
