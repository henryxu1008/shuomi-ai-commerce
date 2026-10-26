> #301 / #961 精确替代顶注：按[legacy-publishing-support-policy-closure-retirement-boundary.md §1–§7（含§4.2/§6.1）](legacy-publishing-support-policy-closure-retirement-boundary.md)，旧 publishing/common 与 jsonx 根、十一闲置 SHEIN 策略族及三个混合策略尾部已在源码完整退出。当前八值合同分归已有 offlinepackage / publishing.Price，本地组装纯行为归 draft；本批表列 KEEP 由该设计矩阵精确替代，其它当前能力/JSON/hash算法/版本/事实owner及独立SHEIN登录与Store同步保持。仅未来新多属性数组固定按原始名排序，不重写已有载荷。以下原文、数字、勾选与历史证据逐字保留，不作为恢复旧owner或运行操作的授权。

> #301 / #939 **SOURCE_IMPLEMENTED / IN_REVIEW**：按[用户确认的旧全局配置与日志便利尾部完整闭包](legacy-global-config-logging-tail-closure-retirement-boundary.md) §1–§7/§4.1/§6.1整批退出旧core/config HTTP/Retry门面、根Logging/Redis/Prompts投影及其专属尾部，两个闲置日志方法及对应保活；仅准确替代本文涉及这些对象的广泛KEEP，不撤销整个Config/日志/Redis能力。有效加载/弃用日志仍消费platform/logging，完整inventory/CURRENT零改；RedisConfig类型、SHEIN私有配置/原生Redis/登录与Store同步、当前日志/Retry/HTTP/AI/图像/JSON/采集保持。port-forward仅去两条旧Redis建议输出，其余转发/参数/Pester和#357不改。原完整Git blob及历史/计数/批准/失败逐字保留为后缀；workflow/依赖/保护/既有实例/数据/凭据/进程不动，不增试用前置，#301 OPEN。

> #301 / #934 **USER_CONFIRMED**：[旧闲置 adapter/helper 完整退休闭包](legacy-idle-adapter-helper-closure-retirement-boundary.md) §1–§7（含 §6.1）已获用户确认，Reuse Existing Architecture / Ready 与 Issue/PR 读回先于源码修改。仅替代 UnmarshalBytes 的 KEEP/原正向保活行：#926 已退休最后旧 Amazon API 消费者，现仅删除该孤立泛型函数；UnmarshalString/CleanLLMResponse、严格 JSON/版本/hash/读取拒绝、原浏览器 Processor 与其它有效能力保持。只按准确矩阵替代本项旧 KEEP；以下原文/已有顶注/数字/勾选/批准/历史证据完整逐字保留。workflow/完整 inventory/依赖/保护及既有实例/数据/凭据/进程零改，保留独立 SHEIN 登录与 Store 同步，不增试用前置，#301 OPEN。

# 旧闲置配置、并行管理与 JSON 辅助合同完整退休闭包设计

