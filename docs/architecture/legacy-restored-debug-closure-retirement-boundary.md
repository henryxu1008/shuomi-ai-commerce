# #301 恢复文件背后的旧调试与采集执行链退休设计

## §1 用户结果与准入

本批优先处理用户曾删除、后因已有引用恢复的代码所承载的业务逻辑。维护者不再需要维护旧手工调试、浏览器执行、prompt 回放、身份可行性探针及人工工具整链；当前产品采集、图像、身份权限与 Store 观察同步继续由原 owner 提供。恢复的 24 个文件不是整删清单，历史 KEEP 也不是永久产品要求。

- 当前三文档阶段：Design Basis: N/A / Ready，仅允许调查与设计/导航文档，不改正式源码或测试。
- 未来实施：Design Basis: Independent Architecture / DESIGN_ONLY / SOURCE_NOT_STARTED / NOT_READY。完整旧子系统退出、配置输入合同和安装能力 owner 变化须先完成适用复核与用户确认，不能先写实现。
- 调查基线：main 94d6e2b7fd6e8ae90d1f94291f596fbe9d1e761a。#1007 已合并，准确 main CI 40994719660 attempt 1 整 run SUCCESS；新优先级及接单见 [#301](https://github.com/henryxu1008/shuomi-ai-commerce/issues/301#issuecomment-6514179818)。
- 一个 Writer、原 #301 独立工作区、一个主要 PR。用户主工作区五项删除及 [#1009](https://github.com/henryxu1008/shuomi-ai-commerce/pull/1009) 保持原 owner，本批不重复删除或代替授权合并。

Must：保留当前采集、图像、身份权限、Store 观察同步、有效 JSON/hash/版本、凭据保护、租户隔离、事务、deadline/取消和 UNKNOWN/副作用 owner。Out of Scope：workflow、完整 inventory、历史计数、ceilings、保护规则、工具版本修改；操作既有实例、数据、配置/凭据、浏览器 profile、轨迹或进程；执行下载、模型回放、身份探针或 DB 查询；部署、运行验收、关单。无新增试用前置，#301 保持 OPEN。不恢复已退休的独立 SHEIN 登录或身份回写 worker。

## §2 依据与前批 KEEP 替代

适用 [AGENTS.md](../../AGENTS.md)、[Issue 驱动规则](../engineering/issue-driven-development.md)、[Hard-Cut Policy](../refactoring/legacy-hard-cut-policy.md)、[全新系统产品基线](../product/greenfield-no-legacy-migration.md)。没有 UI/IA/Figma 增量，不扩大 Threat Model，无新增 Accepted Risk。用户目前授权的是优先核对业务逻辑；下列取舍仍是待确认提案。

