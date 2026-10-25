> #301 / #946 **USER_CONFIRMED / SOURCE_IMPLEMENTED / IN_REVIEW**。用户已确认冻结设计a306b024a的§1–§7（含§6.1）；Reuse Existing Architecture / Ready及Issue/PR读回先于生产修改（6394663649 / 6394679078）。新managed独立工作区issue301-browser-tail-946接续同一分支/PR，四组26工件完整收齐，不机械拆批；两整退、十生产混合、五测试混合、原聚合仅加一个guard、五旧依据/当前说明及三当前文档，无新生产EXTRACT/owner。32闲置生产函数/方法、3类型及字段/注入/计数/测试/文档尾部退出；实际指纹/JSON、代理全部评分状态、质量重试/有效断言、周期健康日志、当前采集/原Amazon浏览器debug及独立SHEIN登录/Store同步保留。原guard真实RED→GREEN，其它原检查不改；原历史/计数/勾选、四旧依据及本冻结设计原Git blob完整后缀保持。workflow/完整inventory/依赖/保护及既有实例/数据/凭据/进程不动，不增试用前置，#301 OPEN。必要自检、准确最终HEAD/原CI/源码最终复核留同一PR，设计复核不替代源码复核，合并另授权。以下冻结设计全文逐字保留，其DESIGN_ONLY/待确认措辞仅为原阶段证据。

# #301 旧浏览器管理便利入口、配置旁路、实例诊断与闲置统计完整退休边界

Status: **DESIGN_ONLY / AWAITING_USER_CONFIRMATION**。本轮用户只授权设计；设计阶段 **Design Basis: N/A / Ready**，仅调查及本文、架构索引、Legacy Register 三文档。未来实施拟 **Reuse Existing Architecture / NOT_READY**；确认 §1–§7（含 §6.1）并在 Issue/PR 记录、读回 Ready 后才可改生产/测试。#301 保持 OPEN，实施、合并、部署、关单分别授权。

## 1. 用户结果、范围与 Design Basis

使用者是维护当前采集、原 Amazon 浏览器/debug 和独立 SHEIN 登录/Store 同步的开发者。当前维护问题是无消费者的旧管理入口、不可达配置分支和无读取者的统计仍让旧合同、字段、测试及说明继续保活；不声称它们正在阻塞用户采集，也不因此增加试用门禁。交付是四组完整源码退休闭包及准确保留说明，同一主要 PR 连续完成，不按 getter/helper/文件数机械拆批。

1. 共享 Manager 的两项指纹便利转发、配置摘要/性能试验和闲置 locale 查询。
2. Amazon 恒真配置判断、不可达传统配置分支、闲置配置 getter/预设信息及重复持有字段。
3. Amazon 闲置实例校验、信息查询、手工重启及最后消费者退出后的 CloseInstance 包装。
4. 无调用方的质量/代理/池统计投影，以及只服务这些投影的 recorder、字段、计数、注入、测试和文档尾部。

**必须保留**：实际指纹生成及注入、全部现役浏览器配置/JSON、代理健康评分/冷却/轮换、周期健康统计/日志、质量校验/有限重试/邮编规则、取消/deadline/失败证据/关闭结果、当前采集、原 Amazon 浏览器与有效 debug、独立 SHEIN 登录和 Store 同步及原 owner/claim/fence/UNKNOWN。

非目标：退休爬虫、改变配置策略或指纹算法、改进健康恢复协议、替换日志/指标平台、增加 telemetry/exporter/管理 API、兼容旧统计 JSON、迁移/清理真实状态、修改 workflow/inventory/依赖/保护、运行浏览器/provider/安装器、访问既有配置/凭据/数据/进程。没有新用户操作、UI/Figma 增量、当前事实 owner、事务、状态机、授权/租户、计费或副作用边界；Figma Authority **N/A**。

未来实施复用原 crawler/amazon、shared/browser、SHEIN runtime 的当前 owner 和本文件引用的已批准合同；不改仍被消费的技术合同/行为。移除内部无消费者的便利/诊断方法不产生新领域合同。范围共26工件、单一浏览器采集能力边界；没有新架构阈值命中需要发明额外系统。若真实 diff 必须改变上述边界，先报告并重新判断 Independent Architecture，不以本文预授权。

