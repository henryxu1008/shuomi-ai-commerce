> #301 / #972 窄范围替代：本文涉及旧1688 LocalAgent PoC CLI/client/deviceauth/未挂载HTTP/内存Job/Runner、旧integration/crawler/a1688及父crawler空壳、sds/adapter/product_source、asset/policy和两个专属Prepare的KEEP/实施/操作要求，已由本批完整退休设计替代。这不取消当前公共/扩展/批量/原浏览器采集、SDS/POD、资产审批/有效规则、共享身份权限、独立SHEIN登录或Store同步；其它产品边界与永久禁令保持。以下起始Git blob、历史批准/失败/数字/勾选逐字保留，不恢复迁移或试用/生产前置，不授权任何既有资源操作。

> #958 实施准入：用户已确认冻结设计 `378066fe91ce734d44c2ac95a1c707919e32ebdc` §1–§7（含 §6.1）。准确设计 CI 与一次授权独立设计复核已通过，Issue/PR 已记录并读回 **Independent Architecture / IMPLEMENTATION_READY / Ready → In Progress**。三组35工件在本 PR 整批实施；原有效配置规则与断言保留，workflow/完整 inventory 不改，独立 SHEIN 登录与 Store 同步保持。不操作既有实例、数据、凭据或进程，不增加试用前置，#301 OPEN。准确源码 HEAD、自检/CI/最终复核另在 PR 记录，运行/产品验收 NOT_RUN；以下冻结设计全文及其当时设计阶段状态保留。

# #301 闲置 Gemini 客户端与宿主辅助入口完整退休设计