| 前批依据 | 本设计确认后替代的准确范围 | 继续有效 |
| --- | --- | --- |
| [#901](legacy-amazon-source-adapter-closure-retirement-boundary.md)、[#942](legacy-1688-captcha-experiment-closure-retirement-boundary.md)、[#946](legacy-browser-management-tail-closure-retirement-boundary.md)、[#947](legacy-source-url-timeout-closure-retirement-boundary.md)、[#948](legacy-crawler-facade-diagnostics-closure-retirement-boundary.md)、[#964](legacy-source-result-contract-closure-retirement-boundary.md) | 原 Go Amazon/1688 浏览器 Processor、只服务它们的模型/算法/浏览器池与三个手工入口不再单独保留，包括旧 PoolManager、验证码解题和轨迹算法；不为“算法还能运行”抽取闲置模块 | 当前 acquisition、extension、公开采集器的 deadline/取消/限流/挑战策略，镜像使用的 Chromium 安装能力 |
| [#844 诊断工具](legacy-task-diagnostics-retirement-boundary.md) | prompt 回放、db-query、json-simplifier 的人工辅助用途不再作为本产品的保留要求，连专属测试/wrapper 收齐 | 当前 AI/provider 客户端、DB 基础层、日志与有效业务规则 |
| [#852](legacy-api-token-tools-retirement-boundary.md)、[#891](legacy-listing-preview-download-closure-retirement-boundary.md) | Login V2 非生产可行性探针及专属 client/runner 退出；不包装进当前身份服务 | 当前身份/注册/Auth.js handoff、安全边界，历史研究证据 |
| [#917](legacy-config-facade-closure-retirement-boundary.md)、[#939](legacy-global-config-logging-tail-closure-retirement-boundary.md)、[#1007](legacy-config-projection-dub-closure-retirement-boundary.md) | 旧爬虫 Browser/Amazon/Platforms 配置、隐式 LoadConfig 与三份公开 YAML 退出，当前命令要求显式私有路径 | 当前显式加载/有效校验、BrowserCollector、私有 YAML、current-application 私有 JSON、现役环境变量与身份拒绝 |
| [#819](legacy-temu-listing-closure-retirement-boundary.md) | 无源码读取者的 tracked data/sensitive_words.json 退出 | 当前清理/规则；外部词表文件与已有数据不操作 |

## §3 实际消费者与根因

调查覆盖 tracked 根模块、hack/debug、tools 的 import、构造者和装配点，没有执行业务入口。

| 工件 | 实际调用与副作用 | 结论 |
| --- | --- | --- |
| test-amazon | 唯一包外构造 AmazonProcessor；启动旧浏览器池，抓取并写本地商品 JSON | 不在当前 HTTP/runtime 装配，与整个旧执行子树同退 |
| test-1688、test-real-track-slider | 唯二包外构造 Alibaba1688Processor；浏览器/profile/验证码，前者还写 JSON | 与旧 1688 执行子树同退，不只删 main |
| internal/crawler | 9 包、114 生产 Go 文件、43 测试文件；树外生产 importer 只有上述三个 debug 入口与 fingerprint-browser-installer | 数量只作调查快照；先抽出安装器闭包，再整退旧树。capturedata/model 也无当前包外生产消费者 |
| replay-sale-attribute | 从 debug JSON 取 system/user prompt，直接请求当前 OpenAI client，分析 variants/截断并写报告 | 没有当前业务的 debug JSON 生产者/注入；不是当前业务重试协议。CLI/专属分析同退，当前 AI client/合同不退 |
| listingkit-phone-onboarding-preflight | 唯一使用 phoneonboardingpreflight 包；独立 provision/login token，临时 org/user、SMS、session proof 与 cleanup | 旧非生产探针整链退出；不能从“仅人工用”推定当前登录依赖它 |
| tools/db-query、scripts/db-query.ps1 | 手工任意表/where/fields；固定连接串仍写在源码，且先连接再解析 --help | 不能因为名字“通用”而保活。整退，不另造 SQL 工具；未输出连接字面值，也未连接目标 |
| tools/json-simplifier | 文件/stdin 的有损 JSON 缩短：数组仅第一项、改写长串/URL/ID | 无当前业务调用，整退；不抽到商品 JSON/hash 路径 |
| 三份公开 YAML | 当前 schema CLI/独立 worker 默认路径与样例检查引用它们 | 文件退出必须同时处理默认路径和有效检查，不能整体删除 YAML 加载器 |
| data/sensitive_words.json | tracked 源码无该路径/词表加载器消费者 | 仅删除 tracked 工件，不读取/处理外部词表 |

当前产品走另一条链：currentapplication/runtime.go、data_services.go 使用 integration/acquisition/amazon；Amazon 扩展载荷由 product/sourcing/amazon_browser_capture.go 严格验证。1688 正式 collector 使用 integration/acquisition/a1688/browser，经 browserclient/browsercollector 注入 HTTP；本地 agent/batchcapture 不 import 旧 crawler。

保留边界：cmd/current-application/image_set_worker.go 读取私有 worker YAML；app/worker/imageagent/dependencies.go 仍用 LoadConfigFromFile。删除公开样例不等于退休当前配置合同。fingerprint-browser-installer 仍被 Docker 构建使用，不可随 shared/browser 一并误删。

## §4 owner、输入与持久化

### §4.1 两项最小 EXTRACT

1. 将 ChromeDownloader 的现役下载/版本解析/平台选择/缓存查找/解压闭包及最小私有日志 port 移到 internal/integration/browserinstall。原 cmd/fingerprint-browser-installer 直接调用；复用已有 HTTP/timeout 与 fingerprint-chromium，不造下载框架、不升级版本。只留现役 NewChromeDownloaderWithTimeout；CheckAndDownload 改为无参（原命令始终传空串），旧 configPath 旁路和无消费者便利构造退出。命令 env/日志注入/下载 timeout/归档行为不变。
2. BrowserCollectorConfig 当前混放在 core/config/type_browser.go，原样移到同包 type_browser_collector.go；字段、tag、JSON 与调用者不变。不能因为同名 Browser 前缀删掉公开采集器配置。

旧指纹随机化、代理评分、旧质量重试和验证码算法只服务旧手工链，无当前独立消费者，随链退出；不凭历史 KEEP 再抽成闲置共享模块。

### §4.2 需要确认的合同取舍

- 旧 Go 浏览器人工执行、旧商品 JSON 输出、profile/轨迹续用、旧 prompt debug 重放不再承诺可用。已有文件/缓存/profile/进程不扫描、不迁移、不清理。
- 当前公开采集器的 challenge 拒绝、冷却/限流与 deadline/取消保留，不把旧自动解题接入当前链。
- 旧 Login V2 探针创建/验证/cleanup 全部退出，不保留后台清理、不调用 provider、不转移 token；当前身份 owner 与安全测试保留。
- 浏览器安装副作用仍只归原 installer 命令；不加应用启动时自动下载、第二事实源、新 retry 或后台 owner。本次不运行安装。
- 四个现役命令 image-agent-temporal-worker、source-account-registry-schema-init、product-listing-api-schema-migrate、listingkit-schema-migrate 改为必须显式提供私有 -config。空白/缺参在依赖解析、DB/provider/worker 启动前失败；原 manifest/owner/role、SQL 和 canary 语义不变。current-application 原私有 JSON 输入不改。
- 仅旧 Browser/Amazon/Platforms 配置根与已枚举旧 env 退出；复用现有 retired-config 入口拒绝旧输入，不静默忽略、不 fallback、不恢复旧别名。当前 BrowserCollector 与其它现役配置/环境输入保留。
- 准确输入边界：YAML 的 browser、amazon、platforms 三根（包含空对象）拒绝；五条现有 Primary 为 TASK_PROCESSOR_BROWSER_PATH、TASK_PROCESSOR_BROWSER_USER_DATA_DIR、TASK_PROCESSOR_BROWSER_HEADLESS、TASK_PROCESSOR_BROWSER_POOL_SIZE、TASK_PROCESSOR_PLATFORM_1688_ENABLED。环境拒绝只涵盖 TASK_PROCESSOR_BROWSER_、TASK_PROCESSOR_AMAZON_、TASK_PROCESSOR_PLATFORMS_ALIBABA1688_、TASK_PROCESSOR_PLATFORM_1688_ 四个旧命名空间；仅按变量名/存在性判断，空值也拒绝，不记录值。当前 TASK_PROCESSOR_BROWSER_COLLECTOR_ 是保留输入，不能被宽泛 browser 前缀误拒；#1007 的三项 image 自动变量映射与身份拒绝不变。
- 没有业务 schema/表、持久化 JSON/hash/版本、事务、权限或 UNKNOWN owner 改变。所有真实环境与旧产物处置均 NOT_AUTHORIZED，运行/产品验收 NOT_RUN。

## §5 完整删除/保留矩阵

完整子树表示调查基线的全部 tracked 文件；混合文件只改所列声明。最终基线出现新当前消费者必须先报告，不能照旧快照强删。

| 范围 | 处置 |
| --- | --- |
| internal/crawler/amazon/**、alibaba1688/**、shared/browser/** | 安装 EXTRACT 完成后 RETIRE 全部旧包/模型/结果合同/提取器/variations/浏览器/helper/专属测试；不留 Go 包、alias、空兼容壳 |
| internal/integration/browserinstall；cmd/fingerprint-browser-installer/main.go；core/config/type_browser_collector.go | §4.1 两项最小抽取，原命令切 import/无参方法；原 installer main_test.go 不改语义 |
| hack/debug/test-amazon/**、test-1688/**、test-real-track-slider/**、replay-sale-attribute/**、listingkit-phone-onboarding-preflight/**；internal/listingkit/phoneonboardingpreflight/** | RETIRE 五入口与全部专属实现/类型/测试；当前身份/AI 不新增替代 wrapper |
| tools/db-query/**、json-simplifier/**；scripts/db-query.ps1 | RETIRE 工具/专属测试/唯一 wrapper；不自建替代工具 |
| config/config-dev.yaml、config-prod.yaml、config-test.yaml；data/sensitive_words.json | RETIRE tracked 工件，不读取真实私有配置/词表；有效检查按 §6.1 改接 |
| core/config/type_browser.go、type_amazon.go、type_platform.go、validator_browser.go、validator_amazon.go、validator_platform.go | Collector 类型抽取后 RETIRE 旧类型与校验 |
| core/config/config.go、defaults.go、loader.go、loader_builder.go、loader_helpers.go、validator.go、retired_product_runtime.go | 仅退出旧三个配置根/隐式 LoadConfig 的字段、装配、defaults/env/helper/错误组尾部；保留当前显式 loader、dotenv/私有保护、现役域校验与 #1007 拒绝。旧输入拒绝仍用原入口 |
| 四个 cmd 的 main.go；app/runtime/listingkitschemamigrate/options.go 与原调用点 | 删除公开样例默认路径，显式输入/缺参在依赖前失败；不动业务 owner/SQL/grants/wire-mode/canary |
| hack/debug/go.mod、go.sum、README.md；scripts/test-all.ps1 | 仅在 #1009 原 owner 合并其两个 debug 入口删除后，剩余五入口随本批退出，才删除空 module；原多模块测试脚本只去掉 debug 项，不先遗留他人源码 |
| docs/CODE_WIKI.md、amazon-listing-workbench-api.md、database-connection-optimization.md、hack/debug/README.md、tmp/README.md 与当前维护导航 | 撤掉本批现行操作指引；历史证据文档加准确退休注记，保留原正文，不为清链接抹历史 |
| 本设计、architecture/README.md、refactoring/legacy-register.md、code-health-decisions.md、phase2-runtime-inventory.md、module-target-mapping.md、development/repository-structure.md | 窄当前说明替代本批 KEEP；历史正文/计数/ceilings/勾选不改。完整 CURRENT inventory 仍零改，真实失配先交准确提案 |
| tools/go.mod、go.sum、tools.go；workflow、Docker、有效 cmd 清单 | KEEP。tools module 和 tools/tools.go 仍固定 CI golangci-lint；不因工具 main 退出删 module、升级依赖或全量 tidy；安装器/Playwright 构建命令保留 |
| 当前 acquisition/extension/batchcapture、product facts/sourcing、图像/Artwork、AI、身份、Resource、Store 观察、DB/Redis/日志/Retry | KEEP 原业务/JSON/hash/版本/事务/安全断言，不从旧树新增依赖，不复活 SHEIN worker |

Legacy decision: EXTRACT | RETIRE。Reusable behavior: 现役 Chromium 安装和当前 Collector 配置。Current owner: integration/browserinstall、core/config 与原当前业务 owner。Cutover/deletion condition: 原安装器切换后整退旧树，适用检查/CI 通过；用户确认与独立准入先于生产源码，合并另行授权。

## §6 必要验证

设计阶段仅三文档 diff/相对链接与原架构目录/Greenfield 检查。源码阶段用合成 fixture 和原测试；缺参/旧输入拒绝先 RED，再最小实现。有效行为不凭旧测试名保活，旧实现细节不成为当前 Architecture Authority。

必要自检：新安装叶子的原 timeout/logger 断言（不下载）、core/config 当前有效校验/身份拒绝、四命令缺参无依赖调用、schema/worker 原 owner/role/canary/取消断言、当前采集原预算/挑战/限流/取消与本批退休 guard。入口仅编译当前应用、采集器/local-agent/batch-import、image worker、schema CLI、安装器。完整稳定候选交原 CI，不手工重复全 CI，不加 DB/浏览器/SMS/provider 演练。

### §6.1 原检查处置边界

本表为待复核提案。无 missing-skip，不放宽 ceiling、检查算法或安全结果。未列当前业务/安全断言不能因为旧实现退出就删掉。

| 原位置 | 本批允许的准确转换 | 保留 |
| --- | --- | --- |
| crawler 整树 43 测试文件 | 专属运行时/算法/DTO 测试退出；ChromeDownloader 两 timeout 测试及注入日志有效断言迁到新叶子，适配 §4.1 | 当前 integration/acquisition 的预算/取消/挑战/解析检查，不伪造旧 PoolManager 的替代执行证据 |
| phoneonboardingpreflight、debug 探针/回放与 db-query 专属测试 | 随专属程序退出，不搬进当前业务保活 | 当前身份/AI/DB 安全与业务测试 |
| core/config/browser_user_data_dir_test.go、proxy_pool_config_test.go；config_test.go 的旧 Browser/Amazon/1688 用例 | 专属旧域用例退出；混合表仅去旧域条目，当前最小配置与 TestConfigValidation 的现役规则保留 | 当前 OpenAI/图像/DB/身份/Workbench 与旧输入拒绝 |
| config_env_test.go 两个 Committed*ConfigArtifacts 测试 | 样例保活改为文件缺席；有效 owner-scope 禁入送原 loader 合成 fixture，不能直接丢断言 | LoadConfigFromFileDoesNotExposeListingKitOwnerScopeToggle、CannotDisableListingKitAuthentication、LegacyUsernameAllowlistInputsFailClosed 等 |
| validator_dependencies_test.go、validator_format_test.go | 错误/顺序 fixture 仅退出 Browser/Amazon/Platforms 组，以原当前域输入继续验证 | 现役域规则/顺序/错误格式 |
| listingkit_backend_retirement_test.go：SourceResultContractClosure、IdleProviderBootstrapClosure、CrawlerFacadeDiagnosticsClosure、SourceURLAndTimeoutClosure、BrowserManagementTailClosure、Alibaba1688CaptchaExperimentClosure、AmazonSourceAdapterClosure、RuntimeSkeletonAndIdleConfigClosure、NumericAcquisitionAndPlatformTailClosure、ListingKitTaskBackendSourcesAreRetired、assertLegacyLoggerFacadeRetired 中本批旧树 KEEP | 旧 crawler/debug/配置保活改显式缺席；旧根遍历上移 retained internal，原 Below 过滤/读解析错误失败保留；安装器 KEEP 改接新叶子 | 全部其它退休禁令与当前 runtime/acquisition/图像/身份/Store/日志/Retry KEEP |
| 同文件 SDSRuntimeClosure、HostConfigFeatureFlagTracingClosure、WatermarkClosure、RetiredTaskAndListingRuntimeConfigClosure；logging_config_test.go | 公共 YAML 存在/扫描改缺席；原退休键用合成 fixture 送原 loader 验证拒绝 | SDS/宿主/水印/任务/全局输入禁令及当前权限安全 |
| target_architecture_phase2_test.go::TestPhase2LegacyRootsDoNotGrow | crawler 与已退 infra 同样从 retained internal 索引按原子树过滤计数，缺席为 0，读解析错误仍失败 | max=134、所有其它 ceiling/CURRENT 规则/比较不变 |
| CurrentCommandsDoNotCompileLegacyBusinessBackends、ProductionProviderConstructorInventoryIsComplete、catalog/sourcing/import boundary guards | 当前禁令保留；若静态 expected 要求本批旧路径存在，仅改显式缺席、安装器准确改新叶子；当前 provider 构造无变化 | 不删现役构造/依赖项，不把 guard 当运行证明 |
| 四命令 main 测试；listingkitschemamigrate/runtime_test.go::TestResolveConfigPathAndParseFlags | 公共路径默认改显式输入/缺参先失败，复用原依赖 spy | productlistingschemamigrate 三 owner/manifest 拒绝及 worker 原安全测试 |
| repository_structure_test.go、architecture_docs_test.go、Greenfield/链接检查 | 只跟随准确维护目录/链接退出；历史文档保留注记，合成安全负例不认作旧消费者 | 当前入口 owner、禁迁移/兼容/旧依赖语义、完整 inventory 零改 |

发现表外真实当前消费者、不同安全结果、CURRENT 失配或新持久化/副作用义务，先报告根因、同根路径和准确补丁，不重跑掩盖或放宽门禁。最终基线必须逐项复核，本文不是提前放行未知改动。

## §7 准入和停止点

1. 当前只形成三文档设计；准确设计 HEAD、原 CI 与适用独立复核留主要 PR。本文尚未 IMPLEMENTATION_READY，复核只聚焦真实旧 KEEP 替代、安装/配置 owner 和保留能力，不重开无关架构。
2. #1009 五删除仍由原 owner 处理；本批实施前同步其实际已合入 main，核对 debug module 不遗留他人入口，不改主工作区。它是源码所有权依赖，不是内部试用前置。
3. 用户确认 §1–§7（重点 §4.2/§6.1）及适用独立准入后，同一 Writer/PR 整批实施；不按文件数、模块或测试机械拆批。
4. 未经用户本轮授权不默认新增 AI 互验线程；设计与源码最终复核分别绑定准确 HEAD，未变证据只作明确复用。自动评审没有结论不等于通过。
5. 源级/CI 不等于运行或产品验收。合并、merge SHA 的完整 push/main CI、部署、真实清理与关单分别遵守授权；#301 保持 OPEN。