## 2. 权威、基线与前批 KEEP 的准确替代

产品权威为用户本轮四组完整闭包设计指令、[Greenfield 决定](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)及 [AGENTS.md](../../AGENTS.md)。只有 `EXTRACT | RETIRE`；无旧数据迁移/compatibility、fallback/双读写或第二事实源。

调查基线为 actual main **8eba97e73748d07ad877cb988aa509195fae099b**，tree **775dfcd0078b54ff105b9fbb06181fc5d6aa26a1**。#942 已合并且准确 push/main CI40167654475 SUCCESS；那是前批证据，不是本批设计/源码验收。本轮唯一 Writer 会话01a133ef-757c-73e3-b8f6-76958c642397，复用独立 worktree/新 codex 分支；活跃 #945 仅 capture 扩展与 sourcing fixture，不接管该 owner；#931/#692 无本批源码重叠。

下表是**待用户确认的精确替代提案**，当前设计阶段不修改四份旧依据；确认实施时仅前置窄注，原完整 Git blob/历史数字/勾选/证据逐字保留。

| 前批依据 | 本批拟替代的 KEEP 含义 | 仍有效的保留决定 |
| --- | --- | --- |
| [#867 数字采集尾链](legacy-numeric-acquisition-tail-closure-retirement-boundary.md) | qualityMetrics 及从 service_metrics 原样抽出的 normalizeMetricsRegion/copyInt64Map 曾有真实消费者；其最终快照现无外部调用，拟连唯一 owner 整退 | CookieData/buildCookiesFromJSON、当前 SessionTransport/采集/JSON 与其它共享规则不动；不回滚当时正确的抽取证据 |
| [#893 日志门面](legacy-logger-facade-closure-retirement-boundary.md) | 对原 browser/shared 构造、辅助和诊断的广泛 KEEP，仅在 §5 精确声明/字段/遥测注入内替代 | 原 Logrus 注入、当前构造/导航/安装/关闭/健康/失败日志和全部其它业务行为保持；不让现役入口静默 |
| [#901 来源适配层](legacy-amazon-source-adapter-closure-retirement-boundary.md) | 原 Processor 全方法/池/诊断的广泛保留，仅退出无人调用的三统计方法和 §5 专属尾部 | NewAmazonProcessor、Process/单批/context/Shutdown、有效池大小与 factory_test、原浏览器/debug 和当前 acquisition owner 保持 |
| [#822 Amazon 刊登闭包](legacy-amazon-listing-closure-retirement-boundary.md) | shared metrics 不因同名删除的保留条款，按本批消费者证据只替代质量快照链和诊断投影 | 不按名字整退 Task/metrics/worker；保留真实健康评分、健康日志、采集/共享 owner；其它 KEEP 不受影响 |
| [当前 Amazon runtime flow](amazon-crawler-runtime-flow.md) | “qualityMetrics 保留/region normalization/map copy 归此消费者”及外层池统计说明不再作为保活依据 | 当前应用 acquisition 与原 debug 两路径、池生命周期/代理/质量/失败证据、SHEIN/Store 仍是各自 owner |

只有新设计获确认的准确矩阵替代旧 KEEP；旧历史结果不改为失败，不以当时无默认应用装配取消现在明确保留的原浏览器/debug。

## 3. 实际消费者：contract → implementation → injection → consumer

全追踪3260 Go 文件做 AST 标识符/声明调查（含测试、独立 debug 模块与不同 build tag 源），结合全仓文本/非 Go 配置检索和 receiver/字段调用核对。标识符同名不等于同一方法；未运行应用、业务构造器或真实资源。静态无调用不证明既有进程退出。

| 链路 | 真实消费者/断点 | 决定 |
| --- | --- | --- |
| shared.Manager.GenerateRandomFingerprint/GenerateStableFingerprint → Manager.generator | 两 wrapper 无外部调用；generator 字段仅为它们持有；NewManager 构造了第二个无实际用途的 generator | wrapper、字段及构造赋值同退；不是删除真实 FingerprintGenerator |
| BrowserPool.fingerprintGen → GetFingerprintGenerator → InstanceManager.CreateInstance → FingerprintGenerator 两生成方法 → SetFingerprint | CreateInstance 的 random/stable/default 分支真实调用，继而 Manager.Launch → ContextLauncher 使用 fingerprint | 整条 KEEP；fingerprint.go 全文件、种子、GPU/WebRTC/languages JSON 原样 |
| SHEIN automation → shared.NewManager → SetFingerprint(buildAutomationFingerprint) → Launch | 独立 worker 直接注入其当前指纹，不经上述闲置 wrapper | 独立登录/自动化/Profile/Redis/Store owner 与调用全部 KEEP |
| ShouldUseRandomConfig → NewBrowserManagerWithConfig | 唯一调用返回恒 true，外层 else 永不可达；RandomConfig.Enabled 仍决定是否覆盖策略/预设 | 只删恒真判断/死分支，直接执行原可达分支；保留 Enabled 内层判断和 GenerateBrowserConfig/LogConfigStrategy 的原顺序 |
| BrowserManager.GetConfigManager / config / configManager；ConfigManager.GetPresetInfo | getter、信息投影无调用；两持有字段无其它读取 | 同退闲置字段/赋值，不退局部 ConfigManager/configGen/presets、GetAvailablePresets（被 LogConfigStrategy 调用） |
| PrintConfigSummary / TestRandomConfigPerformance / GetLocaleForRegion | 全追踪源无调用；performance 是生产 .go 内的未调用试验，并非 go test | 精确 RETIRE；实际随机/stable/windows/preset、ValidateConfig、IP 时区查询 KEEP |
| InstanceManager.ValidateInstance → GetInstanceInfo | 只有自身信息查询和专属测试；无生产消费者 | 两方法及旧 map 形状测试退出；“无活动 context 不可创建页面”有效断言转接已存在 Manager.NewPage |
| RestartInstance → CloseInstance → closeManagerWithTimeout | RestartInstance 无调用，CloseInstance 只由它调用 | 前两方法连退；closeManagerWithTimeout 仍被同步/异步重建调用，必须 KEEP |
| AmazonProcessor.QualityStats → qualityMetrics.Snapshot；构造→字段→SetQualityMetricsRecorder→InstanceProcessor.Record* | 最终 QualityStats 零调用；其余只有生产写入和测试读取，无 exporter/HTTP/debug 消费者 | 整退无读者的生产/注入/计数/快照/helper 链；真实质量重试/上下文决策/日志原样 |
| AmazonProcessor.ProxyStats → BrowserPool.ProxyStats → ProxyPool.Snapshot | 外层零调用；共享 Snapshot 另只有专属测试 | 三层投影同退；ProxyPool 的全部评分状态、Acquire/MarkSuccess/MarkFailure/评分函数原字节保留 |
| AmazonProcessor.PoolStats → BrowserPool.PoolStats → poolStats/ActiveRebuildCount | 外层仅一专属测试；中层另有计数测试。五累计值、三 recorder 与 ActiveRebuildCount 无其它读取 | 投影、只写计数/字段/构造/调用、统计 getter 同退；rebuildingIDs/锁/去重/回收/重试全保留 |
| StartHealthCheckRoutine → performHealthCheck → LogPoolStats → HealthChecker.GetPoolStats → GetInstancesSnapshot/HealthCheck | 是实际周期任务，有真实日志与健康恢复消费者 | 全部 KEEP；不能把同名 PoolStats 当作无用统计整体删除 |

关键区别：ProxyPool 的 assignments/successes/failures/cooldowns/lastSuccessAt/lastFailureAt/cooldownUntil 都参与当前健康评分/选择。即使 Snapshot 退出，字段和写入仍是有效状态，零改。相反 poolStats 五累计值和 qualityMetrics 只服务闲置快照，必须连生产侧收齐，不能只删 getter 留下死链。

## 4. 配置、持久化与副作用 owner

| 边界 | 当前 owner / 本轮准确变化 | 必须保持 |
| --- | --- | --- |
| 浏览器配置/指纹 | core/config 与 shared BrowserConfig/FingerprintConfig，Amazon ConfigManager/BrowserPool，SHEIN automation；只去无读字段/不可达分支 | 配置字段/default/env/样例/JSON、random/stable/windows/preset、seed、Enabled 策略覆盖、日志和 launcher 参数全部不改 |
| 实例/页面/关闭/重建 | 现役 Manager/BrowserPool/PoolManager/InstanceManager/HealthChecker；只去无人调用的管理入口及统计写入 | Create/同步异步 Recreate、rebuildingIDs/锁/去重、UpdateInstance、channel 满/关闭处理、retry/backoff/超时/取消、CloseWithResult/UNKNOWN 全保留，不修改原恢复协议 |
| 代理状态 | shared.ProxyPool 的内存评分状态，BrowserPool 获取/成功/失败标记 | 所有评分字段和写入/时间戳/冷却/健康选择/round-robin 保留；Snapshot map 只是无消费者的投影，无需新 owner |
| 质量校验/重试/目标上下文 | InstanceProcessor/resultValidator/ZipcodeValidator/FailureArtifactStore | 返回产品/错误、有限重试次数、reload 次数、US 默认邮编/JP 不回退、原日志/失败证据/取消/deadline 不变；只删 Record* 块与专属 hadRetry 记账变量 |
| qualityMetrics/poolStats/诊断 map | 纯进程内计数与临时 map，无 SQL/Redis/文件 schema，无当前持久化 JSON consumer | 整退不迁移/保留快照 JSON，不新建替代 store/exporter；不删除真实日志/结果文件 |
| 商品/lineage、SHEIN/Store | 当前 acquisition/dataacquisition/Sourcing/Catalog、独立登录与 Store 各 owner | 身份/tenant/授权/绑定/claim/fence/UNKNOWN、事务/幂等/计量/保存/同步不变，不读取或操作既有资源 |

没有新增持久化、事务或跨系统一致性问题，不需要 outbox/Saga/队列/迁移/兼容。删除旧 RestartInstance 源码不是执行重启/停止或确认运行态已清空。

## 5. 完整删除 / 保留 / 抽取矩阵

以下 **26工件** 同一批实施；本阶段只改最后三当前文档，其余是提案。RETIRE 32个生产函数/方法与3专属类型及字段/注入尾部；计数仅用于核对，不能取代保留行为验收。

### 5.1 两整文件 RETIRE

| 工件 | 删除内容 / 保留边界 |
| --- | --- |
| `internal/crawler/amazon/quality_metrics.go` | 整退 qualityMetricsRecorder、qualityMetrics、newQualityMetrics、五 Record*、Snapshot、copyInt64Map、normalizeMetricsRegion；后两原抽取函数已失去最后有效读取链，不搬到新 owner |
| `internal/crawler/amazon/quality_metrics_test.go` | 整退 TestQualityMetricsRetainsProcessorStatistics；只验证旧计数/快照，不代替其它有效质量测试 |

### 5.2 十生产混合工件

| 工件 | 仅退休 / 最小修改 | 保留 |
| --- | --- | --- |
| `internal/crawler/shared/browser/manager.go` | Manager 两 Generate*Fingerprint wrapper；generator 字段和 NewManager 中该赋值 | SetFingerprint、全部实际配置/指纹字段、userAgents、installer、Install/Launch/NewPage/Close/CloseWithResult/UNKNOWN 等其它实现与日志 |
| `internal/crawler/shared/browser/random_config.go` | PrintConfigSummary、TestRandomConfigPerformance 及失去唯一用途的 Logrus import | 所有 RandomConfigGenerator 类型/数据/生成算法、随机种子/time、presets、ValidateConfig 原实现 |
| `internal/crawler/shared/browser/ip_location.go` | GetLocaleForRegion 全函数及其局部 localeMap | IPLocationInfo/JSON、GetLocationInfoByIP/HTTP timeout、GetTimezoneForRegion、extractIPFromProxy 原样；不发 IP 请求 |
| `internal/crawler/shared/browser/proxy_pool.go` | 仅 ProxyPool.Snapshot | 其它类型/字段/全部函数含 calculateProxyHealthScore 原字节，不删任一评分计数/时间戳/锁/状态写入 |
| `internal/crawler/amazon/browser/manager.go` | BrowserManager.GetConfigManager、config/configManager 字段和返回值赋值；外层恒真 if 和不可达 else，保留其原 then 内容 | cfg 参数/局部 ConfigManager、Enabled 内层判断、生成与日志顺序、NewBrowserManager 两构造及导航/语言 Cookie/错误全部保持 |
| `internal/crawler/amazon/browser/config_manager.go` | ShouldUseRandomConfig、GetPresetInfo、专属 fmt import | ConfigManager/configGen/presets、生成策略/原既有默认选择、applyBaseConfig/copyBrowserConfig、GetAvailablePresets/LogConfigStrategy 不改 |
| `internal/crawler/amazon/browser/instance_manager.go` | ValidateInstance/GetInstanceInfo/RestartInstance/专属 CloseInstance；ActiveRebuildCount；同步2处、异步3处 record* 调用 | closeManagerWithTimeout 和其调用、CreateInstance/实际指纹、同步/异步重建所有分支/锁/去重/backoff/回收/日志保持；不删 rebuildingIDs |
| `internal/crawler/amazon/browser/browser_pool.go` | ProxyStats/PoolStats/三 record*、poolStats 类型和 BrowserPool.stats 字段/初始化、Initialize 中一处 recordInitFailure 调用 | 真实池初始化/错误、指纹、风险/代理、实例使用计数、周期健康启动/健康快照与日志、Acquire/Release/Shutdown 全部保持 |
| `internal/crawler/amazon/processor.go` | QualityStats/ProxyStats/PoolStats；qualityMetrics 字段、构造局部/赋值及一处 SetQualityMetricsRecorder 注入 | initErr/initMu/ensureReady/retryInit、当前 Process/Batch/context/Shutdown/错误、pool config、失败证据和质量配置注入不改 |
| `internal/crawler/amazon/instance_processor.go` | qualityMetrics 字段/SetQualityMetricsRecorder；全部 Record* 调用所在的五个纯遥测条件块；hadRetry 声明/赋值 | extractWithQualityRetry/目标上下文函数仅删这些记账语句；attempt/attempts、errors.As、产品/错误返回、prepareRetry、邮编/日志/ctx/timeout 和其它函数原样 |

### 5.3 五混合测试工件与一个原聚合工件

| 工件 | 精确处置 |
| --- | --- |
| `internal/crawler/amazon/browser/instance_manager_test.go` | 退两个 GetInstanceInfo map 形状测试；将 TestValidateInstanceRequiresActiveContext 转为 TestBrowserInstanceWithoutContextCannotCreatePage，沿原合成 Manager 调已存在 NewPage，必须返回 nil page 和“浏览器上下文未初始化”错误；不启动 browser/driver。有效 context 拒绝转接当前 owner，非删掉安全断言 |
| `internal/crawler/shared/browser/proxy_pool_test.go` | 只退 TestProxyPoolSnapshotIncludesCounters；原 round-robin/cooldown 与 health-aware 两测试的 fixture/断言完整保持 |
| `internal/crawler/amazon/browser/browser_pool_rotation_test.go` | 只退 TestPoolStatsIncludesRebuildAndInitCounters；core/config import 仍被三个保留测试的 Config/AmazonConfig/AmazonProxyPoolConfig 使用，完整保留；原三项轮换/错误释放测试及 mock/计数/归还池断言原样 |
| `internal/crawler/amazon/processor_test.go` | 只退 TestAmazonProcessorPoolStatsIncludesInitError；当前单批 init error 传播、缺 manager 拒绝和 ensureReady 重试恢复断言完整保留 |
| `internal/crawler/amazon/instance_processor_test.go` | 三测试仅删 metrics 构造/setter/快照及 metric-only switch/断言；重试恢复的产品/attempts=2/reloads=1、重试上限的 nil/错误/attempts=2、四 target-context case/US值/JP 禁调用和 fixture 原样 |
| `tests/listingkit_backend_retirement_test.go` | 只新增一个本闭包 guard，复用既有 trackedFiles/AST/decoded import/失败关闭；所有原函数/helper/import和其它断言不变，§6.1 给准确边界 |

### 5.4 五旧依据 / 当前说明与三当前文档

| 工件 | 处置 |
| --- | --- |
| `docs/architecture/legacy-numeric-acquisition-tail-closure-retirement-boundary.md` | 仅前置 §2 的 qualityMetrics/两纯 helper 精确 KEEP 替代注记；原 blob 完整作为后缀 |
| `docs/architecture/legacy-logger-facade-closure-retirement-boundary.md` | 仅前置 §5 准确辅助/诊断尾部替代；有效日志注入/原历史完整保留 |
| `docs/architecture/legacy-amazon-source-adapter-closure-retirement-boundary.md` | 仅前置本批闲置管理/统计矩阵替代，原七工件退休及 browser/debug/当前采集 KEEP 保持 |
| `docs/architecture/legacy-amazon-listing-closure-retirement-boundary.md` | 仅前置质量快照/无消费者诊断精确替代；不整体取消 metrics/worker 保留 |
| `docs/architecture/amazon-crawler-runtime-flow.md` | 精确将外层统计说明改为仍保留的周期健康统计/日志，删除 qualityMetrics 当前 owner 一条，指向本设计；其余路径/历史/运行权限不改，不顺带重写其它批说明 |
| 本文、`docs/architecture/README.md`、`docs/refactoring/legacy-register.md` | 设计阶段仅这三文档；索引/register 仅前置本批状态/链接，原完整内容保留。确认后本文只前置实施状态，冻结正文逐字保留 |

```text
Legacy decision: RETIRE
Reusable behavior: 当前指纹/配置/代理健康/质量重试/周期健康/上下文拒绝已在现有 owner；仅把一项有效测试接到 Manager.NewPage，无新生产 EXTRACT
Current owner: 原 shared/browser 与 crawler/amazon 的实际执行 owner；当前 acquisition/采集、SHEIN 登录及 Store owner 不变
Cutover/deletion condition: 用户确认 §1–§7/§6.1 → Reuse Existing Architecture / Ready 读回 → 原 guard RED → 全闭包退出/GREEN → 有效规则自检/原 CI/最终交付复核 → 另获准确 HEAD 合并授权；不是运行态清理
```

## 6. 验证与交付边界

设计阶段限全追踪 AST/调用/文档与链接核对、原 blob 保留核验、git diff --check、适用原文档/Phase2 检查及原 PR CI；不跑业务构造器、浏览器/采集、安装/下载、HTTP/IP lookup 或既有资源。设计 CI 不是源码实现或运行验收。

未来实施先用既有聚合文件内单 guard 对原代码确认 RED，整批退出得 GREEN。必要开发自检限受影响原内存/mock 测试、当前 context 拒绝/指纹与代理/质量/轮换/processor 的有效断言及原 import/文档 guard。新增配置回归仅在本批原 instance_manager_test.go 内以纯 Config/固定 preset 验证 Enabled=false 原 windows 分支及 Enabled=true 原 preset 覆盖；不调用构造之外的 browser/安装/网络，不为随机输出制造脆弱精确值断言。这些验证不改变策略/默认。

只编译当前 application、独立 SHEIN worker、1688 local-agent/public collector 和原 debug test-amazon/test-1688；共享 browser helper 被当前指纹 installer 消费时一并只编译该入口。使用任务 TEMP 输出、不执行产物；不扩张成全量验收矩阵/新 runner。稳定候选交原完整 PR CI，未变代码的有效证据明确复用，不手工重复 CI 全量检查。

源码完成、准确 PR HEAD/CI、实际复核、用户确认、正常合并和 actual merge SHA 的完整 event=push/head_branch=main CI 分开记录。真实浏览器/provider/实例/产品验收 **NOT_RUN**，既有资源操作 **NOT_AUTHORIZED**。没有新增用户步骤、试用前置或部署声明。

### 6.1 原检查、文档、inventory 与 workflow 精确处置

| 检查 / 对象 | 只允许的转换 / 必须保持 |
| --- | --- |
| 新 TestLegacyBrowserManagementTailClosureIsRetired，放原聚合文件 | 原码 RED/整退 GREEN；两 quality_metrics 路径缺席；按三个准确 owner/receiver 禁入 §5 的32函数/3类型及专属字段。Manager.Generate* 禁入不能误禁 FingerprintGenerator 同名生成方法；ProxyPool.Snapshot 禁入不能全局禁其它 Snapshot；只退 InstanceManager 的 CloseInstance，不动底层关闭 owner。遍历全追踪 Go/含 debug/build tags，原读取/解析失败即失败，不 missing-skip/dummy/空壳 |
| 同一新 guard 的 KEEP | 必须找到实际 FingerprintGenerator/SetFingerprint/CreateInstance 与 SHEIN 注入；ProxyPool 全部评分字段/Acquire/MarkSuccess/MarkFailure/selectHealthiestAvailable/calculateProxyHealthScore；HealthChecker 周期链/GetPoolStats/LogPoolStats、BrowserPool.GetInstancesSnapshot；重建锁/map/实际方法、质量重试/验证/错误与当前 debug/应用入口。字段/签名按原 owner 正向验证，不以全局同名 token 代替 |
| instance_manager_test.go | 两旧 map 测试退出、一 context 安全测试转接 NewPage；仅增加前述两项纯配置回归及所需标准/现有配置 imports。不构造 Native Playwright、不修改生产 owner 来迎合测试 |
| 其余四混合测试/一整退测试 | 只按 §5.1/§5.3 退出旧统计断言/fixture与孤立 import；有效产品/错误/attempt/reload/目标上下文/轮换/代理健康断言保持，不加 skip，不放宽 deadline/取消 |
| TestLegacyAmazonSourceAdapterClosureIsRetired / TestLegacyLoggerFacadeClosureIsRetired / TestListingKitTaskBackendSourcesAreRetired 等原聚合 | 已核对：无本批32方法/3类型的正向保活要求；#901 正向要求的是实际 Processor/Process/Batch/context/Shutdown/池大小/debug，全部 KEEP。原函数/helper/读取/遍历/断言零改 |
| 原 architecture/import/depguard/Code Health/安全/必需 gate | 没有本批准许放宽项；原逻辑全保持。当前有效四条 CURRENT 消费者仍不因本批改变，无新增 import edge，不允许用删表或 skip 处理失败 |
| 完整 inventory / workflow / 依赖 / 保护 | 全零改，包含 CURRENT/历史/计数/ceilings/检查逻辑；不更换编译目标或 CI gate |
| 文档/历史 | 只按 §5.4 校准五项，四旧依据原 Git blob 为逐字后缀，索引/register 全文后缀保留；#301 marker 后历史/旧勾选逐字保持，正文不超过65536字符，状态 OPEN |

若实施真实检查出现矩阵外冲突，先定位根因和同根兄弟路径，交出准确提案；不能因为“整批退休”自行删 KEEP、改变 guard 或添加绕过。

## 7. 顺序、评审与停止点

1. 本轮只交三设计文档和同一主要 PR，准确设计 HEAD/原 CI/复核证据留 PR；用户确认前不应用 §5 其它23工件。
2. 用户确认后刷新 main/实际消费者/重叠 PR，仅处理真实组合变化；先写并读回 Reuse Existing Architecture / Ready，再由唯一 Writer 在同一 PR 实施四组完整闭包、有效自检及原 CI。
3. 独立复核核对真实 diff、§5/§6.1、#867/#893/#901/#822 精确 KEEP 替代、无持久化迁移、代理评分/质量/指纹/周期健康与 SHEIN/Store 保留。优先复用原 GitHub 复核；同 SHA 正在运行不重复触发，具体 finding 先分类，修复只复核增量。**未经用户本轮授权不默认新增 AI 互验线程**；复核不可用如实记录并交用户决定，不借旧批授权派代理或以作者自检/CI 冒充独立验收。
4. 完成后交准确实现 HEAD、完整原 CI/最终复核和限制，等待另行合并授权；授权后仅正常 expected-head 合并并确认实际 merge SHA 的完整 push/main CI。不得自行合并、关闭 #301、部署或执行真实验收。

出现真实当前消费者/外部可观察持久化合同、必须改现役指纹/配置语义/代理评分/质量重试/健康恢复/取消/授权/事实 owner/副作用边界，先报告具体阻碍、根因、最小改动和成本，暂停受影响部分。保持当前 Must，不派生无关工具/治理或增加试用前置；#301 始终 OPEN。