> Design Basis: **Independent Architecture / DESIGN_ONLY / DESIGNING / NOT_READY**。本轮只交设计，生产源码尚未实施。
> 用户同意三组同批设计，设计确认后再实施；接单与前一当前段归档见 [#301 评论6405893102](https://github.com/qq550723504/shuomi-ai-commerce/issues/301#issuecomment-6405893102)。调查基线 `8424696ea31eeb9f13931a2e1d1db675b549fd21`，复用已有独立工作区，单 Writer。 最终候选同步实际main `4ec492737e85dc4dca187b749219436eb8dc20c8`（#957）；三组生产源码/消费者未变化，README保留该main完整blob后缀。

## 1. 本轮做什么、保留什么

本轮三件事，在同一主要 PR 收齐完整专属尾部：

1. 退休没有生产注入的 `integration/geminiimage` 原生图片客户端及专属日志合同、测试。它是未接线的旧准备代码；当前 Google 文本执行和实际图像能力保留。
2. 退休 `pkg/runtimepath` 安装目录命名空间及唯一包外旧测试。公共1688仍不复用持久化 Profile，独立 SHEIN 登录的 Profile/Store 所有权继续由原 owner 管理。
3. 退休正式 `config.LoadFromBytes` 入口；19个有效配置测试、21处调用改接当前 Viper/Build/Validate owner。配置规则、环境优先级、安全拒绝和实际文件加载入口不退。

使用者：维护当前采集/图像/审核/登录与同步的开发者。当前操作是把旧代码与有效规则分开，避免无消费者入口和旧 KEEP 继续被当作新接线合同。没有声称当前业务 happy path 被这些闲置入口阻塞，也不建设新的底层能力或试用前置。

本轮产物是明确的删除边界和保留行为，不是新产品功能。Must：当前图像/AI/JSON/有效配置规则、当前采集、真实 PoolManager deadline/取消、独立 SHEIN 登录与 Store 同步保持。Should：复用现有 owner 和成熟 Viper，减少正式 API 和测试对旧入口的依赖。Out of Scope：SDK/依赖升级、新 provider、当前路由/预算/计量/UNKNOWN、迁移/兼容/新 schema、运行资源和部署。没有新增 Accepted Risk。

依据：[全新系统产品基线](../product/greenfield-no-legacy-migration.md)、[Hard-Cut](../refactoring/legacy-hard-cut-policy.md)、[目标架构](project-target-architecture.md)、[项目边界](project-boundaries.md)、[Issue驱动规则](../engineering/issue-driven-development.md)及本轮用户决定。无 UI/IA/交互变化，Figma Authority **N/A**。跨 Integration/配置/旧浏览器测试和35路径，按 Independent Architecture 明确完整基线；实际生产 owner、持久化与安全边界均不新增。

## 2. 旧 KEEP 的准确替代

以下 KEEP 在用户确认本设计、适用 review 和 Ready 记录之前仍是原批准边界。本设计不是仅以“生产调用为0”撤销有效规则；规则及当前需求由下述真实 owner 承接。只替代三组明确符号，旧文档正文/历史/计数完整保留。

| 旧依据 | 本轮替代 | 继续保持 |
| --- | --- | --- |
| [#891](legacy-listing-preview-download-closure-retirement-boundary.md)“geminiimage当前合同/准备能力” | 未接线原生 Gemini 图片 SDK 准备代码整根退出；后续接入不恢复旧 API | 当前 ImageAgent、OpenAI/GRSAI/HTTP 图像合同；Dub/OpenMeter准备边界不在本批 |
| [#939](legacy-global-config-logging-tail-closure-retirement-boundary.md) §3/§4的geminiimage、runtimepath、LoadFromBytes | 仅退出本设计三组；19个配置测试改接现有规则 owner | 其余共享层、配置字段/优先级/错误、日志注入、JSON/hash及当前能力 |
| [#948](legacy-crawler-facade-diagnostics-closure-retirement-boundary.md) §4.1、§5 runtimepath与namespace测试/LoadFromBytes | namespace工具和专属旧测试退出；有效配置测试保留并改接 | NewPublicBrowserManager/public配置投影、空userDataDir、不复用账号Profile、其它有效断言/失败关闭 |
| [#878](legacy-host-config-featureflag-tracing-retirement-boundary.md) §2/§4/§5的两个辅助入口 | runtimepath整根与正式LoadFromBytes退出 | 当前Viper/文件加载/scoped .env/环境绑定/验证与拒绝语义 |
| [#880](legacy-redis-proxy-shared-closure-retirement-boundary.md) §5 runtimepath共享层KEEP | 仅已无生产消费者的runtimepath退出 | 有效DB/Redis、通用Retry/并行器、当前Profile owner等 |
| [#917](legacy-config-facade-closure-retirement-boundary.md) §2/§5 loader.go KEEP | 仅LoadFromBytes与失去用途的bytes/fmt imports退出 | NewDefaultConfig函数/字面值、当前Build/Validate与所有正式文件加载函数 |
| [#926](legacy-platform-api-closure-retirement-boundary.md)配置测试调用形式 | 原有效配置期望由当前owner继续验证，旧入口名称退出 | 有效transport/Retry规则及全部当前API/JSON边界 |
| [#872](legacy-queue-proxy-config-tail-closure-retirement-boundary.md)默认/Build/LoadFromBytes测试 | 保留默认/Build/退役字段拒绝测试，替换字节入口调用 | 通用队列/Retry与当前身份/图像/浏览器配置 |
| [#836](legacy-runtime-config-closure-retirement-boundary.md) §5.2/§6旧测试入口 | 只替代剩余LoadFromBytes符号 | 当前配置默认/来源/校验/错误聚合及其它KEEP |

Gemini没有生产注入，不等于取消当前 Google 能力：`integration/googleinteractions` → `integration/aicapability/einomodel` 的受治理文本调用保持；实际图片沿现有 `app/worker/imageagent` → OpenAI/GRSAI/HTTP 与当前治理执行。共享 `internal/ai` DTO/接口保持。没有本批以外能力撤销或新风险批准。

## 3. 真实消费者与 owner

基线全追踪 Go（含cmd/scripts/tools/hack/debug和negative fixture）调查；真实 import 经原AST工具核对，字符串/历史引用另列。

| 目标 | 实际消费者/接线 | 判定 |
| --- | --- | --- |
| integration/geminiimage | 包外实际import仅 `internal/ai/import_boundary_test.go`、`tests/integration_provider_logger_contract_test.go`；无生产import/构造注入。包内client/logger测试自持 | 旧整根 RETIRE，35函数方法/14类型随2生产文件及2专属测试退出 |
| pkg/runtimepath | 包外实际import仅 `crawler/alibaba1688/browser_manager_test.go` 的 `TestAlibaba1688BrowserProfileNamespaceIsStableAcrossRestarts`；包内paths测试自持 | 4函数和专属测试 RETIRE；不抽取新的目录工具 |
| config.LoadFromBytes | 无生产调用，9测试文件的19测试函数/21调用；入口读dotenv再用Viper解析/校验 | 正式入口 RETIRE；有效19函数全部保留并改接，不删除配置规则 |
| 当前采集 | 1688 `NewPublicBrowserManager` 使用public投影，userDataDir固定为空；当前localagent/来源适配仍有真实消费者 | 零生产diff，不能按legacy名字整退采集 |
| SHEIN持久化 | `sheinlogin/profile_owner.go`、`profile_runtime.go`、当前Store/执行归属/fence/UNKNOWN | 原owner与边界保持，不用runtimepath替代，也不碰既有目录/Redis/DB |

现有 `TestProductionProviderConstructorInventoryIsComplete` 列出的真实构造者为OpenAI/GRSAI，没有Gemini。其扫描器继续检测旧Gemini构造回流；不为删除包放宽构造计数或支持新的注入位置。

调查时已核对 #956 Store浏览器、#957 Amazon变体、#931依赖与#692安装边界，无本批生产重叠 owner；最终同步已合并#957，仅带入其既有插件/来源测试和README，不改它们的生产文件。三组目标未新增consumer，未变化调查证据复用；共享README采用实际main完整blob后缀。调查只读，不运行构造器/应用/浏览器、不查询进程/实例、不读取现有凭据或.env。

## 4. 有效行为、持久化及副作用边界

### 4.1 现役配置继续归当前 owner

正式路径保持 `LoadConfig / LoadConfigFromFile / LoadConfigFromFileWithoutValidation` → 当前Viper/BuildConfig → 当前校验；`newViper`、`loadWithViper`、`loadWithViperWithoutValidation`、默认值、ENV绑定、scoped .env先后/非覆盖、错误聚合/格式和退役输入拒绝均不改。Config字段/JSON/YAML/默认/hash不变。

19个测试在原文件改调**仅_test.go内** `loadConfigFixture(t, data)`，定义放既有config_test.go。辅助只用现有 `newViper()`、Viper `ReadConfig(strings.NewReader(string(data)))`、`loadWithViper(v)` 安排合成YAML输入，空输入保留默认路径，解析错误直接返回；不复制校验/字段映射，不调用dotenv/隐式文件扫描，不保留LoadFromBytes同名alias/包装或新生产入口。实际.env及文件加载行为的原有效独立测试保持原样，不在本批重写。

这是测试输入安排，不是把旧生产API搬进_test保活；生产LoadFromBytes删除且新guard禁止返回。OpenAI/Browser/代理、身份/Workbench安全拒绝、图像artifact/admission/上传/定价和退役配置拒绝的原fixture/期望/断言全部保持。原19函数清单见§6.1。

### 4.2 不新增事实源或接管运行资源

旧Gemini客户端的可能历史上游任务/凭据、runtimepath旧目录/浏览器状态属于运行事实 **UNKNOWN**；不扫描、读取、迁移、删除、撤销或自动接管。代码退出不证明它们不存在。当前SHEIN Profile/Store记录/Redis归属控制表、当前配置来源和image ledger/retry/UNKNOWN owner原样，DB/schema/迁移/JSON/事务/授权/租户/幂等边界零变化，无新side-effect owner。

旧Gemini独占请求/素材内存预算与协议测试随旧实现退出；当前HTTP/OpenAI/GRSAI的SSRF、内容/字节限制、并发预算、取消、重试/禁止重发和usage/UNKNOWN原断言保持。本批不把共享安全规则归为旧实现细节删除。

## 5. 删除/保留矩阵与完整闭包

Legacy decision: **RETIRE**。Reusable behavior：有效配置/图片/采集/身份规则已在当前owner，原样复用；没有新生产EXTRACT/新SDK。Current owner：配置当前Viper/Build/Validate；图像当前Integration/App治理；Profile与Store原owner。Cutover/deletion condition：19测试改接完成、原有效断言保留、三组旧入口/根与实际Go import缺席、准确检查转换/CI/最终复核满足；运行资源不在删除条件内。

| 组 | 允许删除/修改 | 必须保留 |
| --- | --- | --- |
| Gemini | 整根4工件，2实际测试import和仅Gemini的compile/logger/root存在条目退出；当前project-boundaries只去该一owner行；Integration README同步Gemini历史退休说明 | 其余provider的DTO/ports/构造/日志/HTTP/协议和原检查算法；共享AI接口、Google文本/当前ImageAgent |
| runtimepath | paths.go/paths_test.go整退；只退包外namespace测试函数及其独占runtimepath/strings imports | 同文件全部public配置/无Profile/失败关闭断言；生产crawler、共享Browser.Manager、SHEIN目录/Store owner零diff |
| 字节入口 | loader.go只删LoadFromBytes与专属bytes/fmt imports；原19函数仅call/t参数/对应名称转换、config_test.go加一个测试辅助 | NewDefaultConfig全部原字节、其余生产配置/Loader/校验代码、全部fixture/数值/错误/角色/退役拒绝断言 |
| 文档 | 9份旧依据仅前置精确替代注；README/register前置新当前说明；本设计；当前project-boundaries一行、Integration README的Gemini当前行/历史说明 | 原文历史/计数/勾选/完整blob后缀；其余有日期superpowers历史不重写 |

未来同一PR **35唯一路径：21 Go +14文档**。本阶段实际只改本设计/README/register三文档。此清单不是任意语义修改授权，§5/§6.1的更窄规则优先。

| 工件路径 | 准确处置 |
| --- | --- |
| `docs/architecture/README.md` | 当前前置索引/登记，原完整blob保留 |
| `docs/architecture/legacy-config-facade-closure-retirement-boundary.md` | 仅准确替代顶注，原全文保留 |
| `docs/architecture/legacy-crawler-facade-diagnostics-closure-retirement-boundary.md` | 仅准确替代顶注，原全文保留 |
| `docs/architecture/legacy-global-config-logging-tail-closure-retirement-boundary.md` | 仅准确替代顶注，原全文保留 |
| `docs/architecture/legacy-host-config-featureflag-tracing-retirement-boundary.md` | 仅准确替代顶注，原全文保留 |
| `docs/architecture/legacy-idle-provider-bootstrap-closure-retirement-boundary.md` | 本设计，阶段证据前置 |
| `docs/architecture/legacy-listing-preview-download-closure-retirement-boundary.md` | 仅准确替代顶注，原全文保留 |
| `docs/architecture/legacy-platform-api-closure-retirement-boundary.md` | 仅准确替代顶注，原全文保留 |
| `docs/architecture/legacy-queue-proxy-config-tail-closure-retirement-boundary.md` | 仅准确替代顶注，原全文保留 |
| `docs/architecture/legacy-redis-proxy-shared-closure-retirement-boundary.md` | 仅准确替代顶注，原全文保留 |
| `docs/architecture/legacy-runtime-config-closure-retirement-boundary.md` | 仅准确替代顶注，原全文保留 |
| `docs/architecture/project-boundaries.md` | 当前owner一行，见§6.1.8 |
| `docs/refactoring/legacy-register.md` | 当前前置索引/登记，原完整blob保留 |
| `internal/ai/import_boundary_test.go` | 准确原检查转换，见§6.1 |
| `internal/core/config/ai_capability_test.go` | 原有效fixture改接，见§6.1.6 |
| `internal/core/config/browser_user_data_dir_test.go` | 原有效fixture改接，见§6.1.6 |
| `internal/core/config/config_env_test.go` | 原有效fixture改接，见§6.1.6 |
| `internal/core/config/config_test.go` | 原有效fixture改接，见§6.1.6 |
| `internal/core/config/image_generation_test.go` | 原有效fixture改接，见§6.1.6 |
| `internal/core/config/loader.go` | 仅LoadFromBytes/专属imports，见§5 |
| `internal/core/config/proxy_pool_config_test.go` | 原有效fixture改接，见§6.1.6 |
| `internal/core/config/task16c_imageagent_config_test.go` | 原有效fixture改接，见§6.1.6 |
| `internal/core/config/task16c_retired_config_test.go` | 原有效fixture改接，见§6.1.6 |
| `internal/core/config/validator_listingkit_test.go` | 原有效fixture改接，见§6.1.6 |
| `internal/crawler/alibaba1688/browser_manager_test.go` | 仅旧namespace测试/独占imports，见§6.1.5 |
| `internal/integration/README.md` | Gemini当前行/准确历史说明，见§6.1.8 |
| `internal/integration/geminiimage/client.go` | 整根退休，见§5 |
| `internal/integration/geminiimage/client_test.go` | 整根退休，见§5 |
| `internal/integration/geminiimage/logger.go` | 整根退休，见§5 |
| `internal/integration/geminiimage/logger_test.go` | 整根退休，见§5 |
| `internal/pkg/runtimepath/paths.go` | 整根退休，见§5 |
| `internal/pkg/runtimepath/paths_test.go` | 整根退休，见§5 |
| `tests/integration_provider_logger_contract_test.go` | 准确原检查转换，见§6.1 |
| `tests/listingkit_backend_retirement_test.go` | 准确原检查转换，见§6.1 |
| `tests/target_architecture_phase2_test.go` | 准确原检查转换，见§6.1 |

## 6. 验证与验收

本设计阶段只做：真实import/构造/调用/前批KEEP调查、35路径唯一性/存在性、旧README/register完整main blob后缀、链接/diffcheck及适用原文档/consumer检查；准确设计HEAD原CI、适用独立设计复核另列。作者不运行生产配置加载/构造器/业务测试或真实资源。本阶段CI不是实施/运行验收。

确认/Ready后：在原聚合新增一个完整guard并取得真实RED → 同批完成旧根/入口退出、19测试改接和§6.1转换 → GREEN。只运行必要原配置fixture/AI合同/provider矩阵/原guard与入口仅编译；完整原CI留稳定候选。新guard验证根/函数/decoded import缺席及当前配置/图像/采集/SHEIN/Store必需owner存在。19规则保留原独立期望，不用自比较或改golden掩盖差异。

本地仅合成fixture/任务TEMP输入，清除所跑fixture涉及的既有环境入口而不读取其值，不运行会读取既有.env/连接既有DSN/启动浏览器的测试。现有文件/dotenv独立测试源码与断言保留，由原CI在其自有隔离环境验证；本批不新建测试平台/runner/故障矩阵。源码、原CI合成执行、真实运行和产品验收分别报告，后两项 **NOT_RUN**，不增加试用前置。

### 6.1 原检查与测试的准确转换边界

1. `tests/listingkit_backend_retirement_test.go`：仅新增 `TestLegacyIdleProviderBootstrapClosureIsRetired` 一个完整guard，复用原trackedFiles/AST/decoded import/失败关闭helpers；按owner检查三组，不全局禁Client/Config/Load等通名。另只从 `TestLegacyRuntimeSkeletonAndIdleConfigClosureIsRetired` 的core/config必需声明表删除 `LoadFromBytes` 一项，由新guard强制其缺席；其它函数/helpers/imports和所有保留项原样。
2. `internal/ai/import_boundary_test.go`：只去Gemini实际import、一个ImageGenerator compile断言和必需读取client.go路径；OpenAI/GRSAI断言与原读取/失败逻辑不改。
3. `tests/integration_provider_logger_contract_test.go`：只在 `TestIntegrationProviderLoggerMethodSetsMatch` 去Gemini row/import；`TestOpenAIConfigsExposeExplicitNonSerializedLogger` 去Gemini.Config的Logger缺席块；`TestIntegrationProviderLoggingUsesOnlyLocalExplicitPorts` 去Gemini遍历名。原OpenAI/GRSAI/S3方法集/Logger非序列化/全局日志禁入/AST断言和失败关闭不改。**scanProviderConstructors、providerConstructorCounts.gemini及两个Production构造/调用inventory测试逐字保持**，继续发现旧构造回流。
4. `tests/target_architecture_phase2_test.go`：`integrationProviderAdapterImportViolations` 仅从providerRoots表和实际存在/遍历名单去Gemini；保留其它当前root必须存在/目录/解析失败逻辑。`TestIntegrationProviderAdaptersEnforceRootSpecificContractMatrix` 仅去Gemini case及公共fixture根名单该项；其它allowed/forbidden矩阵保持。`TestIntegrationProviderAdaptersRejectNonContractProductAndRuntimeDependencies` 仅去Gemini fixture根，根数乘数4→3，单根10禁入及原比较/错误保持。`TestIntegrationProviderBoundaryDecodesAllowedAndForbiddenGoImportLiterals` 只去fixture setup根名单Gemini；OpenAI raw/escaped输入和已获批**三项**禁入期望逐字保持，不再误改为2。`isAllowedIntegrationProviderAdapterImport` 与其它函数/所有ceilings/历史Gemini max=1 **不改**。
5. runtimepath只整退其2文件及browser_manager_test.go的 `TestAlibaba1688BrowserProfileNamespaceIsStableAcrossRestarts` 及其独占runtimepath/strings imports；它仅验证已失去生产consumer的工具。该文件其它函数、public-first空目录/配置/失败关闭断言逐字保持，生产browser_manager.go零diff。
6. 配置9文件/19函数、21调用仅 `LoadFromBytes(data)` → `loadConfigFixture(t,data)`；只有以`TestLoadFromBytes`开头的17函数同前缀更名`TestConfigFixture`，其余两个当前名称不改。仅config_test.go新增§4.1的一个测试输入辅助；所需测试import最小增删。原fixture字面量、t.Setenv/临时代理文件、错误文本期望/数值/角色/所有断言、其它函数保持：

| 测试文件 | 准确函数（当前名称） |
| --- | --- |
| `internal/core/config/ai_capability_test.go` | `TestLoadFromBytesRejectsRetiredStudioImageRoutingModeEnvironment`, `TestLoadFromBytesRejectsRetiredStudioImageRoutingModeYAML` |
| `internal/core/config/browser_user_data_dir_test.go` | `TestLoadFromBytes_AppliesBrowserUserDataDirEnvironmentOverride` |
| `internal/core/config/config_env_test.go` | `TestLoadFromBytesRejectsEnabledWorkbenchWithoutCompleteZitadelConfig`, `TestLoadFromBytes_AppliesEnvironmentOverrides`, `TestLoadFromBytes_AppliesViperManagedEnvironmentOverrides` |
| `internal/core/config/config_test.go` | `TestLoadFromBytesRetains1688BrowserConfiguration` |
| `internal/core/config/image_generation_test.go` | `TestImageGenerationPriceIsExplicitAndHasNoDefault` |
| `internal/core/config/proxy_pool_config_test.go` | `TestLoadFromBytesLoadsAmazonProxyPool`, `TestLoadFromBytesLoadsAmazonProxyPoolFromFile` |
| `internal/core/config/task16c_imageagent_config_test.go` | `TestLoadFromBytesBindsImageAgentAdmissionEnvironment`, `TestLoadFromBytesBindsImageAgentArtifactStoreEnvironment`, `TestLoadFromBytesBindsImageAgentArtifactStoreYAML`, `TestLoadFromBytesBindsListingKitImageUploadS3Environment`, `TestLoadFromBytesBindsListingKitOwnedImageUploadYAML`, `TestLoadFromBytesDoesNotDefaultListingKitImageUploadToLocal` |
| `internal/core/config/task16c_retired_config_test.go` | `TestLoadFromBytesRejectsRetiredProductRuntimeEnvironment`, `TestLoadFromBytesRejectsRetiredProductRuntimeYAML` |
| `internal/core/config/validator_listingkit_test.go` | `TestListingKitLegacyUsernameAllowlistInputsFailClosed` |

7. `.golangci.yml`、`tests/depguard_config_test.go`、`tests/architecture_docs_test.go` 及**完整phase2 inventory**零diff。旧infra/Gemini永久deny、provider历史计数/ceilings、Gemini legacy目标及零当前消费者核对保持；不删目标或制造missing-skip。当前无Gemini CURRENT实际消费者行可删，不借本轮改历史计数。
8. §5的9旧文档只前置三组准确替代及新设计链接，原全文后缀不变；project-boundaries只去当前foundation owners中的Gemini一行。`internal/integration/README.md`仅将Owned external adapters的“Gemini and GRSAI image providers”改为“GRSAI image providers”，并加准确Gemini历史退休说明/当前Google文本owner，原其它adapter及边界义务保持；原architecture_docs_test对Gemini字样的要求继续由明确历史说明满足，不修改检查器或假称现役Gemini。README/register保留最终main完整blob后缀；不改政策/Must、历史PASS/数字/勾选。本批原检查不在上述允许列表者零改；真实CI若暴露遗漏，先查同根sibling并提出精确最小补丁，再取用户例外，不能自行把本节当删检查授权。
9. #301每次更新前新读正文，仅改当前段，从“以下保留 #726 已合并批次与历史证据，不扩大本轮权限。”起逐字保留，正文≤65536字符且OPEN。准确设计/实现HEAD、CI、review、批准留主要PR。

## 7. 评审、用户确认和停止点

- 当前只设计三文档 → 适用原自检/准确设计HEAD原CI → Independent Architecture Review → 用户确认本§1–§7/§6.1 → Issue/PR明确IMPLEMENTATION_READY/Ready并读回，才准同一PR生产实施。设计未通过/用户未确认前为SOURCE_NOT_STARTED/NOT_READY。
- 一个Writer、一个主要PR，三组35路径完整闭包不拆小批。起始已同步main，最终候选前再核对实际main组合，不为始终最新反复merge；未变证据可明确复用，新增组合风险另验。
- 适用架构复核最多两轮，finding先按AGENTS分类；只有当前Must的真实BLOCKER重开架构。源码形成后一次最终交付复核，修复只查实际增量。设计review不替代源码review，另一AI或CI不等于用户验收。
- GitHub复核启动失败至多一次重试；**未经用户本轮授权不默认新增AI互验线程**。#953的旧设计/源码子代理授权不扩用本设计或未来实现。不新增实现者、runner或治理平台，不重复触发运行中的GitHub评审。
- 准确源码HEAD原CI整run及最终复核满足后另交用户授权正常expected-head合并，不绕过/强推/改保护。合并后只认实际mergeSHA的push/main整runcompleted/SUCCESS，设计CI/PRCI不替代。
- #301保持OPEN。workflow/完整inventory/依赖/保护、既有实例/数据/凭据/目录/进程不动；保留当前能力、独立SHEIN登录与Store同步，不新增内部试用前置。真实环境操作/部署/关单另授权。