Refs [#301](https://github.com/qq550723504/shuomi-ai-commerce/issues/301)。#301 保持 OPEN，#29 不重开。

- Status: **USER_CONFIRMED / SOURCE_IMPLEMENTED / FINAL_REVIEW_PENDING**。用户已确认设计 c1f956766 的 §1–§7（含 §6.1），同一 PR #914 整批实施；准确实现 HEAD、原 CI、最终交付检查与另行合并授权留 PR。无真实运行或产品验收结论。
- Design Basis：设计交付为 **N/A**；当前整批实施为 **Reuse Existing Architecture / Ready**。用户确认与 Issue/PR Ready 记录读回先于源码修改，复用当前 Config/HTTP/Retry、浏览器 Processor、context/time、JSON/LLM/严格 codec 及 Store/登录 owner，不新增领域合同、状态/权限/事实/恢复/副作用边界。
- 下列 §1–§7 保留已确认设计的矩阵和阶段时态；源码状态以上述 Status 和主要 PR 的准确证据为准，不为滚动证据反复改仓库。
- 调查基线：main **2bbaea9027a4a529fb77e1d96a3df54ad13faa34**。唯一 Writer 使用原独占 worktree、分支 codex/issue301-idle-support-contract-closure-design，无新增实现者或 AI 线程。起始 main 已同步；#913 四个扩展/交接文件不触及本矩阵。

## 1. 用户结果、产品依据与完整范围

使用者是维护当前采集、平台规则、登录与 Store 源码的维护者。交付结果是一份可确认的四组完整删除/保留矩阵：移除已失去当前需求与消费者的旧配置/管理/序列化分支，保留同包真实能力，消除历史 KEEP 检查对闲置声明的保活。没有当前核心用户操作被这次整理阻塞；它不是内部试用的新前置，不建设配置、并发或 JSON 产品。

一个 Delivery Batch / 一个主要 PR 一起覆盖：① core/config 的孤立 CostCalculationConfig、TimeoutConfig、RateLimitConfig 及专属测试；② pkg/goroutine 未使用 Manager 分支；③ pkg/timeout 无消费者常量/包装；④ pkg/jsonx/marshal.go 全部未使用的序列化/格式/校验/写文件 helper。范围包括原 guard 和直接冲突的旧 KEEP 文档尾部，不按包或文件机械拆批，不删除四个仍有保留侧的包。

Product Authority 为用户本轮“先设计、确认后实施”的明确范围、[全新系统产品基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[Legacy Register](../refactoring/legacy-register.md)、[Issue 驱动规则](../engineering/issue-driven-development.md) 及当前各消费 owner。UI/IA/Figma N/A，没有页面或交互变化。既有源码和旧测试只是调查证据，不自行构成当前产品需求；本矩阵获确认才窄替代 §2 的旧 KEEP。

Must：四组连同专属尾部整批收齐；当前采集/原 Amazon 浏览器与 debug、图像、平台 API/DTO/规则、HTTP/Retry/日志/配置加载、草稿/严格解码/规范化/hash/版本、共享合同、独立 SHEIN 登录与 Store 同步保持。Should：复用原 AST guard、已有测试和原 CI，不复制通用 helper 或制造替代库。Out of Scope：其它配置治理、整体搬包、统一超时/并发框架、新 provider、旧 JSON 迁移/兼容、业务数据清理、workflow/inventory/依赖/锁/保护修改、真实环境操作。Accepted Risks：无新增；既有物理状态 UNKNOWN，真实环境与产品验收 NOT_RUN。

Threat Model 限于误删混合文件中的当前行为、把同名类型/字段当全局禁词、误改当前严格解码或并发/超时语义、误把源码删除当历史数据处置。此前“全部 Cookie API 退休”提议已被用户撤回，本批不恢复该范围。

## 2. 根因、Design Basis 与前批 KEEP 的替代

根因是旧数字刊登/宿主产品消费者已退出，零调用声明仍由宽泛的 helper 暂留要求和存在性 guard 保活。无需给旧 helper 安排新消费者、搬目录、包一层服务或新造公式。无消费者不代表算法数学错误；当前没有要求把它们变成新产品能力。

| 原依据 | 当时 KEEP | 本次拟替代的准确边界 |
| --- | --- | --- |
| [#867 数字采集尾部](legacy-numeric-acquisition-tail-closure-retirement-boundary.md) | 删除旧 pricing.CostCalculator 时指向 core/config 成本纯公式 | 仅 CostCalculationConfig/default/CalculateTotalCost 与专属测试退出；不表示当前平台定价 API 或成本字段可删，不复制原固定成本/百分比公式 |
| [#819 TEMU 整链](legacy-temu-listing-closure-retirement-boundary.md) | platform_common.go 中保留 CostCalculationConfig | 本批该文件只剩这条孤立链，可连唯一测试整退；同名产品/平台 DTO 和当前校验不动 |
| [#897 执行骨架/配置](legacy-runtime-skeleton-config-closure-retirement-boundary.md) | TimeoutConfig/RateLimitConfig/HTTPClientConfig/RetryConfig 及 defaults | 仅前两对及四项专属测试转 RETIRE；HTTP/Retry、Config/LoggingConfig、现役 Registry 路由与 platform 日志继续 KEEP |
| [#870 宿主运行层](legacy-host-runtime-closure-retirement-boundary.md) | pkg/timeout 和共享 helper 暂留 | 仅 §5 列出的闲置 timeout 声明、goroutine.Manager 和 jsonx/marshal.go 退出；三个包仍有效，不能整包禁入或借此删 current caller |

拟实施 **Reuse Existing Architecture** 的轻量依据：当前配置仍由原 Config/Viper/加载/环境覆盖/校验持有；并行处理由原 Processor 持有；timeout 仍为标准 context/time 值；JSON 仍由各当前 decoder/codec 和 LLM 清理 owner 持有。调用链见 §3；授权/租户、事实 owner、状态机、事务、幂等、retry/UNKNOWN 和外部副作用均不改变。只有 15 个拟实施工件，无新增生产代码，不命中新边界；若发现真实消费者或持久化/权限合同必须变更，停止相关删除，报告后升级准入，不在实施中临时设计兼容或迁移。

这不是“cmd 没引用就可删”的规则：原浏览器 debug 路径是明确 KEEP，即使不在默认 cmd 传递图；平台 API 同样完整保留。旧 Phase2 inventory 的迁移表、历史 test-baseline 与计划文档不是本轮改写对象，不为形式追加无直接冲突的文档。

## 3. 实际消费者：contract → implementation → injection → consumer

基线只读检查全仓追踪的 Go import、owner 限定符号、同包调用和测试、非 Go 直接引用，检查目标包文件与 sibling 路径；未见四包 dot import。go list -mod=readonly 的根模块 ./cmd/... 成功：31 个命令，其中 core/config、goroutine、timeout、jsonx 分别被 14/0/5/2 个命令传递依赖；这是包级图，不是每个声明的使用证明，更不是整包可删依据。

原 Amazon debug 在 hack/debug 独立 module（replace task-processor => ../..）。根模块直接 go list ./hack/debug/test-amazon 曾因模块边界失败；改在 hack/debug 执行 go list -mod=readonly ./test-amazon 成功，其图包含 core/config、goroutine 与 timeout。两次都只读取 Go 元数据，未执行 main、LoadConfig、浏览器或外网业务请求，未改依赖文件。

| 合同/实现 | 实际注入与消费者 | 决策 / 当前 owner |
| --- | --- | --- |
| core/config.CostCalculationConfig、DefaultCostCalculationConfig、CalculateTotalCost | 唯一引用是本文件和 platform_common_test.go；无当前生产 caller，不属于根 Config/schema 字段 | 全链 RETIRE，不抽取公式，不删平台同名费用 DTO |
| core/config.TimeoutConfig/RateLimitConfig 及 default | 仅 common_types_test.go 和原存在性 guard；无生产构造、根 Config 嵌入或读取 | 精确 RETIRE；同文件 HTTPClientConfig/default KEEP，Retry 在原 owner 不变 |
| pkg/goroutine.TaskInfo/Manager/NewManager 与全部 Manager 方法 | 仅本文件 Manager 分支互调，无包外生产构造/调用，没有专属测试文件 | 闲置内存注册/计数/等待分支 RETIRE，不补其并发缺陷或建新 lifecycle |
| pkg/goroutine.Processor/Task/Result/ProcessFunc | crawler/amazon/extractor/extractor.go 调用 NewProcessor、Task、ProcessParallel；由原 Amazon browser processor/debug 入口到达 | 原 owner 全 KEEP，包括默认值、timeout、日志、panic/结果/错误和并发语义 |
| pkg/timeout.WithAIShortTimeout → AIShortTimeout | publishing/shein/title_resolution.go 与 crawler/amazon/extractor/rating_extractor.go | 两现役 caller 与 30 秒原值/标准 context cancel 保持 |
| pkg/timeout.DownloadTimeout / DownloadLongTimeout | crawler/shared/browser/chrome_downloader.go 用前者；fingerprint-browser-installer/main.go 及其测试、downloader 测试用后者 | 10/30 分钟原值 KEEP；WithDownloadTimeout 包装无 caller，不能因包装退休删除常量 |
| 其余 11 timeout 常量 / 13 包装 | 全仓无限定生产或测试 caller；只存在各包装互引、声明/历史记录 | §5 精确 RETIRE，不改请求 timeout、deadline 或真实 worker |
| pkg/jsonx/marshal.go 九函数 | 没有包外调用；MarshalPretty/ToJSONString 等仅文件内互调，SaveToFile 无 caller | 整文件 RETIRE，不执行写文件 helper，不替换其它 owner 同名 UnmarshalStrict |
| pkg/jsonx/unmarshal.go、llm.go | amazon/api/auth.go、request.go、uploads.go 用 UnmarshalBytes；publishing/shein/title_resolution.go 用 UnmarshalString/CleanLLMResponse | 原泛型解析/LLM清理文件与当前严格草稿 decoder 完整 KEEP |
| 当前 Store 与独立登录 | Store 持久化由 internal/storecenter 原 owner 持有；独立登录路径 cmd/shein-login-worker → internal/app/runtime/sheinloginworker/runtime.go → internal/sheinlogin/bootstrap/build.go / internal/sheinlogin/worker.go，仍用原模型/注入/执行 fence；无拟退声明调用 | 原 Store 同步与独立登录、UNKNOWN 不自动接管保持，无新 owner |

以上区分测试、注释、同包互调、默认命令和独立 browser/debug。实施前重查实际消费者和 main 组合；不能只搜名字不看 alias/receiver，也不把其它 owner 的 Manager/TimeoutConfig/UnmarshalStrict 或平台费用字段误判为拟退合同。

## 4. 配置、持久化与副作用边界

CostCalculationConfig 的 FixedCostAmount/FixedCostPercent/ShippingCost/ProcessingFee/PlatformCommission，TimeoutConfig 的 ConnectTimeout/ReadTimeout/WriteTimeout/IdleTimeout，RateLimitConfig 的 Enabled/RequestsPerSecond/BurstSize 只是孤立 Go 声明及 JSON/YAML tags，没有当前 Config 嵌入、仓库/schema/迁移、加载/注入或 Marshal/Unmarshal caller。删除其类型不等于删除任何真实配置键/记录，不全局禁用字段字符串或同名 DTO。当前 Viper、环境覆盖、默认值、校验、HTTP/Retry/Logging 配置及样例全保持，无新配置 owner 或迁移任务。

Manager 的 tasks/map/wg/running 只有未使用分支内存状态，没有持久化、自动恢复或外部调用入口；本批不接管运行中的 goroutine，也不修原 Processor 的行为。保留的并行处理、浏览器、爬虫生命周期和真实 workerpool/Temporal 仍由原 owner 负责。

jsonx.SaveToFile 曾可创建 logs/ 并写文件，但无当前 caller；退休代码不运行此函数，不读取、遍历、删除或转换任何已有 logs/、JSON、token、凭据文件。当前 debug 自有 saveProductToFile 使用 encoding/json/os 是另一 owner，完整保留。删除闲置 UnmarshalStrict 不修改当前草稿的严格解析、拒绝未知字段、规范化/hash、版本、fixture 或持久化合同；不提供 fallback/双读/双写/第二事实源。

无新请求、provider 或平台 mutation，无新权限/租户边界、幂等键、事务/恢复或 retry/UNKNOWN 语义。响应丢失、重试、取消、并发、重启的行为继续由保留侧 owner 承担；本批没有新的跨持久化步骤，无需新 outbox/Saga/reconciler。SHEIN 登录的授权/fence/UNKNOWN、Store observation 的原事务/归属/刷新协议全部保持。既有运行态和数据是否存在 UNKNOWN；源码缺席不作为真实清理或验收结论。

## 5. 删除、保留与不抽取矩阵

### 5.1 七个原源码/测试工件

| 精确工件 | 拟删除完整闭包 | 必须保留 |
| --- | --- | --- |
| internal/core/config/platform_common.go | 整文件：CostCalculationConfig、DefaultCostCalculationConfig、CostCalculationConfig.CalculateTotalCost | 其它 config 文件/平台定价 API/DTO 和有效规则 |
| internal/core/config/platform_common_test.go | 整文件：TestDefaultCostCalculationConfig、TestCalculateTotalCost、BenchmarkCalculateTotalCost | 不把旧算法测试复制到新 owner |
| internal/core/config/common_types.go | TimeoutConfig/DefaultTimeoutConfig、RateLimitConfig/DefaultRateLimitConfig 四声明及专属注释 | HTTPClientConfig/DefaultHTTPClientConfig 全部字段/默认值/注释与其 time import |
| internal/core/config/common_types_test.go | TestDefaultTimeoutConfig、TestTimeoutConfigCustom、TestTimeoutConfigZeroValues、TestDefaultRateLimitConfig 四函数 | TestDefaultRetryConfig、TestRetryConfigCustom、TestRetryConfigZeroValues、TestDefaultHTTPClientConfig、TestHTTPClientConfigCustom 五函数原样 |
| internal/pkg/goroutine/goroutine.go | TaskInfo、Manager、NewManager；Manager.Start/Wait/WaitWithTimeout/GetRunningCount/GetStatus 及专属注释 | Task/Result/ProcessFunc/Processor/NewProcessor 和 Processor.ProcessParallel/worker/executeTask 全部原行为；保留侧仍需 context/fmt/sync/time/Logrus |
| internal/pkg/timeout/timeout.go | §5.2 的 11 常量/13 函数及专属注释 | AIShortTimeout、DownloadTimeout、DownloadLongTimeout、WithAIShortTimeout 和原值/实现，context/time imports |
| internal/pkg/jsonx/marshal.go | 整文件：MarshalWithoutHTMLEscape、MarshalIndentWithoutHTMLEscape、MarshalPretty、UnmarshalStrict、ToJSONString、ToJSONStringPretty、IsValidJSON、CompactJSON、SaveToFile | unmarshal.go、llm.go 全文件及当前 callers/严格草稿 codec 原样 |

共五个生产文件、两个原测试文件；三整文件删除、四混合文件精确删除。没有新 EXTRACT 生产文件、替代 library、包装或目录迁移。混合文件只去准确声明/专属注释，除必要 gofmt 对齐外保留侧 AST/实现/字段/默认值不变；两个 JSON KEEP 文件逐字不变。没有其它包内测试或 README 专属尾部；不制造空壳。

### 5.2 timeout 精确集合

RETIRE 常量（11）：AITimeout、AILongTimeout、HTTPTimeout、HTTPShortTimeout、HTTPLongTimeout、TaskTimeout、TaskShortTimeout、TaskLongTimeout、TaskExtraTimeout、ShutdownTimeout、HealthTimeout。

RETIRE 函数（13）：WithAITimeout、WithAILongTimeout、WithHTTPTimeout、WithHTTPShortTimeout、WithHTTPLongTimeout、WithTaskTimeout、WithTaskShortTimeout、WithTaskLongTimeout、WithTaskExtraTimeout、WithDownloadTimeout、WithShutdownTimeout、WithHealthTimeout、WithCustomTimeout。

KEEP：AIShortTimeout = 30*time.Second，DownloadTimeout = 10*time.Minute，DownloadLongTimeout = 30*time.Minute；WithAIShortTimeout 原 context.WithTimeout(parent, AIShortTimeout) 实现不变。不全局禁用 TaskTimeout 等名字，不修改其它 owner 中的自有 timeout 或 SDK 参数。

### 5.3 永久保留与明确非目标

四包都保留有效声明，不能删包根、全局禁 import 或取消 pkg/jsonx 解析。Config/LoggingConfig/HTTPClientConfig/RetryConfig、kernel/module Module/Registry/路由、platform/logging、真实 workerpool/Temporal/队列、DB/Redis、AI/prompt/预算/百炼参数、图像与当前规则保留。当前采集与原浏览器/debug、平台 API/DTO/transport、草稿/预览/严格 JSON/版本/fixture、共享 MappingInfo/SourceIdentity、独立 SHEIN 登录及 Store 同步全部保留。当前四条 CURRENT inventory 边（logging→core/config、logging→pkg/appenv、openai→core/config、httpimage→imageagent）仍真实，不删除或改动任何行。

```text
Legacy decision: RETIRE
Reusable behavior: 原 HTTP/Retry、Processor、AI 短超时与下载常量、泛型 JSON/LLM 清理已由当前 owner 提供；本轮无新抽取。
Current owner: 原 Config/HTTP/Retry、crawler/amazon/extractor 与 goroutine.Processor、标准 context/time 与当前消费端、jsonx/unmarshal.go/llm.go 和各当前严格 codec。
Cutover/deletion condition: 用户确认本矩阵；实际 caller 复查不变，原 guard 精确转换、必要自检/原 CI 与最终交付检查完成；合并须另获准确 HEAD 授权。不附加环境清理或试用门槛。
```

## 6. 原检查与文档闭包

本轮设计交付只改三文档：本文、架构 README 新当前入口、Legacy Register 新当前注记；后两者原正文逐字作为后缀保留。确认后的整批实施共 **15 工件**：§5.1 七个旧源码/测试 + 下列一个原检查文件 + 四个旧 KEEP 文档 + 本三文档。不顺带改 repository-structure/module-target-mapping，它们没有本轮准确 helper 直接引用。

### 6.1 冻结的检查/文档转换与零改边界

仅修改 tests/listingkit_backend_retirement_test.go 中 **TestLegacyRuntimeSkeletonAndIdleConfigClosureIsRetired** 这一原函数；不新建 checker 或拆 test 文件，不修改其它原 guard。具体允许：

1. 保留原 Temporal bundle 两路径缺席断言；另对 §5 的三个整退文件加精确缺席检查，使用 fail-closed os.Lstat（存在/其它读取错误均失败，只有不存在通过），不靠静默跳过缺文件的 source helper。
2. core/config 的 retained 表只移除 TimeoutConfig/DefaultTimeoutConfig/RateLimitConfig/DefaultRateLimitConfig 四项；原 retiredConfig 加这四项与 CostCalculationConfig/DefaultCostCalculationConfig，并对原 receiver CostCalculationConfig.CalculateTotalCost 精确禁入。Config/LoggingConfig/HTTPClientConfig/default/RetryConfig/default、platform 日志及全部 kernel/HTTP 既有断言保持。
3. 扩展同函数现有 trackedFiles 全追踪 Go 扫描到三个准确 owner internal/pkg/goroutine、internal/pkg/timeout、internal/pkg/jsonx，继续原 parser.ParseFile 失败即失败及原 kernel decoded import 失败路径。保留原 kernel/module、app/httpapi、core/config、platform/logging owner 扫描。TypeSpec/ValueSpec/FuncDecl 分别检查声明；区分接收者与顶层函数，不全局扫字符串。没有全包/全 import 禁令，因为这四包继续有效。
4. goroutine：精确拒绝 TaskInfo/Manager/NewManager 及 Manager 的全部五方法；新增正向保留 Task/Result/ProcessFunc/Processor/NewProcessor/Processor.ProcessParallel/worker/executeTask。timeout：拒绝 §5.2 的 11 常量/13 函数，正向保留三常量与 WithAIShortTimeout。jsonx：拒绝 §5.1 九函数，正向保留 UnmarshalString/UnmarshalBytes/CleanLLMResponse；泛型 FuncDecl 不得漏检。保留侧原值/行为由差异核对及已有测试/编译验证，不把“名字存在”冒充行为等价。
5. 原 kernel retiredKernelTypes、Registry methods/fields、HTTP runtimeBundle.pools/workerPools、workerpool decoded import 与 retained 循环全部保留。仅加上述 owner/声明，不更换 trackedFiles、解析器、读取失败策略、断言工具或其它 guard 逻辑；新增常量 ValueSpec 处理不能跳过原 TypeSpec 字段检查。
6. 四旧依据仅在首部加窄 superseding 注记，链接本文并精确说明 §2 的被替代声明；其原全部正文逐字作为后缀保留：legacy-numeric-acquisition-tail-closure-retirement-boundary.md、legacy-temu-listing-closure-retirement-boundary.md、legacy-runtime-skeleton-config-closure-retirement-boundary.md、legacy-host-runtime-closure-retirement-boundary.md。其余 KEEP/历史计数/失败与证据不改；后三个仍有真实 timeout/HTTP/Retry/共享消费者，不把历史整个作废。
7. 本文/README/Register 只校准本批当前状态与入口，原历史不改。完整 phase2-runtime-inventory.md（CURRENT 四行、历史、数字、ceilings）、import_boundaries_test.go、lint、两套及其它 workflow、其它检查、test-baseline.txt、旧 Phase2 计划、依赖/锁/保护规则全部零改。若原 CI 发现矩阵外 stale check，先报告准确冲突、分类及最小补丁，未经用户决定不静默放松。

不得把前批 KEEP 词串改成全仓禁词，也不因三包有历史分类而更新 inventory。源码/测试/旧依据在用户确认前不修改。

## 7. 实施顺序、验证与交接停止点

### 7.1 设计交付与开工门槛

先交付这三文档、只读消费者证据和原文档检查/PR CI；记录准确设计 HEAD、适用设计复核、finding 分类以及用户确认。文档 CI 通过不自动等于实施或产品验收。用户确认后，#301 当前段写 Reuse Existing Architecture 的依据、Scope/Out of Scope、owner/路径/不变量及 Ready，并读回再由原唯一 Writer 同一 PR 实施。#301 历史后缀逐字保留，当前段不超过正文上限；#301 OPEN，#29 不重开。

### 7.2 整批实施与必要验证

实现期先扩展 §6.1 原 guard，在旧目标存在时确认准确 RED；再完成七工件退休及四旧文档尾部，确认 GREEN。保留混合文件的 HTTP/Retry 五原测试；业务规则没有新实现，不制造公式替身或无意义 RED。检查真实差异只含 15 工件，无新生产代码/禁改项、四旧正文精确后缀、两个 JSON KEEP 文件不变、混合文件保留侧语义/值不变；重查包外/alias/测试及 sibling 消费者。

必要开发自检复用现有 core/config 与相关 guard 测试、当前 JSON/发布/采集/登录/Store 的受影响既有测试及适用编译，不人工重复完整 CI。核对 current-application、image-agent-temporal-worker、shein-login-worker、product-listing-api-schema-migrate、1688-local-agent、1688-batch-import 等原保留入口可编译；原 Amazon debug 独立 module 用正确 module 的 readonly 依赖核对/编译，不运行 main。全程不启用真实 provider/浏览器/DB/Redis，不读取已有配置/凭据，不开发新验收环境、故障平台或工具的工具。若测试必须接触现有资源，保留 NOT_RUN 并报告，不能从测试授权推导真实环境权限。

稳定候选只跟进其准确 HEAD 的原 Task Processor CI，完整 run 的完成结果与原条件 skip 分别记录；不改 workflow、检查或门禁来换绿。最终候选再同步一次实际 main，只处理真实组合问题；未变化证据可明确复用，不冒充新 SHA 的直接执行。完整闭包形成后一次最终交付检查；finding 先按 AGENTS 分类，具体 Must 缺口在原 PR 收敛，不无限重开架构或机械拆批。

### 7.3 Review、权限与最终交接

本批沿用已有 GitHub 复核入口，不对同 SHA 已运行复核重复触发；启动失败据实记录，至多一次针对启动失败的重试。**未经用户本轮授权不默认新增 AI 互验线程**；此前 #912 的授权不覆盖本轮。需要备用独立只读设计/最终源码复核时，先给准确 HEAD、限定差异、失败证据与只读范围，请用户决定，不自动派 Writer/Reviewer 或恢复旧线程。

设计确认与实现完成各自交接，CI/AI 复核不等于用户产品验收。合并、部署、真实资源操作、数据/凭据/进程操作及 #301 关单均 NOT_AUTHORIZED；#301 保持 OPEN。本批不新增内部试用前置，不停止现役能力。后续若获准确 HEAD 合并授权，正常 expected-head 合并后只以实际 merge SHA 对应 event=push/head_branch=main 的完整 CI completed/SUCCESS 确认，不拿 PR CI、单独 Gate 或其它 main 代替。
